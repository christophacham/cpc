package normalizer

import (
	"strings"

	"github.com/raulc0399/cpc/internal/database"
)

// StandardUnitNormalizer implements UnitNormalizer interface
type StandardUnitNormalizer struct{}

// NewStandardUnitNormalizer creates a new standard unit normalizer
func NewStandardUnitNormalizer() *StandardUnitNormalizer {
	return &StandardUnitNormalizer{}
}

// NormalizeUnit normalizes pricing units to standardized units
func (n *StandardUnitNormalizer) NormalizeUnit(provider, originalUnit string) string {
	unit := strings.ToLower(strings.TrimSpace(originalUnit))

	// Provider-specific normalization first
	if provider == database.ProviderAWS {
		return n.normalizeAWSUnit(unit)
	} else if provider == database.ProviderAzure {
		return n.normalizeAzureUnit(unit)
	}

	// Fallback to generic normalization
	return n.normalizeGenericUnit(unit)
}

// normalizeAWSUnit normalizes AWS-specific units
func (n *StandardUnitNormalizer) normalizeAWSUnit(unit string) string {
	awsUnitMap := map[string]string{
		"hrs":                    database.UnitHour,
		"hour":                   database.UnitHour,
		"hours":                  database.UnitHour,
		"gb-mo":                  database.UnitGBMonth,
		"gb-month":               database.UnitGBMonth,
		"requests":               database.UnitRequest,
		"request":                database.UnitRequest,
		"1m requests":            database.UnitMillionRequests,
		"million requests":       database.UnitMillionRequests,
		"gb":                     database.UnitGB,
		"gigabyte":               database.UnitGB,
		"tb":                     database.UnitTB,
		"terabyte":               database.UnitTB,
		"instances":              database.UnitInstance,
		"instance":               database.UnitInstance,
		"second":                 "second",
		"seconds":                "second",
		"lambda-gb-second":       "lambda_gb_second",
		"vcpu-hours":             "vcpu_hour",
		"api call":               database.UnitRequest,
		"api calls":              database.UnitRequest,
	}

	if normalized, exists := awsUnitMap[unit]; exists {
		return normalized
	}

	return n.normalizeGenericUnit(unit)
}

// normalizeAzureUnit normalizes Azure-specific units
func (n *StandardUnitNormalizer) normalizeAzureUnit(unit string) string {
	azureUnitMap := map[string]string{
		"1 hour":                 database.UnitHour,
		"hour":                   database.UnitHour,
		"hours":                  database.UnitHour,
		"1 gb/month":             database.UnitGBMonth,
		"gb/month":               database.UnitGBMonth,
		"gb-month":               database.UnitGBMonth,
		"1m requests":            database.UnitMillionRequests,
		"1 million requests":     database.UnitMillionRequests,
		"million requests":       database.UnitMillionRequests,
		"10k requests":           database.UnitRequest, // Convert to base unit
		"10000 requests":         database.UnitRequest, // Convert to base unit
		"1 gb":                   database.UnitGB,
		"gb":                     database.UnitGB,
		"1 tb":                   database.UnitTB,
		"tb":                     database.UnitTB,
		"1 transaction":          database.UnitTransaction,
		"transaction":            database.UnitTransaction,
		"transactions":           database.UnitTransaction,
		"vcpu hour":              "vcpu_hour",
		"vcpu hours":             "vcpu_hour",
		"compute unit":           "compute_unit",
		"compute units":          "compute_unit",
	}

	if normalized, exists := azureUnitMap[unit]; exists {
		return normalized
	}

	return n.normalizeGenericUnit(unit)
}

// normalizeGenericUnit normalizes common units across providers
func (n *StandardUnitNormalizer) normalizeGenericUnit(unit string) string {
	genericUnitMap := map[string]string{
		"hour":         database.UnitHour,
		"hours":        database.UnitHour,
		"hr":           database.UnitHour,
		"hrs":          database.UnitHour,
		"gb":           database.UnitGB,
		"gigabyte":     database.UnitGB,
		"gigabytes":    database.UnitGB,
		"tb":           database.UnitTB,
		"terabyte":     database.UnitTB,
		"terabytes":    database.UnitTB,
		"request":      database.UnitRequest,
		"requests":     database.UnitRequest,
		"transaction":  database.UnitTransaction,
		"transactions": database.UnitTransaction,
		"instance":     database.UnitInstance,
		"instances":    database.UnitInstance,
	}

	if normalized, exists := genericUnitMap[unit]; exists {
		return normalized
	}

	// Return original unit if no mapping found
	return unit
}

// ConvertUnitValue converts pricing values when changing units
func (n *StandardUnitNormalizer) ConvertUnitValue(originalUnit, targetUnit string, value float64) float64 {
	// Handle special conversions
	if originalUnit == "10k requests" && targetUnit == database.UnitRequest {
		return value / 10000 // Convert per-10k-request to per-request
	}

	if originalUnit == "10000 requests" && targetUnit == database.UnitRequest {
		return value / 10000 // Convert per-10k-request to per-request
	}

	// Add more conversions as needed
	return value
}

// GetUnitCategory returns the category of a unit for grouping
func (n *StandardUnitNormalizer) GetUnitCategory(unit string) string {
	timeUnits := []string{database.UnitHour, "second", "minute", "vcpu_hour", "lambda_gb_second"}
	storageUnits := []string{database.UnitGB, database.UnitTB, database.UnitGBMonth}
	requestUnits := []string{database.UnitRequest, database.UnitMillionRequests, database.UnitTransaction}
	resourceUnits := []string{database.UnitInstance, "compute_unit"}

	for _, timeUnit := range timeUnits {
		if unit == timeUnit {
			return "time"
		}
	}

	for _, storageUnit := range storageUnits {
		if unit == storageUnit {
			return "storage"
		}
	}

	for _, requestUnit := range requestUnits {
		if unit == requestUnit {
			return "requests"
		}
	}

	for _, resourceUnit := range resourceUnits {
		if unit == resourceUnit {
			return "resources"
		}
	}

	return "other"
}