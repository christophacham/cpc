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

// AzurePricingResponse represents the API response
type AzurePricingResponse struct {
	BillingCurrency    string              `json:"BillingCurrency"`
	CustomerEntityID   string              `json:"CustomerEntityId"`
	CustomerEntityType string              `json:"CustomerEntityType"`
	Items              []AzurePricingItem  `json:"Items"`
	NextPageLink       string              `json:"NextPageLink"`
	Count              int                 `json:"Count"`
}

// AzurePricingItem represents a single pricing item from API
type AzurePricingItem struct {
	CurrencyCode         string  `json:"currencyCode"`
	TierMinimumUnits     float64 `json:"tierMinimumUnits"`
	RetailPrice          float64 `json:"retailPrice"`
	UnitPrice            float64 `json:"unitPrice"`
	ArmRegionName        string  `json:"armRegionName"`
	Location             string  `json:"location"`
	EffectiveStartDate   string  `json:"effectiveStartDate"`
	MeterID              string  `json:"meterId"`
	MeterName            string  `json:"meterName"`
	ProductID            string  `json:"productId"`
	SkuID                string  `json:"skuId"`
	ProductName          string  `json:"productName"`
	SkuName              string  `json:"skuName"`
	ServiceName          string  `json:"serviceName"`
	ServiceID            string  `json:"serviceId"`
	ServiceFamily        string  `json:"serviceFamily"`
	UnitOfMeasure        string  `json:"unitOfMeasure"`
	Type                 string  `json:"type"`
	IsPrimaryMeterRegion bool    `json:"isPrimaryMeterRegion"`
	ArmSkuName           string  `json:"armSkuName"`
	ReservationTerm      string  `json:"reservationTerm,omitempty"`
}

func main() {
	fmt.Println("Azure Database Collector - Fetching and Storing Pricing Data")
	fmt.Println("===========================================================")
	fmt.Println()

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

	fmt.Println("✅ Connected to database")

	// Create database handler
	dbHandler := database.New(db)

	// Start collection
	version, err := dbHandler.StartAzureCollection()
	if err != nil {
		log.Fatalf("Failed to start collection: %v", err)
	}

	fmt.Printf("🚀 Started collection version %d\n\n", version)

	// Test with a small region first
	region := "eastus"
	fmt.Printf("Collecting pricing data from %s region...\n", region)

	startTime := time.Now()
	items, err := collectRegionPricing(region, 1000) // Limit to 1000 items for testing
	if err != nil {
		dbHandler.FailAzureCollection(version, err.Error())
		log.Fatalf("Failed to collect pricing: %v", err)
	}

	fmt.Printf("📦 Collected %d items in %.2f seconds\n", len(items), time.Since(startTime).Seconds())

	// Convert to database format
	fmt.Println("🔄 Converting data for database insertion...")
	dbItems := convertToDBFormat(items, version)

	// Insert into database
	fmt.Println("💾 Inserting data into database...")
	insertStart := time.Now()
	err = dbHandler.BulkInsertAzurePricing(dbItems)
	if err != nil {
		dbHandler.FailAzureCollection(version, err.Error())
		log.Fatalf("Failed to insert pricing data: %v", err)
	}

	insertDuration := time.Since(insertStart)
	fmt.Printf("✅ Inserted %d records in %.2f seconds\n", len(dbItems), insertDuration.Seconds())

	// Complete collection
	err = dbHandler.CompleteAzureCollection(version, len(dbItems), []string{region})
	if err != nil {
		log.Printf("Warning: Failed to mark collection complete: %v", err)
	}

	totalDuration := time.Since(startTime)
	fmt.Printf("\n🎉 Collection completed successfully!\n")
	fmt.Printf("Total time: %.2f seconds\n", totalDuration.Seconds())
	fmt.Printf("Collection version: %d\n", version)

	// Show some statistics
	showStatistics(dbItems)
}

// collectRegionPricing collects pricing data from a specific region
func collectRegionPricing(region string, maxItems int) ([]AzurePricingItem, error) {
	var allItems []AzurePricingItem
	pageSize := 1000
	pageCount := 0

	baseURL := "https://prices.azure.com/api/retail/prices"
	params := url.Values{}
	params.Add("$filter", fmt.Sprintf("armRegionName eq '%s' and priceType eq 'Consumption'", region))
	params.Add("$top", fmt.Sprintf("%d", pageSize))

	nextURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for nextURL != "" && len(allItems) < maxItems {
		pageCount++
		fmt.Printf("\r  Fetching page %d... (items: %d)", pageCount, len(allItems))

		resp, err := client.Get(nextURL)
		if err != nil {
			return allItems, fmt.Errorf("failed to make request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if len(allItems) > 0 {
				fmt.Printf("\n⚠️  Stopping due to API error after %d items\n", len(allItems))
				return allItems, nil
			}
			return allItems, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result AzurePricingResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return allItems, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(result.Items) == 0 {
			break // No more items
		}

		allItems = append(allItems, result.Items...)

		// Check if we have enough items
		if len(allItems) >= maxItems {
			allItems = allItems[:maxItems]
			break
		}

		nextURL = result.NextPageLink

		// Be nice to the API
		if nextURL != "" {
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Println() // New line after progress
	return allItems, nil
}

// convertToDBFormat converts API items to database format
func convertToDBFormat(items []AzurePricingItem, version int) []database.AzurePricingInsert {
	var dbItems []database.AzurePricingInsert

	for _, item := range items {
		// Parse effective date
		effectiveDate, err := time.Parse("2006-01-02T15:04:05Z", item.EffectiveStartDate)
		if err != nil {
			// Try alternative format
			effectiveDate, err = time.Parse("2006-01-02T00:00:00Z", item.EffectiveStartDate)
			if err != nil {
				log.Printf("Warning: Could not parse date %s, using current date", item.EffectiveStartDate)
				effectiveDate = time.Now().Truncate(24 * time.Hour)
			}
		}

		dbItem := database.AzurePricingInsert{
			ServiceName:          item.ServiceName,
			ServiceFamily:        item.ServiceFamily,
			ProductName:          item.ProductName,
			ProductID:            item.ProductID,
			SKUName:              item.SkuName,
			SKUID:                item.SkuID,
			ARMSKUName:           item.ArmSkuName,
			ARMRegionName:        item.ArmRegionName,
			DisplayName:          item.Location,
			MeterID:              item.MeterID,
			MeterName:            item.MeterName,
			RetailPrice:          item.RetailPrice,
			UnitPrice:            item.UnitPrice,
			TierMinimumUnits:     item.TierMinimumUnits,
			CurrencyCode:         item.CurrencyCode,
			UnitOfMeasure:        item.UnitOfMeasure,
			PriceType:            item.Type,
			ReservationTerm:      item.ReservationTerm,
			EffectiveStartDate:   effectiveDate,
			IsPrimaryMeterRegion: item.IsPrimaryMeterRegion,
			CollectionVersion:    version,
		}

		dbItems = append(dbItems, dbItem)
	}

	return dbItems
}

// showStatistics displays collection statistics
func showStatistics(items []database.AzurePricingInsert) {
	fmt.Println("\n📊 Collection Statistics:")
	fmt.Println("========================")

	// Count unique services
	services := make(map[string]bool)
	products := make(map[string]bool)
	skus := make(map[string]bool)
	regions := make(map[string]bool)

	for _, item := range items {
		services[item.ServiceName] = true
		products[item.ProductName] = true
		skus[item.SKUName] = true
		regions[item.ARMRegionName] = true
	}

	fmt.Printf("Unique services: %d\n", len(services))
	fmt.Printf("Unique products: %d\n", len(products))
	fmt.Printf("Unique SKUs: %d\n", len(skus))
	fmt.Printf("Unique regions: %d\n", len(regions))

	// Show top services
	serviceCounts := make(map[string]int)
	for _, item := range items {
		serviceCounts[item.ServiceName]++
	}

	fmt.Println("\nTop 5 Services:")
	topServices := getTopN(serviceCounts, 5)
	for i, service := range topServices {
		fmt.Printf("%d. %-30s: %d items\n", i+1, service.Key, service.Value)
	}
}

// Helper functions
type KeyValue struct {
	Key   string
	Value int
}

func getTopN(m map[string]int, n int) []KeyValue {
	var items []KeyValue
	for k, v := range m {
		items = append(items, KeyValue{k, v})
	}

	// Simple sort by value
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Value > items[i].Value {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	if n > len(items) {
		n = len(items)
	}
	return items[:n]
}