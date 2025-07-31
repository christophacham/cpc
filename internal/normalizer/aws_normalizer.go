package normalizer

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/raulc0399/cpc/internal/database"
)

// AWSNormalizer handles normalization of AWS pricing data
type AWSNormalizer struct {
	db *database.DB
}

// NewAWSNormalizer creates a new AWS pricing normalizer
func NewAWSNormalizer(db *database.DB) *AWSNormalizer {
	return &AWSNormalizer{db: db}
}

// NormalizeAWSPricing normalizes raw AWS pricing data into the standardized format
func (n *AWSNormalizer) NormalizeAWSPricing(input database.NormalizationInput) (*database.NormalizationResult, error) {
	log.Printf("🔄 Normalizing AWS pricing data for service: %s, region: %s", input.ServiceCode, input.Region)

	var awsProduct map[string]interface{}
	if err := json.Unmarshal(input.RawData, &awsProduct); err != nil {
		return &database.NormalizationResult{
			Success:      false,
			ErrorCount:   1,
			Errors:       []string{fmt.Sprintf("failed to unmarshal AWS product JSON: %v", err)},
		}, nil
	}

	// Extract product information
	product, ok := awsProduct["product"].(map[string]interface{})
	if !ok {
		return &database.NormalizationResult{
			Success:      false,
			ErrorCount:   1,
			Errors:       []string{"invalid AWS product structure: missing product field"},
		}, nil
	}

	// Get service mapping
	serviceMapping, err := n.db.GetServiceMappingByProvider(database.ProviderAWS, input.ServiceCode)
	if err != nil {
		return &database.NormalizationResult{
			Success:      false,
			ErrorCount:   1,
			Errors:       []string{fmt.Sprintf("failed to get service mapping: %v", err)},
		}, nil
	}
	if serviceMapping == nil {
		log.Printf("⚠️ No service mapping found for AWS service: %s", input.ServiceCode)
		return &database.NormalizationResult{
			Success:      false,
			SkippedCount: 1,
			Errors:       []string{fmt.Sprintf("no service mapping found for: %s", input.ServiceCode)},
		}, nil
	}

	// Get normalized region
	normalizedRegion, err := n.db.GetNormalizedRegionByProvider(database.ProviderAWS, input.Region)
	if err != nil {
		return &database.NormalizationResult{
			Success:      false,
			ErrorCount:   1,
			Errors:       []string{fmt.Sprintf("failed to get normalized region: %v", err)},
		}, nil
	}
	if normalizedRegion == nil {
		log.Printf("⚠️ No normalized region found for AWS region: %s", input.Region)
		return &database.NormalizationResult{
			Success:      false,
			SkippedCount: 1,
			Errors:       []string{fmt.Sprintf("no normalized region found for: %s", input.Region)},
		}, nil
	}

	// Extract base product attributes
	attributes, _ := product["attributes"].(map[string]interface{})
	productSKU, _ := product["sku"].(string)
	serviceName, _ := attributes["serviceName"].(string)
	location, _ := attributes["location"].(string)

	// Parse pricing terms
	terms, ok := awsProduct["terms"].(map[string]interface{})
	if !ok {
		return &database.NormalizationResult{
			Success:      false,
			ErrorCount:   1,
			Errors:       []string{"no pricing terms found in AWS product"},
		}, nil
	}

	var normalizedRecords []database.NormalizedPricing
	var errors []string

	// Process On-Demand pricing
	if onDemand, exists := terms["OnDemand"].(map[string]interface{}); exists {
		records, errs := n.processAWSTerms(onDemand, database.PricingModelOnDemand, serviceMapping, normalizedRegion, productSKU, serviceName, location, attributes, input.RawDataID)
		normalizedRecords = append(normalizedRecords, records...)
		errors = append(errors, errs...)
	}

	// Process Reserved pricing
	if reserved, exists := terms["Reserved"].(map[string]interface{}); exists {
		records, errs := n.processAWSTerms(reserved, database.PricingModelReserved1Yr, serviceMapping, normalizedRegion, productSKU, serviceName, location, attributes, input.RawDataID)
		normalizedRecords = append(normalizedRecords, records...)
		errors = append(errors, errs...)
	}

	log.Printf("✅ Normalized %d AWS pricing records for service: %s", len(normalizedRecords), input.ServiceCode)

	return &database.NormalizationResult{
		Success:           len(normalizedRecords) > 0,
		NormalizedRecords: normalizedRecords,
		ErrorCount:        len(errors),
		Errors:            errors,
	}, nil
}

// processAWSTerms processes AWS pricing terms for a specific pricing model
func (n *AWSNormalizer) processAWSTerms(
	terms map[string]interface{},
	pricingModel string,
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	productSKU, serviceName, location string,
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

		priceDimensions, ok := termMap["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract term attributes for reserved instances
		termAttributes, _ := termMap["termAttributes"].(map[string]interface{})

		for _, dimension := range priceDimensions {
			dimMap, ok := dimension.(map[string]interface{})
			if !ok {
				continue
			}

			record, err := n.createNormalizedRecord(
				dimMap, pricingModel, serviceMapping, normalizedRegion,
				productSKU, serviceName, location, attributes, termAttributes, rawDataID,
			)
			if err != nil {
				errors = append(errors, err.Error())
				continue
			}

			if record != nil {
				records = append(records, *record)
			}
		}
	}

	return records, errors
}

// createNormalizedRecord creates a single normalized pricing record from AWS data
func (n *AWSNormalizer) createNormalizedRecord(
	priceDimension map[string]interface{},
	pricingModel string,
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	productSKU, serviceName, location string,
	attributes, termAttributes map[string]interface{},
	rawDataID int,
) (*database.NormalizedPricing, error) {
	// Extract pricing information
	pricePerUnitMap, ok := priceDimension["pricePerUnit"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing pricePerUnit in price dimension")
	}

	unit, _ := priceDimension["unit"].(string)
	description, _ := priceDimension["description"].(string)

	// Get price (AWS typically uses USD)
	var pricePerUnit float64
	var currency string = "USD"
	for curr, priceStr := range pricePerUnitMap {
		currency = curr
		if priceString, ok := priceStr.(string); ok {
			var err error
			pricePerUnit, err = strconv.ParseFloat(priceString, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse price: %v", err)
			}
		}
		break // Take first (usually only) currency
	}

	// Skip zero-cost items (often indicates missing data)
	if pricePerUnit == 0 {
		return nil, nil
	}

	// Normalize the unit
	normalizedUnit := n.normalizeAWSUnit(unit)

	// Extract resource specifications
	resourceSpecs := n.extractAWSResourceSpecs(attributes, serviceMapping.NormalizedServiceType)

	// Create resource name
	resourceName := n.createAWSResourceName(attributes, serviceMapping.NormalizedServiceType)

	// Extract pricing details for reserved instances
	pricingDetails := n.extractAWSPricingDetails(termAttributes, pricingModel)

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
		ResourceDescription: &description,
		ResourceSpecs:       resourceSpecs,
		PricePerUnit:        pricePerUnit,
		Unit:                normalizedUnit,
		Currency:            currency,
		PricingModel:        pricingModel,
		PricingDetails:      pricingDetails,
		EffectiveDate:       nil, // AWS doesn't typically provide this
		MinimumCommitment:   1,
		AWSRawID:            &rawDataID,
	}

	return &record, nil
}

// extractAWSResourceSpecs extracts and normalizes resource specifications from AWS attributes
func (n *AWSNormalizer) extractAWSResourceSpecs(attributes map[string]interface{}, serviceType string) database.ResourceSpecs {
	specs := database.ResourceSpecs{}

	// Common fields
	if vcpuStr, ok := attributes["vcpu"].(string); ok {
		if vcpu, err := strconv.Atoi(vcpuStr); err == nil {
			specs.VCPU = &vcpu
		}
	}

	if memoryStr, ok := attributes["memory"].(string); ok {
		// Parse memory like "4 GiB" or "3.75 GiB"
		memoryStr = strings.ReplaceAll(memoryStr, " GiB", "")
		memoryStr = strings.ReplaceAll(memoryStr, " GB", "")
		if memory, err := strconv.ParseFloat(memoryStr, 64); err == nil {
			specs.MemoryGB = &memory
		}
	}

	if storageStr, ok := attributes["storage"].(string); ok && storageStr != "NA" && storageStr != "EBS only" {
		// Parse storage like "1 x 150 SSD" or "2 x 300 SSD"
		if strings.Contains(storageStr, "SSD") || strings.Contains(storageStr, "HDD") {
			storageType := "ssd"
			if strings.Contains(storageStr, "HDD") {
				storageType = "hdd"
			}
			specs.StorageType = &storageType

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
						specs.StorageGB = &size
						break
					}
				}
			}
		}
	}

	if networkPerf, ok := attributes["networkPerformance"].(string); ok {
		specs.NetworkPerformance = &networkPerf
	}

	if instanceType, ok := attributes["instanceType"].(string); ok {
		// Extract additional specs from instance type
		if strings.Contains(instanceType, "t2") || strings.Contains(instanceType, "t3") || strings.Contains(instanceType, "t4g") {
			burstable := true
			specs.Burstable = &burstable
		}

		// Extract processor info
		if procType, ok := attributes["processorType"].(string); ok {
			specs.ProcessorType = &procType
		}

		if procFeatures, ok := attributes["physicalProcessor"].(string); ok {
			features := []string{procFeatures}
			specs.ProcessorFeatures = features
		}
	}

	// GPU specifications for GPU instances
	if gpu, ok := attributes["gpu"].(string); ok && gpu != "NA" {
		if gpuCount, err := strconv.Atoi(gpu); err == nil {
			specs.GPUCount = &gpuCount
		}
	}

	return specs
}

// createAWSResourceName creates a standardized resource name from AWS attributes
func (n *AWSNormalizer) createAWSResourceName(attributes map[string]interface{}, serviceType string) string {
	switch serviceType {
	case "Virtual Machines":
		if instanceType, ok := attributes["instanceType"].(string); ok {
			return instanceType
		}
	case "Serverless Functions":
		// Lambda pricing is usually by request/duration
		if arch, ok := attributes["architecture"].(string); ok {
			return fmt.Sprintf("Lambda (%s)", arch)
		}
		return "Lambda"
	case "Serverless Containers":
		return "Fargate"
	}

	// Fallback to instance type or service name
	if instanceType, ok := attributes["instanceType"].(string); ok {
		return instanceType
	}
	if serviceName, ok := attributes["serviceName"].(string); ok {
		return serviceName
	}

	return "Unknown"
}

// extractAWSPricingDetails extracts additional pricing details for reserved instances
func (n *AWSNormalizer) extractAWSPricingDetails(termAttributes map[string]interface{}, pricingModel string) database.PricingDetails {
	details := database.PricingDetails{}

	if termAttributes == nil {
		return details
	}

	// Extract lease contract length
	if leaseLength, ok := termAttributes["LeaseContractLength"].(string); ok {
		details.TermLength = &leaseLength
	}

	// Extract purchase option
	if purchaseOption, ok := termAttributes["PurchaseOption"].(string); ok {
		details.PaymentOption = &purchaseOption
	}

	// Note: Upfront costs and hourly rates would need to be calculated from
	// multiple price dimensions in the AWS pricing data, which is complex
	// For now, we'll leave these as nil and calculate them in a separate process

	return details
}

// normalizeAWSUnit normalizes AWS pricing units to standardized units
func (n *AWSNormalizer) normalizeAWSUnit(unit string) string {
	unit = strings.ToLower(strings.TrimSpace(unit))

	switch unit {
	case "hrs", "hour", "hours":
		return database.UnitHour
	case "gb-mo", "gb-month":
		return database.UnitGBMonth
	case "requests", "request":
		return database.UnitRequest
	case "1m requests", "million requests":
		return database.UnitMillionRequests
	case "gb", "gigabyte":
		return database.UnitGB
	case "tb", "terabyte":
		return database.UnitTB
	case "instances", "instance":
		return database.UnitInstance
	default:
		// Return original unit if no mapping found
		return unit
	}
}