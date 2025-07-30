package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AzurePricingResponse represents the API response
type AzurePricingResponse struct {
	BillingCurrency string      `json:"BillingCurrency"`
	CustomerEntityID string     `json:"CustomerEntityId"`
	CustomerEntityType string   `json:"CustomerEntityType"`
	Items           []PriceItem `json:"Items"`
	NextPageLink    string      `json:"NextPageLink"`
	Count           int         `json:"Count"`
}

// PriceItem represents a single pricing item
type PriceItem struct {
	CurrencyCode         string   `json:"currencyCode"`
	TierMinimumUnits     float64  `json:"tierMinimumUnits"`
	RetailPrice          float64  `json:"retailPrice"`
	UnitPrice            float64  `json:"unitPrice"`
	ArmRegionName        string   `json:"armRegionName"`
	Location             string   `json:"location"`
	EffectiveStartDate   string   `json:"effectiveStartDate"`
	MeterID              string   `json:"meterId"`
	MeterName            string   `json:"meterName"`
	ProductID            string   `json:"productId"`
	SkuID                string   `json:"skuId"`
	ProductName          string   `json:"productName"`
	SkuName              string   `json:"skuName"`
	ServiceName          string   `json:"serviceName"`
	ServiceID            string   `json:"serviceId"`
	ServiceFamily        string   `json:"serviceFamily"`
	UnitOfMeasure        string   `json:"unitOfMeasure"`
	Type                 string   `json:"type"`
	IsPrimaryMeterRegion bool     `json:"isPrimaryMeterRegion"`
	ArmSkuName           string   `json:"armSkuName"`
	ReservationTerm      string   `json:"reservationTerm,omitempty"`
}

// ServiceExample represents a service to query for each category
type ServiceExample struct {
	Category    string
	ServiceName string
	Filter      string
}

func main() {
	fmt.Println("Azure Pricing API Explorer - Focused Query")
	fmt.Println("==========================================")
	fmt.Println()

	// Define one specific service example for each category
	services := []ServiceExample{
		{
			Category:    "General",
			ServiceName: "Virtual Network (IP Addresses)",
			Filter:      "serviceName eq 'Virtual Network' and armRegionName eq 'eastus' and productName eq 'IP Addresses' and skuName eq 'Basic' and priceType eq 'Consumption'",
		},
		{
			Category:    "Networking",
			ServiceName: "Content Delivery Network",
			Filter:      "serviceName eq 'Content Delivery Network' and armRegionName eq 'eastus' and priceType eq 'Consumption'",
		},
		{
			Category:    "Compute & Web",
			ServiceName: "Virtual Machines (B2s)",
			Filter:      "serviceName eq 'Virtual Machines' and armRegionName eq 'eastus' and armSkuName eq 'Standard_B2s' and priceType eq 'Consumption' and contains(productName, 'Windows') eq false",
		},
		{
			Category:    "Containers",
			ServiceName: "Container Instances",
			Filter:      "serviceName eq 'Container Instances' and armRegionName eq 'eastus' and skuName eq 'Standard' and meterName eq 'Standard vCPU Duration' and priceType eq 'Consumption'",
		},
		{
			Category:    "Databases",
			ServiceName: "Azure SQL Database (Basic)",
			Filter:      "serviceName eq 'SQL Database' and armRegionName eq 'eastus' and skuName eq 'Basic' and priceType eq 'Consumption'",
		},
		{
			Category:    "Storage",
			ServiceName: "Storage (Blob Hot)",
			Filter:      "serviceName eq 'Storage' and armRegionName eq 'eastus' and skuName eq 'Hot LRS' and meterName eq 'Hot LRS Data Stored' and priceType eq 'Consumption'",
		},
		{
			Category:    "AI & ML",
			ServiceName: "Azure OpenAI",
			Filter:      "serviceName eq 'Cognitive Services' and productName eq 'Azure OpenAI' and armRegionName eq 'eastus' and contains(meterName, 'gpt-3.5') and priceType eq 'Consumption'",
		},
		{
			Category:    "Analytics & IoT",
			ServiceName: "Event Hubs (Basic)",
			Filter:      "serviceName eq 'Event Hubs' and armRegionName eq 'eastus' and skuName eq 'Basic' and priceType eq 'Consumption'",
		},
		{
			Category:    "Virtual Desktop",
			ServiceName: "Windows 365",
			Filter:      "serviceName eq 'Windows 365' and priceType eq 'Consumption'",
		},
		{
			Category:    "Dev Tools",
			ServiceName: "Azure DevOps",
			Filter:      "serviceName eq 'Azure DevOps' and productName eq 'Azure Artifacts' and priceType eq 'Consumption'",
		},
		{
			Category:    "Integration",
			ServiceName: "Logic Apps (Standard)",
			Filter:      "serviceName eq 'Logic Apps' and armRegionName eq 'eastus' and skuName eq 'Standard' and meterName eq 'Standard Actions' and priceType eq 'Consumption'",
		},
		{
			Category:    "Migration",
			ServiceName: "Azure Database Migration",
			Filter:      "serviceName eq 'Azure Database Migration Service' and armRegionName eq 'eastus' and priceType eq 'Consumption'",
		},
		{
			Category:    "Management",
			ServiceName: "Azure Monitor (Logs)",
			Filter:      "serviceName eq 'Azure Monitor' and productName eq 'Azure Monitor Logs' and armRegionName eq 'eastus' and priceType eq 'Consumption'",
		},
	}

	// Query each service
	fmt.Println("Fetching one representative service from each category...")
	fmt.Println()
	
	for _, service := range services {
		fmt.Printf("Category: %s\n", service.Category)
		fmt.Printf("Service: %s\n", service.ServiceName)
		fmt.Println(strings.Repeat("-", 60))
		
		items, err := queryAzurePricing(service.Filter, 1) // Get only 1 item
		if err != nil {
			log.Printf("Error querying %s: %v\n", service.ServiceName, err)
			fmt.Println()
			continue
		}

		if len(items) == 0 {
			fmt.Printf("No pricing found. Trying broader search...\n")
			// Try a simpler filter
			simpleFilter := fmt.Sprintf("serviceName eq '%s' and priceType eq 'Consumption'", strings.Split(service.Filter, "'")[1])
			items, err = queryAzurePricing(simpleFilter, 1)
			if err != nil || len(items) == 0 {
				fmt.Printf("Still no pricing found for %s\n", service.ServiceName)
				fmt.Println()
				continue
			}
		}

		// Display item details
		item := items[0]
		fmt.Printf("  Service Name: %s\n", item.ServiceName)
		fmt.Printf("  Product: %s\n", item.ProductName)
		fmt.Printf("  SKU: %s\n", item.SkuName)
		if item.ArmSkuName != "" {
			fmt.Printf("  ARM SKU: %s\n", item.ArmSkuName)
		}
		fmt.Printf("  Meter: %s\n", item.MeterName)
		fmt.Printf("  Price: $%.6f per %s\n", item.RetailPrice, item.UnitOfMeasure)
		fmt.Printf("  Region: %s (%s)\n", item.Location, item.ArmRegionName)
		fmt.Printf("  Service Family: %s\n", item.ServiceFamily)
		fmt.Printf("  Type: %s\n", item.Type)
		fmt.Printf("  Effective Date: %s\n", item.EffectiveStartDate)
		fmt.Println()
	}

	// Summary
	fmt.Println("\nKey Observations:")
	fmt.Println("- Azure Retail Pricing API provides detailed pricing without authentication")
	fmt.Println("- Prices vary by region, SKU, and consumption type")
	fmt.Println("- Service names and filters need to be precise for accurate results")
	fmt.Println("- The API returns prices in various units (per hour, per GB, per month, etc.)")
}

func queryAzurePricing(filter string, maxResults int) ([]PriceItem, error) {
	baseURL := "https://prices.azure.com/api/retail/prices"
	
	// Build query parameters
	params := url.Values{}
	params.Add("$filter", filter)
	params.Add("$top", fmt.Sprintf("%d", maxResults))
	
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Make request
	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var result AzurePricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Items, nil
}