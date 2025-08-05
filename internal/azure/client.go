package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client provides Azure pricing API access
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Azure pricing API client
func NewClient() *Client {
	return &Client{
		baseURL: "https://prices.azure.com/api/retail/prices",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QueryPricing queries Azure pricing API with filters
func (c *Client) QueryPricing(filter string, maxResults int) ([]PricingItem, error) {
	params := url.Values{}
	params.Add("$filter", filter)
	if maxResults > 0 {
		params.Add("$top", fmt.Sprintf("%d", maxResults))
	}

	fullURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AzureAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to standardized format
	items := make([]PricingItem, len(result.Items))
	for i, rawItem := range result.Items {
		items[i] = convertRawToPricingItem(rawItem)
	}

	return items, nil
}

// QueryPricingWithPagination queries with full pagination support
func (c *Client) QueryPricingWithPagination(filter string, maxItems int) ([]PricingItem, error) {
	var allItems []PricingItem
	nextURL := c.buildURL(filter, 1000)

	for nextURL != "" && (maxItems == 0 || len(allItems) < maxItems) {
		resp, err := c.httpClient.Get(nextURL)
		if err != nil {
			return allItems, fmt.Errorf("failed to make request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if len(allItems) > 0 {
				return allItems, nil // Return partial results
			}
			return allItems, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result AzureAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return allItems, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(result.Items) == 0 {
			break
		}

		// Convert to standardized format
		for _, rawItem := range result.Items {
			if maxItems > 0 && len(allItems) >= maxItems {
				break
			}
			allItems = append(allItems, convertRawToPricingItem(rawItem))
		}

		nextURL = result.NextPageLink
		
		// Rate limiting
		if nextURL != "" {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return allItems, nil
}

// QueryRawWithPagination returns raw API response for database storage
func (c *Client) QueryRawWithPagination(filter string, maxItems int) ([]map[string]interface{}, error) {
	var allItems []map[string]interface{}
	nextURL := c.buildURL(filter, 1000)

	for nextURL != "" && (maxItems == 0 || len(allItems) < maxItems) {
		resp, err := c.httpClient.Get(nextURL)
		if err != nil {
			return allItems, fmt.Errorf("failed to make request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if len(allItems) > 0 {
				return allItems, nil // Return partial results
			}
			return allItems, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var result AzureAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return allItems, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(result.Items) == 0 {
			break
		}

		// Store raw items
		for _, rawItem := range result.Items {
			if maxItems > 0 && len(allItems) >= maxItems {
				break
			}
			allItems = append(allItems, rawItem)
		}

		nextURL = result.NextPageLink
		
		// Rate limiting
		if nextURL != "" {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return allItems, nil
}

func (c *Client) buildURL(filter string, pageSize int) string {
	params := url.Values{}
	params.Add("$filter", filter)
	params.Add("$top", fmt.Sprintf("%d", pageSize))
	return fmt.Sprintf("%s?%s", c.baseURL, params.Encode())
}

func convertRawToPricingItem(raw map[string]interface{}) PricingItem {
	item := PricingItem{}
	
	if v, ok := raw["currencyCode"].(string); ok {
		item.CurrencyCode = v
	}
	if v, ok := raw["tierMinimumUnits"].(float64); ok {
		item.TierMinimumUnits = v
	}
	if v, ok := raw["retailPrice"].(float64); ok {
		item.RetailPrice = v
	}
	if v, ok := raw["unitPrice"].(float64); ok {
		item.UnitPrice = v
	}
	if v, ok := raw["armRegionName"].(string); ok {
		item.ArmRegionName = v
	}
	if v, ok := raw["location"].(string); ok {
		item.Location = v
	}
	if v, ok := raw["effectiveStartDate"].(string); ok {
		item.EffectiveStartDate = v
	}
	if v, ok := raw["meterId"].(string); ok {
		item.MeterID = v
	}
	if v, ok := raw["meterName"].(string); ok {
		item.MeterName = v
	}
	if v, ok := raw["productId"].(string); ok {
		item.ProductID = v
	}
	if v, ok := raw["skuId"].(string); ok {
		item.SkuID = v
	}
	if v, ok := raw["productName"].(string); ok {
		item.ProductName = v
	}
	if v, ok := raw["skuName"].(string); ok {
		item.SkuName = v
	}
	if v, ok := raw["serviceName"].(string); ok {
		item.ServiceName = v
	}
	if v, ok := raw["serviceId"].(string); ok {
		item.ServiceID = v
	}
	if v, ok := raw["serviceFamily"].(string); ok {
		item.ServiceFamily = v
	}
	if v, ok := raw["unitOfMeasure"].(string); ok {
		item.UnitOfMeasure = v
	}
	if v, ok := raw["type"].(string); ok {
		item.Type = v
	}
	if v, ok := raw["isPrimaryMeterRegion"].(bool); ok {
		item.IsPrimaryMeterRegion = v
	}
	if v, ok := raw["armSkuName"].(string); ok {
		item.ArmSkuName = v
	}
	if v, ok := raw["reservationTerm"].(string); ok {
		item.ReservationTerm = v
	}
	
	return item
}