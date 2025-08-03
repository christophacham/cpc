package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/raulc0399/cpc/internal/database"
	"github.com/raulc0399/cpc/internal/normalizer"
)

func main() {
	// Connect to database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://cpc_user:cpc_password@localhost/cpc_db?sslmode=disable"
	}
	
	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	db := database.New(sqlDB)

	// Create normalizer components
	logger := normalizer.NewSimpleLogger()
	serviceMappingRepo := normalizer.NewServiceMappingRepository(db, logger)
	regionMappingRepo := normalizer.NewRegionMappingRepository(db, logger)
	unitNormalizer := normalizer.NewStandardUnitNormalizer()
	validator := normalizer.NewInputValidator()
	
	// Create Azure normalizer
	azureNormalizer := normalizer.NewAzureNormalizerV2(
		serviceMappingRepo,
		regionMappingRepo,
		unitNormalizer,
		validator,
		logger,
	)
	
	ctx := context.Background()
	
	// Test service mapping lookup
	fmt.Println("🔍 Testing service mapping lookup...")
	serviceMapping, err := serviceMappingRepo.GetServiceMappingByProvider(ctx, "azure", "Storage")
	if err != nil {
		log.Fatalf("Failed to get service mapping: %v", err)
	}
	if serviceMapping == nil {
		log.Fatalf("Service mapping not found for azure:Storage")
	}
	fmt.Printf("✅ Service mapping found: %+v\n", serviceMapping)
	
	// Test region mapping lookup
	fmt.Println("🔍 Testing region mapping lookup...")
	regionMapping, err := regionMappingRepo.GetNormalizedRegionByProvider(ctx, "azure", "eastus")
	if err != nil {
		log.Fatalf("Failed to get region mapping: %v", err)
	}
	if regionMapping == nil {
		log.Fatalf("Region mapping not found for azure:eastus")
	}
	fmt.Printf("✅ Region mapping found: %+v\n", regionMapping)
	
	// Get a sample Storage record
	fmt.Println("🔍 Getting sample Storage record...")
	var rawRecord struct {
		ID           int
		Region       string
		ServiceName  string
		Data         json.RawMessage
		CollectionID string
	}
	
	query := `SELECT id, region, service_name, data, collection_id FROM azure_pricing_raw WHERE service_name = 'Storage' LIMIT 1`
	err = sqlDB.QueryRow(query).Scan(&rawRecord.ID, &rawRecord.Region, &rawRecord.ServiceName, &rawRecord.Data, &rawRecord.CollectionID)
	if err != nil {
		log.Fatalf("Failed to get sample record: %v", err)
	}
	
	fmt.Printf("✅ Sample record: ID=%d, Region=%s, Service=%s\n", rawRecord.ID, rawRecord.Region, rawRecord.ServiceName)
	
	// Test normalization
	fmt.Println("🔍 Testing normalization...")
	input := database.NormalizationInput{
		Provider:     database.ProviderAzure,
		ServiceCode:  rawRecord.ServiceName,
		Region:       rawRecord.Region,
		RawData:      rawRecord.Data,
		RawDataID:    rawRecord.ID,
		CollectionID: rawRecord.CollectionID,
	}
	
	result, err := azureNormalizer.NormalizePricing(ctx, input)
	if err != nil {
		log.Fatalf("Normalization failed with error: %v", err)
	}
	
	fmt.Printf("✅ Normalization result: Success=%v, Records=%d, Skipped=%d, Errors=%d\n", 
		result.Success, len(result.NormalizedRecords), result.SkippedCount, result.ErrorCount)
	
	if len(result.Errors) > 0 {
		fmt.Printf("🔍 Errors: %v\n", result.Errors)
	}
	
	if len(result.NormalizedRecords) > 0 {
		fmt.Printf("✅ First normalized record: %+v\n", result.NormalizedRecords[0])
	}
}