package normalizer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/raulc0399/cpc/internal/database"
)

// RefactoredAWSNormalizer handles normalization of AWS pricing data with improved modularity
type RefactoredAWSNormalizer struct {
	serviceMappingRepo ServiceMappingRepository
	regionMappingRepo  RegionMappingRepository
	unitNormalizer     UnitNormalizer
	validator          *InputValidator
	specExtractor      *AWSResourceSpecExtractor
}

// NewRefactoredAWSNormalizer creates a new refactored AWS pricing normalizer
func NewRefactoredAWSNormalizer(
	serviceMappingRepo ServiceMappingRepository,
	regionMappingRepo RegionMappingRepository,
	unitNormalizer UnitNormalizer,
	validator *InputValidator,
) *RefactoredAWSNormalizer {
	return &RefactoredAWSNormalizer{
		serviceMappingRepo: serviceMappingRepo,
		regionMappingRepo:  regionMappingRepo,
		unitNormalizer:     unitNormalizer,
		validator:          validator,
		specExtractor:      NewAWSResourceSpecExtractor(),
	}
}

// GetSupportedProvider returns the provider this normalizer supports
func (n *RefactoredAWSNormalizer) GetSupportedProvider() string {
	return database.ProviderAWS
}

// ValidateInput validates the input data before normalization
func (n *RefactoredAWSNormalizer) ValidateInput(input database.NormalizationInput) error {
	if err := n.validator.ValidateNormalizationInput(input); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if input.Provider != database.ProviderAWS {
		return NormalizationError{
			Provider:    input.Provider,
			ServiceCode: input.ServiceCode,
			Region:      input.Region,
			Message:     "unsupported provider for AWS normalizer",
		}
	}

	return nil
}

// NormalizePricing normalizes raw AWS pricing data into the standardized format
func (n *RefactoredAWSNormalizer) NormalizePricing(ctx context.Context, input database.NormalizationInput) (*database.NormalizationResult, error) {
	log.Printf("🔄 Normalizing AWS pricing data for service: %s, region: %s", input.ServiceCode, input.Region)

	// Validate input
	if err := n.ValidateInput(input); err != nil {
		return &database.NormalizationResult{
			Success:    false,
			ErrorCount: 1,
			Errors:     []string{err.Error()},
		}, nil
	}

	// Parse AWS product JSON
	awsProduct, err := n.parseAWSProduct(input.RawData)
	if err != nil {
		return &database.NormalizationResult{
			Success:    false,
			ErrorCount: 1,
			Errors:     []string{fmt.Sprintf("failed to parse AWS product: %v", err)},
		}, nil
	}

	// Get dependencies
	serviceMapping, err := n.serviceMappingRepo.GetServiceMappingByProvider(ctx, database.ProviderAWS, input.ServiceCode)
	if err != nil {
		return n.createErrorResult(fmt.Sprintf("failed to get service mapping: %v", err)), nil
	}
	if serviceMapping == nil {
		log.Printf("⚠️ No service mapping found for AWS service: %s", input.ServiceCode)
		return &database.NormalizationResult{
			Success:      false,
			SkippedCount: 1,
			Errors:       []string{fmt.Sprintf("no service mapping found for: %s", input.ServiceCode)},
		}, nil
	}

	normalizedRegion, err := n.regionMappingRepo.GetNormalizedRegionByProvider(ctx, database.ProviderAWS, input.Region)
	if err != nil {
		return n.createErrorResult(fmt.Sprintf("failed to get normalized region: %v", err)), nil
	}
	if normalizedRegion == nil {
		log.Printf("⚠️ No normalized region found for AWS region: %s", input.Region)
		return &database.NormalizationResult{
			Success:      false,
			SkippedCount: 1,
			Errors:       []string{fmt.Sprintf("no normalized region found for: %s", input.Region)},
		}, nil
	}

	// Process pricing terms
	result := n.processPricingTerms(awsProduct, serviceMapping, normalizedRegion, input.RawDataID)
	
	log.Printf("✅ Normalized %d AWS pricing records for service: %s", len(result.NormalizedRecords), input.ServiceCode)
	return result, nil
}

// parseAWSProduct parses AWS product JSON
func (n *RefactoredAWSNormalizer) parseAWSProduct(rawData json.RawMessage) (map[string]interface{}, error) {
	var awsProduct map[string]interface{}
	if err := json.Unmarshal(rawData, &awsProduct); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AWS product JSON: %w", err)
	}

	// Validate product structure
	if _, ok := awsProduct["product"]; !ok {
		return nil, fmt.Errorf("invalid AWS product structure: missing product field")
	}

	if _, ok := awsProduct["terms"]; !ok {
		return nil, fmt.Errorf("invalid AWS product structure: missing terms field")
	}

	return awsProduct, nil
}

// processPricingTerms processes all pricing terms in the AWS product
func (n *RefactoredAWSNormalizer) processPricingTerms(
	awsProduct map[string]interface{},
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	rawDataID int,
) *database.NormalizationResult {
	var normalizedRecords []database.NormalizedPricing
	var errors []string

	// Extract base product information
	product := awsProduct["product"].(map[string]interface{})
	attributes, _ := product["attributes"].(map[string]interface{})
	productSKU, _ := product["sku"].(string)

	// Process terms
	terms, _ := awsProduct["terms"].(map[string]interface{})

	// Process On-Demand pricing
	if onDemand, exists := terms["OnDemand"].(map[string]interface{}); exists {
		records, errs := n.processTermType(
			onDemand, database.PricingModelOnDemand, 
			serviceMapping, normalizedRegion, productSKU, attributes, rawDataID,
		)
		normalizedRecords = append(normalizedRecords, records...)
		errors = append(errors, errs...)
	}

	// Process Reserved pricing
	if reserved, exists := terms["Reserved"].(map[string]interface{}); exists {
		records, errs := n.processTermType(
			reserved, database.PricingModelReserved1Yr,
			serviceMapping, normalizedRegion, productSKU, attributes, rawDataID,
		)
		normalizedRecords = append(normalizedRecords, records...)
		errors = append(errors, errs...)
	}

	return &database.NormalizationResult{
		Success:           len(normalizedRecords) > 0,
		NormalizedRecords: normalizedRecords,
		ErrorCount:        len(errors),
		Errors:            errors,
	}
}

// processTermType processes a specific pricing term type
func (n *RefactoredAWSNormalizer) processTermType(
	terms map[string]interface{},
	pricingModel string,
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	productSKU string,
	attributes map[string]interface{},
	rawDataID int,
) ([]database.NormalizedPricing, []string) {
	var records []database.NormalizedPricing
	var errors []string

	for _, termData := range terms {
		termMap, ok := termData.(map[string]interface{})
		if !ok {
			continue
		}

		termRecords, termErrors := n.processTermInstance(
			termMap, pricingModel, serviceMapping, normalizedRegion,
			productSKU, attributes, rawDataID,
		)
		records = append(records, termRecords...)
		errors = append(errors, termErrors...)
	}

	return records, errors
}

// processTermInstance processes a single term instance
func (n *RefactoredAWSNormalizer) processTermInstance(
	termMap map[string]interface{},
	pricingModel string,
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	productSKU string,
	attributes map[string]interface{},
	rawDataID int,
) ([]database.NormalizedPricing, []string) {
	var records []database.NormalizedPricing
	var errors []string

	priceDimensions, ok := termMap["priceDimensions"].(map[string]interface{})
	if !ok {
		return records, errors
	}

	termAttributes, _ := termMap["termAttributes"].(map[string]interface{})

	for _, dimension := range priceDimensions {
		dimMap, ok := dimension.(map[string]interface{})
		if !ok {
			continue
		}

		record, err := n.createNormalizedRecord(
			dimMap, pricingModel, serviceMapping, normalizedRegion,
			productSKU, attributes, termAttributes, rawDataID,
		)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		if record != nil {
			records = append(records, *record)
		}
	}

	return records, errors
}

// createNormalizedRecord creates a single normalized pricing record
func (n *RefactoredAWSNormalizer) createNormalizedRecord(
	priceDimension map[string]interface{},
	pricingModel string,
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	productSKU string,
	attributes, termAttributes map[string]interface{},
	rawDataID int,
) (*database.NormalizedPricing, error) {
	// Extract pricing information
	priceInfo, err := n.extractPricingInfo(priceDimension)
	if err != nil {
		return nil, err
	}

	// Skip zero-cost items
	if priceInfo.PricePerUnit == 0 {
		return nil, nil
	}

	// Normalize unit
	normalizedUnit := n.unitNormalizer.NormalizeUnit(database.ProviderAWS, priceInfo.Unit)

	// Extract resource specifications
	resourceSpecs, err := n.specExtractor.ExtractResourceSpecs(
		database.ProviderAWS, serviceMapping.NormalizedServiceType, attributes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to extract resource specs: %w", err)
	}

	// Create resource name
	resourceName := n.createResourceName(attributes, serviceMapping.NormalizedServiceType)

	// Extract pricing details
	pricingDetails := n.extractPricingDetails(termAttributes, pricingModel)

	// Create the normalized record
	record := database.NormalizedPricing{
		Provider:            database.ProviderAWS,
		ProviderServiceCode: *serviceMapping.ProviderServiceCode,
		ProviderSKU:         &productSKU,
		ServiceMappingID:    &serviceMapping.ID,
		ServiceCategory:     serviceMapping.ServiceCategory,
		ServiceFamily:       serviceMapping.ServiceFamily,
		ServiceType:         serviceMapping.NormalizedServiceType,
		RegionID:            &normalizedRegion.ID,
		NormalizedRegion:    normalizedRegion.NormalizedCode,
		ProviderRegion:      *normalizedRegion.AWSRegion,
		ResourceName:        resourceName,
		ResourceDescription: &priceInfo.Description,
		ResourceSpecs:       resourceSpecs,
		PricePerUnit:        priceInfo.PricePerUnit,
		Unit:                normalizedUnit,
		Currency:            priceInfo.Currency,
		PricingModel:        pricingModel,
		PricingDetails:      pricingDetails,
		MinimumCommitment:   1,
		AWSRawID:            &rawDataID,
	}

	// Validate the created record
	if err := n.validator.ValidateNormalizedPricing(record); err != nil {
		return nil, fmt.Errorf("created record validation failed: %w", err)
	}

	return &record, nil
}

// PricingInfo holds extracted pricing information
type PricingInfo struct {
	PricePerUnit float64
	Unit         string
	Currency     string
	Description  string
}

// extractPricingInfo extracts pricing information from a price dimension
func (n *RefactoredAWSNormalizer) extractPricingInfo(priceDimension map[string]interface{}) (*PricingInfo, error) {
	pricePerUnitMap, ok := priceDimension["pricePerUnit"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing pricePerUnit in price dimension")
	}

	unit, _ := priceDimension["unit"].(string)
	description, _ := priceDimension["description"].(string)

	var pricePerUnit float64
	var currency string = "USD"

	for curr, priceStr := range pricePerUnitMap {
		currency = curr
		if priceString, ok := priceStr.(string); ok {
			var err error
			pricePerUnit, err = strconv.ParseFloat(priceString, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse price: %w", err)
			}
		}
		break // Take first (usually only) currency
	}

	return &PricingInfo{
		PricePerUnit: pricePerUnit,
		Unit:         unit,
		Currency:     currency,
		Description:  description,
	}, nil
}

// createResourceName creates a standardized resource name
func (n *RefactoredAWSNormalizer) createResourceName(attributes map[string]interface{}, serviceType string) string {
	switch serviceType {
	case "Virtual Machines":
		if instanceType, ok := attributes["instanceType"].(string); ok {
			return instanceType
		}
	case "Serverless Functions":
		if arch, ok := attributes["architecture"].(string); ok {
			return fmt.Sprintf("Lambda (%s)", arch)
		}
		return "Lambda"
	case "Serverless Containers":
		return "Fargate"
	}

	// Fallback
	if instanceType, ok := attributes["instanceType"].(string); ok {
		return instanceType
	}
	if serviceName, ok := attributes["serviceName"].(string); ok {
		return serviceName
	}

	return "Unknown"
}

// extractPricingDetails extracts additional pricing details
func (n *RefactoredAWSNormalizer) extractPricingDetails(termAttributes map[string]interface{}, pricingModel string) database.PricingDetails {
	details := database.PricingDetails{}

	if termAttributes == nil {
		return details
	}

	if leaseLength, ok := termAttributes["LeaseContractLength"].(string); ok {
		details.TermLength = &leaseLength
	}

	if purchaseOption, ok := termAttributes["PurchaseOption"].(string); ok {
		details.PaymentOption = &purchaseOption
	}

	return details
}

// createErrorResult creates a standardized error result
func (n *RefactoredAWSNormalizer) createErrorResult(message string) *database.NormalizationResult {
	return &database.NormalizationResult{
		Success:    false,
		ErrorCount: 1,
		Errors:     []string{message},
	}
}

// AWSResourceSpecExtractor extracts resource specifications from AWS attributes
type AWSResourceSpecExtractor struct{}

// NewAWSResourceSpecExtractor creates a new AWS resource spec extractor
func NewAWSResourceSpecExtractor() *AWSResourceSpecExtractor {
	return &AWSResourceSpecExtractor{}
}

// ExtractResourceSpecs extracts and normalizes resource specifications from AWS attributes
func (e *AWSResourceSpecExtractor) ExtractResourceSpecs(provider, serviceType string, data map[string]interface{}) (database.ResourceSpecs, error) {
	specs := database.ResourceSpecs{}

	// Extract common fields
	if vcpuStr, ok := data["vcpu"].(string); ok {
		if vcpu, err := strconv.Atoi(vcpuStr); err == nil {
			specs.VCPU = &vcpu
		}
	}

	if memoryStr, ok := data["memory"].(string); ok {
		memory := e.parseMemoryString(memoryStr)
		if memory > 0 {
			specs.MemoryGB = &memory
		}
	}

	if storageStr, ok := data["storage"].(string); ok {
		storageGB, storageType := e.parseStorageString(storageStr)
		if storageGB > 0 {
			specs.StorageGB = &storageGB
		}
		if storageType != "" {
			specs.StorageType = &storageType
		}
	}

	if networkPerf, ok := data["networkPerformance"].(string); ok {
		specs.NetworkPerformance = &networkPerf
	}

	// Handle instance-type specific attributes
	if instanceType, ok := data["instanceType"].(string); ok {
		e.enrichSpecsFromInstanceType(&specs, instanceType)
		
		if procType, ok := data["processorType"].(string); ok {
			specs.ProcessorType = &procType
		}
	}

	// GPU specifications
	if gpu, ok := data["gpu"].(string); ok && gpu != "NA" {
		if gpuCount, err := strconv.Atoi(gpu); err == nil {
			specs.GPUCount = &gpuCount
		}
	}

	return specs, nil
}

// parseMemoryString parses memory strings like "4 GiB" or "3.75 GiB"
func (e *AWSResourceSpecExtractor) parseMemoryString(memoryStr string) float64 {
	memoryStr = strings.ReplaceAll(memoryStr, " GiB", "")
	memoryStr = strings.ReplaceAll(memoryStr, " GB", "")
	if memory, err := strconv.ParseFloat(memoryStr, 64); err == nil {
		return memory
	}
	return 0
}

// parseStorageString parses storage strings like "1 x 150 SSD"
func (e *AWSResourceSpecExtractor) parseStorageString(storageStr string) (float64, string) {
	if storageStr == "NA" || storageStr == "EBS only" {
		return 0, ""
	}

	storageType := "ssd"
	if strings.Contains(storageStr, "HDD") {
		storageType = "hdd"
	}

	// Extract storage size
	parts := strings.Fields(storageStr)
	for i, part := range parts {
		if i > 0 && (parts[i-1] == "x" || parts[i-1] == "×") {
			if size, err := strconv.ParseFloat(part, 64); err == nil {
				// Check if there's a multiplier
				if i >= 2 {
					if multiplier, err := strconv.Atoi(parts[i-2]); err == nil {
						size *= float64(multiplier)
					}
				}
				return size, storageType
			}
		}
	}

	return 0, storageType
}

// enrichSpecsFromInstanceType adds instance-type specific attributes
func (e *AWSResourceSpecExtractor) enrichSpecsFromInstanceType(specs *database.ResourceSpecs, instanceType string) {
	// Check for burstable instances
	if strings.Contains(instanceType, "t2") || strings.Contains(instanceType, "t3") || strings.Contains(instanceType, "t4g") {
		burstable := true
		specs.Burstable = &burstable
	}

	// Set architecture for ARM instances
	if strings.Contains(instanceType, "g") && (strings.Contains(instanceType, "t4g") || strings.Contains(instanceType, "m6g") || strings.Contains(instanceType, "c6g")) {
		arch := "arm64"
		specs.Architecture = &arch
	} else {
		arch := "x86_64"
		specs.Architecture = &arch
	}
}