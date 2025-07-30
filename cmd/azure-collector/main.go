package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// AzurePricingResponse represents the API response
type AzurePricingResponse struct {
	BillingCurrency    string      `json:"BillingCurrency"`
	CustomerEntityID   string      `json:"CustomerEntityId"`
	CustomerEntityType string      `json:"CustomerEntityType"`
	Items              []PriceItem `json:"Items"`
	NextPageLink       string      `json:"NextPageLink"`
	Count              int         `json:"Count"`
}

// PriceItem represents a single pricing item
type PriceItem struct {
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

// CollectorStats tracks collection statistics
type CollectorStats struct {
	TotalItems      int
	UniqueServices  map[string]int
	UniqueRegions   map[string]int
	UniqueSKUs      map[string]int
	StartTime       time.Time
	Duration        time.Duration
}

func main() {
	fmt.Println("Azure Pricing Collector - Fetching All Regions")
	fmt.Println("==============================================")
	fmt.Println()

	// First, let's get a small sample to see what regions are available
	fmt.Println("Step 1: Discovering available regions...")
	regions := discoverRegions()
	fmt.Printf("Found %d unique regions\n", len(regions))
	
	// For now, let's just work with a few regions to test
	testRegions := []string{"eastus", "westus", "northeurope", "southeastasia"}
	
	fmt.Printf("\nStep 2: Collecting pricing data from %d test regions...\n", len(testRegions))
	allItems, stats := collectFromRegions(testRegions)
	
	// Display statistics
	fmt.Println("\nCollection Statistics:")
	fmt.Println("---------------------")
	fmt.Printf("Total items collected: %d\n", stats.TotalItems)
	fmt.Printf("Unique services: %d\n", len(stats.UniqueServices))
	fmt.Printf("Unique regions: %d\n", len(stats.UniqueRegions))
	fmt.Printf("Unique SKUs: %d\n", len(stats.UniqueSKUs))
	fmt.Printf("Collection time: %.2f seconds\n", stats.Duration.Seconds())
	
	// Show top services
	fmt.Println("\nTop 10 Services by Item Count:")
	topServices := getTopN(stats.UniqueServices, 10)
	for i, service := range topServices {
		fmt.Printf("%2d. %-40s: %d items\n", i+1, service.Key, service.Value)
	}
	
	// Show data shape analysis
	fmt.Println("\nData Shape Analysis:")
	analyzeDataShape(allItems[:min(100, len(allItems))])
	
	// Show unique units of measure
	fmt.Println("\nUnique Units of Measure:")
	units := getUniqueUnits(allItems)
	for unit := range units {
		fmt.Printf("- %s\n", unit)
	}
}

// discoverRegions fetches a sample to discover available regions
func discoverRegions() []string {
	regions := make(map[string]bool)
	
	// Query without region filter to get all regions
	filter := "priceType eq 'Consumption'"
	items, err := queryAzurePricing(filter, 1000)
	if err != nil {
		log.Printf("Error discovering regions: %v", err)
		return []string{}
	}
	
	for _, item := range items {
		if item.ArmRegionName != "" {
			regions[item.ArmRegionName] = true
		}
	}
	
	// Convert map to slice
	result := make([]string, 0, len(regions))
	for region := range regions {
		result = append(result, region)
	}
	
	return result
}

// collectFromRegions collects pricing data from specified regions
func collectFromRegions(regions []string) ([]PriceItem, CollectorStats) {
	stats := CollectorStats{
		UniqueServices: make(map[string]int),
		UniqueRegions:  make(map[string]int),
		UniqueSKUs:     make(map[string]int),
		StartTime:      time.Now(),
	}
	
	var allItems []PriceItem
	
	for _, region := range regions {
		fmt.Printf("\nCollecting from %s...", region)
		
		// Get all services from each region (limited for testing)
		filter := fmt.Sprintf("armRegionName eq '%s' and priceType eq 'Consumption'", region)
		items, err := queryAzurePricingWithPagination(filter, 500) // Limit to 500 items per region for testing
		
		if err != nil {
			log.Printf("Error collecting from %s: %v", region, err)
			continue
		}
		
		fmt.Printf(" found %d items", len(items))
		allItems = append(allItems, items...)
		
		// Update statistics
		for _, item := range items {
			stats.UniqueServices[item.ServiceName]++
			stats.UniqueRegions[item.ArmRegionName]++
			stats.UniqueSKUs[item.SkuName]++
		}
	}
	
	stats.TotalItems = len(allItems)
	stats.Duration = time.Since(stats.StartTime)
	
	return allItems, stats
}

// queryAzurePricing queries the Azure Pricing API
func queryAzurePricing(filter string, maxResults int) ([]PriceItem, error) {
	baseURL := "https://prices.azure.com/api/retail/prices"
	
	params := url.Values{}
	params.Add("$filter", filter)
	params.Add("$top", fmt.Sprintf("%d", maxResults))
	
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var result AzurePricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Items, nil
}

// queryAzurePricingWithPagination queries with pagination support
func queryAzurePricingWithPagination(filter string, maxItems int) ([]PriceItem, error) {
	var allItems []PriceItem
	pageSize := 100
	
	baseURL := "https://prices.azure.com/api/retail/prices"
	params := url.Values{}
	params.Add("$filter", filter)
	params.Add("$top", fmt.Sprintf("%d", pageSize))
	
	nextURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	for nextURL != "" && len(allItems) < maxItems {
		resp, err := client.Get(nextURL)
		if err != nil {
			return allItems, fmt.Errorf("failed to make request: %w", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return allItems, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}
		
		var result AzurePricingResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return allItems, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()
		
		allItems = append(allItems, result.Items...)
		
		// Check if we have enough items
		if len(allItems) >= maxItems {
			allItems = allItems[:maxItems]
			break
		}
		
		nextURL = result.NextPageLink
	}
	
	return allItems, nil
}

// analyzeDataShape analyzes the structure of pricing items
func analyzeDataShape(items []PriceItem) {
	if len(items) == 0 {
		fmt.Println("No items to analyze")
		return
	}
	
	fmt.Println("---------------------")
	fmt.Println("Sample item structure:")
	
	// Take first item as example
	item := items[0]
	
	fmt.Printf("\nService: %s\n", item.ServiceName)
	fmt.Printf("Product: %s\n", item.ProductName)
	fmt.Printf("SKU: %s (ARM: %s)\n", item.SkuName, item.ArmSkuName)
	fmt.Printf("Meter: %s\n", item.MeterName)
	fmt.Printf("Region: %s (%s)\n", item.Location, item.ArmRegionName)
	fmt.Printf("Price: %.6f %s per %s\n", item.RetailPrice, item.CurrencyCode, item.UnitOfMeasure)
	fmt.Printf("Type: %s\n", item.Type)
	fmt.Printf("Service Family: %s\n", item.ServiceFamily)
	
	// Analyze field usage
	fmt.Println("\nField Usage Analysis (sample of 100):")
	fieldUsage := analyzeFieldUsage(items)
	for field, count := range fieldUsage {
		percentage := float64(count) * 100 / float64(len(items))
		fmt.Printf("%-20s: %3.0f%% (%d/%d)\n", field, percentage, count, len(items))
	}
}

// analyzeFieldUsage counts non-empty fields
func analyzeFieldUsage(items []PriceItem) map[string]int {
	usage := make(map[string]int)
	
	for _, item := range items {
		if item.CurrencyCode != "" {
			usage["CurrencyCode"]++
		}
		if item.RetailPrice > 0 {
			usage["RetailPrice"]++
		}
		if item.ArmRegionName != "" {
			usage["ArmRegionName"]++
		}
		if item.Location != "" {
			usage["Location"]++
		}
		if item.ServiceName != "" {
			usage["ServiceName"]++
		}
		if item.ProductName != "" {
			usage["ProductName"]++
		}
		if item.SkuName != "" {
			usage["SkuName"]++
		}
		if item.ArmSkuName != "" {
			usage["ArmSkuName"]++
		}
		if item.MeterName != "" {
			usage["MeterName"]++
		}
		if item.UnitOfMeasure != "" {
			usage["UnitOfMeasure"]++
		}
		if item.ServiceFamily != "" {
			usage["ServiceFamily"]++
		}
		if item.Type != "" {
			usage["Type"]++
		}
		if item.ReservationTerm != "" {
			usage["ReservationTerm"]++
		}
	}
	
	return usage
}

// Helper types and functions
type KeyValue struct {
	Key   string
	Value int
}

func getTopN(m map[string]int, n int) []KeyValue {
	// Convert map to slice
	var items []KeyValue
	for k, v := range m {
		items = append(items, KeyValue{k, v})
	}
	
	// Sort by value
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Value > items[i].Value {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	
	// Return top N
	if n > len(items) {
		n = len(items)
	}
	return items[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getUniqueUnits(items []PriceItem) map[string]bool {
	units := make(map[string]bool)
	for _, item := range items {
		if item.UnitOfMeasure != "" {
			units[item.UnitOfMeasure] = true
		}
	}
	return units
}