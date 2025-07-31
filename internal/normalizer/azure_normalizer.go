package normalizer

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/raulc0399/cpc/internal/database"
)

// AzureNormalizer handles normalization of Azure pricing data
type AzureNormalizer struct {
	db *database.DB
}

// NewAzureNormalizer creates a new Azure pricing normalizer
func NewAzureNormalizer(db *database.DB) *AzureNormalizer {
	return &AzureNormalizer{db: db}
}

// NormalizeAzurePricing normalizes raw Azure pricing data into the standardized format
func (n *AzureNormalizer) NormalizeAzurePricing(input database.NormalizationInput) (*database.NormalizationResult, error) {
	log.Printf("🔄 Normalizing Azure pricing data for service: %s, region: %s", input.ServiceCode, input.Region)

	// Azure raw data is typically an array of pricing items
	var azureItems []map[string]interface{}
	if err := json.Unmarshal(input.RawData, &azureItems); err != nil {
		return &database.NormalizationResult{
			Success:      false,
			ErrorCount:   1,
			Errors:       []string{fmt.Sprintf("failed to unmarshal Azure pricing JSON: %v", err)},
		}, nil
	}

	var normalizedRecords []database.NormalizedPricing
	var errors []string
	var skippedCount int

	for _, item := range azureItems {
		serviceName, _ := item["serviceName"].(string)
		
		// Get service mapping
		serviceMapping, err := n.db.GetServiceMappingByProvider(database.ProviderAzure, serviceName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to get service mapping for %s: %v", serviceName, err))
			continue
		}
		if serviceMapping == nil {
			log.Printf("⚠️ No service mapping found for Azure service: %s", serviceName)
			skippedCount++
			continue
		}

		// Get Azure region from the item
		azureRegion, _ := item["armRegionName"].(string)
		if azureRegion == "" {
			skippedCount++
			continue
		}

		// Get normalized region
		normalizedRegion, err := n.db.GetNormalizedRegionByProvider(database.ProviderAzure, azureRegion)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to get normalized region for %s: %v", azureRegion, err))
			continue
		}
		if normalizedRegion == nil {
			log.Printf("⚠️ No normalized region found for Azure region: %s", azureRegion)
			skippedCount++
			continue
		}

		// Create normalized record
		record, err := n.createNormalizedRecord(item, serviceMapping, normalizedRegion, input.RawDataID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to create normalized record: %v", err))
			continue
		}

		if record != nil {
			normalizedRecords = append(normalizedRecords, *record)
		} else {
			skippedCount++
		}
	}

	log.Printf("✅ Normalized %d Azure pricing records, skipped %d", len(normalizedRecords), skippedCount)

	return &database.NormalizationResult{
		Success:           len(normalizedRecords) > 0,
		NormalizedRecords: normalizedRecords,
		SkippedCount:      skippedCount,
		ErrorCount:        len(errors),
		Errors:            errors,
	}, nil
}

// createNormalizedRecord creates a single normalized pricing record from Azure data
func (n *AzureNormalizer) createNormalizedRecord(
	item map[string]interface{},
	serviceMapping *database.ServiceMapping,
	normalizedRegion *database.NormalizedRegion,
	rawDataID int,
) (*database.NormalizedPricing, error) {
	// Extract Azure pricing fields
	serviceName, _ := item["serviceName"].(string)
	productName, _ := item["productName"].(string)
	skuName, _ := item["skuName"].(string)
	meterName, _ := item["meterName"].(string)
	retailPrice, ok := item["retailPrice"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid retailPrice")
	}
	unitOfMeasure, _ := item["unitOfMeasure"].(string)
	currency, _ := item["currencyCode"].(string)
	if currency == "" {
		currency = "USD"
	}
	armRegionName, _ := item["armRegionName"].(string)
	priceType, _ := item["type"].(string)

	// Skip zero-cost items (often indicates missing data or free tier)
	if retailPrice == 0 {
		return nil, nil
	}

	// Determine pricing model
	pricingModel := n.determineAzurePricingModel(priceType, productName, skuName)

	// Normalize the unit
	normalizedUnit := n.normalizeAzureUnit(unitOfMeasure)

	// Extract resource specifications
	resourceSpecs := n.extractAzureResourceSpecs(productName, skuName, meterName, serviceMapping.NormalizedServiceType)

	// Create resource name
	resourceName := n.createAzureResourceName(productName, skuName, serviceMapping.NormalizedServiceType)

	// Create the normalized record
	record := database.NormalizedPricing{
		Provider:            database.ProviderAzure,
		ProviderServiceCode: serviceName,
		ProviderSKU:         &skuName,
		ServiceMappingID:    &serviceMapping.ID,
		ServiceCategory:     serviceMapping.ServiceCategory,
		ServiceFamily:       serviceMapping.ServiceFamily,
		ServiceType:         serviceMapping.NormalizedServiceType,
		RegionID:            &normalizedRegion.ID,
		NormalizedRegion:    normalizedRegion.NormalizedCode,
		ProviderRegion:      armRegionName,
		ResourceName:        resourceName,
		ResourceDescription: &meterName,
		ResourceSpecs:       resourceSpecs,
		PricePerUnit:        retailPrice,
		Unit:                normalizedUnit,
		Currency:            currency,
		PricingModel:        pricingModel,
		PricingDetails:      database.PricingDetails{}, // Azure doesn't provide detailed term info in retail API
		MinimumCommitment:   1,
		AzureRawID:          &rawDataID,
	}

	return &record, nil
}

// extractAzureResourceSpecs extracts and normalizes resource specifications from Azure fields
func (n *AzureNormalizer) extractAzureResourceSpecs(productName, skuName, meterName, serviceType string) database.ResourceSpecs {
	specs := database.ResourceSpecs{}

	switch serviceType {
	case "Virtual Machines":
		n.parseAzureVMSpecs(&specs, skuName, productName)
	case "Serverless Functions":
		n.parseAzureFunctionSpecs(&specs, meterName)
	case "Serverless Containers":
		n.parseAzureContainerSpecs(&specs, meterName)
	case "Managed Kubernetes":
		n.parseAzureAKSSpecs(&specs, meterName)
	}

	return specs
}

// parseAzureVMSpecs parses VM specifications from Azure VM SKU names
func (n *AzureNormalizer) parseAzureVMSpecs(specs *database.ResourceSpecs, skuName, productName string) {
	// Azure VM SKU format examples:
	// Standard_D2s_v3 = D-series, 2 vCPU, s=SSD, v3=version
	// Standard_B1ms = B-series (burstable), 1 vCPU, m=memory optimized, s=SSD
	// Standard_NC6s_v3 = N-series (GPU), C=compute optimized, 6 vCPU, s=SSD

	// Extract vCPU count from SKU name
	re := regexp.MustCompile(`Standard_[A-Z]+(\d+)`)
	matches := re.FindStringSubmatch(skuName)
	if len(matches) > 1 {
		if vcpu, err := strconv.Atoi(matches[1]); err == nil {
			specs.VCPU = &vcpu
		}
	}

	// Memory calculation based on Azure VM series patterns
	if specs.VCPU != nil {
		vcpu := *specs.VCPU
		var memoryGB float64

		// Series-specific memory calculations
		if strings.Contains(skuName, "_D") && strings.Contains(skuName, "s_v") {
			// D-series: 4GB per vCPU
			memoryGB = float64(vcpu) * 4
		} else if strings.Contains(skuName, "_B") {
			// B-series (burstable): varies, but roughly 1-4GB per vCPU
			burstable := true
			specs.Burstable = &burstable
			memoryGB = float64(vcpu) * 2 // Average
		} else if strings.Contains(skuName, "_F") {
			// F-series (compute optimized): 2GB per vCPU
			memoryGB = float64(vcpu) * 2
		} else if strings.Contains(skuName, "_E") {
			// E-series (memory optimized): 8GB per vCPU
			memoryGB = float64(vcpu) * 8
		} else if strings.Contains(skuName, "_M") {
			// M-series (memory optimized): 28-29GB per vCPU
			memoryGB = float64(vcpu) * 28
		} else if strings.Contains(skuName, "_G") {
			// G-series (memory optimized): 14GB per vCPU
			memoryGB = float64(vcpu) * 14
		} else if strings.Contains(skuName, "_N") {
			// N-series (GPU): 6GB per vCPU + GPU
			memoryGB = float64(vcpu) * 6
			
			// GPU count estimation (rough)
			if strings.Contains(skuName, "_NC6") || strings.Contains(skuName, "_NV6") {
				gpuCount := 1
				specs.GPUCount = &gpuCount
			} else if strings.Contains(skuName, "_NC12") || strings.Contains(skuName, "_NV12") {
				gpuCount := 2
				specs.GPUCount = &gpuCount
			} else if strings.Contains(skuName, "_NC24") || strings.Contains(skuName, "_NV24") {
				gpuCount := 4
				specs.GPUCount = &gpuCount
			}
		} else {
			// Default: 4GB per vCPU
			memoryGB = float64(vcpu) * 4
		}

		if memoryGB > 0 {
			specs.MemoryGB = &memoryGB
		}
	}

	// Storage type
	if strings.Contains(skuName, "s_") || strings.Contains(skuName, "_s") {
		storageType := "ssd"
		specs.StorageType = &storageType
	}

	// Network performance (rough estimation)
	if specs.VCPU != nil {
		vcpu := *specs.VCPU
		var networkPerf string
		if vcpu <= 2 {
			networkPerf = "low"
		} else if vcpu <= 8 {
			networkPerf = "moderate"
		} else if vcpu <= 16 {
			networkPerf = "high"
		} else {
			networkPerf = "very high"
		}
		specs.NetworkPerformance = &networkPerf
	}

	// Architecture (rough estimation)
	if strings.Contains(productName, "v5") || strings.Contains(skuName, "v5") {
		arch := "x64 (Ice Lake)"
		specs.Architecture = &arch
	} else if strings.Contains(productName, "v4") || strings.Contains(skuName, "v4") {
		arch := "x64 (Cascade Lake)"
		specs.Architecture = &arch
	} else if strings.Contains(productName, "v3") || strings.Contains(skuName, "v3") {
		arch := "x64 (Skylake)"
		specs.Architecture = &arch
	}
}

// parseAzureFunctionSpecs parses Azure Functions specifications
func (n *AzureNormalizer) parseAzureFunctionSpecs(specs *database.ResourceSpecs, meterName string) {
	// Azure Functions is serverless, so no fixed vCPU/memory
	// We can extract some info from meter name if needed
	if strings.Contains(strings.ToLower(meterName), "premium") {
		arch := "Premium"
		specs.Architecture = &arch
	}
}

// parseAzureContainerSpecs parses Azure Container Instance specifications
func (n *AzureNormalizer) parseAzureContainerSpecs(specs *database.ResourceSpecs, meterName string) {
	// Extract vCPU and memory from meter name like "1 vCPU 1.5 GB"
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*vCPU\s*(\d+(?:\.\d+)?)\s*GB`)
	matches := re.FindStringSubmatch(meterName)
	if len(matches) > 2 {
		if vcpu, err := strconv.ParseFloat(matches[1], 64); err == nil {
			vcpuInt := int(vcpu)
			specs.VCPU = &vcpuInt
		}
		if memory, err := strconv.ParseFloat(matches[2], 64); err == nil {
			specs.MemoryGB = &memory
		}
	}
}

// parseAzureAKSSpecs parses Azure Kubernetes Service specifications
func (n *AzureNormalizer) parseAzureAKSSpecs(specs *database.ResourceSpecs, meterName string) {
	// AKS node specifications would be in the meter name
	// This is complex as it depends on the underlying VM SKU
	// For now, we'll leave this simple
}

// createAzureResourceName creates a standardized resource name from Azure fields
func (n *AzureNormalizer) createAzureResourceName(productName, skuName, serviceType string) string {
	switch serviceType {
	case "Virtual Machines":
		return skuName
	case "Serverless Functions":
		return "Azure Functions"
	case "Serverless Containers":
		return "Container Instances"
	case "Managed Kubernetes":
		return "AKS"
	}

	// Fallback to SKU name or product name
	if skuName != "" {
		return skuName
	}
	return productName
}

// determineAzurePricingModel determines the pricing model from Azure fields
func (n *AzureNormalizer) determineAzurePricingModel(priceType, productName, skuName string) string {
	// Convert to lowercase for easier matching
	priceTypeLower := strings.ToLower(priceType)
	productLower := strings.ToLower(productName)
	skuLower := strings.ToLower(skuName)

	// Check for spot instances
	if strings.Contains(productLower, "spot") || strings.Contains(skuLower, "spot") {
		return database.PricingModelSpot
	}

	// Check for reserved instances
	if strings.Contains(priceTypeLower, "reservation") || 
	   strings.Contains(productLower, "reservation") ||
	   strings.Contains(productLower, "reserved") {
		// Try to determine term length
		if strings.Contains(productLower, "3 year") || strings.Contains(productLower, "3year") {
			return database.PricingModelReserved3Yr
		}
		return database.PricingModelReserved1Yr
	}

	// Check for savings plans
	if strings.Contains(productLower, "savings") {
		return database.PricingModelSavingsPlan
	}

	// Default to on-demand
	return database.PricingModelOnDemand
}

// normalizeAzureUnit normalizes Azure pricing units to standardized units
func (n *AzureNormalizer) normalizeAzureUnit(unit string) string {
	unit = strings.ToLower(strings.TrimSpace(unit))

	switch unit {
	case "1 hour", "hour", "hours":
		return database.UnitHour
	case "1 gb/month", "gb/month", "gb-month":
		return database.UnitGBMonth
	case "1m requests", "1 million requests", "million requests":
		return database.UnitMillionRequests
	case "10k requests", "10000 requests":
		return database.UnitRequest // Convert to base unit
	case "1 gb", "gb":
		return database.UnitGB
	case "1 tb", "tb":
		return database.UnitTB
	case "1 transaction", "transaction", "transactions":
		return database.UnitTransaction
	default:
		// Return original unit if no mapping found
		return unit
	}
}