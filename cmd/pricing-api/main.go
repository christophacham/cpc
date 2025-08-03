package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// PricingResponse represents pricing data for cloud services
type PricingResponse struct {
	Provider string                 `json:"provider"`
	Region   string                 `json:"region"`
	Compute  map[string]float64     `json:"compute"`
	Storage  map[string]float64     `json:"storage"`
	Transfer map[string]float64     `json:"transfer"`
	Raw      map[string]interface{} `json:"raw,omitempty"`
}

var db *sql.DB

func main() {
	// Connect to database
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully!")

	// Set up routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/pricing/aws", awsPricingHandler)
	http.HandleFunc("/pricing/azure", azurePricingHandler)
	http.HandleFunc("/pricing/unified", unifiedPricingHandler)
	http.HandleFunc("/health", healthHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting pricing API server on http://localhost:%s/", port)
	log.Printf("Endpoints:")
	log.Printf("  GET /pricing/aws?region=us-east-1")
	log.Printf("  GET /pricing/azure?region=eastus")
	log.Printf("  GET /pricing/unified?aws_region=us-east-1&azure_region=eastus")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service": "Cloud Price Compare API",
		"version": "1.0",
		"endpoints": map[string]string{
			"aws_pricing":     "/pricing/aws?region=us-east-1",
			"azure_pricing":   "/pricing/azure?region=eastus",
			"unified_pricing": "/pricing/unified?aws_region=us-east-1&azure_region=eastus",
			"health":          "/health",
		},
	}
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func awsPricingHandler(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "us-east-1"
	}

	response := PricingResponse{
		Provider: "aws",
		Region:   region,
		Compute:  make(map[string]float64),
		Storage:  make(map[string]float64),
		Transfer: make(map[string]float64),
	}

	// Get compute pricing from raw data
	computePrices := getAWSComputePrices(region)
	for k, v := range computePrices {
		response.Compute[k] = v
	}

	// Get storage pricing
	storagePrices := getAWSStoragePrices(region)
	for k, v := range storagePrices {
		response.Storage[k] = v
	}

	// Set transfer pricing
	response.Transfer["in"] = 0.0
	response.Transfer["out"] = 0.09

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func azurePricingHandler(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "eastus"
	}

	response := PricingResponse{
		Provider: "azure",
		Region:   region,
		Compute:  make(map[string]float64),
		Storage:  make(map[string]float64),
		Transfer: make(map[string]float64),
	}

	// Get compute pricing from raw data
	computePrices := getAzureComputePrices(region)
	for k, v := range computePrices {
		response.Compute[k] = v
	}

	// Get storage pricing
	storagePrices := getAzureStoragePrices(region)
	for k, v := range storagePrices {
		response.Storage[k] = v
	}

	// Set transfer pricing
	response.Transfer["in"] = 0.0
	response.Transfer["out"] = 0.0877

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func unifiedPricingHandler(w http.ResponseWriter, r *http.Request) {
	awsRegion := r.URL.Query().Get("aws_region")
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}
	azureRegion := r.URL.Query().Get("azure_region")
	if azureRegion == "" {
		azureRegion = "eastus"
	}

	response := map[string]interface{}{
		"aws": PricingResponse{
			Provider: "aws",
			Region:   awsRegion,
			Compute:  getAWSComputePrices(awsRegion),
			Storage:  getAWSStoragePrices(awsRegion),
			Transfer: map[string]float64{"in": 0.0, "out": 0.09},
		},
		"azure": PricingResponse{
			Provider: "azure",
			Region:   azureRegion,
			Compute:  getAzureComputePrices(azureRegion),
			Storage:  getAzureStoragePrices(azureRegion),
			Transfer: map[string]float64{"in": 0.0, "out": 0.0877},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Database query functions

func getAWSComputePrices(region string) map[string]float64 {
	prices := make(map[string]float64)
	
	// Map region to AWS location name
	locationMap := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "EU (Ireland)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
	}
	
	location, ok := locationMap[region]
	if !ok {
		location = region
	}

	// Query for common instance types
	instanceTypes := []string{"t3.micro", "t3.small", "t3.medium", "m5.large", "m5.xlarge", "c5.large", "c5.xlarge", "r5.large", "r5.xlarge"}
	
	for _, instanceType := range instanceTypes {
		query := `
			SELECT 
				CAST(data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')->jsonb_object_keys(
					data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')
				)->'priceDimensions'->jsonb_object_keys(
					data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')->jsonb_object_keys(
						data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')
					)->'priceDimensions'
				)->'pricePerUnit'->>'USD' AS FLOAT)
			FROM aws_pricing_raw
			WHERE service_code = 'AmazonEC2'
			AND data->>'productFamily' = 'Compute Instance'
			AND data->'attributes'->>'instanceType' = $1
			AND data->'attributes'->>'location' = $2
			AND data->'terms'->'OnDemand' IS NOT NULL
			LIMIT 1
		`
		
		var price float64
		err := db.QueryRow(query, instanceType, location).Scan(&price)
		if err == nil && price > 0 {
			// Convert instance type to key format
			key := strings.Replace(instanceType, ".", "_", -1)
			prices["ec2_"+key] = price
		}
	}

	// Add fallback prices if nothing found
	if len(prices) == 0 {
		prices["ec2_t3_micro"] = 0.0104
		prices["ec2_t3_small"] = 0.0208
		prices["ec2_t3_medium"] = 0.0416
		prices["ec2_m5_large"] = 0.096
		prices["ec2_m5_xlarge"] = 0.192
	}

	return prices
}

func getAWSStoragePrices(region string) map[string]float64 {
	prices := make(map[string]float64)
	
	// Map region to AWS location name
	locationMap := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "EU (Ireland)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
	}
	
	location, ok := locationMap[region]
	if !ok {
		location = region
	}
	
	// Query for different S3 storage classes
	storageClasses := map[string]string{
		"s3_standard":           "General Purpose",
		"s3_standard_ia":        "Standard - Infrequent Access",
		"s3_one_zone_ia":        "One Zone - Infrequent Access",
		"s3_glacier_instant":    "Amazon Glacier Instant Retrieval",
		"s3_glacier_flexible":   "Amazon Glacier Flexible Retrieval",
		"s3_glacier_deep":       "Amazon Glacier Deep Archive",
	}
	
	for key, storageClass := range storageClasses {
		query := `
			SELECT 
				CAST(data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')->jsonb_object_keys(
					data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')
				)->'priceDimensions'->jsonb_object_keys(
					data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')->jsonb_object_keys(
						data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')
					)->'priceDimensions'
				)->'pricePerUnit'->>'USD' AS FLOAT)
			FROM aws_pricing_raw
			WHERE service_code = 'AmazonS3'
			AND data->>'productFamily' = 'Storage'
			AND data->'attributes'->>'storageClass' ILIKE $1
			AND data->'attributes'->>'location' = $2
			AND data->'terms'->'OnDemand' IS NOT NULL
			LIMIT 1
		`
		
		var price float64
		err := db.QueryRow(query, "%"+storageClass+"%", location).Scan(&price)
		if err == nil && price > 0 {
			prices[key] = price
		}
	}
	
	// Add default fallback prices if nothing found
	if len(prices) == 0 {
		prices["s3_standard"] = 0.023
		prices["s3_standard_ia"] = 0.0125
		prices["s3_glacier_flexible"] = 0.004
		prices["s3_glacier_deep"] = 0.00099
	}
	
	// Add egress pricing
	prices["s3_data_transfer_out"] = getAWSEgressPricing(region)
	
	return prices
}

func getAWSEgressPricing(region string) float64 {
	// Map region to AWS location name
	locationMap := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "EU (Ireland)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
	}
	
	location, ok := locationMap[region]
	if !ok {
		location = region
	}
	
	// Query for AWS data transfer out pricing
	query := `
		SELECT 
			CAST(data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')->jsonb_object_keys(
				data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')
			)->'priceDimensions'->jsonb_object_keys(
				data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')->jsonb_object_keys(
					data->'terms'->'OnDemand'->jsonb_object_keys(data->'terms'->'OnDemand')
				)->'priceDimensions'
			)->'pricePerUnit'->>'USD' AS FLOAT)
		FROM aws_pricing_raw
		WHERE service_code = 'AmazonEC2'
		AND data->>'productFamily' = 'Data Transfer'
		AND data->'attributes'->>'transferType' = 'AWS Outbound'
		AND data->'attributes'->>'location' = $1
		AND data->'terms'->'OnDemand' IS NOT NULL
		LIMIT 1
	`
	
	var price float64
	err := db.QueryRow(query, location).Scan(&price)
	if err == nil && price > 0 {
		return price
	}
	
	// Default fallback
	return 0.09
}

func getAzureComputePrices(region string) map[string]float64 {
	prices := make(map[string]float64)
	
	// Query for common VM sizes
	vmSizes := []string{"Standard_B1s", "Standard_B2s", "Standard_D2s_v3", "Standard_D4s_v3", "Standard_F2s_v2", "Standard_F4s_v2", "Standard_NC8as_T4_v3", "Standard_NC16as_T4_v3"}
	
	for _, vmSize := range vmSizes {
		query := `
			SELECT CAST(data->>'retailPrice' AS FLOAT)
			FROM azure_pricing_raw
			WHERE data->>'serviceName' = 'Virtual Machines'
			AND data->>'armRegionName' = $1
			AND data->>'armSkuName' = $2
			AND data->>'type' = 'Consumption'
			LIMIT 1
		`
		
		var price float64
		err := db.QueryRow(query, region, vmSize).Scan(&price)
		if err == nil && price > 0 {
			// Convert VM size to key format
			key := strings.ToLower(strings.Replace(vmSize, "Standard_", "vm_", 1))
			prices[key] = price
		}
	}

	// Add fallback prices if nothing found
	if len(prices) == 0 {
		prices["vm_b1s"] = 0.0104
		prices["vm_b2s"] = 0.0416
		prices["vm_d2s_v3"] = 0.096
		prices["vm_d4s_v3"] = 0.192
		prices["vm_f2s_v2"] = 0.085
		prices["vm_f4s_v2"] = 0.17
		prices["vm_nc8as_t4_v3"] = 1.204
		prices["vm_nc16as_t4_v3"] = 2.408
	}

	return prices
}

func getAzureStoragePrices(region string) map[string]float64 {
	prices := make(map[string]float64)
	
	// Query for Azure storage pricing by meter name
	storageQueries := map[string]string{
		"hot_lrs":     "Hot LRS Data Stored",
		"cool_lrs":    "Cool LRS Data Stored", 
		"archive_lrs": "Archive LRS Data Stored",
		"hot_grs":     "Hot GRS Data Stored",
		"cool_grs":    "Cool GRS Data Stored",
		"archive_grs": "Archive GRS Data Stored",
	}
	
	for key, meterName := range storageQueries {
		query := `
			SELECT CAST(data->>'retailPrice' AS FLOAT)
			FROM azure_pricing_raw
			WHERE data->>'serviceName' = 'Storage'
			AND data->>'armRegionName' = $1
			AND data->>'meterName' ILIKE $2
			AND data->>'type' = 'Consumption'
			LIMIT 1
		`
		
		var price float64
		err := db.QueryRow(query, region, "%"+meterName+"%").Scan(&price)
		if err == nil && price > 0 {
			prices[key] = price
		}
	}
	
	// Add default fallback prices if nothing found
	if len(prices) == 0 {
		prices["hot_lrs"] = 0.0184
		prices["cool_lrs"] = 0.01
		prices["archive_lrs"] = 0.00099
	}
	
	// Add egress pricing
	prices["egress_data_transfer"] = getAzureEgressPricing(region)
	
	return prices
}

func getAzureEgressPricing(region string) float64 {
	// Query for Azure bandwidth/egress pricing
	query := `
		SELECT CAST(data->>'retailPrice' AS FLOAT)
		FROM azure_pricing_raw
		WHERE data->>'serviceName' = 'Bandwidth'
		AND data->>'armRegionName' = $1
		AND data->>'meterName' = 'Standard Data Transfer Out'
		AND data->>'type' = 'Consumption'
		LIMIT 1
	`
	
	var price float64
	err := db.QueryRow(query, region).Scan(&price)
	if err == nil && price > 0 {
		return price
	}
	
	// Try without region filter
	query = `
		SELECT CAST(data->>'retailPrice' AS FLOAT)
		FROM azure_pricing_raw
		WHERE data->>'serviceName' = 'Bandwidth'
		AND data->>'meterName' = 'Standard Data Transfer Out'
		AND data->>'type' = 'Consumption'
		LIMIT 1
	`
	
	err = db.QueryRow(query).Scan(&price)
	if err == nil && price > 0 {
		return price
	}
	
	// Default fallback
	return 0.0877
}