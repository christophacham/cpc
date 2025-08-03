package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Helper functions to query raw pricing data directly

func (c *AWSCompute) getInstancePriceFromRaw(ctx context.Context, instanceType string) (float64, error) {
	// Normalize region (us-east-1 -> US East (N. Virginia))
	regionMapping := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "EU (Ireland)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
	}
	
	displayRegion, ok := regionMapping[c.region]
	if !ok {
		displayRegion = c.region
	}

	query := `
		SELECT data
		FROM aws_pricing_raw
		WHERE service_code = 'AmazonEC2'
		AND location = $1
		AND data->>'productFamily' = 'Compute Instance'
		AND data->'attributes'->>'instanceType' = $2
		LIMIT 1
	`

	var rawData json.RawMessage
	err := c.resolver.DB.QueryRow(query, displayRegion, instanceType).Scan(&rawData)
	if err != nil {
		// Try without location filter if not found
		query = `
			SELECT data
			FROM aws_pricing_raw
			WHERE service_code = 'AmazonEC2'
			AND data->>'productFamily' = 'Compute Instance'
			AND data->'attributes'->>'instanceType' = $1
			LIMIT 1
		`
		err = c.resolver.DB.QueryRow(query, instanceType).Scan(&rawData)
		if err != nil {
			return 0.0, fmt.Errorf("instance type %s not found: %w", instanceType, err)
		}
	}

	// Parse the pricing data
	var pricing struct {
		Terms struct {
			OnDemand map[string]map[string]struct {
				PriceDimensions map[string]struct {
					PricePerUnit struct {
						USD string `json:"USD"`
					} `json:"pricePerUnit"`
				} `json:"priceDimensions"`
			} `json:"OnDemand"`
		} `json:"terms"`
	}

	if err := json.Unmarshal(rawData, &pricing); err != nil {
		return 0.0, fmt.Errorf("failed to parse pricing data: %w", err)
	}

	// Extract price from nested structure
	for _, skuData := range pricing.Terms.OnDemand {
		for _, termData := range skuData {
			for _, dimension := range termData.PriceDimensions {
				if price := dimension.PricePerUnit.USD; price != "" && price != "0.0000000000" {
					var priceFloat float64
					fmt.Sscanf(price, "%f", &priceFloat)
					return priceFloat, nil
				}
			}
		}
	}

	return 0.0, fmt.Errorf("no pricing found for instance type %s", instanceType)
}

func (s *AWSStorage) getStoragePriceFromRaw(ctx context.Context, tier string) (float64, error) {
	// Map tier names to S3 storage class names
	tierMapping := map[string]string{
		"standard":          "General Purpose",
		"infrequent_access": "Infrequent Access",
		"glacier":           "Amazon Glacier",
		"deep_archive":      "Amazon Glacier Deep Archive",
	}

	storageClass, ok := tierMapping[tier]
	if !ok {
		storageClass = "General Purpose"
	}

	query := `
		SELECT data
		FROM aws_pricing_raw
		WHERE service_code = 'AmazonS3'
		AND data->>'productFamily' = 'Storage'
		AND data->'attributes'->>'storageClass' ILIKE $1
		AND data->'attributes'->>'location' = $2
		LIMIT 1
	`

	// Map region to location
	regionMapping := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "EU (Ireland)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
	}

	displayRegion, ok := regionMapping[s.region]
	if !ok {
		displayRegion = s.region
	}

	var rawData json.RawMessage
	err := s.resolver.DB.QueryRow(query, "%"+storageClass+"%", displayRegion).Scan(&rawData)
	if err != nil {
		// Try without location filter
		query = `
			SELECT data
			FROM aws_pricing_raw
			WHERE service_code = 'AmazonS3'
			AND data->>'productFamily' = 'Storage'
			AND data->'attributes'->>'storageClass' ILIKE $1
			LIMIT 1
		`
		err = s.resolver.DB.QueryRow(query, "%"+storageClass+"%").Scan(&rawData)
		if err != nil {
			return 0.0, fmt.Errorf("storage tier %s not found: %w", tier, err)
		}
	}

	// Parse the pricing data
	var pricing struct {
		Terms struct {
			OnDemand map[string]map[string]struct {
				PriceDimensions map[string]struct {
					PricePerUnit struct {
						USD string `json:"USD"`
					} `json:"pricePerUnit"`
				} `json:"priceDimensions"`
			} `json:"OnDemand"`
		} `json:"terms"`
	}

	if err := json.Unmarshal(rawData, &pricing); err != nil {
		return 0.0, fmt.Errorf("failed to parse storage pricing data: %w", err)
	}

	// Extract price per GB-month
	for _, skuData := range pricing.Terms.OnDemand {
		for _, termData := range skuData {
			for _, dimension := range termData.PriceDimensions {
				if price := dimension.PricePerUnit.USD; price != "" && price != "0.0000000000" {
					var priceFloat float64
					fmt.Sscanf(price, "%f", &priceFloat)
					return priceFloat, nil
				}
			}
		}
	}

	return 0.0, fmt.Errorf("no pricing found for storage tier %s", tier)
}

func (c *AzureCompute) getVMPriceFromRaw(ctx context.Context, size string) (float64, error) {
	// Normalize region
	regionMapping := map[string]string{
		"eastus":        "eastus",
		"westus":        "westus",
		"westeurope":    "westeurope",
		"southeastasia": "southeastasia",
	}

	azureRegion, ok := regionMapping[c.region]
	if !ok {
		azureRegion = c.region
	}

	query := `
		SELECT data->>'retailPrice' as price
		FROM azure_pricing_raw
		WHERE data->>'serviceName' = 'Virtual Machines'
		AND data->>'armRegionName' = $1
		AND data->>'armSkuName' = $2
		AND data->>'type' = 'Consumption'
		LIMIT 1
	`

	var priceStr string
	err := c.resolver.DB.QueryRow(query, azureRegion, size).Scan(&priceStr)
	if err != nil {
		// Try without region filter
		query = `
			SELECT data->>'retailPrice' as price
			FROM azure_pricing_raw
			WHERE data->>'serviceName' = 'Virtual Machines'
			AND data->>'armSkuName' = $1
			AND data->>'type' = 'Consumption'
			LIMIT 1
		`
		err = c.resolver.DB.QueryRow(query, size).Scan(&priceStr)
		if err != nil {
			return 0.0, fmt.Errorf("VM size %s not found: %w", size, err)
		}
	}

	var price float64
	fmt.Sscanf(priceStr, "%f", &price)
	return price, nil
}

func (s *AzureStorage) getStoragePriceFromRaw(ctx context.Context, tier string) (float64, error) {
	// Map tier names to Azure storage tiers
	tierMapping := map[string]string{
		"hot":     "Hot LRS",
		"cool":    "Cool LRS",
		"archive": "Archive LRS",
	}

	azureTier, ok := tierMapping[tier]
	if !ok {
		azureTier = "Hot LRS"
	}

	// Normalize region
	regionMapping := map[string]string{
		"eastus":        "eastus",
		"westus":        "westus",
		"westeurope":    "westeurope",
		"southeastasia": "southeastasia",
	}

	azureRegion, ok := regionMapping[s.region]
	if !ok {
		azureRegion = s.region
	}

	query := `
		SELECT data->>'retailPrice' as price
		FROM azure_pricing_raw
		WHERE data->>'serviceName' = 'Storage'
		AND data->>'armRegionName' = $1
		AND data->>'meterName' ILIKE $2
		AND data->>'type' = 'Consumption'
		AND data->>'unitOfMeasure' ILIKE '%GB%'
		LIMIT 1
	`

	var priceStr string
	err := s.resolver.DB.QueryRow(query, azureRegion, "%"+azureTier+"%").Scan(&priceStr)
	if err != nil {
		// Try without region filter
		query = `
			SELECT data->>'retailPrice' as price
			FROM azure_pricing_raw
			WHERE data->>'serviceName' = 'Storage'
			AND data->>'meterName' ILIKE $1
			AND data->>'type' = 'Consumption'
			AND data->>'unitOfMeasure' ILIKE '%GB%'
			LIMIT 1
		`
		err = s.resolver.DB.QueryRow(query, "%"+azureTier+"%").Scan(&priceStr)
		if err != nil {
			return 0.0, fmt.Errorf("storage tier %s not found: %w", tier, err)
		}
	}

	var price float64
	fmt.Sscanf(priceStr, "%f", &price)
	return price, nil
}