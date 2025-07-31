package normalizer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/raulc0399/cpc/internal/database"
)

func TestAzureNormalizerV2_GetSupportedProvider(t *testing.T) {
	normalizer := createTestAzureNormalizerV2()
	assert.Equal(t, database.ProviderAzure, normalizer.GetSupportedProvider())
}

func TestAzureNormalizerV2_ValidateInput(t *testing.T) {
	normalizer := createTestAzureNormalizerV2()

	tests := []struct {
		name        string
		input       database.NormalizationInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid Azure input",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "VirtualMachines",
				Region:      "eastus",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			expectError: false,
		},
		{
			name: "invalid provider",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			expectError: true,
			errorMsg:    "unsupported provider for Azure normalizer",
		},
		{
			name: "empty provider",
			input: database.NormalizationInput{
				Provider:    "",
				ServiceCode: "VirtualMachines",
				Region:      "eastus",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			expectError: true,
			errorMsg:    "provider cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizer.ValidateInput(tt.input)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAzureNormalizerV2_NormalizePricing(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name              string
		input             database.NormalizationInput
		setupMocks        func(*MockServiceMappingRepository, *MockRegionMappingRepository, *MockUnitNormalizer)
		expectedSuccess   bool
		expectedRecords   int
		expectedErrors    int
		expectedSkipped   int
	}{
		{
			name: "successful VM normalization",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "Virtual Machines",
				Region:      "eastus",
				RawData:     getValidVMPricingJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				// Add service mapping
				serviceRepo.AddMapping(database.ProviderAzure, "Virtual Machines", &database.ServiceMapping{
					ID:                    1,
					Provider:              database.ProviderAzure,
					ProviderServiceName:   "Virtual Machines",
					ProviderServiceCode:   stringPtr("VirtualMachines"),
					NormalizedServiceType: "Virtual Machines",
					ServiceCategory:       "Compute & Web",
					ServiceFamily:         "Virtual Machines",
				})
				
				// Add region mapping
				regionRepo.AddRegion(database.ProviderAzure, "eastus", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AzureRegion:    stringPtr("eastus"),
					DisplayName:    "East US",
				})
				
				// Add unit mapping
				unitNorm.AddMapping(database.ProviderAzure, "1 Hour", database.UnitHour)
			},
			expectedSuccess: true,
			expectedRecords: 1,
			expectedErrors:  0,
			expectedSkipped: 0,
		},
		{
			name: "service not mapped - skip",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "UnmappedService",
				Region:      "eastus",
				RawData:     getValidVMPricingJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				// Don't add service mapping - should skip
				regionRepo.AddRegion(database.ProviderAzure, "eastus", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AzureRegion:    stringPtr("eastus"),
				})
			},
			expectedSuccess: false,
			expectedRecords: 0,
			expectedErrors:  0,
			expectedSkipped: 1,
		},
		{
			name: "invalid JSON structure",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "Virtual Machines",
				Region:      "eastus",
				RawData:     json.RawMessage(`{"invalid": "structure"}`),
				RawDataID:   1,
			},
			setupMocks:      func(*MockServiceMappingRepository, *MockRegionMappingRepository, *MockUnitNormalizer) {},
			expectedSuccess: false,
			expectedRecords: 0,
			expectedErrors:  1,
			expectedSkipped: 0,
		},
		{
			name: "zero price - skip",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "Virtual Machines",
				Region:      "eastus",
				RawData:     getZeroPriceVMJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				serviceRepo.AddMapping(database.ProviderAzure, "Virtual Machines", &database.ServiceMapping{
					ID:                    1,
					Provider:              database.ProviderAzure,
					ProviderServiceName:   "Virtual Machines",
					NormalizedServiceType: "Virtual Machines",
					ServiceCategory:       "Compute & Web",
				})
				regionRepo.AddRegion(database.ProviderAzure, "eastus", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AzureRegion:    stringPtr("eastus"),
				})
			},
			expectedSuccess: false,
			expectedRecords: 0,
			expectedErrors:  0,
			expectedSkipped: 1,
		},
		{
			name: "reserved VM pricing",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "Virtual Machines",
				Region:      "eastus",
				RawData:     getReservedVMJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				serviceRepo.AddMapping(database.ProviderAzure, "Virtual Machines", &database.ServiceMapping{
					ID:                    1,
					Provider:              database.ProviderAzure,
					ProviderServiceName:   "Virtual Machines",
					NormalizedServiceType: "Virtual Machines",
					ServiceCategory:       "Compute & Web",
				})
				regionRepo.AddRegion(database.ProviderAzure, "eastus", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AzureRegion:    stringPtr("eastus"),
				})
				unitNorm.AddMapping(database.ProviderAzure, "1 Hour", database.UnitHour)
			},
			expectedSuccess: true,
			expectedRecords: 1,
			expectedErrors:  0,
			expectedSkipped: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test normalizer with mocks
			serviceRepo := NewMockServiceMappingRepository()
			regionRepo := NewMockRegionMappingRepository()
			unitNorm := NewMockUnitNormalizer()
			logger := NewMockLogger()
			validator := NewInputValidator()
			
			tt.setupMocks(serviceRepo, regionRepo, unitNorm)
			
			normalizer := NewAzureNormalizerV2(
				serviceRepo,
				regionRepo,
				unitNorm,
				validator,
				logger,
			)
			
			// Execute normalization
			result, err := normalizer.NormalizePricing(ctx, tt.input)
			
			// Verify results
			require.NoError(t, err)
			require.NotNil(t, result)
			
			assert.Equal(t, tt.expectedSuccess, result.Success)
			assert.Len(t, result.NormalizedRecords, tt.expectedRecords)
			assert.Equal(t, tt.expectedErrors, result.ErrorCount)
			assert.Equal(t, tt.expectedSkipped, result.SkippedCount)
			
			// Verify normalized records if any
			if tt.expectedRecords > 0 {
				for _, record := range result.NormalizedRecords {
					assert.Equal(t, database.ProviderAzure, record.Provider)
					assert.NotEmpty(t, record.ResourceName)
					assert.Greater(t, record.PricePerUnit, 0.0)
					assert.NotEmpty(t, record.Unit)
					assert.NotEmpty(t, record.Currency)
					assert.NotEmpty(t, record.PricingModel)
				}
			}
		})
	}
}

func TestAzureNormalizerV2_ExtractPricingFromAzureItem(t *testing.T) {
	normalizer := createTestAzureNormalizerV2()
	
	tests := []struct {
		name     string
		pricing  AzurePricing
		expected *PricingInfo
	}{
		{
			name: "with retail price",
			pricing: AzurePricing{
				RetailPrice:   0.096,
				UnitPrice:     0.0,
				UnitOfMeasure: "1 Hour",
				CurrencyCode:  "USD",
				ProductName:   "Virtual Machines",
				MeterName:     "D2s v3",
			},
			expected: &PricingInfo{
				PricePerUnit: 0.096,
				Unit:         "1 Hour",
				Currency:     "USD",
				Description:  "Virtual Machines - D2s v3",
			},
		},
		{
			name: "with unit price only",
			pricing: AzurePricing{
				RetailPrice:   0.0,
				UnitPrice:     0.087,
				UnitOfMeasure: "1 Hour",
				CurrencyCode:  "USD",
				ProductName:   "Virtual Machines",
				MeterName:     "D2s v3",
			},
			expected: &PricingInfo{
				PricePerUnit: 0.087,
				Unit:         "1 Hour",
				Currency:     "USD",
				Description:  "Virtual Machines - D2s v3",
			},
		},
		{
			name: "both prices - uses retail",
			pricing: AzurePricing{
				RetailPrice:   0.096,
				UnitPrice:     0.087,
				UnitOfMeasure: "1 Hour",
				CurrencyCode:  "USD",
				ProductName:   "Virtual Machines",
				MeterName:     "D2s v3",
			},
			expected: &PricingInfo{
				PricePerUnit: 0.096,
				Unit:         "1 Hour",
				Currency:     "USD",
				Description:  "Virtual Machines - D2s v3",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.extractPricingFromAzureItem(&tt.pricing)
			assert.Equal(t, tt.expected.PricePerUnit, result.PricePerUnit)
			assert.Equal(t, tt.expected.Unit, result.Unit)
			assert.Equal(t, tt.expected.Currency, result.Currency)
			assert.Equal(t, tt.expected.Description, result.Description)
		})
	}
}

func TestAzureNormalizerV2_CreateResourceName(t *testing.T) {
	normalizer := createTestAzureNormalizerV2()
	
	tests := []struct {
		name         string
		pricing      AzurePricing
		serviceType  string
		expected     string
	}{
		{
			name: "VM with ARM SKU name",
			pricing: AzurePricing{
				ArmSKUName: "Standard_D2s_v3",
				SKUName:    "D2s v3",
			},
			serviceType: "Virtual Machines",
			expected:    "Standard_D2s_v3",
		},
		{
			name: "VM without ARM SKU name",
			pricing: AzurePricing{
				ArmSKUName: "",
				SKUName:    "D2s v3",
			},
			serviceType: "Virtual Machines",
			expected:    "D2s v3",
		},
		{
			name: "Azure Functions",
			pricing: AzurePricing{
				ProductName: "Azure Functions",
			},
			serviceType: "Serverless Functions",
			expected:    "Azure Functions",
		},
		{
			name: "Hot Storage",
			pricing: AzurePricing{
				MeterName: "Hot LRS Data Stored",
			},
			serviceType: "Storage",
			expected:    "Hot Storage",
		},
		{
			name: "Cool Storage",
			pricing: AzurePricing{
				MeterName: "Cool LRS Data Stored",
			},
			serviceType: "Storage",
			expected:    "Cool Storage",
		},
		{
			name: "Archive Storage",
			pricing: AzurePricing{
				MeterName: "Archive LRS Data Stored",
			},
			serviceType: "Storage",
			expected:    "Archive Storage",
		},
		{
			name: "Generic Storage",
			pricing: AzurePricing{
				MeterName: "Standard LRS Data Stored",
			},
			serviceType: "Storage",
			expected:    "Storage",
		},
		{
			name: "Unknown service fallback",
			pricing: AzurePricing{
				ArmSKUName:  "GP_Gen5_2",
				SKUName:     "General Purpose",
				ProductName: "Azure Database for MySQL",
			},
			serviceType: "Other",
			expected:    "GP_Gen5_2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.createResourceName(&tt.pricing, tt.serviceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAzureNormalizerV2_DeterminePricingModel(t *testing.T) {
	normalizer := createTestAzureNormalizerV2()
	
	tests := []struct {
		name     string
		pricing  AzurePricing
		expected string
	}{
		{
			name: "on-demand VM",
			pricing: AzurePricing{
				ProductName: "Virtual Machines",
				SKUName:     "D2s v3",
			},
			expected: database.PricingModelOnDemand,
		},
		{
			name: "1-year reserved VM",
			pricing: AzurePricing{
				ProductName: "Virtual Machines Reserved VM Instance",
				SKUName:     "Standard_D2s_v3 Reserved",
			},
			expected: database.PricingModelReserved1Yr,
		},
		{
			name: "3-year reserved VM",
			pricing: AzurePricing{
				ProductName: "Virtual Machines Reserved VM Instance 3 Year",
				SKUName:     "Standard_D2s_v3",
			},
			expected: database.PricingModelReserved3Yr,
		},
		{
			name: "spot VM",
			pricing: AzurePricing{
				ProductName: "Virtual Machines Spot",
				SKUName:     "D2s v3",
			},
			expected: database.PricingModelSpot,
		},
		{
			name: "reserved in SKU name",
			pricing: AzurePricing{
				ProductName: "Azure Database for MySQL",
				SKUName:     "GP_Gen5_2 Reserved",
			},
			expected: database.PricingModelReserved1Yr,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.determinePricingModel(&tt.pricing)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAzureNormalizerV2_ExtractPricingDetails(t *testing.T) {
	normalizer := createTestAzureNormalizerV2()
	
	tests := []struct {
		name         string
		pricing      AzurePricing
		pricingModel string
		expected     database.PricingDetails
	}{
		{
			name: "on-demand - no details",
			pricing: AzurePricing{
				ProductName: "Virtual Machines",
			},
			pricingModel: database.PricingModelOnDemand,
			expected:     database.PricingDetails{},
		},
		{
			name: "1-year reserved",
			pricing: AzurePricing{
				ProductName: "Virtual Machines Reserved VM Instance 1 Year",
			},
			pricingModel: database.PricingModelReserved1Yr,
			expected: database.PricingDetails{
				TermLength:    stringPtr("1yr"),
				PaymentOption: stringPtr("All Upfront"),
			},
		},
		{
			name: "3-year reserved",
			pricing: AzurePricing{
				ProductName: "Virtual Machines Reserved VM Instance 3 Year",
			},
			pricingModel: database.PricingModelReserved3Yr,
			expected: database.PricingDetails{
				TermLength:    stringPtr("3yr"),
				PaymentOption: stringPtr("All Upfront"),
			},
		},
		{
			name: "reserved without explicit year",
			pricing: AzurePricing{
				ProductName: "Virtual Machines Reserved VM Instance",
			},
			pricingModel: database.PricingModelReserved1Yr,
			expected: database.PricingDetails{
				PaymentOption: stringPtr("All Upfront"),
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.extractPricingDetails(&tt.pricing, tt.pricingModel)
			
			if tt.expected.TermLength != nil {
				require.NotNil(t, result.TermLength)
				assert.Equal(t, *tt.expected.TermLength, *result.TermLength)
			} else {
				assert.Nil(t, result.TermLength)
			}
			
			if tt.expected.PaymentOption != nil {
				require.NotNil(t, result.PaymentOption)
				assert.Equal(t, *tt.expected.PaymentOption, *result.PaymentOption)
			} else {
				assert.Nil(t, result.PaymentOption)
			}
		})
	}
}

// Helper functions

func createTestAzureNormalizerV2() *AzureNormalizerV2 {
	return NewAzureNormalizerV2(
		NewMockServiceMappingRepository(),
		NewMockRegionMappingRepository(),
		NewMockUnitNormalizer(),
		NewInputValidator(),
		NewMockLogger(),
	)
}

func getValidVMPricingJSON() json.RawMessage {
	return json.RawMessage(`{
		"currencyCode": "USD",
		"tierMinimumUnits": 0,
		"retailPrice": 0.096,
		"unitPrice": 0.096,
		"armRegionName": "eastus",
		"location": "US East",
		"effectiveDate": "2021-01-01T00:00:00Z",
		"meterId": "abc123",
		"meterName": "D2s v3",
		"productId": "xyz789",
		"productName": "Virtual Machines",
		"skuId": "sku456",
		"skuName": "D2s v3",
		"serviceName": "Virtual Machines",
		"serviceId": "svc123",
		"serviceFamily": "Compute",
		"unitOfMeasure": "1 Hour",
		"type": "Consumption",
		"isPrimaryMeterRegion": true,
		"armSkuName": "Standard_D2s_v3"
	}`)
}

func getZeroPriceVMJSON() json.RawMessage {
	return json.RawMessage(`{
		"currencyCode": "USD",
		"tierMinimumUnits": 0,
		"retailPrice": 0.0,
		"unitPrice": 0.0,
		"armRegionName": "eastus",
		"location": "US East",
		"effectiveDate": "2021-01-01T00:00:00Z",
		"meterId": "free123",
		"meterName": "B1s",
		"productId": "xyz789",
		"productName": "Virtual Machines",
		"skuId": "sku456",
		"skuName": "B1s",
		"serviceName": "Virtual Machines",
		"serviceId": "svc123",
		"serviceFamily": "Compute",
		"unitOfMeasure": "1 Hour",
		"type": "Consumption",
		"isPrimaryMeterRegion": true,
		"armSkuName": "Standard_B1s"
	}`)
}

func getReservedVMJSON() json.RawMessage {
	return json.RawMessage(`{
		"currencyCode": "USD",
		"tierMinimumUnits": 0,
		"retailPrice": 0.062,
		"unitPrice": 0.062,
		"armRegionName": "eastus",
		"location": "US East",
		"effectiveDate": "2021-01-01T00:00:00Z",
		"meterId": "res123",
		"meterName": "D2s v3 Reserved",
		"productId": "xyz789",
		"productName": "Virtual Machines Reserved VM Instance 3 Year",
		"skuId": "sku456",
		"skuName": "Standard_D2s_v3 Reserved",
		"serviceName": "Virtual Machines",
		"serviceId": "svc123",
		"serviceFamily": "Compute",
		"unitOfMeasure": "1 Hour",
		"type": "Reservation",
		"isPrimaryMeterRegion": true,
		"armSkuName": "Standard_D2s_v3"
	}`)
}