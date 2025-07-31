package normalizer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/raulc0399/cpc/internal/database"
)

func TestInputValidator_ValidateNormalizationInput(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		input       database.NormalizationInput
		expectError bool
		errorType   string
	}{
		{
			name: "valid AWS input",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{"product": {"sku": "test"}}`),
				RawDataID:   1,
			},
			expectError: false,
		},
		{
			name: "valid Azure input",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "Virtual Machines",
				Region:      "eastus",
				RawData:     json.RawMessage(`[{"serviceName": "Virtual Machines"}]`),
				RawDataID:   1,
			},
			expectError: false,
		},
		{
			name: "empty provider",
			input: database.NormalizationInput{
				Provider:    "",
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   1,
			},
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name: "unsupported provider",
			input: database.NormalizationInput{
				Provider:    "gcp",
				ServiceCode: "Compute Engine",
				Region:      "us-central1",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   1,
			},
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name: "empty service code",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   1,
			},
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name: "empty region",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   1,
			},
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name: "empty raw data",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			expectError: false, // Empty JSON object is valid
		},
		{
			name: "invalid JSON",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{invalid json`),
				RawDataID:   1,
			},
			expectError: true,
			errorType:   "ValidationError",
		},
		{
			name: "invalid raw data ID",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   0,
			},
			expectError: true,
			errorType:   "ValidationError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateNormalizationInput(tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorType == "ValidationError" {
					var validationErr ValidationError
					assert.ErrorAs(t, err, &validationErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInputValidator_ValidateNormalizedPricing(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name        string
		pricing     database.NormalizedPricing
		expectError bool
	}{
		{
			name: "valid pricing record",
			pricing: database.NormalizedPricing{
				Provider:            database.ProviderAWS,
				ProviderServiceCode: "AmazonEC2",
				ResourceName:        "t3.medium",
				PricePerUnit:        0.0416,
				Unit:                database.UnitHour,
				Currency:            "USD",
				PricingModel:        database.PricingModelOnDemand,
			},
			expectError: false,
		},
		{
			name: "invalid price - negative",
			pricing: database.NormalizedPricing{
				Provider:            database.ProviderAWS,
				ProviderServiceCode: "AmazonEC2",
				ResourceName:        "t3.medium",
				PricePerUnit:        -0.01,
				Unit:                database.UnitHour,
				Currency:            "USD",
				PricingModel:        database.PricingModelOnDemand,
			},
			expectError: true,
		},
		{
			name: "invalid price - too high",
			pricing: database.NormalizedPricing{
				Provider:            database.ProviderAWS,
				ProviderServiceCode: "AmazonEC2",
				ResourceName:        "t3.medium",
				PricePerUnit:        1000000.00,
				Unit:                database.UnitHour,
				Currency:            "USD",
				PricingModel:        database.PricingModelOnDemand,
			},
			expectError: true,
		},
		{
			name: "invalid currency",
			pricing: database.NormalizedPricing{
				Provider:            database.ProviderAWS,
				ProviderServiceCode: "AmazonEC2",
				ResourceName:        "t3.medium",
				PricePerUnit:        0.0416,
				Unit:                database.UnitHour,
				Currency:            "INVALID",
				PricingModel:        database.PricingModelOnDemand,
			},
			expectError: true,
		},
		{
			name: "invalid pricing model",
			pricing: database.NormalizedPricing{
				Provider:            database.ProviderAWS,
				ProviderServiceCode: "AmazonEC2",
				ResourceName:        "t3.medium",
				PricePerUnit:        0.0416,
				Unit:                database.UnitHour,
				Currency:            "USD",
				PricingModel:        "invalid_model",
			},
			expectError: true,
		},
		{
			name: "empty resource name",
			pricing: database.NormalizedPricing{
				Provider:            database.ProviderAWS,
				ProviderServiceCode: "AmazonEC2",
				ResourceName:        "",
				PricePerUnit:        0.0416,
				Unit:                database.UnitHour,
				Currency:            "USD",
				PricingModel:        database.PricingModelOnDemand,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateNormalizedPricing(tt.pricing)

			if tt.expectError {
				assert.Error(t, err)
				var validationErr ValidationError
				assert.ErrorAs(t, err, &validationErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "provider",
		Value:   "invalid",
		Message: "unsupported provider",
	}

	assert.Equal(t, "unsupported provider", err.Error())
	assert.Equal(t, "provider", err.Field)
	assert.Equal(t, "invalid", err.Value)
}

func TestNormalizationError(t *testing.T) {
	baseErr := ValidationError{Field: "test", Message: "test error"}
	err := NormalizationError{
		Provider:    "aws",
		ServiceCode: "AmazonEC2",
		Region:      "us-east-1",
		Message:     "normalization failed",
		Cause:       baseErr,
	}

	assert.Contains(t, err.Error(), "normalization failed")
	assert.Contains(t, err.Error(), "test error")
	assert.Equal(t, "aws", err.Provider)
	assert.Equal(t, "AmazonEC2", err.ServiceCode)
}