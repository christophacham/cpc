package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"github.com/lib/pq"
)

// AWSPricingClient handles AWS Pricing API operations
type AWSPricingClient struct {
	client *pricing.Client
	region string
}

// AWSPricingItem represents a single AWS pricing item
type AWSPricingItem struct {
	ServiceCode    string                 `json:"serviceCode"`
	ServiceName    string                 `json:"serviceName"`
	Location       string                 `json:"location"`
	InstanceType   string                 `json:"instanceType,omitempty"`
	PricePerUnit   float64               `json:"pricePerUnit"`
	Unit           string                `json:"unit"`
	Currency       string                `json:"currency"`
	TermType       string                `json:"termType"` // OnDemand, Reserved
	Attributes     map[string]interface{} `json:"attributes"`
	RawProduct     json.RawMessage       `json:"rawProduct"`
}

// AWSRegionLocationMap maps AWS regions to location names used in pricing API
var AWSRegionLocationMap = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)", 
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"eu-west-1":      "Europe (Ireland)",
	"eu-west-2":      "Europe (London)",
	"eu-west-3":      "Europe (Paris)",
	"eu-central-1":   "Europe (Frankfurt)",
	"eu-north-1":     "Europe (Stockholm)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ca-central-1":   "Canada (Central)",
	"sa-east-1":      "South America (São Paulo)",
}

// NewAWSPricingClient creates a new AWS Pricing API client
func NewAWSPricingClient() (*AWSPricingClient, error) {
	// Validate that required environment variables are set
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID environment variable is required")
	}
	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY environment variable is required")
	}

	// Load AWS configuration - this will use credentials from environment variables
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := pricing.NewFromConfig(cfg)
	
	return &AWSPricingClient{
		client: client,
		region: "us-east-1", // Pricing API only works in us-east-1
	}, nil
}

// GetAllServicePricing retrieves ALL pricing data for specified services and regions
func (c *AWSPricingClient) GetAllServicePricing(serviceCodes []string, regions []string) ([]AWSPricingItem, error) {
	var allPricing []AWSPricingItem
	
	for _, serviceCode := range serviceCodes {
		log.Printf("🔥 Collecting ALL pricing data for service: %s", serviceCode)
		
		// Get ALL pricing for this service (no filters except service code)
		pricing, err := c.getAllPricingForService(serviceCode, regions)
		if err != nil {
			log.Printf("WARNING: Failed to get pricing for service %s: %v", serviceCode, err)
			continue
		}
		allPricing = append(allPricing, pricing...)
		log.Printf("✅ Collected %d pricing items for service: %s", len(pricing), serviceCode)
	}
	
	return allPricing, nil
}

// GetEC2Pricing retrieves EC2 pricing for specific instance types and regions (legacy method)
func (c *AWSPricingClient) GetEC2Pricing(instanceTypes []string, regions []string) ([]AWSPricingItem, error) {
	// For backward compatibility, call the comprehensive method
	return c.GetAllServicePricing([]string{"AmazonEC2"}, regions)
}

// getAllPricingForService gets ALL pricing data for a service across all regions
func (c *AWSPricingClient) getAllPricingForService(serviceCode string, regions []string) ([]AWSPricingItem, error) {
	var allPricing []AWSPricingItem
	
	if len(regions) == 0 {
		// If no regions specified, get global pricing (no location filter)
		log.Printf("🌍 Collecting GLOBAL pricing for service: %s", serviceCode)
		pricing, err := c.getPricingWithMinimalFilters(serviceCode, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get global pricing for %s: %w", serviceCode, err)
		}
		allPricing = append(allPricing, pricing...)
	} else {
		// Collect for each specified region
		for _, region := range regions {
			locationName, exists := AWSRegionLocationMap[region]
			if !exists {
				log.Printf("WARNING: Unknown region %s, using region name as location", region)
				locationName = region // Use region name if not in map
			}
			
			log.Printf("📍 Collecting pricing for service %s in region: %s (%s)", serviceCode, region, locationName)
			
			pricing, err := c.getPricingWithMinimalFilters(serviceCode, locationName)
			if err != nil {
				log.Printf("WARNING: Failed to get pricing for %s in %s: %v", serviceCode, region, err)
				continue
			}
			allPricing = append(allPricing, pricing...)
		}
	}
	
	return allPricing, nil
}

// getPricingWithMinimalFilters gets pricing with only service code and optional location filter
func (c *AWSPricingClient) getPricingWithMinimalFilters(serviceCode string, locationName string) ([]AWSPricingItem, error) {
	// Minimal filters - just service code and optionally location
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("ServiceCode"),
			Value: awsString(serviceCode),
		},
	}
	
	// Add location filter only if specified
	if locationName != "" {
		filters = append(filters, types.Filter{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("location"),
			Value: awsString(locationName),
		})
	}

	input := &pricing.GetProductsInput{
		ServiceCode: awsString(serviceCode),
		Filters:     filters,
		MaxResults:  awsInt32(100), // AWS maximum
	}

	var pricingItems []AWSPricingItem
	pageCount := 0
	
	for {
		pageCount++
		log.Printf("📄 Processing page %d for service %s...", pageCount, serviceCode)
		
		result, err := c.client.GetProducts(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to get products on page %d: %w", pageCount, err)
		}

		// Process all products on this page
		for _, product := range result.PriceList {
			items, err := c.parseAWSProduct(product, locationName)
			if err != nil {
				log.Printf("WARNING: Failed to parse product: %v", err)
				continue
			}
			pricingItems = append(pricingItems, items...)
		}
		
		log.Printf("📊 Page %d: processed %d products, total items so far: %d", 
			pageCount, len(result.PriceList), len(pricingItems))

		// Check if there are more pages
		if result.NextToken == nil {
			log.Printf("✅ Completed collection for %s: %d total items across %d pages", 
				serviceCode, len(pricingItems), pageCount)
			break
		}
		
		input.NextToken = result.NextToken
		
		// Safety break to avoid infinite loops (AWS typically has 20-50 pages per service)
		if pageCount > 100 {
			log.Printf("⚠️ Reached page limit (%d) for service %s, stopping", pageCount, serviceCode)
			break
		}
	}

	return pricingItems, nil
}

// getEC2PricingForInstance gets pricing for a specific EC2 instance type
func (c *AWSPricingClient) getEC2PricingForInstance(instanceType, locationName, region string) ([]AWSPricingItem, error) {
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("ServiceCode"),
			Value: awsString("AmazonEC2"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("instanceType"),
			Value: awsString(instanceType),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("location"),
			Value: awsString(locationName),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("tenancy"),
			Value: awsString("Shared"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("operatingSystem"),
			Value: awsString("Linux"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: awsString("preInstalledSw"),
			Value: awsString("NA"),
		},
	}

	input := &pricing.GetProductsInput{
		ServiceCode: awsString("AmazonEC2"),
		Filters:     filters,
		MaxResults:  awsInt32(100),
	}

	var pricingItems []AWSPricingItem
	
	for {
		result, err := c.client.GetProducts(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to get products: %w", err)
		}

		for _, product := range result.PriceList {
			items, err := c.parseAWSProduct(product, region)
			if err != nil {
				log.Printf("WARNING: Failed to parse product: %v", err)
				continue
			}
			pricingItems = append(pricingItems, items...)
		}

		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}

	return pricingItems, nil
}

// GetS3Pricing retrieves S3 storage pricing
func (c *AWSPricingClient) GetS3Pricing(regions []string) ([]AWSPricingItem, error) {
	var allPricing []AWSPricingItem
	
	for _, region := range regions {
		locationName, exists := AWSRegionLocationMap[region]
		if !exists {
			log.Printf("WARNING: Unknown region %s, skipping", region)
			continue
		}
		
		log.Printf("Collecting S3 pricing for region: %s (%s)", region, locationName)
		
		filters := []types.Filter{
			{
				Type:  types.FilterTypeTermMatch,
				Field: awsString("ServiceCode"),
				Value: awsString("AmazonS3"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: awsString("location"),
				Value: awsString(locationName),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: awsString("storageClass"),
				Value: awsString("General Purpose"),
			},
		}

		input := &pricing.GetProductsInput{
			ServiceCode: awsString("AmazonS3"),
			Filters:     filters,
			MaxResults:  awsInt32(100),
		}

		for {
			result, err := c.client.GetProducts(context.TODO(), input)
			if err != nil {
				log.Printf("WARNING: Failed to get S3 products for %s: %v", region, err)
				break
			}

			for _, product := range result.PriceList {
				items, err := c.parseAWSProduct(product, region)
				if err != nil {
					log.Printf("WARNING: Failed to parse S3 product: %v", err)
					continue
				}
				allPricing = append(allPricing, items...)
			}

			if result.NextToken == nil {
				break
			}
			input.NextToken = result.NextToken
		}
	}
	
	return allPricing, nil
}

// parseAWSProduct parses an AWS product JSON into pricing items
func (c *AWSPricingClient) parseAWSProduct(productJSON string, region string) ([]AWSPricingItem, error) {
	var product map[string]interface{}
	if err := json.Unmarshal([]byte(productJSON), &product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal product: %w", err)
	}

	// Extract product attributes
	productData, ok := product["product"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid product structure")
	}

	attributes, _ := productData["attributes"].(map[string]interface{})
	serviceCode, _ := productData["productFamily"].(string)
	
	// Extract pricing terms
	terms, ok := product["terms"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no pricing terms found")
	}

	var pricingItems []AWSPricingItem

	// Parse On-Demand pricing
	if onDemand, exists := terms["OnDemand"].(map[string]interface{}); exists {
		for _, termData := range onDemand {
			if termMap, ok := termData.(map[string]interface{}); ok {
				items := c.parseTermPricing(termMap, attributes, serviceCode, region, "OnDemand", productJSON)
				pricingItems = append(pricingItems, items...)
			}
		}
	}

	// Parse Reserved pricing
	if reserved, exists := terms["Reserved"].(map[string]interface{}); exists {
		for _, termData := range reserved {
			if termMap, ok := termData.(map[string]interface{}); ok {
				items := c.parseTermPricing(termMap, attributes, serviceCode, region, "Reserved", productJSON)
				pricingItems = append(pricingItems, items...)
			}
		}
	}

	return pricingItems, nil
}

// parseTermPricing parses pricing terms into individual pricing items
func (c *AWSPricingClient) parseTermPricing(term map[string]interface{}, attributes map[string]interface{}, serviceCode, region, termType, rawProduct string) []AWSPricingItem {
	var items []AWSPricingItem

	priceDimensions, ok := term["priceDimensions"].(map[string]interface{})
	if !ok {
		return items
	}

	serviceName, _ := attributes["serviceName"].(string)
	instanceType, _ := attributes["instanceType"].(string)
	location, _ := attributes["location"].(string)

	for _, dimension := range priceDimensions {
		if dimMap, ok := dimension.(map[string]interface{}); ok {
			pricePerUnit, currency, unit := c.extractPriceInfo(dimMap)
			
			item := AWSPricingItem{
				ServiceCode:  serviceCode,
				ServiceName:  serviceName,
				Location:     location,
				InstanceType: instanceType,
				PricePerUnit: pricePerUnit,
				Unit:         unit,
				Currency:     currency,
				TermType:     termType,
				Attributes:   attributes,
				RawProduct:   json.RawMessage(rawProduct),
			}
			items = append(items, item)
		}
	}

	return items
}

// extractPriceInfo extracts price, currency, and unit from price dimension
func (c *AWSPricingClient) extractPriceInfo(dimension map[string]interface{}) (float64, string, string) {
	pricePerUnit := make(map[string]interface{})
	if ppu, exists := dimension["pricePerUnit"].(map[string]interface{}); exists {
		pricePerUnit = ppu
	}

	var price float64
	var currency string
	
	// AWS pricing typically has USD as the currency
	for curr, priceStr := range pricePerUnit {
		currency = curr
		if priceString, ok := priceStr.(string); ok {
			fmt.Sscanf(priceString, "%f", &price)
		}
		break // Take the first (usually only) currency
	}

	unit, _ := dimension["unit"].(string)
	
	return price, currency, unit
}

// GetAllAWSServices returns a comprehensive list of AWS services to collect
func GetAllAWSServices() []string {
	return []string{
		// Core Compute
		"AmazonEC2", "AWSLambda", "AmazonECS", "AmazonEKS",
		
		// Storage
		"AmazonS3", "AmazonEBS", "AmazonEFS", "AmazonFSx",
		
		// Database
		"AmazonRDS", "AmazonDynamoDB", "AmazonElastiCache", "AmazonRedshift",
		"AmazonNeptune", "AmazonDocumentDB", "AmazonMemoryDB",
		
		// Networking
		"AmazonVPC", "AmazonCloudFront", "AmazonRoute53", "AWSELB",
		"AWSDirectConnect", "AmazonVPCEndpoint", "AWSTransitGateway",
		
		// Analytics
		"AmazonEMR", "AmazonKinesis", "AmazonAthena", "AWSGlue",
		"AmazonQuickSight", "AmazonOpenSearch",
		
		// AI/ML
		"AmazonSageMaker", "AmazonRekognition", "AmazonComprehend",
		"AmazonTranscribe", "AmazonPolly", "AmazonTranslate",
		
		// Application Integration
		"AmazonSQS", "AmazonSNS", "AmazonSWF", "AWSStepFunctions",
		"AmazonMQ", "AmazonEventBridge",
		
		// Developer Tools
		"AWSCodeCommit", "AWSCodeBuild", "AWSCodeDeploy", "AWSCodePipeline",
		
		// Security & Management
		"AWSCloudTrail", "AmazonCloudWatch", "AWSConfig", "AWSSecurityHub",
		"AWSKMS", "AWSSecretsManager", "AWSWAF", "AWSShield",
		
		// Migration & Transfer
		"AWSDataSync", "AWSSnowball", "AWSStorageGateway", "AWSDMS",
		
		// Enterprise Applications
		"AmazonWorkSpaces", "AmazonAppStream", "AmazonConnect",
		
		// IoT
		"AWSIoTCore", "AWSIoTAnalytics", "AWSIoTEvents",
		
		// Containers
		"AWSFargate", "AmazonECR",
		
		// Data Transfer (Global)
		"AWSDataTransfer",
	}
}

// GetMajorAWSServices returns the most commonly used AWS services (80/20 rule)
func GetMajorAWSServices() []string {
	return []string{
		"AmazonEC2", "AmazonS3", "AmazonEBS", "AmazonRDS", 
		"AWSLambda", "AmazonVPC", "AmazonCloudFront", "AWSELB",
		"AmazonDynamoDB", "AmazonCloudWatch", "AWSDataTransfer",
		"AmazonRoute53", "AmazonElastiCache", "AmazonEMR",
	}
}

// StoreAWSPricing stores AWS pricing data in the database using raw JSONB approach
func StoreAWSPricing(db *sql.DB, pricingItems []AWSPricingItem, collectionID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO aws_pricing_raw (
			collection_id, service_code, service_name, location,
			data, collected_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range pricingItems {
		// Store the complete raw product JSON as the data field
		_, err = stmt.Exec(
			collectionID,
			item.ServiceCode,
			item.ServiceName,
			item.Location,
			item.RawProduct, // Store entire raw AWS product JSON
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert pricing item: %w", err)
		}
	}

	return tx.Commit()
}

// GetAWSCollections retrieves AWS collection tracking records
func (db *DB) GetAWSCollections() ([]map[string]interface{}, error) {
	query := `
		SELECT id, collection_id, service_codes, regions, status, 
			   started_at, completed_at, total_items, error_message, metadata
		FROM aws_collections 
		ORDER BY started_at DESC
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query AWS collections: %w", err)
	}
	defer rows.Close()
	
	var results []map[string]interface{}
	
	for rows.Next() {
		var id int
		var collectionID string
		var serviceCodes pq.StringArray
		var regions pq.StringArray
		var status string
		var startedAt time.Time
		var completedAt sql.NullTime
		var totalItems int
		var errorMessage sql.NullString
		var metadataJSON sql.NullString
		
		err := rows.Scan(&id, &collectionID, &serviceCodes, &regions, &status, 
						&startedAt, &completedAt, &totalItems, &errorMessage, &metadataJSON)
		if err != nil {
			return nil, err
		}
		
		collection := map[string]interface{}{
			"id":           fmt.Sprintf("%d", id),
			"collectionId": collectionID,
			"serviceCodes": []string(serviceCodes),
			"regions":      []string(regions),
			"status":       status,
			"startedAt":    startedAt.Format(time.RFC3339),
			"totalItems":   totalItems,
		}
		
		if completedAt.Valid {
			collection["completedAt"] = completedAt.Time.Format(time.RFC3339)
			duration := completedAt.Time.Sub(startedAt)
			collection["duration"] = fmt.Sprintf("%.3fs", duration.Seconds())
		}
		
		if errorMessage.Valid {
			collection["errorMessage"] = errorMessage.String
		}
		
		if metadataJSON.Valid {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err == nil {
				collection["metadata"] = metadata
			}
		}
		
		results = append(results, collection)
	}
	
	return results, rows.Err()
}

// Helper functions to create pointers (AWS SDK requirement)
func awsString(s string) *string {
	return &s
}

func awsInt32(i int32) *int32 {
	return &i
}