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
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/raulc0399/cpc/internal/database"
)

// Azure regions list - all available regions
var azureRegions = []string{
	"eastus", "eastus2", "southcentralus", "westus2", "westus3", "australiaeast",
	"southeastasia", "northeurope", "swedencentral", "uksouth", "westeurope",
	"centralus", "southafricanorth", "centralindia", "eastasia", "japaneast",
	"koreacentral", "canadacentral", "francecentral", "germanywestcentral",
	"norwayeast", "switzerlandnorth", "uaenorth", "brazilsouth", "eastus2euap",
	"qatarcentral", "centralusstage", "eastusstage", "eastus2stage", "northcentralusstage",
	"southcentralusstage", "westusstage", "westus2stage", "asia", "asiapacific",
	"australia", "brazil", "canada", "europe", "france", "germany", "global",
	"india", "japan", "korea", "norway", "southafrica", "switzerland", "uae",
	"uk", "unitedstates", "unitedstateseuap", "eastasiastage", "southeastasiastage",
	"northcentralus", "westus", "jioindiawest", "centraluseuap", "westcentralus",
	"southafricawest", "australiacentral", "australiacentral2", "australiasoutheast",
	"japanwest", "jioindiacentral", "koreasouth", "southindia", "westindia",
	"canadaeast", "francesouth", "germanynorth", "norwaywest", "switzerlandwest",
	"ukwest", "uaecentral", "brazilsoutheast",
}

// AzureAPIResponse represents the Azure pricing API response
type AzureAPIResponse struct {
	BillingCurrency    string                   `json:"BillingCurrency"`
	CustomerEntityID   string                   `json:"CustomerEntityId"`
	CustomerEntityType string                   `json:"CustomerEntityType"`
	Items              []map[string]interface{} `json:"Items"`
	NextPageLink       string                   `json:"NextPageLink"`
	Count              int                      `json:"Count"`
}

// Progress tracking structure
type ProgressTracker struct {
	mu                sync.RWMutex
	totalRegions      int
	completedRegions  int
	failedRegions     int
	currentRegions    map[int]string
	startTime         time.Time
}

func (p *ProgressTracker) start(totalRegions int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalRegions = totalRegions
	p.completedRegions = 0
	p.failedRegions = 0
	p.currentRegions = make(map[int]string)
	p.startTime = time.Now()
}

func (p *ProgressTracker) setWorking(workerID int, region string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentRegions[workerID] = region
}

func (p *ProgressTracker) complete(workerID int, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.currentRegions, workerID)
	if success {
		p.completedRegions++
	} else {
		p.failedRegions++
	}
}

func (p *ProgressTracker) getStatus() (int, int, int, map[int]string, time.Duration) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	elapsed := time.Since(p.startTime)
	currentRegionsCopy := make(map[int]string)
	for k, v := range p.currentRegions {
		currentRegionsCopy[k] = v
	}
	return p.completedRegions, p.failedRegions, p.totalRegions, currentRegionsCopy, elapsed
}

func main() {
	log.Printf("🌍 Starting Azure all-regions data collection")

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

	// Get concurrency limit from args or env (default 3)
	concurrency := 3
	if len(os.Args) > 1 {
		if c := os.Args[1]; c != "" {
			if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
				concurrency = parsed
			}
		}
	}

	log.Printf("📊 Configuration:")
	log.Printf("   - Total regions: %d", len(azureRegions))
	log.Printf("   - Concurrency: %d workers", concurrency)
	log.Printf("   - Estimated time: %d-%d minutes", len(azureRegions)/concurrency, len(azureRegions)*2/concurrency)

	// Initialize progress tracker
	var progress ProgressTracker
	progress.start(len(azureRegions))

	// Start progress reporter goroutine
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				completed, failed, total, working, elapsed := progress.getStatus()
				remaining := total - completed - failed
				
				log.Printf("📈 PROGRESS: %d/%d completed, %d failed, %d remaining (%.1f%%) - Elapsed: %v", 
					completed, total, failed, remaining, 
					float64(completed+failed)/float64(total)*100, 
					elapsed.Truncate(time.Second))
				
				if len(working) > 0 {
					log.Printf("🔄 Currently processing:")
					for workerID, region := range working {
						log.Printf("   Worker %d: %s", workerID, region)
					}
				}
				
				if completed+failed >= total {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Create worker pool
	var wg sync.WaitGroup
	regionChan := make(chan string, len(azureRegions))
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for region := range regionChan {
				progress.setWorking(workerID, region)
				log.Printf("🚀 Worker %d: Starting region %s", workerID, region)
				
				err := collectRegionData(dbHandler, region, workerID)
				success := err == nil
				
				if success {
					log.Printf("✅ Worker %d: Successfully collected %s", workerID, region)
				} else {
					log.Printf("❌ Worker %d: Failed to collect %s: %v", workerID, region, err)
				}
				
				progress.complete(workerID, success)
				
				// Small delay between regions to be nice to the API
				time.Sleep(2 * time.Second)
			}
		}(i)
	}

	// Queue all regions
	for _, region := range azureRegions {
		regionChan <- region
	}
	close(regionChan)

	// Wait for all workers to complete
	wg.Wait()
	done <- true

	// Final summary
	completed, failed, total, _, elapsed := progress.getStatus()
	log.Printf("🎉 ALL REGIONS COLLECTION COMPLETED!")
	log.Printf("📊 Final Summary:")
	log.Printf("   - Total regions: %d", total)
	log.Printf("   - Successfully collected: %d", completed)
	log.Printf("   - Failed: %d", failed)
	log.Printf("   - Success rate: %.1f%%", float64(completed)/float64(total)*100)
	log.Printf("   - Total time: %v", elapsed.Truncate(time.Second))
	
	if failed > 0 {
		log.Printf("⚠️  Some regions failed - check logs above for details")
	}
}

func collectRegionData(db *database.DB, region string, workerID int) error {
	// Start collection
	collectionID, err := db.StartAzureRawCollection(region)
	if err != nil {
		return fmt.Errorf("failed to start collection: %w", err)
	}

	log.Printf("🆔 [Worker %d] Started collection %s for region %s", workerID, collectionID[:8], region)

	// Collect data
	totalItems, err := collectAzureData(db, collectionID, region, workerID)
	if err != nil {
		log.Printf("❌ [Worker %d] Collection failed for %s: %v", workerID, region, err)
		if failErr := db.FailAzureRawCollection(collectionID, err.Error()); failErr != nil {
			log.Printf("Failed to mark collection as failed: %v", failErr)
		}
		return err
	}

	// Complete collection
	err = db.CompleteAzureRawCollection(collectionID, totalItems)
	if err != nil {
		return fmt.Errorf("failed to complete collection: %w", err)
	}

	log.Printf("✅ [Worker %d] Collection %s completed for %s: %d items", workerID, collectionID[:8], region, totalItems)
	return nil
}

func collectAzureData(db *database.DB, collectionID string, region string, workerID int) (int, error) {
	baseURL := "https://prices.azure.com/api/retail/prices"
	
	// Build filter for the specific region
	filter := fmt.Sprintf("armRegionName eq '%s'", region)
	
	totalItems := 0
	nextLink := ""
	pageCount := 0
	estimatedTotalPages := 0
	
	log.Printf("🚀 [Worker %d] Starting data collection for region: %s", workerID, region)
	
	for {
		pageCount++
		
		// Update progress in database
		statusMsg := fmt.Sprintf("[Worker %d] Fetching page %d for region %s", workerID, pageCount, region)
		err := db.UpdateAzureRawCollectionProgress(collectionID, pageCount, estimatedTotalPages, totalItems, statusMsg)
		if err != nil {
			log.Printf("⚠️  [Worker %d] Failed to update progress: %v", workerID, err)
		}
		
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
			if retries < 2 {
				log.Printf("⚠️  [Worker %d] API request failed for %s (retry %d), retrying...", workerID, region, retries+1)
				time.Sleep(time.Duration(retries+1) * time.Second)
			}
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
		
		// Store raw data in database
		if len(apiResp.Items) > 0 {
			err = db.BulkInsertAzureRawPricing(collectionID, region, apiResp.Items)
			if err != nil {
				return totalItems, fmt.Errorf("failed to store data: %w", err)
			}
		}
		
		totalItems += len(apiResp.Items)
		
		// Estimate total pages based on first page
		if pageCount == 1 && len(apiResp.Items) > 0 {
			estimatedTotalPages = 10 // Rough estimate
		}
		
		// Update progress after successful page
		statusMsg = fmt.Sprintf("[Worker %d] Completed page %d for region %s (%d items)", workerID, pageCount, region, totalItems)
		db.UpdateAzureRawCollectionProgress(collectionID, pageCount, estimatedTotalPages, totalItems, statusMsg)
		
		// Reduced logging frequency for all-regions collection
		if pageCount%3 == 1 || len(apiResp.Items) == 0 || apiResp.NextPageLink == "" {
			log.Printf("📊 [Worker %d] %s: Page %d, Items: %d (Total: %d)", workerID, region, pageCount, len(apiResp.Items), totalItems)
		}
		
		// Check if there's more data
		if apiResp.NextPageLink == "" || len(apiResp.Items) == 0 {
			log.Printf("✅ [Worker %d] %s: Collection complete - %d pages, %d items", workerID, region, pageCount, totalItems)
			break
		}
		
		nextLink = apiResp.NextPageLink
		
		// Add delay between pages (shorter for concurrent collection)
		time.Sleep(200 * time.Millisecond)
		
		// Safety break to avoid infinite loops
		if pageCount > 20 {
			log.Printf("⚠️  [Worker %d] %s: Reached page limit (%d), stopping collection", workerID, region, pageCount)
			break
		}
	}
	
	return totalItems, nil
}