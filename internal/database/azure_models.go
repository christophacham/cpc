package database

import (
	"time"
)

// AzureService represents an Azure service
type AzureService struct {
	ID            int       `db:"id"`
	ServiceName   string    `db:"service_name"`
	ServiceFamily string    `db:"service_family"`
	CategoryID    *int      `db:"category_id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// AzureRegion represents an Azure region
type AzureRegion struct {
	ID            int       `db:"id"`
	ARMRegionName string    `db:"arm_region_name"`
	DisplayName   string    `db:"display_name"`
	CreatedAt     time.Time `db:"created_at"`
}

// AzureProduct represents a product within a service
type AzureProduct struct {
	ID          int       `db:"id"`
	ServiceID   int       `db:"service_id"`
	ProductName string    `db:"product_name"`
	ProductID   *string   `db:"product_id"`
	CreatedAt   time.Time `db:"created_at"`
}

// AzureSKU represents a SKU within a product
type AzureSKU struct {
	ID         int       `db:"id"`
	ProductID  int       `db:"product_id"`
	SKUName    string    `db:"sku_name"`
	SKUID      *string   `db:"sku_id"`
	ARMSKUName *string   `db:"arm_sku_name"`
	CreatedAt  time.Time `db:"created_at"`
}

// AzurePricing represents a pricing record
type AzurePricing struct {
	ID                   int64     `db:"id"`
	ServiceID            int       `db:"service_id"`
	ProductID            int       `db:"product_id"`
	SKUID                int       `db:"sku_id"`
	RegionID             int       `db:"region_id"`
	MeterID              string    `db:"meter_id"`
	MeterName            string    `db:"meter_name"`
	RetailPrice          float64   `db:"retail_price"`
	UnitPrice            float64   `db:"unit_price"`
	TierMinimumUnits     float64   `db:"tier_minimum_units"`
	CurrencyCode         string    `db:"currency_code"`
	UnitOfMeasure        string    `db:"unit_of_measure"`
	PriceType            string    `db:"price_type"`
	ReservationTerm      *string   `db:"reservation_term"`
	EffectiveStartDate   time.Time `db:"effective_start_date"`
	IsPrimaryMeterRegion bool      `db:"is_primary_meter_region"`
	CreatedAt            time.Time `db:"created_at"`
	CollectionVersion    int       `db:"collection_version"`
}

// AzureCollectionRun tracks collection runs
type AzureCollectionRun struct {
	ID                  int       `db:"id"`
	Version             int       `db:"version"`
	StartedAt           time.Time `db:"started_at"`
	CompletedAt         *time.Time `db:"completed_at"`
	Status              string    `db:"status"`
	TotalItems          int       `db:"total_items"`
	RegionsCollected    []string  `db:"regions_collected"`
	ErrorMessage        *string   `db:"error_message"`
	CollectionMetadata  string    `db:"collection_metadata"` // JSON
}

// AzurePricingInsert represents data for bulk insert
type AzurePricingInsert struct {
	ServiceName          string
	ServiceFamily        string
	ProductName          string
	ProductID            string
	SKUName              string
	SKUID                string
	ARMSKUName           string
	ARMRegionName        string
	DisplayName          string
	MeterID              string
	MeterName            string
	RetailPrice          float64
	UnitPrice            float64
	TierMinimumUnits     float64
	CurrencyCode         string
	UnitOfMeasure        string
	PriceType            string
	ReservationTerm      string
	EffectiveStartDate   time.Time
	IsPrimaryMeterRegion bool
	CollectionVersion    int
}