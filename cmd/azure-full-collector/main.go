package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

// ServiceSummary summarizes pricing for a service
type ServiceSummary struct {
	ServiceName    string
	ServiceFamily  string
	TotalItems     int
	UniqueProducts map[string]int
	UniqueSKUs     map[string]int
	UniqueMeters   map[string]int
	PriceRange     struct {
		Min float64
		Max float64
	}
	SampleItems []PriceItem
}

func main() {
	fmt.Println("Azure Full Pricing Collector - Complete Region Analysis")
	fmt.Println("======================================================")
	fmt.Println()

	region := "eastus"
	fmt.Printf("Collecting ALL pricing data from %s region...\n\n", region)
	
	startTime := time.Now()
	
	// Collect all data from one region
	filter := fmt.Sprintf("armRegionName eq '%s' and priceType eq 'Consumption'", region)
	allItems, err := collectAllPricing(filter)
	
	if err != nil {
		log.Fatalf("Error collecting pricing: %v", err)
	}
	
	duration := time.Since(startTime)
	
	fmt.Printf("\nCollection completed in %.2f seconds\n", duration.Seconds())
	fmt.Printf("Total items collected: %d\n\n", len(allItems))
	
	// Analyze by service
	serviceSummaries := analyzeByService(allItems)
	
	// Display service summaries
	fmt.Println("Service Summary:")
	fmt.Println("================")
	fmt.Printf("Total unique services: %d\n\n", len(serviceSummaries))
	
	// Show all services with counts
	fmt.Println("All Services (sorted by item count):")
	fmt.Println("------------------------------------")
	for i, summary := range serviceSummaries {
		fmt.Printf("%3d. %-50s [%s] - %d items\n", 
			i+1, summary.ServiceName, summary.ServiceFamily, summary.TotalItems)
	}
	
	// Export data structure for analysis
	fmt.Println("\nExporting sample data to azure_pricing_sample.json...")
	exportSampleData(allItems[:min(1000, len(allItems))])
	
	// Show detailed breakdown of a few services
	fmt.Println("\nDetailed Service Analysis (top 5):")
	fmt.Println("==================================")
	for i := 0; i < min(5, len(serviceSummaries)); i++ {
		summary := serviceSummaries[i]
		fmt.Printf("\n%d. %s\n", i+1, summary.ServiceName)
		fmt.Printf("   Service Family: %s\n", summary.ServiceFamily)
		fmt.Printf("   Total Items: %d\n", summary.TotalItems)
		fmt.Printf("   Unique Products: %d\n", len(summary.UniqueProducts))
		fmt.Printf("   Unique SKUs: %d\n", len(summary.UniqueSKUs))
		fmt.Printf("   Price Range: $%.6f - $%.6f\n", summary.PriceRange.Min, summary.PriceRange.Max)
		
		// Show top products
		fmt.Println("   Top Products:")
		topProducts := getTopN(summary.UniqueProducts, 3)
		for _, prod := range topProducts {
			fmt.Printf("     - %s (%d items)\n", prod.Key, prod.Value)
		}
	}
}

// collectAllPricing collects all pricing data with pagination
func collectAllPricing(filter string) ([]PriceItem, error) {
	var allItems []PriceItem
	pageSize := 1000
	pageCount := 0
	maxPages := 10 // Limit for safety
	
	baseURL := "https://prices.azure.com/api/retail/prices"
	params := url.Values{}
	params.Add("$filter", filter)
	params.Add("$top", fmt.Sprintf("%d", pageSize))
	
	nextURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	for nextURL != "" && pageCount < maxPages {
		pageCount++
		fmt.Printf("\rFetching page %d... (items: %d)", pageCount, len(allItems))
		
		resp, err := client.Get(nextURL)
		if err != nil {
			return allItems, fmt.Errorf("failed to make request: %w", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			// If we get an error after collecting some data, return what we have
			if len(allItems) > 0 {
				fmt.Printf("\nStopping due to API error after %d items\n", len(allItems))
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
		nextURL = result.NextPageLink
		
		// Add small delay to be nice to the API
		if nextURL != "" {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	fmt.Println() // New line after progress
	return allItems, nil
}

// analyzeByService groups and analyzes items by service
func analyzeByService(items []PriceItem) []ServiceSummary {
	serviceMap := make(map[string]*ServiceSummary)
	
	for _, item := range items {
		if _, exists := serviceMap[item.ServiceName]; !exists {
			serviceMap[item.ServiceName] = &ServiceSummary{
				ServiceName:    item.ServiceName,
				ServiceFamily:  item.ServiceFamily,
				UniqueProducts: make(map[string]int),
				UniqueSKUs:     make(map[string]int),
				UniqueMeters:   make(map[string]int),
				PriceRange: struct {
					Min float64
					Max float64
				}{Min: 999999999, Max: 0},
			}
		}
		
		summary := serviceMap[item.ServiceName]
		summary.TotalItems++
		summary.UniqueProducts[item.ProductName]++
		summary.UniqueSKUs[item.SkuName]++
		summary.UniqueMeters[item.MeterName]++
		
		// Update price range
		if item.RetailPrice > 0 {
			if item.RetailPrice < summary.PriceRange.Min {
				summary.PriceRange.Min = item.RetailPrice
			}
			if item.RetailPrice > summary.PriceRange.Max {
				summary.PriceRange.Max = item.RetailPrice
			}
		}
		
		// Keep a few sample items
		if len(summary.SampleItems) < 5 {
			summary.SampleItems = append(summary.SampleItems, item)
		}
	}
	
	// Convert map to sorted slice
	var summaries []ServiceSummary
	for _, summary := range serviceMap {
		summaries = append(summaries, *summary)
	}
	
	// Sort by item count
	for i := 0; i < len(summaries); i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].TotalItems > summaries[i].TotalItems {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}
	
	return summaries
}

// exportSampleData exports sample data to JSON file
func exportSampleData(items []PriceItem) {
	file, err := os.Create("azure_pricing_sample.json")
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(items); err != nil {
		log.Printf("Error encoding JSON: %v", err)
		return
	}
	
	fmt.Println("Sample data exported successfully!")
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
	
	// Sort by value
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}