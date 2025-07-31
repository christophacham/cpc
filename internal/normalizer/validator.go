package normalizer

import (
	"encoding/json"
	"strings"

	"github.com/raulc0399/cpc/internal/database"
)

// InputValidator validates normalization input data
type InputValidator struct{}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// ValidateNormalizationInput validates a normalization input
func (v *InputValidator) ValidateNormalizationInput(input database.NormalizationInput) error {
	// Validate provider
	if err := v.validateProvider(input.Provider); err != nil {
		return err
	}

	// Validate service code
	if err := v.validateServiceCode(input.ServiceCode); err != nil {
		return err
	}

	// Validate region
	if err := v.validateRegion(input.Region); err != nil {
		return err
	}

	// Validate raw data
	if err := v.validateRawData(input.RawData); err != nil {
		return err
	}

	// Validate raw data ID
	if err := v.validateRawDataID(input.RawDataID); err != nil {
		return err
	}

	return nil
}

// validateProvider validates the provider field
func (v *InputValidator) validateProvider(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return ValidationError{
			Field:   "provider",
			Value:   provider,
			Message: ErrMsgEmptyProvider,
		}
	}

	// Check if provider is supported
	supportedProviders := []string{database.ProviderAWS, database.ProviderAzure}
	for _, supported := range supportedProviders {
		if provider == supported {
			return nil
		}
	}

	return ValidationError{
		Field:   "provider",
		Value:   provider,
		Message: ErrMsgUnsupportedProvider,
	}
}

// validateServiceCode validates the service code field
func (v *InputValidator) validateServiceCode(serviceCode string) error {
	serviceCode = strings.TrimSpace(serviceCode)
	if serviceCode == "" {
		return ValidationError{
			Field:   "serviceCode",
			Value:   serviceCode,
			Message: ErrMsgEmptyServiceCode,
		}
	}

	if len(serviceCode) > MaxServiceCodeLength {
		return ValidationError{
			Field:   "serviceCode",
			Value:   serviceCode,
			Message: "service code too long",
		}
	}

	return nil
}

// validateRegion validates the region field
func (v *InputValidator) validateRegion(region string) error {
	region = strings.TrimSpace(region)
	if region == "" {
		return ValidationError{
			Field:   "region",
			Value:   region,
			Message: ErrMsgEmptyRegion,
		}
	}

	if len(region) > MaxRegionLength {
		return ValidationError{
			Field:   "region",
			Value:   region,
			Message: "region name too long",
		}
	}

	return nil
}

// validateRawData validates the raw data field
func (v *InputValidator) validateRawData(rawData json.RawMessage) error {
	if len(rawData) == 0 {
		return ValidationError{
			Field:   "rawData",
			Value:   rawData,
			Message: ErrMsgEmptyRawData,
		}
	}

	// Validate that it's valid JSON
	var temp interface{}
	if err := json.Unmarshal(rawData, &temp); err != nil {
		return ValidationError{
			Field:   "rawData",
			Value:   string(rawData),
			Message: "invalid JSON format",
		}
	}

	return nil
}

// validateRawDataID validates the raw data ID field
func (v *InputValidator) validateRawDataID(rawDataID int) error {
	if rawDataID <= 0 {
		return ValidationError{
			Field:   "rawDataID",
			Value:   rawDataID,
			Message: "raw data ID must be positive",
		}
	}

	return nil
}

// ValidateNormalizedPricing validates a normalized pricing record before insertion
func (v *InputValidator) ValidateNormalizedPricing(pricing database.NormalizedPricing) error {
	// Validate provider
	if err := v.validateProvider(pricing.Provider); err != nil {
		return err
	}

	// Validate service code
	if err := v.validateServiceCode(pricing.ProviderServiceCode); err != nil {
		return err
	}

	// Validate resource name
	if err := v.validateResourceName(pricing.ResourceName); err != nil {
		return err
	}

	// Validate price
	if err := v.validatePrice(pricing.PricePerUnit); err != nil {
		return err
	}

	// Validate unit
	if err := v.validateUnit(pricing.Unit); err != nil {
		return err
	}

	// Validate currency
	if err := v.validateCurrency(pricing.Currency); err != nil {
		return err
	}

	// Validate pricing model
	if err := v.validatePricingModel(pricing.PricingModel); err != nil {
		return err
	}

	return nil
}

// validateResourceName validates the resource name field
func (v *InputValidator) validateResourceName(resourceName string) error {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return ValidationError{
			Field:   "resourceName",
			Value:   resourceName,
			Message: "resource name cannot be empty",
		}
	}

	if len(resourceName) > MaxResourceNameLength {
		return ValidationError{
			Field:   "resourceName",
			Value:   resourceName,
			Message: "resource name too long",
		}
	}

	return nil
}

// validatePrice validates the price field
func (v *InputValidator) validatePrice(price float64) error {
	if price < MinPriceValue || price > MaxPriceValue {
		return ValidationError{
			Field:   "pricePerUnit",
			Value:   price,
			Message: ErrMsgInvalidPrice,
		}
	}

	return nil
}

// validateUnit validates the unit field
func (v *InputValidator) validateUnit(unit string) error {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return ValidationError{
			Field:   "unit",
			Value:   unit,
			Message: "unit cannot be empty",
		}
	}

	return nil
}

// validateCurrency validates the currency field
func (v *InputValidator) validateCurrency(currency string) error {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return ValidationError{
			Field:   "currency",
			Value:   currency,
			Message: "currency cannot be empty",
		}
	}

	// Validate currency format (should be 3-letter ISO code)
	if len(currency) != 3 {
		return ValidationError{
			Field:   "currency",
			Value:   currency,
			Message: "currency must be 3-letter ISO code",
		}
	}

	return nil
}

// validatePricingModel validates the pricing model field
func (v *InputValidator) validatePricingModel(pricingModel string) error {
	pricingModel = strings.TrimSpace(pricingModel)
	if pricingModel == "" {
		return ValidationError{
			Field:   "pricingModel",
			Value:   pricingModel,
			Message: "pricing model cannot be empty",
		}
	}

	// Check if pricing model is supported
	supportedModels := []string{
		database.PricingModelOnDemand,
		database.PricingModelReserved1Yr,
		database.PricingModelReserved3Yr,
		database.PricingModelSpot,
		database.PricingModelSavingsPlan,
	}

	for _, supported := range supportedModels {
		if pricingModel == supported {
			return nil
		}
	}

	return ValidationError{
		Field:   "pricingModel",
		Value:   pricingModel,
		Message: "unsupported pricing model",
	}
}