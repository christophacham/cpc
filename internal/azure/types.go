package azure

import "time"

// AzureAPIResponse represents the Azure pricing API response
type AzureAPIResponse struct {
	BillingCurrency    string                   `json:"BillingCurrency"`
	CustomerEntityID   string                   `json:"CustomerEntityId"`
	CustomerEntityType string                   `json:"CustomerEntityType"`
	Items              []map[string]interface{} `json:"Items"`
	NextPageLink       string                   `json:"NextPageLink"`
	Count              int                      `json:"Count"`
}

// PricingItem represents a standardized pricing item structure
type PricingItem struct {
	CurrencyCode         string  `json:"currencyCode"`
	TierMinimumUnits     float64 `json:"tierMinimumUnits"`
	RetailPrice          float64 `json:"retailPrice"`
	UnitPrice            float64 `json:"unitPrice"`
	ArmRegionName        string  `json:"armRegionName"`
	Location             string  `json:"location"`
	EffectiveStartDate   string  `json:"effectiveStartDate"`
	MeterID              string  `json:"meterId"`
	MeterName            string  `json:"meterName"`
	ProductID            string  `json:"productId"`
	SkuID                string  `json:"skuId"`
	ProductName          string  `json:"productName"`
	SkuName              string  `json:"skuName"`
	ServiceName          string  `json:"serviceName"`
	ServiceID            string  `json:"serviceId"`
	ServiceFamily        string  `json:"serviceFamily"`
	UnitOfMeasure        string  `json:"unitOfMeasure"`
	Type                 string  `json:"type"`
	IsPrimaryMeterRegion bool    `json:"isPrimaryMeterRegion"`
	ArmSkuName           string  `json:"armSkuName"`
	ReservationTerm      string  `json:"reservationTerm,omitempty"`
}

// CollectionConfig holds configuration for Azure data collection
type CollectionConfig struct {
	Mode            string   `json:"mode"`            // "production", "explorer", "admin"
	Regions         string   `json:"regions"`         // "single", "all", "limited"
	StorageType     string   `json:"storage_type"`    // "jsonb", "database", "none"
	OutputType      string   `json:"output_type"`     // "console", "json-export", "database"
	Concurrency     int      `json:"concurrency"`     // for multi-region concurrent workers
	EnableTracking  bool     `json:"enable_tracking"` // progress tracking toggle
	SamplingEnabled bool     `json:"sampling"`        // for explorer mode sampling
	MaxItems        int      `json:"max_items"`       // limit for testing/exploration
	ExportFile      string   `json:"export_file"`     // JSON export file path
	TargetRegion    string   `json:"target_region"`   // specific region for single mode
	TestRegions     []string `json:"test_regions"`    // regions for limited mode
}

// DefaultConfig returns a default configuration for production use
func DefaultConfig() CollectionConfig {
	return CollectionConfig{
		Mode:            "production",
		Regions:         "single",
		StorageType:     "jsonb",
		OutputType:      "database",
		Concurrency:     3,
		EnableTracking:  true,
		SamplingEnabled: false,
		MaxItems:        0,
		TargetRegion:    "eastus",
		TestRegions:     []string{"eastus", "westus", "northeurope", "southeastasia"},
	}
}

// CollectionStats tracks collection statistics
type CollectionStats struct {
	TotalItems      int                     `json:"total_items"`
	UniqueServices  map[string]int          `json:"unique_services"`
	UniqueRegions   map[string]int          `json:"unique_regions"`
	UniqueSKUs      map[string]int          `json:"unique_skus"`
	StartTime       time.Time               `json:"start_time"`
	Duration        time.Duration           `json:"duration"`
	RegionProgress  map[string]RegionStatus `json:"region_progress"`
}

// RegionStatus tracks the status of region collection
type RegionStatus struct {
	Status      string    `json:"status"`      // "pending", "running", "completed", "failed"
	ItemCount   int       `json:"item_count"`
	StartTime   time.Time `json:"start_time"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}