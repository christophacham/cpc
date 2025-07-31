package database

import (
	"encoding/json"
	"time"
)

// NormalizedPricing represents a unified pricing record from AWS or Azure
type NormalizedPricing struct {
	ID                   int                    `json:"id" db:"id"`
	Provider             string                 `json:"provider" db:"provider"`
	ProviderServiceCode  string                 `json:"providerServiceCode" db:"provider_service_code"`
	ProviderSKU          *string                `json:"providerSku,omitempty" db:"provider_sku"`
	ServiceMappingID     *int                   `json:"serviceMappingId,omitempty" db:"service_mapping_id"`
	ServiceCategory      string                 `json:"serviceCategory" db:"service_category"`
	ServiceFamily        string                 `json:"serviceFamily" db:"service_family"`
	ServiceType          string                 `json:"serviceType" db:"service_type"`
	RegionID             *int                   `json:"regionId,omitempty" db:"region_id"`
	NormalizedRegion     string                 `json:"normalizedRegion" db:"normalized_region"`
	ProviderRegion       string                 `json:"providerRegion" db:"provider_region"`
	ResourceName         string                 `json:"resourceName" db:"resource_name"`
	ResourceDescription  *string                `json:"resourceDescription,omitempty" db:"resource_description"`
	ResourceSpecs        ResourceSpecs          `json:"resourceSpecs" db:"resource_specs"`
	PricePerUnit         float64                `json:"pricePerUnit" db:"price_per_unit"`
	Unit                 string                 `json:"unit" db:"unit"`
	Currency             string                 `json:"currency" db:"currency"`
	PricingModel         string                 `json:"pricingModel" db:"pricing_model"`
	PricingDetails       PricingDetails         `json:"pricingDetails,omitempty" db:"pricing_details"`
	EffectiveDate        *time.Time             `json:"effectiveDate,omitempty" db:"effective_date"`
	ExpirationDate       *time.Time             `json:"expirationDate,omitempty" db:"expiration_date"`
	MinimumCommitment    int                    `json:"minimumCommitment" db:"minimum_commitment"`
	AWSRawID             *int                   `json:"awsRawId,omitempty" db:"aws_raw_id"`
	AzureRawID           *int                   `json:"azureRawId,omitempty" db:"azure_raw_id"`
	CreatedAt            time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time              `json:"updatedAt" db:"updated_at"`
}

// ResourceSpecs represents the standardized resource specifications
type ResourceSpecs struct {
	VCPU               *int     `json:"vcpu,omitempty"`
	MemoryGB           *float64 `json:"memory_gb,omitempty"`
	StorageGB          *float64 `json:"storage_gb,omitempty"`
	GPUCount           *int     `json:"gpu_count,omitempty"`
	GPUMemoryGB        *float64 `json:"gpu_memory_gb,omitempty"`
	NetworkPerformance *string  `json:"network_performance,omitempty"`
	StorageType        *string  `json:"storage_type,omitempty"`
	ProcessorType      *string  `json:"processor_type,omitempty"`
	ProcessorFeatures  []string `json:"processor_features,omitempty"`
	Architecture       *string  `json:"architecture,omitempty"`
	ClockSpeedGHz      *float64 `json:"clock_speed_ghz,omitempty"`
	Burstable          *bool    `json:"burstable,omitempty"`
}

// PricingDetails represents additional pricing information for reserved/savings plans
type PricingDetails struct {
	TermLength     *string  `json:"term_length,omitempty"`     // "1yr", "3yr"
	PaymentOption  *string  `json:"payment_option,omitempty"`  // "all_upfront", "partial_upfront", "no_upfront"
	UpfrontCost    *float64 `json:"upfront_cost,omitempty"`
	HourlyRate     *float64 `json:"hourly_rate,omitempty"`
	SavingsPercent *float64 `json:"savings_percent,omitempty"` // Compared to on-demand
}

// ServiceMapping represents the mapping between provider services and normalized categories
type ServiceMapping struct {
	ID                     int    `json:"id" db:"id"`
	Provider               string `json:"provider" db:"provider"`
	ProviderServiceName    string `json:"providerServiceName" db:"provider_service_name"`
	ProviderServiceCode    *string `json:"providerServiceCode,omitempty" db:"provider_service_code"`
	NormalizedServiceType  string `json:"normalizedServiceType" db:"normalized_service_type"`
	ServiceCategory        string `json:"serviceCategory" db:"service_category"`
	ServiceFamily          string `json:"serviceFamily" db:"service_family"`
}

// NormalizedRegion represents the standardized region mapping
type NormalizedRegion struct {
	ID             int     `json:"id" db:"id"`
	NormalizedCode string  `json:"normalizedCode" db:"normalized_code"`
	AWSRegion      *string `json:"awsRegion,omitempty" db:"aws_region"`
	AzureRegion    *string `json:"azureRegion,omitempty" db:"azure_region"`
	DisplayName    string  `json:"displayName" db:"display_name"`
	Country        *string `json:"country,omitempty" db:"country"`
	Continent      *string `json:"continent,omitempty" db:"continent"`
}

// NormalizationInput represents the input for pricing normalization
type NormalizationInput struct {
	Provider         string          `json:"provider"`
	ServiceCode      string          `json:"serviceCode"`
	Region           string          `json:"region"`
	RawData          json.RawMessage `json:"rawData"`
	RawDataID        int             `json:"rawDataId"`
	CollectionID     string          `json:"collectionId"`
}

// NormalizationResult represents the result of pricing normalization
type NormalizationResult struct {
	Success           bool                `json:"success"`
	NormalizedRecords []NormalizedPricing `json:"normalizedRecords"`
	SkippedCount      int                 `json:"skippedCount"`
	ErrorCount        int                 `json:"errorCount"`
	Errors            []string            `json:"errors,omitempty"`
}

// PricingComparison represents a comparison between equivalent services
type PricingComparison struct {
	ServiceType      string                   `json:"serviceType"`
	NormalizedRegion string                   `json:"normalizedRegion"`
	PricingModel     string                   `json:"pricingModel"`
	ResourceSpecs    ResourceSpecs            `json:"resourceSpecs"`
	AWS              *NormalizedPricing       `json:"aws,omitempty"`
	Azure            *NormalizedPricing       `json:"azure,omitempty"`
	PriceDifference  *float64                 `json:"priceDifference,omitempty"` // AWS - Azure
	CheaperProvider  *string                  `json:"cheaperProvider,omitempty"`
	SavingsPercent   *float64                 `json:"savingsPercent,omitempty"`
}

// PricingFilter represents filters for normalized pricing queries
type PricingFilter struct {
	Provider         *string       `json:"provider,omitempty"`
	ServiceCategory  *string       `json:"serviceCategory,omitempty"`
	ServiceFamily    *string       `json:"serviceFamily,omitempty"`
	ServiceType      *string       `json:"serviceType,omitempty"`
	NormalizedRegion *string       `json:"normalizedRegion,omitempty"`
	PricingModel     *string       `json:"pricingModel,omitempty"`
	Currency         *string       `json:"currency,omitempty"`
	ResourceSpecs    *ResourceSpecs `json:"resourceSpecs,omitempty"`
	MaxPricePerUnit  *float64      `json:"maxPricePerUnit,omitempty"`
	MinPricePerUnit  *float64      `json:"minPricePerUnit,omitempty"`
	Limit            *int          `json:"limit,omitempty"`
	Offset           *int          `json:"offset,omitempty"`
	OrderBy          *string       `json:"orderBy,omitempty"` // "price_per_unit", "resource_name", etc.
	OrderDirection   *string       `json:"orderDirection,omitempty"` // "ASC", "DESC"
}

// Constants for standardized values
const (
	// Providers
	ProviderAWS   = "aws"
	ProviderAzure = "azure"

	// Pricing Models
	PricingModelOnDemand    = "on_demand"
	PricingModelReserved1Yr = "reserved_1yr"
	PricingModelReserved3Yr = "reserved_3yr"
	PricingModelSpot        = "spot"
	PricingModelSavingsPlan = "savings_plan"

	// Units (standardized)
	UnitHour            = "hour"
	UnitGBMonth         = "gb_month"
	UnitRequest         = "request"
	UnitMillionRequests = "million_requests"
	UnitTransaction     = "transaction"
	UnitGB              = "gb"
	UnitTB              = "tb"
	UnitInstance        = "instance"

	// Service Categories
	CategoryGeneral         = "General"
	CategoryNetworking      = "Networking"
	CategoryComputeWeb      = "Compute & Web"
	CategoryContainers      = "Containers"
	CategoryDatabases       = "Databases"
	CategoryStorage         = "Storage"
	CategoryAIML            = "AI & ML"
	CategoryAnalyticsIoT    = "Analytics & IoT"
	CategoryVirtualDesktop  = "Virtual Desktop"
	CategoryDevTools        = "Dev Tools"
	CategoryIntegration     = "Integration"
	CategoryMigration       = "Migration"
	CategoryManagement      = "Management"

	// Service Families
	ServiceFamilyVirtualMachines    = "Virtual Machines"
	ServiceFamilyServerless         = "Serverless"
	ServiceFamilyContainerOrch      = "Container Orchestration"
	ServiceFamilyServerlessContainers = "Serverless Containers"
	ServiceFamilyPaaS               = "PaaS"
	ServiceFamilyBatchComputing     = "Batch Computing"
	ServiceFamilyHPC                = "HPC"
	ServiceFamilyHybridCloud        = "Hybrid Cloud"
	ServiceFamilyEdgeComputing      = "Edge Computing"
	ServiceFamilyGPUComputing       = "GPU Computing"
	ServiceFamilyQuantumComputing   = "Quantum Computing"
	ServiceFamilyDevelopment        = "Development"
	ServiceFamilyTesting            = "Testing"
	ServiceFamilyIoTEdge            = "IoT Edge"
)

// Helper methods for ResourceSpecs

// HasGPU returns true if the resource has GPU capabilities
func (rs ResourceSpecs) HasGPU() bool {
	return rs.GPUCount != nil && *rs.GPUCount > 0
}

// HasStorage returns true if the resource has local storage
func (rs ResourceSpecs) HasStorage() bool {
	return rs.StorageGB != nil && *rs.StorageGB > 0
}

// IsBurstable returns true if the resource supports burstable performance
func (rs ResourceSpecs) IsBurstable() bool {
	return rs.Burstable != nil && *rs.Burstable
}

// GetVCPU returns the vCPU count or 0 if not specified
func (rs ResourceSpecs) GetVCPU() int {
	if rs.VCPU != nil {
		return *rs.VCPU
	}
	return 0
}

// GetMemoryGB returns the memory in GB or 0 if not specified
func (rs ResourceSpecs) GetMemoryGB() float64 {
	if rs.MemoryGB != nil {
		return *rs.MemoryGB
	}
	return 0
}

// Helper methods for NormalizedPricing

// IsAWS returns true if this is an AWS pricing record
func (np NormalizedPricing) IsAWS() bool {
	return np.Provider == ProviderAWS
}

// IsAzure returns true if this is an Azure pricing record
func (np NormalizedPricing) IsAzure() bool {
	return np.Provider == ProviderAzure
}

// IsOnDemand returns true if this is on-demand pricing
func (np NormalizedPricing) IsOnDemand() bool {
	return np.PricingModel == PricingModelOnDemand
}

// IsReserved returns true if this is reserved instance pricing
func (np NormalizedPricing) IsReserved() bool {
	return np.PricingModel == PricingModelReserved1Yr || np.PricingModel == PricingModelReserved3Yr
}

// IsSpot returns true if this is spot/preemptible pricing
func (np NormalizedPricing) IsSpot() bool {
	return np.PricingModel == PricingModelSpot
}

// GetHourlyPrice returns the effective hourly price considering upfront costs
func (np NormalizedPricing) GetHourlyPrice() float64 {
	if np.Unit == UnitHour {
		return np.PricePerUnit
	}
	
	// For reserved instances, calculate effective hourly rate
	if np.IsReserved() && np.PricingDetails.HourlyRate != nil {
		return *np.PricingDetails.HourlyRate
	}
	
	return np.PricePerUnit
}