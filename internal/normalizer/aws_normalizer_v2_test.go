package normalizer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/raulc0399/cpc/internal/database"
)

func TestAWSNormalizerV2_GetSupportedProvider(t *testing.T) {
	normalizer := createTestAWSNormalizerV2()
	assert.Equal(t, database.ProviderAWS, normalizer.GetSupportedProvider())
}

func TestAWSNormalizerV2_ValidateInput(t *testing.T) {
	normalizer := createTestAWSNormalizerV2()

	tests := []struct {
		name        string
		input       database.NormalizationInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid AWS input",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			expectError: false,
		},
		{
			name: "invalid provider",
			input: database.NormalizationInput{
				Provider:    database.ProviderAzure,
				ServiceCode: "VirtualMachines",
				Region:      "eastus",
				RawData:     json.RawMessage(`{}`),
				RawDataID:   1,
			},
			expectError: true,
			errorMsg:    "unsupported provider for AWS normalizer",
		},
		{
			name: "empty provider",
			input: database.NormalizationInput{
				Provider:    "",
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
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

func TestAWSNormalizerV2_NormalizePricing(t *testing.T) {
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
			name: "successful EC2 normalization",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     getValidEC2PricingJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				// Add service mapping
				serviceRepo.AddMapping(database.ProviderAWS, "AmazonEC2", &database.ServiceMapping{
					ID:                    1,
					Provider:              database.ProviderAWS,
					ProviderServiceName:   "Amazon Elastic Compute Cloud",
					ProviderServiceCode:   stringPtr("AmazonEC2"),
					NormalizedServiceType: "Virtual Machines",
					ServiceCategory:       "Compute & Web",
					ServiceFamily:         "Virtual Machines",
				})
				
				// Add region mapping
				regionRepo.AddRegion(database.ProviderAWS, "us-east-1", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AWSRegion:      stringPtr("us-east-1"),
					DisplayName:    "US East (N. Virginia)",
				})
				
				// Add unit mapping
				unitNorm.AddMapping(database.ProviderAWS, "Hrs", database.UnitHour)
			},
			expectedSuccess: true,
			expectedRecords: 1,
			expectedErrors:  0,
			expectedSkipped: 0,
		},
		{
			name: "service not mapped - skip",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "UnmappedService",
				Region:      "us-east-1",
				RawData:     getValidEC2PricingJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				// Don't add service mapping - should skip
				regionRepo.AddRegion(database.ProviderAWS, "us-east-1", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AWSRegion:      stringPtr("us-east-1"),
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
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
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
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     getZeroPriceEC2JSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				serviceRepo.AddMapping(database.ProviderAWS, "AmazonEC2", &database.ServiceMapping{
					ID:                    1,
					Provider:              database.ProviderAWS,
					ProviderServiceName:   "Amazon Elastic Compute Cloud",
					NormalizedServiceType: "Virtual Machines",
					ServiceCategory:       "Compute & Web",
				})
				regionRepo.AddRegion(database.ProviderAWS, "us-east-1", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AWSRegion:      stringPtr("us-east-1"),
				})
			},
			expectedSuccess: false,
			expectedRecords: 0,
			expectedErrors:  0,
			expectedSkipped: 1,
		},
		{
			name: "reserved instance pricing",
			input: database.NormalizationInput{
				Provider:    database.ProviderAWS,
				ServiceCode: "AmazonEC2",
				Region:      "us-east-1",
				RawData:     getReservedInstanceJSON(),
				RawDataID:   1,
			},
			setupMocks: func(serviceRepo *MockServiceMappingRepository, regionRepo *MockRegionMappingRepository, unitNorm *MockUnitNormalizer) {
				serviceRepo.AddMapping(database.ProviderAWS, "AmazonEC2", &database.ServiceMapping{
					ID:                    1,
					Provider:              database.ProviderAWS,
					ProviderServiceName:   "Amazon Elastic Compute Cloud",
					NormalizedServiceType: "Virtual Machines",
					ServiceCategory:       "Compute & Web",
				})
				regionRepo.AddRegion(database.ProviderAWS, "us-east-1", &database.NormalizedRegion{
					ID:             1,
					NormalizedCode: "us-east",
					AWSRegion:      stringPtr("us-east-1"),
				})
				unitNorm.AddMapping(database.ProviderAWS, "Hrs", database.UnitHour)
			},
			expectedSuccess: true,
			expectedRecords: 3, // 1yr upfront fee + 1yr hourly + 3yr hourly
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
			
			normalizer := NewAWSNormalizerV2(
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
					assert.Equal(t, database.ProviderAWS, record.Provider)
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

func TestAWSNormalizerV2_ExtractPricingFromDimension(t *testing.T) {
	normalizer := createTestAWSNormalizerV2()
	
	tests := []struct {
		name        string
		dimension   AWSPriceDimension
		expectError bool
		expected    *PricingInfo
	}{
		{
			name: "valid dimension",
			dimension: AWSPriceDimension{
				Description: "On Demand Linux t3.medium",
				Unit:        "Hrs",
				PricePerUnit: map[string]string{
					"USD": "0.0416",
				},
			},
			expectError: false,
			expected: &PricingInfo{
				PricePerUnit: 0.0416,
				Unit:         "Hrs",
				Currency:     "USD",
				Description:  "On Demand Linux t3.medium",
			},
		},
		{
			name: "empty price map",
			dimension: AWSPriceDimension{
				Description:  "Test",
				Unit:         "Hrs",
				PricePerUnit: map[string]string{},
			},
			expectError: true,
		},
		{
			name: "invalid price format",
			dimension: AWSPriceDimension{
				Description: "Test",
				Unit:        "Hrs",
				PricePerUnit: map[string]string{
					"USD": "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "multiple currencies - uses first",
			dimension: AWSPriceDimension{
				Description: "Test",
				Unit:        "Hrs",
				PricePerUnit: map[string]string{
					"USD": "1.00",
					"EUR": "0.85",
				},
			},
			expectError: false,
			expected: &PricingInfo{
				PricePerUnit: 1.00,
				Unit:         "Hrs",
				Currency:     "USD", // Or EUR, depends on map iteration
				Description:  "Test",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.extractPricingFromDimension(&tt.dimension)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				// For multiple currencies test, just check that we got a valid result
				if len(tt.dimension.PricePerUnit) > 1 {
					assert.Contains(t, []string{"USD", "EUR"}, result.Currency)
					assert.Contains(t, []float64{1.00, 0.85}, result.PricePerUnit)
				} else {
					assert.Equal(t, tt.expected.PricePerUnit, result.PricePerUnit)
					assert.Equal(t, tt.expected.Unit, result.Unit)
					assert.Equal(t, tt.expected.Currency, result.Currency)
					assert.Equal(t, tt.expected.Description, result.Description)
				}
			}
		})
	}
}

func TestAWSNormalizerV2_CreateResourceName(t *testing.T) {
	normalizer := createTestAWSNormalizerV2()
	
	tests := []struct {
		name         string
		attributes   map[string]interface{}
		serviceType  string
		expected     string
	}{
		{
			name: "EC2 instance type",
			attributes: map[string]interface{}{
				"instanceType": "t3.medium",
			},
			serviceType: "Virtual Machines",
			expected:    "t3.medium",
		},
		{
			name: "Lambda with architecture",
			attributes: map[string]interface{}{
				"architecture": "x86_64",
			},
			serviceType: "Serverless Functions",
			expected:    "Lambda (x86_64)",
		},
		{
			name: "Lambda without architecture",
			attributes:  map[string]interface{}{},
			serviceType: "Serverless Functions",
			expected:    "Lambda",
		},
		{
			name:        "Fargate",
			attributes:  map[string]interface{}{},
			serviceType: "Serverless Containers",
			expected:    "Fargate",
		},
		{
			name: "fallback to service name",
			attributes: map[string]interface{}{
				"serviceName": "Some Service",
			},
			serviceType: "Other",
			expected:    "Some Service",
		},
		{
			name:        "unknown",
			attributes:  map[string]interface{}{},
			serviceType: "Other",
			expected:    "Unknown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.createResourceName(tt.attributes, tt.serviceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAWSNormalizerV2_DetermineReservedPricingModel(t *testing.T) {
	normalizer := createTestAWSNormalizerV2()
	
	tests := []struct {
		name           string
		termAttributes map[string]interface{}
		expected       string
	}{
		{
			name: "1 year term",
			termAttributes: map[string]interface{}{
				"LeaseContractLength": "1yr",
			},
			expected: database.PricingModelReserved1Yr,
		},
		{
			name: "3 year term",
			termAttributes: map[string]interface{}{
				"LeaseContractLength": "3yr",
			},
			expected: database.PricingModelReserved3Yr,
		},
		{
			name:           "no attributes - default to 1yr",
			termAttributes: nil,
			expected:       database.PricingModelReserved1Yr,
		},
		{
			name:           "empty attributes - default to 1yr",
			termAttributes: map[string]interface{}{},
			expected:       database.PricingModelReserved1Yr,
		},
		{
			name: "unknown lease length - default to 1yr",
			termAttributes: map[string]interface{}{
				"LeaseContractLength": "2yr",
			},
			expected: database.PricingModelReserved1Yr,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.determineReservedPricingModel(tt.termAttributes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func createTestAWSNormalizerV2() *AWSNormalizerV2 {
	return NewAWSNormalizerV2(
		NewMockServiceMappingRepository(),
		NewMockRegionMappingRepository(),
		NewMockUnitNormalizer(),
		NewInputValidator(),
		NewMockLogger(),
	)
}

func getValidEC2PricingJSON() json.RawMessage {
	return json.RawMessage(`{
		"product": {
			"sku": "ABCDEFGH",
			"attributes": {
				"instanceType": "t3.medium",
				"vcpu": "2",
				"memory": "4 GiB",
				"operatingSystem": "Linux",
				"tenancy": "Shared"
			}
		},
		"terms": {
			"OnDemand": {
				"ABCDEFGH.JRTCKXETXF": {
					"offerTermCode": "JRTCKXETXF",
					"sku": "ABCDEFGH",
					"priceDimensions": {
						"ABCDEFGH.JRTCKXETXF.6YS6EN2CT7": {
							"description": "On Demand Linux t3.medium",
							"unit": "Hrs",
							"pricePerUnit": {
								"USD": "0.0416"
							}
						}
					}
				}
			}
		}
	}`)
}

func getZeroPriceEC2JSON() json.RawMessage {
	return json.RawMessage(`{
		"product": {
			"sku": "FREETER",
			"attributes": {
				"instanceType": "t2.micro",
				"vcpu": "1",
				"memory": "1 GiB"
			}
		},
		"terms": {
			"OnDemand": {
				"FREETER.JRTCKXETXF": {
					"offerTermCode": "JRTCKXETXF",
					"sku": "FREETER",
					"priceDimensions": {
						"FREETER.JRTCKXETXF.6YS6EN2CT7": {
							"description": "Free tier t2.micro",
							"unit": "Hrs",
							"pricePerUnit": {
								"USD": "0.0000"
							}
						}
					}
				}
			}
		}
	}`)
}

func getReservedInstanceJSON() json.RawMessage {
	return json.RawMessage(`{
		"product": {
			"sku": "RESERVED123",
			"attributes": {
				"instanceType": "m5.large",
				"vcpu": "2",
				"memory": "8 GiB"
			}
		},
		"terms": {
			"Reserved": {
				"RESERVED123.NQ3QZPMQV9": {
					"offerTermCode": "NQ3QZPMQV9",
					"sku": "RESERVED123",
					"termAttributes": {
						"LeaseContractLength": "1yr",
						"PurchaseOption": "All Upfront"
					},
					"priceDimensions": {
						"RESERVED123.NQ3QZPMQV9.2TG2D8R56U": {
							"description": "Upfront Fee",
							"unit": "Quantity",
							"pricePerUnit": {
								"USD": "500"
							}
						},
						"RESERVED123.NQ3QZPMQV9.6YS6EN2CT7": {
							"description": "Linux m5.large Reserved Instance",
							"unit": "Hrs",
							"pricePerUnit": {
								"USD": "0.057"
							}
						}
					}
				},
				"RESERVED123.38NPMPTW36": {
					"offerTermCode": "38NPMPTW36",
					"sku": "RESERVED123",
					"termAttributes": {
						"LeaseContractLength": "3yr",
						"PurchaseOption": "Partial Upfront"
					},
					"priceDimensions": {
						"RESERVED123.38NPMPTW36.6YS6EN2CT7": {
							"description": "Linux m5.large Reserved Instance",
							"unit": "Hrs",
							"pricePerUnit": {
								"USD": "0.038"
							}
						}
					}
				}
			}
		}
	}`)
}