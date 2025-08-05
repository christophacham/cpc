package azure

import (
	"context"
	"time"
)

// RegionHandler defines the interface for region-based data collection
type RegionHandler interface {
	GetRegions() []string
	Collect(ctx context.Context, region string) ([]map[string]interface{}, error)
	SetMaxItems(max int)
	SetClient(client *Client)
}

// DataStore defines the interface for data storage
type DataStore interface {
	Store(ctx context.Context, collectionID string, region string, data []map[string]interface{}) error
	StartCollection(region string) (string, error)
	CompleteCollection(collectionID string, totalItems int) error
	FailCollection(collectionID string, errorMsg string) error
}

// OutputHandler defines the interface for data output
type OutputHandler interface {
	Write(data []PricingItem) error
	WriteStats(stats CollectionStats) error
	WriteAnalysis(analysis map[string]interface{}) error
}

// ProgressTracker defines the interface for progress tracking
type ProgressTracker interface {
	Start(totalRegions int)
	Update(region string, itemCount int, status string)
	Complete(region string, success bool, itemCount int)
	GetStatus() (completed int, failed int, total int, elapsed time.Duration)
	SetWorking(workerID int, region string)
	ClearWorking(workerID int)
}

// ServiceAnalyzer defines the interface for service-based analysis
type ServiceAnalyzer interface {
	AnalyzeByService(items []PricingItem) map[string]ServiceSummary
	AnalyzeDataShape(items []PricingItem) map[string]interface{}
	GetTopServices(items []PricingItem, n int) []KeyValue
}

// ServiceSummary represents analysis of a specific service
type ServiceSummary struct {
	ServiceName    string             `json:"service_name"`
	ServiceFamily  string             `json:"service_family"`
	TotalItems     int                `json:"total_items"`
	UniqueProducts map[string]int     `json:"unique_products"`
	UniqueSKUs     map[string]int     `json:"unique_skus"`
	UniqueMeters   map[string]int     `json:"unique_meters"`
	PriceRange     PriceRange         `json:"price_range"`
	SampleItems    []PricingItem      `json:"sample_items"`
}

// PriceRange represents min/max price range
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// KeyValue represents a key-value pair for statistics
type KeyValue struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}