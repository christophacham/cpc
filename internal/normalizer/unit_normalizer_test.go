package normalizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/raulc0399/cpc/internal/database"
)

func TestStandardUnitNormalizer_NormalizeUnit(t *testing.T) {
	normalizer := NewStandardUnitNormalizer()

	tests := []struct {
		name         string
		provider     string
		originalUnit string
		expected     string
	}{
		// AWS-specific units
		{
			name:         "AWS hours",
			provider:     database.ProviderAWS,
			originalUnit: "Hrs",
			expected:     database.UnitHour,
		},
		{
			name:         "AWS GB-Mo",
			provider:     database.ProviderAWS,
			originalUnit: "GB-Mo",
			expected:     database.UnitGBMonth,
		},
		{
			name:         "AWS million requests",
			provider:     database.ProviderAWS,
			originalUnit: "1M Requests",
			expected:     database.UnitMillionRequests,
		},
		{
			name:         "AWS Lambda GB-second",
			provider:     database.ProviderAWS,
			originalUnit: "Lambda-GB-Second",
			expected:     "lambda_gb_second",
		},
		
		// Azure-specific units
		{
			name:         "Azure hour",
			provider:     database.ProviderAzure,
			originalUnit: "1 Hour",
			expected:     database.UnitHour,
		},
		{
			name:         "Azure GB/month",
			provider:     database.ProviderAzure,
			originalUnit: "1 GB/Month",
			expected:     database.UnitGBMonth,
		},
		{
			name:         "Azure 10K requests",
			provider:     database.ProviderAzure,
			originalUnit: "10K Requests",
			expected:     database.UnitRequest,
		},
		{
			name:         "Azure vCPU hour",
			provider:     database.ProviderAzure,
			originalUnit: "vCPU Hour",
			expected:     "vcpu_hour",
		},
		
		// Generic units
		{
			name:         "Generic hour",
			provider:     "unknown",
			originalUnit: "hour",
			expected:     database.UnitHour,
		},
		{
			name:         "Generic GB",
			provider:     "unknown",
			originalUnit: "GB",
			expected:     database.UnitGB,
		},
		
		// Unknown units (should return original)
		{
			name:         "Unknown unit",
			provider:     database.ProviderAWS,
			originalUnit: "unknown_unit",
			expected:     "unknown_unit",
		},
		
		// Case insensitive
		{
			name:         "Case insensitive",
			provider:     database.ProviderAWS,
			originalUnit: "HRS",
			expected:     database.UnitHour,
		},
		
		// Whitespace handling
		{
			name:         "Whitespace handling",
			provider:     database.ProviderAWS,
			originalUnit: "  Hrs  ",
			expected:     database.UnitHour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeUnit(tt.provider, tt.originalUnit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStandardUnitNormalizer_ConvertUnitValue(t *testing.T) {
	normalizer := NewStandardUnitNormalizer()

	tests := []struct {
		name         string
		originalUnit string
		targetUnit   string
		value        float64
		expected     float64
	}{
		{
			name:         "10K requests to requests",
			originalUnit: "10k requests",
			targetUnit:   database.UnitRequest,
			value:        1.0,
			expected:     0.0001,
		},
		{
			name:         "10000 requests to requests",
			originalUnit: "10000 requests",
			targetUnit:   database.UnitRequest,
			value:        2.0,
			expected:     0.0002,
		},
		{
			name:         "No conversion needed",
			originalUnit: database.UnitHour,
			targetUnit:   database.UnitHour,
			value:        1.5,
			expected:     1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.ConvertUnitValue(tt.originalUnit, tt.targetUnit, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStandardUnitNormalizer_GetUnitCategory(t *testing.T) {
	normalizer := NewStandardUnitNormalizer()

	tests := []struct {
		name     string
		unit     string
		expected string
	}{
		{
			name:     "Time unit",
			unit:     database.UnitHour,
			expected: "time",
		},
		{
			name:     "Storage unit",
			unit:     database.UnitGB,
			expected: "storage",
		},
		{
			name:     "Request unit",
			unit:     database.UnitRequest,
			expected: "requests",
		},
		{
			name:     "Resource unit",
			unit:     database.UnitInstance,
			expected: "resources",
		},
		{
			name:     "Other unit",
			unit:     "unknown",
			expected: "other",
		},
		{
			name:     "Lambda GB-second",
			unit:     "lambda_gb_second",
			expected: "time",
		},
		{
			name:     "vCPU hour",
			unit:     "vcpu_hour",
			expected: "time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.GetUnitCategory(tt.unit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAWSUnitNormalization(t *testing.T) {
	normalizer := NewStandardUnitNormalizer()

	// Test AWS-specific edge cases
	awsUnits := map[string]string{
		"Hrs":                    database.UnitHour,
		"GB-Mo":                  database.UnitGBMonth,
		"Requests":               database.UnitRequest,
		"1M Requests":            database.UnitMillionRequests,
		"API Call":               database.UnitRequest,
		"API Calls":              database.UnitRequest,
		"vCPU-Hours":             "vcpu_hour",
		"Lambda-GB-Second":       "lambda_gb_second",
		"Second":                 "second",
		"Instances":              database.UnitInstance,
	}

	for originalUnit, expectedUnit := range awsUnits {
		t.Run("AWS_"+originalUnit, func(t *testing.T) {
			result := normalizer.NormalizeUnit(database.ProviderAWS, originalUnit)
			assert.Equal(t, expectedUnit, result, "Failed to normalize AWS unit: %s", originalUnit)
		})
	}
}

func TestAzureUnitNormalization(t *testing.T) {
	normalizer := NewStandardUnitNormalizer()

	// Test Azure-specific edge cases
	azureUnits := map[string]string{
		"1 Hour":                 database.UnitHour,
		"1 GB/Month":             database.UnitGBMonth,
		"1M Requests":            database.UnitMillionRequests,
		"10K Requests":           database.UnitRequest,
		"1 Transaction":          database.UnitTransaction,
		"vCPU Hour":              "vcpu_hour",
		"Compute Unit":           "compute_unit",
		"Compute Units":          "compute_unit",
	}

	for originalUnit, expectedUnit := range azureUnits {
		t.Run("Azure_"+originalUnit, func(t *testing.T) {
			result := normalizer.NormalizeUnit(database.ProviderAzure, originalUnit)
			assert.Equal(t, expectedUnit, result, "Failed to normalize Azure unit: %s", originalUnit)
		})
	}
}