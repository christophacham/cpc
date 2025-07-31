package normalizer

import (
	"context"
	"fmt"
	"strconv"

	"github.com/raulc0399/cpc/internal/database"
)

// AWSNormalizerV2 handles normalization of AWS pricing data using base normalizer
type AWSNormalizerV2 struct {
	*BaseNormalizer
	specExtractor *AWSResourceSpecExtractor
}

// NewAWSNormalizerV2 creates a new AWS pricing normalizer
func NewAWSNormalizerV2(
	serviceMappingRepo ServiceMappingRepository,
	regionMappingRepo RegionMappingRepository,
	unitNormalizer UnitNormalizer,
	validator *InputValidator,
	logger Logger,
) *AWSNormalizerV2 {
	return &AWSNormalizerV2{
		BaseNormalizer: NewBaseNormalizer(
			serviceMappingRepo,
			regionMappingRepo,
			unitNormalizer,
			validator,
			logger,
		),
		specExtractor: NewAWSResourceSpecExtractor(),
	}
}

// GetSupportedProvider returns the provider this normalizer supports
func (n *AWSNormalizerV2) GetSupportedProvider() string {
	return database.ProviderAWS
}

// ValidateInput validates the input data before normalization
func (n *AWSNormalizerV2) ValidateInput(input database.NormalizationInput) error {
	if err := n.ValidateCommonInput(context.Background(), input); err != nil {
		return err
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

// NormalizePricing normalizes raw AWS pricing data
func (n *AWSNormalizerV2) NormalizePricing(ctx context.Context, input database.NormalizationInput) (*database.NormalizationResult, error) {
	n.logger.Info("Starting AWS pricing normalization",
		Field{"service", input.ServiceCode},
		Field{"region", input.Region},
	)

	// Validate input
	if err := n.ValidateInput(input); err != nil {
		return n.CreateErrorResult("validation failed", err), nil
	}

	// Parse AWS product JSON
	var awsProduct AWSProduct
	if err := n.ParseJSONData(input.RawData, &awsProduct); err != nil {
		return n.CreateErrorResult("failed to parse AWS product", err), nil
	}

	// Validate product structure
	if err := n.validateAWSProduct(&awsProduct); err != nil {
		return n.CreateErrorResult("invalid AWS product structure", err), nil
	}

	// Get normalization context
	normCtx, err := n.GetNormalizationContext(ctx, input)
	if err != nil {
		return n.CreateErrorResult("failed to get normalization context", err), nil
	}
	if normCtx == nil {
		return n.CreateSkippedResult("service or region not mapped", 1), nil
	}

	// Process pricing terms
	result := n.processPricingTerms(ctx, &awsProduct, normCtx)

	n.logger.Info("Completed AWS pricing normalization",
		Field{"service", input.ServiceCode},
		Field{"region", input.Region},
		Field{"recordCount", len(result.NormalizedRecords)},
		Field{"errorCount", result.ErrorCount},
		Field{"skippedCount", result.SkippedCount},
	)

	return result, nil
}

// AWSProduct represents the structure of AWS pricing data
type AWSProduct struct {
	Product struct {
		SKU        string                 `json:"sku"`
		Attributes map[string]interface{} `json:"attributes"`
	} `json:"product"`
	Terms struct {
		OnDemand map[string]AWSTermData `json:"OnDemand,omitempty"`
		Reserved map[string]AWSTermData `json:"Reserved,omitempty"`
	} `json:"terms"`
}

// AWSTermData represents AWS pricing term data
type AWSTermData struct {
	OfferTermCode    string                          `json:"offerTermCode"`
	SKU              string                          `json:"sku"`
	PriceDimensions  map[string]AWSPriceDimension   `json:"priceDimensions"`
	TermAttributes   map[string]interface{}         `json:"termAttributes,omitempty"`
}

// AWSPriceDimension represents AWS price dimension
type AWSPriceDimension struct {
	Description  string                 `json:"description"`
	Unit         string                 `json:"unit"`
	PricePerUnit map[string]string     `json:"pricePerUnit"`
	AppliesTo    []string              `json:"appliesTo,omitempty"`
}

// validateAWSProduct validates the AWS product structure
func (n *AWSNormalizerV2) validateAWSProduct(product *AWSProduct) error {
	if product.Product.SKU == "" {
		return fmt.Errorf("missing product SKU")
	}

	// Check if we have any pricing terms
	hasOnDemand := len(product.Terms.OnDemand) > 0
	hasReserved := len(product.Terms.Reserved) > 0

	if !hasOnDemand && !hasReserved {
		return fmt.Errorf("no pricing terms found")
	}

	return nil
}

// processPricingTerms processes all pricing terms in the AWS product
func (n *AWSNormalizerV2) processPricingTerms(
	ctx context.Context,
	awsProduct *AWSProduct,
	normCtx *NormalizationContext,
) *database.NormalizationResult {
	var normalizedRecords []database.NormalizedPricing
	var errors []string
	skippedCount := 0

	// Process On-Demand pricing
	for _, termData := range awsProduct.Terms.OnDemand {
		records, errs, skipped := n.processTermData(
			ctx, &termData, database.PricingModelOnDemand,
			awsProduct.Product.Attributes, normCtx,
		)
		normalizedRecords = append(normalizedRecords, records...)
		errors = append(errors, errs...)
		skippedCount += skipped
	}

	// Process Reserved pricing
	for _, termData := range awsProduct.Terms.Reserved {
		pricingModel := n.determineReservedPricingModel(termData.TermAttributes)
		records, errs, skipped := n.processTermData(
			ctx, &termData, pricingModel,
			awsProduct.Product.Attributes, normCtx,
		)
		normalizedRecords = append(normalizedRecords, records...)
		errors = append(errors, errs...)
		skippedCount += skipped
	}

	return &database.NormalizationResult{
		Success:           len(normalizedRecords) > 0,
		NormalizedRecords: normalizedRecords,
		SkippedCount:      skippedCount,
		ErrorCount:        len(errors),
		Errors:            errors,
	}
}

// processTermData processes a single term's data
func (n *AWSNormalizerV2) processTermData(
	ctx context.Context,
	termData *AWSTermData,
	pricingModel string,
	attributes map[string]interface{},
	normCtx *NormalizationContext,
) ([]database.NormalizedPricing, []string, int) {
	var records []database.NormalizedPricing
	var errors []string
	skippedCount := 0

	for _, dimension := range termData.PriceDimensions {
		// Extract pricing info
		priceInfo, err := n.extractPricingFromDimension(&dimension)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to extract pricing: %v", err))
			continue
		}

		// Skip if price is zero
		if priceInfo.PricePerUnit == 0 {
			skippedCount++
			continue
		}

		// Extract resource specs
		resourceSpecs, err := n.specExtractor.ExtractResourceSpecs(
			database.ProviderAWS,
			normCtx.ServiceMapping.NormalizedServiceType,
			attributes,
		)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to extract resource specs: %v", err))
			continue
		}

		// Create resource name
		resourceName := n.createResourceName(attributes, normCtx.ServiceMapping.NormalizedServiceType)

		// Extract pricing details
		pricingDetails := n.extractPricingDetails(termData.TermAttributes, pricingModel)

		// Create normalized record
		record, err := n.CreateNormalizedRecord(
			ctx, normCtx, *priceInfo, resourceSpecs,
			resourceName, pricingModel, pricingDetails,
		)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		if record != nil {
			// Add provider SKU
			sku := termData.SKU
			record.ProviderSKU = &sku
			records = append(records, *record)
		}
	}

	return records, errors, skippedCount
}

// extractPricingFromDimension extracts pricing info from a price dimension
func (n *AWSNormalizerV2) extractPricingFromDimension(dimension *AWSPriceDimension) (*PricingInfo, error) {
	if len(dimension.PricePerUnit) == 0 {
		return nil, fmt.Errorf("no price information in dimension")
	}

	// Get first currency (usually USD)
	var currency string
	var priceStr string
	for curr, price := range dimension.PricePerUnit {
		currency = curr
		priceStr = price
		break
	}

	// Parse price
	pricePerUnit, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price %s: %w", priceStr, err)
	}

	return &PricingInfo{
		PricePerUnit: pricePerUnit,
		Unit:         dimension.Unit,
		Currency:     currency,
		Description:  dimension.Description,
	}, nil
}

// createResourceName creates a standardized resource name
func (n *AWSNormalizerV2) createResourceName(attributes map[string]interface{}, serviceType string) string {
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
func (n *AWSNormalizerV2) extractPricingDetails(termAttributes map[string]interface{}, pricingModel string) database.PricingDetails {
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

// determineReservedPricingModel determines the specific reserved pricing model
func (n *AWSNormalizerV2) determineReservedPricingModel(termAttributes map[string]interface{}) string {
	if termAttributes == nil {
		return database.PricingModelReserved1Yr
	}

	if leaseLength, ok := termAttributes["LeaseContractLength"].(string); ok {
		if leaseLength == "3yr" {
			return database.PricingModelReserved3Yr
		}
	}

	return database.PricingModelReserved1Yr
}