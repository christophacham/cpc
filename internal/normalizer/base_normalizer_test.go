package normalizer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/raulc0399/cpc/internal/database"
)

func TestBaseNormalizer_ValidateCommonInput(t *testing.T) {
	mockLogger := NewMockLogger()
	validator := NewInputValidator()
	
	base := NewBaseNormalizer(nil, nil, nil, validator, mockLogger)

	tests := []struct {
		name        string
		input       database.NormalizationInput
		expectError bool
	}{
		{
			name: "valid input",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   1,
			},
			expectError: false,
		},
		{
			name: "invalid provider",
			input: database.NormalizationInput{
				Provider:    "",
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{"test": "data"}`),
				RawDataID:   1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger.Messages = nil // Clear messages
			err := base.ValidateCommonInput(context.Background(), tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, mockLogger.HasMessage("ERROR", "Input validation failed"))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBaseNormalizer_GetNormalizationContext(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockServiceRepo := NewMockServiceMappingRepository()
	mockRegionRepo := NewMockRegionMappingRepository()
	mockLogger := NewMockLogger()
	validator := NewInputValidator()
	
	base := NewBaseNormalizer(mockServiceRepo, mockRegionRepo, nil, validator, mockLogger)

	// Add test data
	testServiceMapping := &database.ServiceMapping{
		ID:                    1,
		Provider:              database.ProviderAWS,
		ProviderServiceName:   "Amazon EC2",
		ProviderServiceCode:   stringPtr("AmazonEC2"),
		NormalizedServiceType: "Virtual Machines",
		ServiceCategory:       "Compute & Web",
		ServiceFamily:         "Virtual Machines",
	}
	mockServiceRepo.AddMapping(database.ProviderAWS, "AmazonEC2", testServiceMapping)

	testRegion := &database.NormalizedRegion{
		ID:             1,
		NormalizedCode: "us-east",
		AWSRegion:      stringPtr("us-east-1"),
		DisplayName:    "US East (Virginia)",
	}
	mockRegionRepo.AddRegion(database.ProviderAWS, "us-east-1", testRegion)

	tests := []struct {
		name           string
		input          database.NormalizationInput
		setupMocks     func()
		expectContext  bool
		expectError    bool
	}{
		{
			name: "successful context creation",
			input: database.NormalizationInput{
				Provider:     database.ProviderAWS,
				ServiceCode:  "AmazonEC2",
				Region:       "us-east-1",
				RawData:      json.RawMessage(`{}`),
				RawDataID:    1,
				CollectionID: "test-123",
			},
			setupMocks:    func() {},
			expectContext: true,
			expectError:   false,
		},
		{
			name: "service mapping not found",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "UnknownService",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			setupMocks:    func() {},
			expectContext: false,
			expectError:   false, // Not found is not an error
		},
		{
			name: "region not found",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "unknown-region",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			setupMocks:    func() {},
			expectContext: false,
			expectError:   false, // Not found is not an error
		},
		{
			name: "service mapping error",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			setupMocks: func() {
				mockServiceRepo.Error = errors.New("database error")
			},
			expectContext: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockServiceRepo.Error = nil
			mockRegionRepo.Error = nil
			mockLogger.Messages = nil
			
			tt.setupMocks()
			
			normCtx, err := base.GetNormalizationContext(ctx, tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, normCtx)
			} else {
				assert.NoError(t, err)
				if tt.expectContext {
					require.NotNil(t, normCtx)
					assert.Equal(t, tt.input.Provider, normCtx.Provider)
					assert.NotNil(t, normCtx.ServiceMapping)
					assert.NotNil(t, normCtx.NormalizedRegion)
					assert.Equal(t, tt.input.RawDataID, normCtx.RawDataID)
					assert.Equal(t, tt.input.CollectionID, normCtx.CollectionID)
				} else {
					assert.Nil(t, normCtx)
				}
			}
		})
	}
}

func TestBaseNormalizer_CreateNormalizedRecord(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	mockUnitNormalizer := NewMockUnitNormalizer()
	mockLogger := NewMockLogger()
	validator := NewInputValidator()
	
	base := NewBaseNormalizer(nil, nil, mockUnitNormalizer, validator, mockLogger)

	// Setup unit normalization
	mockUnitNormalizer.AddMapping(database.ProviderAWS, "Hrs", database.UnitHour)

	// Test context
	normCtx := &NormalizationContext{
		Provider: database.ProviderAWS,
		ServiceMapping: &database.ServiceMapping{
			ID:                    1,
			ProviderServiceCode:   stringPtr("AmazonEC2"),
			NormalizedServiceType: "Virtual Machines",
			ServiceCategory:       "Compute & Web",
			ServiceFamily:         "Virtual Machines",
		},
		NormalizedRegion: &database.NormalizedRegion{
			ID:             1,
			NormalizedCode: "us-east",
			AWSRegion:      stringPtr("us-east-1"),
		},
		RawDataID: 123,
	}

	tests := []struct {
		name           string
		priceInfo      PricingInfo
		resourceSpecs  database.ResourceSpecs
		resourceName   string
		pricingModel   string
		pricingDetails database.PricingDetails
		expectRecord   bool
		expectError    bool
	}{
		{
			name: "successful record creation",
			priceInfo: PricingInfo{
				PricePerUnit: 0.0416,
				Unit:         "Hrs",
				Currency:     "USD",
				Description:  "On-Demand Linux t3.medium",
			},
			resourceSpecs: database.ResourceSpecs{
				VCPU:     intPtr(2),
				MemoryGB: float64Ptr(4.0),
			},
			resourceName:   "t3.medium",
			pricingModel:   database.PricingModelOnDemand,
			pricingDetails: database.PricingDetails{},
			expectRecord:   true,
			expectError:    false,
		},
		{
			name: "zero price - skip record",
			priceInfo: PricingInfo{
				PricePerUnit: 0,
				Unit:         "Hrs",
				Currency:     "USD",
				Description:  "Free tier",
			},
			resourceSpecs:  database.ResourceSpecs{},
			resourceName:   "t2.micro",
			pricingModel:   database.PricingModelOnDemand,
			pricingDetails: database.PricingDetails{},
			expectRecord:   false,
			expectError:    false,
		},
		{
			name: "invalid currency",
			priceInfo: PricingInfo{
				PricePerUnit: 0.05,
				Unit:         "Hrs",
				Currency:     "INVALID",
				Description:  "Test",
			},
			resourceSpecs:  database.ResourceSpecs{},
			resourceName:   "test",
			pricingModel:   database.PricingModelOnDemand,
			pricingDetails: database.PricingDetails{},
			expectRecord:   false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger.Messages = nil
			
			record, err := base.CreateNormalizedRecord(
				ctx, normCtx, tt.priceInfo, tt.resourceSpecs,
				tt.resourceName, tt.pricingModel, tt.pricingDetails,
			)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				assert.NoError(t, err)
				if tt.expectRecord {
					require.NotNil(t, record)
					assert.Equal(t, database.ProviderAWS, record.Provider)
					assert.Equal(t, "AmazonEC2", record.ProviderServiceCode)
					assert.Equal(t, tt.resourceName, record.ResourceName)
					assert.Equal(t, tt.priceInfo.PricePerUnit, record.PricePerUnit)
					assert.Equal(t, database.UnitHour, record.Unit) // Normalized
					assert.Equal(t, tt.priceInfo.Currency, record.Currency)
					assert.Equal(t, tt.pricingModel, record.PricingModel)
					assert.NotNil(t, record.AWSRawID)
					assert.Equal(t, 123, *record.AWSRawID)
				} else {
					assert.Nil(t, record)
				}
			}
		})
	}
}

func TestBaseNormalizer_ParseJSONData(t *testing.T) {
	mockLogger := NewMockLogger()
	base := NewBaseNormalizer(nil, nil, nil, nil, mockLogger)

	tests := []struct {
		name        string
		rawData     json.RawMessage
		target      interface{}
		expectError bool
	}{
		{
			name:        "valid JSON object",
			rawData:     json.RawMessage(`{"key": "value", "number": 42}`),
			target:      &map[string]interface{}{},
			expectError: false,
		},
		{
			name:        "valid JSON array",
			rawData:     json.RawMessage(`[1, 2, 3]`),
			target:      &[]int{},
			expectError: false,
		},
		{
			name:        "invalid JSON",
			rawData:     json.RawMessage(`{invalid json`),
			target:      &map[string]interface{}{},
			expectError: true,
		},
		{
			name:        "empty JSON",
			rawData:     json.RawMessage(`{}`),
			target:      &map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger.Messages = nil
			err := base.ParseJSONData(tt.rawData, tt.target)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, mockLogger.HasMessage("ERROR", "Failed to parse JSON"))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimpleLogger(t *testing.T) {
	logger := NewSimpleLogger()

	// Test all log levels
	logger.Debug("debug message", Field{"key", "value"})
	logger.Info("info message", Field{"count", 42})
	logger.Warn("warning message", Field{"error", "potential issue"})
	logger.Error("error message", Field{"error", "actual error"})

	// Test formatting with multiple fields
	logger.Info("multiple fields", Field{"field1", "value1"}, Field{"field2", 2})

	// Test with no fields
	logger.Info("no fields")

	// Manual verification would be needed for actual output
	// This test mainly ensures the logger doesn't panic
	assert.NotNil(t, logger)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}