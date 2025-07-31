package normalizer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/raulc0399/cpc/internal/database"
)

// BaseNormalizer contains common logic for all pricing normalizers
type BaseNormalizer struct {
	serviceMappingRepo ServiceMappingRepository
	regionMappingRepo  RegionMappingRepository
	unitNormalizer     UnitNormalizer
	validator          *InputValidator
	logger             Logger
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// NewBaseNormalizer creates a new base normalizer
func NewBaseNormalizer(
	serviceMappingRepo ServiceMappingRepository,
	regionMappingRepo RegionMappingRepository,
	unitNormalizer UnitNormalizer,
	validator *InputValidator,
	logger Logger,
) *BaseNormalizer {
	return &BaseNormalizer{
		serviceMappingRepo: serviceMappingRepo,
		regionMappingRepo:  regionMappingRepo,
		unitNormalizer:     unitNormalizer,
		validator:          validator,
		logger:             logger,
	}
}

// NormalizationContext holds common normalization data
type NormalizationContext struct {
	Provider         string
	ServiceMapping   *database.ServiceMapping
	NormalizedRegion *database.NormalizedRegion
	RawDataID        int
	CollectionID     string
}

// ValidateCommonInput performs common input validation
func (n *BaseNormalizer) ValidateCommonInput(ctx context.Context, input database.NormalizationInput) error {
	if err := n.validator.ValidateNormalizationInput(input); err != nil {
		n.logger.Error("Input validation failed",
			Field{"provider", input.Provider},
			Field{"service", input.ServiceCode},
			Field{"error", err},
		)
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// GetNormalizationContext retrieves common normalization dependencies
func (n *BaseNormalizer) GetNormalizationContext(
	ctx context.Context,
	input database.NormalizationInput,
) (*NormalizationContext, error) {
	// Get service mapping
	serviceMapping, err := n.serviceMappingRepo.GetServiceMappingByProvider(ctx, input.Provider, input.ServiceCode)
	if err != nil {
		n.logger.Error("Failed to get service mapping",
			Field{"provider", input.Provider},
			Field{"service", input.ServiceCode},
			Field{"error", err},
		)
		return nil, fmt.Errorf("failed to get service mapping: %w", err)
	}
	if serviceMapping == nil {
		n.logger.Warn("No service mapping found",
			Field{"provider", input.Provider},
			Field{"service", input.ServiceCode},
		)
		return nil, nil // This is a valid case - service not mapped
	}

	// Get normalized region
	normalizedRegion, err := n.regionMappingRepo.GetNormalizedRegionByProvider(ctx, input.Provider, input.Region)
	if err != nil {
		n.logger.Error("Failed to get normalized region",
			Field{"provider", input.Provider},
			Field{"region", input.Region},
			Field{"error", err},
		)
		return nil, fmt.Errorf("failed to get normalized region: %w", err)
	}
	if normalizedRegion == nil {
		n.logger.Warn("No normalized region found",
			Field{"provider", input.Provider},
			Field{"region", input.Region},
		)
		return nil, nil // This is a valid case - region not mapped
	}

	return &NormalizationContext{
		Provider:         input.Provider,
		ServiceMapping:   serviceMapping,
		NormalizedRegion: normalizedRegion,
		RawDataID:        input.RawDataID,
		CollectionID:     input.CollectionID,
	}, nil
}

// CreateNormalizedRecord creates a normalized pricing record with validation
func (n *BaseNormalizer) CreateNormalizedRecord(
	ctx context.Context,
	normCtx *NormalizationContext,
	priceInfo PricingInfo,
	resourceSpecs database.ResourceSpecs,
	resourceName string,
	pricingModel string,
	pricingDetails database.PricingDetails,
) (*database.NormalizedPricing, error) {
	// Skip zero-cost items
	if priceInfo.PricePerUnit == 0 {
		n.logger.Debug("Skipping zero-cost item",
			Field{"resource", resourceName},
			Field{"service", normCtx.ServiceMapping.NormalizedServiceType},
		)
		return nil, nil
	}

	// Normalize unit
	normalizedUnit := n.unitNormalizer.NormalizeUnit(normCtx.Provider, priceInfo.Unit)

	// Determine provider region
	var providerRegion string
	if normCtx.Provider == database.ProviderAWS && normCtx.NormalizedRegion.AWSRegion != nil {
		providerRegion = *normCtx.NormalizedRegion.AWSRegion
	} else if normCtx.Provider == database.ProviderAzure && normCtx.NormalizedRegion.AzureRegion != nil {
		providerRegion = *normCtx.NormalizedRegion.AzureRegion
	}

	// Determine provider service code
	var providerServiceCode string
	if normCtx.ServiceMapping.ProviderServiceCode != nil {
		providerServiceCode = *normCtx.ServiceMapping.ProviderServiceCode
	} else {
		providerServiceCode = normCtx.ServiceMapping.ProviderServiceName
	}

	// Create the normalized record
	record := database.NormalizedPricing{
		Provider:            normCtx.Provider,
		ProviderServiceCode: providerServiceCode,
		ServiceMappingID:    &normCtx.ServiceMapping.ID,
		ServiceCategory:     normCtx.ServiceMapping.ServiceCategory,
		ServiceFamily:       normCtx.ServiceMapping.ServiceFamily,
		ServiceType:         normCtx.ServiceMapping.NormalizedServiceType,
		RegionID:            &normCtx.NormalizedRegion.ID,
		NormalizedRegion:    normCtx.NormalizedRegion.NormalizedCode,
		ProviderRegion:      providerRegion,
		ResourceName:        resourceName,
		ResourceDescription: &priceInfo.Description,
		ResourceSpecs:       resourceSpecs,
		PricePerUnit:        priceInfo.PricePerUnit,
		Unit:                normalizedUnit,
		Currency:            priceInfo.Currency,
		PricingModel:        pricingModel,
		PricingDetails:      pricingDetails,
		MinimumCommitment:   1,
	}

	// Set provider-specific raw data ID
	if normCtx.Provider == database.ProviderAWS {
		record.AWSRawID = &normCtx.RawDataID
	} else if normCtx.Provider == database.ProviderAzure {
		record.AzureRawID = &normCtx.RawDataID
	}

	// Validate the created record
	if err := n.validator.ValidateNormalizedPricing(record); err != nil {
		n.logger.Error("Created record validation failed",
			Field{"resource", resourceName},
			Field{"error", err},
		)
		return nil, fmt.Errorf("record validation failed: %w", err)
	}

	n.logger.Debug("Created normalized record",
		Field{"resource", resourceName},
		Field{"price", priceInfo.PricePerUnit},
		Field{"unit", normalizedUnit},
		Field{"model", pricingModel},
	)

	return &record, nil
}

// CreateErrorResult creates a standardized error result
func (n *BaseNormalizer) CreateErrorResult(message string, err error) *database.NormalizationResult {
	if err != nil {
		n.logger.Error(message, Field{"error", err})
		message = fmt.Sprintf("%s: %v", message, err)
	}
	return &database.NormalizationResult{
		Success:    false,
		ErrorCount: 1,
		Errors:     []string{message},
	}
}

// CreateSkippedResult creates a result for skipped items
func (n *BaseNormalizer) CreateSkippedResult(reason string, count int) *database.NormalizationResult {
	n.logger.Info("Items skipped",
		Field{"reason", reason},
		Field{"count", count},
	)
	return &database.NormalizationResult{
		Success:      false,
		SkippedCount: count,
		Errors:       []string{reason},
	}
}

// ParseJSONData parses raw JSON data with error handling
func (n *BaseNormalizer) ParseJSONData(rawData json.RawMessage, target interface{}) error {
	if err := json.Unmarshal(rawData, target); err != nil {
		n.logger.Error("Failed to parse JSON",
			Field{"error", err},
			Field{"dataLength", len(rawData)},
		)
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	return nil
}

// SimpleLogger is a basic logger implementation using standard log package
type SimpleLogger struct{}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, fields ...Field) {
	log.Printf("[DEBUG] %s %s", msg, l.formatFields(fields))
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, fields ...Field) {
	log.Printf("[INFO] %s %s", msg, l.formatFields(fields))
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, fields ...Field) {
	log.Printf("[WARN] %s %s", msg, l.formatFields(fields))
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, fields ...Field) {
	log.Printf("[ERROR] %s %s", msg, l.formatFields(fields))
}

// formatFields formats log fields for output
func (l *SimpleLogger) formatFields(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}
	result := "["
	for i, field := range fields {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("%s=%v", field.Key, field.Value)
	}
	result += "]"
	return result
}