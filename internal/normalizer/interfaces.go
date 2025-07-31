package normalizer

import (
	"context"
	"github.com/raulc0399/cpc/internal/database"
)

// PricingNormalizer defines the interface for pricing data normalization
type PricingNormalizer interface {
	// NormalizePricing normalizes raw pricing data into standardized format
	NormalizePricing(ctx context.Context, input database.NormalizationInput) (*database.NormalizationResult, error)
	
	// GetSupportedProvider returns the provider this normalizer supports
	GetSupportedProvider() string
	
	// ValidateInput validates the input data before normalization
	ValidateInput(input database.NormalizationInput) error
}

// ServiceMappingRepository defines interface for service mapping operations
type ServiceMappingRepository interface {
	GetServiceMappingByProvider(ctx context.Context, provider, serviceName string) (*database.ServiceMapping, error)
	GetAllServiceMappings(ctx context.Context) ([]database.ServiceMapping, error)
}

// RegionMappingRepository defines interface for region mapping operations  
type RegionMappingRepository interface {
	GetNormalizedRegionByProvider(ctx context.Context, provider, providerRegion string) (*database.NormalizedRegion, error)
	GetAllNormalizedRegions(ctx context.Context) ([]database.NormalizedRegion, error)
}

// ResourceSpecExtractor defines interface for extracting resource specifications
type ResourceSpecExtractor interface {
	ExtractResourceSpecs(provider, serviceType string, data map[string]interface{}) (database.ResourceSpecs, error)
}

// UnitNormalizer defines interface for normalizing pricing units
type UnitNormalizer interface {
	NormalizeUnit(provider, originalUnit string) string
}

// PricingModelDetector defines interface for detecting pricing models
type PricingModelDetector interface {
	DetectPricingModel(provider string, data map[string]interface{}) string
}

// NormalizationError represents an error during normalization
type NormalizationError struct {
	Provider    string
	ServiceCode string
	Region      string
	Message     string
	Cause       error
}

func (e NormalizationError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// Constants for validation
const (
	MaxServiceCodeLength = 100
	MaxRegionLength      = 50
	MaxResourceNameLength = 200
	MinPriceValue        = 0.0
	MaxPriceValue        = 999999.99
)

// Standard error messages
const (
	ErrMsgEmptyProvider     = "provider cannot be empty"
	ErrMsgEmptyServiceCode  = "service code cannot be empty"
	ErrMsgEmptyRegion       = "region cannot be empty"
	ErrMsgEmptyRawData      = "raw data cannot be empty"
	ErrMsgInvalidPrice      = "price must be between 0 and 999999.99"
	ErrMsgUnsupportedProvider = "unsupported provider"
)