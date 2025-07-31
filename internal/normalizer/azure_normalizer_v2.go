package normalizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/raulc0399/cpc/internal/database"
	"strconv"
	"regexp"
)

// AzureNormalizerV2 handles normalization of Azure pricing data using base normalizer
type AzureNormalizerV2 struct {
	*BaseNormalizer
	specExtractor *AzureResourceSpecExtractor
}

// NewAzureNormalizerV2 creates a new Azure pricing normalizer
func NewAzureNormalizerV2(
	serviceMappingRepo ServiceMappingRepository,
	regionMappingRepo RegionMappingRepository,
	unitNormalizer UnitNormalizer,
	validator *InputValidator,
	logger Logger,
) *AzureNormalizerV2 {
	return &AzureNormalizerV2{
		BaseNormalizer: NewBaseNormalizer(
			serviceMappingRepo,
			regionMappingRepo,
			unitNormalizer,
			validator,
			logger,
		),
		specExtractor: NewAzureResourceSpecExtractor(),
	}
}

// GetSupportedProvider returns the provider this normalizer supports
func (n *AzureNormalizerV2) GetSupportedProvider() string {
	return database.ProviderAzure
}

// ValidateInput validates the input data before normalization
func (n *AzureNormalizerV2) ValidateInput(input database.NormalizationInput) error {
	if err := n.ValidateCommonInput(context.Background(), input); err != nil {
		return err
	}

	if input.Provider != database.ProviderAzure {
		return NormalizationError{
			Provider:    input.Provider,
			ServiceCode: input.ServiceCode,
			Region:      input.Region,
			Message:     "unsupported provider for Azure normalizer",
		}
	}

	return nil
}

// NormalizePricing normalizes raw Azure pricing data
func (n *AzureNormalizerV2) NormalizePricing(ctx context.Context, input database.NormalizationInput) (*database.NormalizationResult, error) {
	n.logger.Info("Starting Azure pricing normalization",
		Field{"service", input.ServiceCode},
		Field{"region", input.Region},
	)

	// Validate input
	if err := n.ValidateInput(input); err != nil {
		return n.CreateErrorResult("validation failed", err), nil
	}

	// Parse Azure pricing JSON
	var azurePricing AzurePricing
	if err := n.ParseJSONData(input.RawData, &azurePricing); err != nil {
		return n.CreateErrorResult("failed to parse Azure pricing", err), nil
	}

	// Validate pricing structure
	if err := n.validateAzurePricing(&azurePricing); err != nil {
		return n.CreateErrorResult("invalid Azure pricing structure", err), nil
	}

	// Get normalization context
	normCtx, err := n.GetNormalizationContext(ctx, input)
	if err != nil {
		return n.CreateErrorResult("failed to get normalization context", err), nil
	}
	if normCtx == nil {
		return n.CreateSkippedResult("service or region not mapped", 1), nil
	}

	// Process Azure pricing item
	result := n.processPricingItem(ctx, &azurePricing, normCtx)

	n.logger.Info("Completed Azure pricing normalization",
		Field{"service", input.ServiceCode},
		Field{"region", input.Region},
		Field{"recordCount", len(result.NormalizedRecords)},
		Field{"errorCount", result.ErrorCount},
		Field{"skippedCount", result.SkippedCount},
	)

	return result, nil
}

// AzurePricing represents the structure of Azure pricing data
type AzurePricing struct {
	CurrencyCode     string  `json:"currencyCode"`
	TierMinimumUnits int     `json:"tierMinimumUnits"`
	RetailPrice      float64 `json:"retailPrice"`
	UnitPrice        float64 `json:"unitPrice"`
	ArmRegionName    string  `json:"armRegionName"`
	Location         string  `json:"location"`
	EffectiveDate    string  `json:"effectiveDate"`
	MeterID          string  `json:"meterId"`
	MeterName        string  `json:"meterName"`
	ProductID        string  `json:"productId"`
	ProductName      string  `json:"productName"`
	SKUID            string  `json:"skuId"`
	SKUName          string  `json:"skuName"`
	ServiceName      string  `json:"serviceName"`
	ServiceID        string  `json:"serviceId"`
	ServiceFamily    string  `json:"serviceFamily"`
	UnitOfMeasure    string  `json:"unitOfMeasure"`
	Type             string  `json:"type"`
	IsPrimaryRegion  bool    `json:"isPrimaryMeterRegion"`
	ArmSKUName       string  `json:"armSkuName"`
}

// validateAzurePricing validates the Azure pricing structure
func (n *AzureNormalizerV2) validateAzurePricing(pricing *AzurePricing) error {
	if pricing.MeterID == "" {
		return fmt.Errorf("missing meter ID")
	}

	if pricing.ServiceName == "" {
		return fmt.Errorf("missing service name")
	}

	if pricing.UnitOfMeasure == "" {
		return fmt.Errorf("missing unit of measure")
	}

	return nil
}

// processPricingItem processes a single Azure pricing item
func (n *AzureNormalizerV2) processPricingItem(
	ctx context.Context,
	azurePricing *AzurePricing,
	normCtx *NormalizationContext,
) *database.NormalizationResult {
	var normalizedRecords []database.NormalizedPricing
	var errors []string
	skippedCount := 0

	// Extract pricing info
	priceInfo := n.extractPricingFromAzureItem(azurePricing)

	// Skip if price is zero
	if priceInfo.PricePerUnit == 0 {
		skippedCount++
		return &database.NormalizationResult{
			Success:      false,
			SkippedCount: skippedCount,
		}
	}

	// Extract resource specs
	resourceSpecs, err := n.specExtractor.ExtractResourceSpecs(
		database.ProviderAzure,
		normCtx.ServiceMapping.NormalizedServiceType,
		n.createAttributesFromAzureItem(azurePricing),
	)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to extract resource specs: %v", err))
		return &database.NormalizationResult{
			Success:    false,
			ErrorCount: len(errors),
			Errors:     errors,
		}
	}

	// Create resource name
	resourceName := n.createResourceName(azurePricing, normCtx.ServiceMapping.NormalizedServiceType)

	// Determine pricing model
	pricingModel := n.determinePricingModel(azurePricing)

	// Extract pricing details
	pricingDetails := n.extractPricingDetails(azurePricing, pricingModel)

	// Create normalized record
	record, err := n.CreateNormalizedRecord(
		ctx, normCtx, *priceInfo, resourceSpecs,
		resourceName, pricingModel, pricingDetails,
	)
	if err != nil {
		errors = append(errors, err.Error())
		return &database.NormalizationResult{
			Success:    false,
			ErrorCount: len(errors),
			Errors:     errors,
		}
	}

	if record != nil {
		// Add Azure-specific fields
		record.ProviderSKU = &azurePricing.SKUID
		normalizedRecords = append(normalizedRecords, *record)
	}

	return &database.NormalizationResult{
		Success:           len(normalizedRecords) > 0,
		NormalizedRecords: normalizedRecords,
		SkippedCount:      skippedCount,
		ErrorCount:        len(errors),
		Errors:            errors,
	}
}

// extractPricingFromAzureItem extracts pricing info from Azure pricing item
func (n *AzureNormalizerV2) extractPricingFromAzureItem(pricing *AzurePricing) *PricingInfo {
	// Use retail price if available, otherwise unit price
	price := pricing.RetailPrice
	if price == 0 {
		price = pricing.UnitPrice
	}

	return &PricingInfo{
		PricePerUnit: price,
		Unit:         pricing.UnitOfMeasure,
		Currency:     pricing.CurrencyCode,
		Description:  fmt.Sprintf("%s - %s", pricing.ProductName, pricing.MeterName),
	}
}

// createAttributesFromAzureItem creates attributes map from Azure pricing item
func (n *AzureNormalizerV2) createAttributesFromAzureItem(pricing *AzurePricing) map[string]interface{} {
	return map[string]interface{}{
		"productName":    pricing.ProductName,
		"skuName":        pricing.SKUName,
		"armSkuName":     pricing.ArmSKUName,
		"meterName":      pricing.MeterName,
		"serviceName":    pricing.ServiceName,
		"serviceFamily":  pricing.ServiceFamily,
		"location":       pricing.Location,
		"armRegionName":  pricing.ArmRegionName,
		"unitOfMeasure":  pricing.UnitOfMeasure,
		"type":           pricing.Type,
	}
}

// createResourceName creates a standardized resource name for Azure
func (n *AzureNormalizerV2) createResourceName(pricing *AzurePricing, serviceType string) string {
	switch serviceType {
	case "Virtual Machines":
		if pricing.ArmSKUName != "" {
			return pricing.ArmSKUName
		}
		return pricing.SKUName
	case "Serverless Functions":
		return "Azure Functions"
	case "Storage":
		if strings.Contains(pricing.MeterName, "Hot") {
			return "Hot Storage"
		} else if strings.Contains(pricing.MeterName, "Cool") {
			return "Cool Storage"
		} else if strings.Contains(pricing.MeterName, "Archive") {
			return "Archive Storage"
		}
		return "Storage"
	case "Databases":
		if pricing.ArmSKUName != "" {
			return pricing.ArmSKUName
		}
		return pricing.SKUName
	}

	// Fallback logic
	if pricing.ArmSKUName != "" {
		return pricing.ArmSKUName
	}
	if pricing.SKUName != "" {
		return pricing.SKUName
	}
	if pricing.ProductName != "" {
		return pricing.ProductName
	}

	return "Unknown"
}

// determinePricingModel determines the pricing model from Azure data
func (n *AzureNormalizerV2) determinePricingModel(pricing *AzurePricing) string {
	// Azure primarily uses consumption-based pricing
	// Reserved instances are typically identified by specific product names or types
	
	productName := strings.ToLower(pricing.ProductName)
	skuName := strings.ToLower(pricing.SKUName)
	
	if strings.Contains(productName, "reserved") || strings.Contains(skuName, "reserved") {
		if strings.Contains(productName, "3 year") || strings.Contains(skuName, "3 year") {
			return database.PricingModelReserved3Yr
		}
		return database.PricingModelReserved1Yr
	}
	
	if strings.Contains(productName, "spot") || strings.Contains(skuName, "spot") {
		return database.PricingModelSpot
	}
	
	return database.PricingModelOnDemand
}

// extractPricingDetails extracts additional pricing details
func (n *AzureNormalizerV2) extractPricingDetails(pricing *AzurePricing, pricingModel string) database.PricingDetails {
	details := database.PricingDetails{}

	// Extract term length for reserved instances
	if strings.Contains(pricingModel, "reserved") {
		productName := strings.ToLower(pricing.ProductName)
		if strings.Contains(productName, "3 year") {
			termLength := "3yr"
			details.TermLength = &termLength
		} else if strings.Contains(productName, "1 year") {
			termLength := "1yr"
			details.TermLength = &termLength
		}
	}

	// Azure typically uses upfront payment for reserved instances
	if strings.Contains(pricingModel, "reserved") {
		paymentOption := "All Upfront"
		details.PaymentOption = &paymentOption
	}

	return details
}

// AzureResourceSpecExtractor extracts resource specifications from Azure pricing data
type AzureResourceSpecExtractor struct{}

// NewAzureResourceSpecExtractor creates a new Azure resource spec extractor
func NewAzureResourceSpecExtractor() *AzureResourceSpecExtractor {
	return &AzureResourceSpecExtractor{}
}

// ExtractResourceSpecs extracts and normalizes resource specifications from Azure attributes
func (e *AzureResourceSpecExtractor) ExtractResourceSpecs(provider, serviceType string, data map[string]interface{}) (database.ResourceSpecs, error) {
	specs := database.ResourceSpecs{}

	// Extract from ARM SKU name patterns (e.g., Standard_D2s_v3)
	if armSKU, ok := data["armSkuName"].(string); ok && armSKU != "" {
		e.extractFromArmSKU(&specs, armSKU)
	}

	// Extract from SKU name patterns (e.g., D2s v3)
	if skuName, ok := data["skuName"].(string); ok && skuName != "" {
		e.extractFromSKUName(&specs, skuName)
	}

	// Extract from meter name (fallback)
	if meterName, ok := data["meterName"].(string); ok && meterName != "" {
		e.extractFromMeterName(&specs, meterName)
	}

	return specs, nil
}

// extractFromArmSKU extracts specs from ARM SKU name (e.g., Standard_D2s_v3)
func (e *AzureResourceSpecExtractor) extractFromArmSKU(specs *database.ResourceSpecs, armSKU string) {
	// Pattern: Standard_[Family][Size][Generation]_[Version]
	// Example: Standard_D2s_v3 = D family, 2 vCPU, s=SSD, v3 generation
	
	// D-series VMs
	if matched, _ := regexp.MatchString(`Standard_D\d+s?_v\d+`, armSKU); matched {
		e.extractDSeriesSpecs(specs, armSKU)
	}
	
	// F-series VMs (compute optimized)
	if matched, _ := regexp.MatchString(`Standard_F\d+s?_v\d+`, armSKU); matched {
		e.extractFSeriesSpecs(specs, armSKU)
	}
	
	// B-series VMs (burstable)
	if matched, _ := regexp.MatchString(`Standard_B\d+m?s`, armSKU); matched {
		e.extractBSeriesSpecs(specs, armSKU)
	}
	
	// E-series VMs (memory optimized)
	if matched, _ := regexp.MatchString(`Standard_E\d+s?_v\d+`, armSKU); matched {
		e.extractESeriesSpecs(specs, armSKU)
	}
}

// extractDSeriesSpecs extracts D-series VM specifications
func (e *AzureResourceSpecExtractor) extractDSeriesSpecs(specs *database.ResourceSpecs, armSKU string) {
	// Extract vCPU count from number in SKU
	re := regexp.MustCompile(`Standard_D(\d+)s?_v\d+`)
	matches := re.FindStringSubmatch(armSKU)
	if len(matches) > 1 {
		if vcpu, err := strconv.Atoi(matches[1]); err == nil {
			specs.VCPU = &vcpu
			// D-series memory ratio is typically 4GB per vCPU
			memoryGB := float64(vcpu * 4)
			specs.MemoryGB = &memoryGB
		}
	}
	
	// Check for SSD storage (s suffix)
	if strings.Contains(armSKU, "s_") {
		storageType := "Premium SSD"
		specs.StorageType = &storageType
	}
}

// extractFSeriesSpecs extracts F-series VM specifications (compute optimized)
func (e *AzureResourceSpecExtractor) extractFSeriesSpecs(specs *database.ResourceSpecs, armSKU string) {
	// Extract vCPU count
	re := regexp.MustCompile(`Standard_F(\d+)s?_v\d+`)
	matches := re.FindStringSubmatch(armSKU)
	if len(matches) > 1 {
		if vcpu, err := strconv.Atoi(matches[1]); err == nil {
			specs.VCPU = &vcpu
			// F-series memory ratio is typically 2GB per vCPU
			memoryGB := float64(vcpu * 2)
			specs.MemoryGB = &memoryGB
		}
	}
	
	if strings.Contains(armSKU, "s_") {
		storageType := "Premium SSD"
		specs.StorageType = &storageType
	}
}

// extractBSeriesSpecs extracts B-series VM specifications (burstable)
func (e *AzureResourceSpecExtractor) extractBSeriesSpecs(specs *database.ResourceSpecs, armSKU string) {
	// B-series has specific mappings
	bSeriesSpecs := map[string]struct{ vcpu int; memory float64 }{
		"Standard_B1ls": {1, 0.5},
		"Standard_B1s":  {1, 1.0},
		"Standard_B1ms": {1, 2.0},
		"Standard_B2s":  {2, 4.0},
		"Standard_B2ms": {2, 8.0},
		"Standard_B4ms": {4, 16.0},
		"Standard_B8ms": {8, 32.0},
	}
	
	if spec, exists := bSeriesSpecs[armSKU]; exists {
		specs.VCPU = &spec.vcpu
		specs.MemoryGB = &spec.memory
	}
}

// extractESeriesSpecs extracts E-series VM specifications (memory optimized)
func (e *AzureResourceSpecExtractor) extractESeriesSpecs(specs *database.ResourceSpecs, armSKU string) {
	// Extract vCPU count
	re := regexp.MustCompile(`Standard_E(\d+)s?_v\d+`)
	matches := re.FindStringSubmatch(armSKU)
	if len(matches) > 1 {
		if vcpu, err := strconv.Atoi(matches[1]); err == nil {
			specs.VCPU = &vcpu
			// E-series memory ratio is typically 8GB per vCPU
			memoryGB := float64(vcpu * 8)
			specs.MemoryGB = &memoryGB
		}
	}
	
	if strings.Contains(armSKU, "s_") {
		storageType := "Premium SSD"
		specs.StorageType = &storageType
	}
}

// extractFromSKUName extracts specs from friendly SKU name (e.g., "D2s v3")
func (e *AzureResourceSpecExtractor) extractFromSKUName(specs *database.ResourceSpecs, skuName string) {
	// Only extract if we don't already have specs from ARM SKU
	if specs.VCPU != nil {
		return
	}
	
	// Pattern matching for common SKU names
	if matched, _ := regexp.MatchString(`[DFBE]\d+s?\s?v\d+`, skuName); matched {
		re := regexp.MustCompile(`([DFBE])(\d+)(s?)\s?v\d+`)
		matches := re.FindStringSubmatch(skuName)
		if len(matches) > 2 {
			family := matches[1]
			if vcpu, err := strconv.Atoi(matches[2]); err == nil {
				specs.VCPU = &vcpu
				
				// Apply family-specific memory ratios
				var memoryRatio float64
				switch family {
				case "D":
					memoryRatio = 4.0
				case "F":
					memoryRatio = 2.0
				case "E":
					memoryRatio = 8.0
				case "B":
					memoryRatio = 2.0 // Approximate for B-series
				default:
					memoryRatio = 4.0
				}
				
				memoryGB := float64(vcpu) * memoryRatio
				specs.MemoryGB = &memoryGB
				
				// Check for premium storage
				if matches[3] == "s" {
					storageType := "Premium SSD"
					specs.StorageType = &storageType
				}
			}
		}
	}
}

// extractFromMeterName extracts specs from meter name as fallback
func (e *AzureResourceSpecExtractor) extractFromMeterName(specs *database.ResourceSpecs, meterName string) {
	// Only extract if we don't already have specs
	if specs.VCPU != nil {
		return
	}
	
	// Look for patterns like "2 vCPU", "4 Core", etc.
	re := regexp.MustCompile(`(\d+)\s*(vCPU|Core|CPU)`)
	matches := re.FindStringSubmatch(meterName)
	if len(matches) > 1 {
		if vcpu, err := strconv.Atoi(matches[1]); err == nil {
			specs.VCPU = &vcpu
		}
	}
	
	// Look for memory patterns like "8 GB", "16GB RAM"
	re = regexp.MustCompile(`(\d+\.?\d*)\s*(GB|GiB)`)
	matches = re.FindStringSubmatch(meterName)
	if len(matches) > 1 {
		if memory, err := strconv.ParseFloat(matches[1], 64); err == nil {
			specs.MemoryGB = &memory
		}
	}
}