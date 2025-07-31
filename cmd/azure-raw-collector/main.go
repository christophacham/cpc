package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/raulc0399/cpc/internal/database"
)

// AzureAPIResponse represents the Azure pricing API response
type AzureAPIResponse struct {
	BillingCurrency    string                   `json:"BillingCurrency"`
	CustomerEntityID   string                   `json:"CustomerEntityId"`
	CustomerEntityType string                   `json:"CustomerEntityType"`
	Items              []map[string]interface{} `json:"Items"`
	NextPageLink       string                   `json:"NextPageLink"`
	Count              int                      `json:"Count"`
}

func main() {
	// Get region from command line arg or default to eastus
	region := "eastus"
	if len(os.Args) > 1 {
		region = os.Args[1]
	}

	log.Printf("Starting Azure raw data collection for region: %s", region)

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cpc_user:cpc_password@localhost:5432/cpc_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	dbHandler := database.New(db)

	// Start collection
	collectionID, err := dbHandler.StartAzureRawCollection(region)
	if err != nil {
		log.Fatalf("Failed to start collection: %v", err)
	}

	log.Printf("Started collection: %s", collectionID)

	// Collect data
	totalItems, err := collectAzureData(dbHandler, collectionID, region)
	if err != nil {
		log.Printf("Collection failed: %v", err)
		if failErr := dbHandler.FailAzureRawCollection(collectionID, err.Error()); failErr != nil {
			log.Printf("Failed to mark collection as failed: %v", failErr)
		}
		return
	}

	// Complete collection
	err = dbHandler.CompleteAzureRawCollection(collectionID, totalItems)
	if err != nil {
		log.Printf("Failed to mark collection as completed: %v", err)
		return
	}

	log.Printf("✅ Collection completed successfully!")
	log.Printf("📊 Collection ID: %s", collectionID)
	log.Printf("📊 Region: %s", region)
	log.Printf("📊 Total items: %d", totalItems)
}

func collectAzureData(db *database.DB, collectionID string, region string) (int, error) {
	baseURL := "https://prices.azure.com/api/retail/prices"
	
	// Build filter for the specific region
	filter := fmt.Sprintf("armRegionName eq '%s'", region)
	
	totalItems := 0
	nextLink := ""
	pageCount := 0
	estimatedTotalPages := 0
	
	log.Printf("🚀 Starting collection for region: %s", region)
	
	for {
		pageCount++
		
		// Update progress in database
		statusMsg := fmt.Sprintf("Fetching page %d for region %s", pageCount, region)
		err := db.UpdateAzureRawCollectionProgress(collectionID, pageCount, estimatedTotalPages, totalItems, statusMsg)
		if err != nil {
			log.Printf("⚠️  Failed to update progress: %v", err)
		}
		
		log.Printf("📥 [%s] Fetching page %d (total items so far: %d)...", region, pageCount, totalItems)
		
		// Build URL
		var apiURL string
		if nextLink != "" {
			apiURL = nextLink
		} else {
			params := url.Values{}
			params.Add("$filter", filter)
			apiURL = baseURL + "?" + params.Encode()
		}
		
		// Make API request with retry
		var resp *http.Response
		for retries := 0; retries < 3; retries++ {
			resp, err = http.Get(apiURL)
			if err == nil && resp.StatusCode == 200 {
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			log.Printf("⚠️  [%s] API request failed (retry %d), retrying...", region, retries+1)
			time.Sleep(time.Duration(retries+1) * time.Second)
		}
		
		if err != nil {
			return totalItems, fmt.Errorf("failed to fetch data after retries: %w", err)
		}
		
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return totalItems, fmt.Errorf("API returned status %d", resp.StatusCode)
		}
		
		// Read response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return totalItems, fmt.Errorf("failed to read response: %w", err)
		}
		
		// Parse JSON
		var apiResp AzureAPIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return totalItems, fmt.Errorf("failed to parse JSON: %w", err)
		}
		
		log.Printf("📊 [%s] Page %d: %d items (total: %d)", region, pageCount, len(apiResp.Items), totalItems+len(apiResp.Items))
		
		// Store raw data in database
		if len(apiResp.Items) > 0 {
			err = db.BulkInsertAzureRawPricing(collectionID, region, apiResp.Items)
			if err != nil {
				return totalItems, fmt.Errorf("failed to store data: %w", err)
			}
		}
		
		totalItems += len(apiResp.Items)
		
		// Estimate total pages based on first page (rough estimate)
		if pageCount == 1 && len(apiResp.Items) > 0 {
			estimatedTotalPages = 10 // Rough estimate, Azure typically has 5-15 pages per region
		}
		
		// Update progress after successful page
		statusMsg = fmt.Sprintf("Completed page %d for region %s", pageCount, region)
		db.UpdateAzureRawCollectionProgress(collectionID, pageCount, estimatedTotalPages, totalItems, statusMsg)
		
		// Check if there's more data
		if apiResp.NextPageLink == "" || len(apiResp.Items) == 0 {
			log.Printf("✅ [%s] Collection complete - no more pages", region)
			break
		}
		
		nextLink = apiResp.NextPageLink
		
		// Add small delay to be nice to the API
		time.Sleep(100 * time.Millisecond)
		
		// Safety break to avoid infinite loops
		if pageCount > 20 {
			log.Printf("⚠️  [%s] Reached page limit (%d), stopping collection", region, pageCount)
			break
		}
	}
	
	log.Printf("🎉 [%s] Collection completed! Total items: %d, Pages: %d", region, totalItems, pageCount)
	return totalItems, nil
}