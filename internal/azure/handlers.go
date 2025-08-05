package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/raulc0399/cpc/internal/database"
)

// RegionHandler implementations

type SingleRegionHandler struct {
	region   string
	maxItems int
	client   *Client
}

func (h *SingleRegionHandler) GetRegions() []string {
	return []string{h.region}
}

func (h *SingleRegionHandler) Collect(ctx context.Context, region string) ([]map[string]interface{}, error) {
	filter := fmt.Sprintf("armRegionName eq '%s'", region)
	return h.client.QueryRawWithPagination(filter, h.maxItems)
}

func (h *SingleRegionHandler) SetMaxItems(max int) {
	h.maxItems = max
}

func (h *SingleRegionHandler) SetClient(client *Client) {
	h.client = client
}

type MultiRegionHandler struct {
	regions  []string
	maxItems int
	client   *Client
}

func (h *MultiRegionHandler) GetRegions() []string {
	return h.regions
}

func (h *MultiRegionHandler) Collect(ctx context.Context, region string) ([]map[string]interface{}, error) {
	filter := fmt.Sprintf("armRegionName eq '%s'", region)
	return h.client.QueryRawWithPagination(filter, h.maxItems)
}

func (h *MultiRegionHandler) SetMaxItems(max int) {
	h.maxItems = max
}

func (h *MultiRegionHandler) SetClient(client *Client) {
	h.client = client
}

type LimitedRegionHandler struct {
	regions  []string
	maxItems int
	client   *Client
}

func (h *LimitedRegionHandler) GetRegions() []string {
	return h.regions
}

func (h *LimitedRegionHandler) Collect(ctx context.Context, region string) ([]map[string]interface{}, error) {
	filter := fmt.Sprintf("armRegionName eq '%s'", region)
	return h.client.QueryRawWithPagination(filter, h.maxItems)
}

func (h *LimitedRegionHandler) SetMaxItems(max int) {
	h.maxItems = max
}

func (h *LimitedRegionHandler) SetClient(client *Client) {
	h.client = client
}

// DataStore implementations

type JSONBDataStore struct {
	db *database.DB
}

func (ds *JSONBDataStore) Store(ctx context.Context, collectionID string, region string, data []map[string]interface{}) error {
	for _, item := range data {
		jsonData, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}
		
		query := `INSERT INTO azure_pricing_raw (collection_id, region, data, created_at) VALUES ($1, $2, $3, NOW())`
		if _, err := ds.db.Exec(query, collectionID, region, jsonData); err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}
	}
	return nil
}

func (ds *JSONBDataStore) StartCollection(region string) (string, error) {
	collectionID := fmt.Sprintf("%s-%d", region, time.Now().Unix())
	query := `INSERT INTO collection_progress (collection_id, region, status, started_at) VALUES ($1, $2, 'running', NOW())`
	if _, err := ds.db.Exec(query, collectionID, region); err != nil {
		return "", fmt.Errorf("failed to start collection: %w", err)
	}
	return collectionID, nil
}

func (ds *JSONBDataStore) CompleteCollection(collectionID string, totalItems int) error {
	query := `UPDATE collection_progress SET status = 'completed', total_items = $2, completed_at = NOW() WHERE collection_id = $1`
	if _, err := ds.db.Exec(query, collectionID, totalItems); err != nil {
		return fmt.Errorf("failed to complete collection: %w", err)
	}
	return nil
}

func (ds *JSONBDataStore) FailCollection(collectionID string, errorMsg string) error {
	query := `UPDATE collection_progress SET status = 'failed', error_message = $2, completed_at = NOW() WHERE collection_id = $1`
	if _, err := ds.db.Exec(query, collectionID, errorMsg); err != nil {
		return fmt.Errorf("failed to mark collection as failed: %w", err)
	}
	return nil
}

type StructuredDataStore struct {
	db *database.DB
}

func (ds *StructuredDataStore) Store(ctx context.Context, collectionID string, region string, data []map[string]interface{}) error {
	for _, rawItem := range data {
		item := convertRawToPricingItem(rawItem)
		query := `INSERT INTO azure_pricing (
			collection_id, region, currency_code, tier_minimum_units, retail_price, unit_price,
			arm_region_name, location, effective_start_date, meter_id, meter_name, product_id,
			sku_id, product_name, sku_name, service_name, service_id, service_family,
			unit_of_measure, type, is_primary_meter_region, arm_sku_name, reservation_term,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, NOW())`
		
		if _, err := ds.db.Exec(query,
			collectionID, region, item.CurrencyCode, item.TierMinimumUnits, item.RetailPrice,
			item.UnitPrice, item.ArmRegionName, item.Location, item.EffectiveStartDate,
			item.MeterID, item.MeterName, item.ProductID, item.SkuID, item.ProductName,
			item.SkuName, item.ServiceName, item.ServiceID, item.ServiceFamily,
			item.UnitOfMeasure, item.Type, item.IsPrimaryMeterRegion, item.ArmSkuName,
			item.ReservationTerm,
		); err != nil {
			return fmt.Errorf("failed to insert structured item: %w", err)
		}
	}
	return nil
}

func (ds *StructuredDataStore) StartCollection(region string) (string, error) {
	collectionID := fmt.Sprintf("%s-%d", region, time.Now().Unix())
	query := `INSERT INTO collection_progress (collection_id, region, status, started_at) VALUES ($1, $2, 'running', NOW())`
	if _, err := ds.db.Exec(query, collectionID, region); err != nil {
		return "", fmt.Errorf("failed to start collection: %w", err)
	}
	return collectionID, nil
}

func (ds *StructuredDataStore) CompleteCollection(collectionID string, totalItems int) error {
	query := `UPDATE collection_progress SET status = 'completed', total_items = $2, completed_at = NOW() WHERE collection_id = $1`
	if _, err := ds.db.Exec(query, collectionID, totalItems); err != nil {
		return fmt.Errorf("failed to complete collection: %w", err)
	}
	return nil
}

func (ds *StructuredDataStore) FailCollection(collectionID string, errorMsg string) error {
	query := `UPDATE collection_progress SET status = 'failed', error_message = $2, completed_at = NOW() WHERE collection_id = $1`
	if _, err := ds.db.Exec(query, collectionID, errorMsg); err != nil {
		return fmt.Errorf("failed to mark collection as failed: %w", err)
	}
	return nil
}

type NoOpDataStore struct{}

func (ds *NoOpDataStore) Store(ctx context.Context, collectionID string, region string, data []map[string]interface{}) error {
	return nil // No-op
}

func (ds *NoOpDataStore) StartCollection(region string) (string, error) {
	return fmt.Sprintf("%s-%d", region, time.Now().Unix()), nil
}

func (ds *NoOpDataStore) CompleteCollection(collectionID string, totalItems int) error {
	return nil // No-op
}

func (ds *NoOpDataStore) FailCollection(collectionID string, errorMsg string) error {
	return nil // No-op
}

// OutputHandler implementations

type ConsoleOutputHandler struct {
	samplingEnabled bool
}

func (h *ConsoleOutputHandler) Write(data []PricingItem) error {
	if h.samplingEnabled && len(data) > 10 {
		log.Printf("📊 Sample of %d items:", len(data))
		for i := 0; i < 10; i++ {
			item := data[i]
			log.Printf("  %s - %s: $%.4f (%s)", item.ServiceName, item.ProductName, item.UnitPrice, item.Location)
		}
		log.Printf("  ... and %d more items", len(data)-10)
	} else {
		for _, item := range data {
			log.Printf("%s - %s: $%.4f (%s)", item.ServiceName, item.ProductName, item.UnitPrice, item.Location)
		}
	}
	return nil
}

func (h *ConsoleOutputHandler) WriteStats(stats CollectionStats) error {
	log.Printf("📈 Collection Statistics:")
	log.Printf("  Total Items: %d", stats.TotalItems)
	log.Printf("  Unique Services: %d", stats.UniqueServices)
	log.Printf("  Collection Time: %v", stats.Duration)
	return nil
}

func (h *ConsoleOutputHandler) WriteAnalysis(analysis map[string]interface{}) error {
	log.Printf("📊 Analysis Results:")
	for key, value := range analysis {
		log.Printf("  %s: %v", key, value)
	}
	return nil
}

type JSONExportOutputHandler struct {
	filePath string
}

func (h *JSONExportOutputHandler) Write(data []PricingItem) error {
	file, err := os.Create(h.filePath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	log.Printf("💾 Exported %d items to %s", len(data), h.filePath)
	return nil
}

func (h *JSONExportOutputHandler) WriteStats(stats CollectionStats) error {
	statsFile := fmt.Sprintf("%s.stats", h.filePath)
	file, err := os.Create(statsFile)
	if err != nil {
		return fmt.Errorf("failed to create stats file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(stats)
}

func (h *JSONExportOutputHandler) WriteAnalysis(analysis map[string]interface{}) error {
	analysisFile := fmt.Sprintf("%s.analysis", h.filePath)
	file, err := os.Create(analysisFile)
	if err != nil {
		return fmt.Errorf("failed to create analysis file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(analysis)
}

type DatabaseOutputHandler struct{}

func (h *DatabaseOutputHandler) Write(data []PricingItem) error {
	return nil // Data already stored via DataStore
}

func (h *DatabaseOutputHandler) WriteStats(stats CollectionStats) error {
	return nil // Stats handled separately
}

func (h *DatabaseOutputHandler) WriteAnalysis(analysis map[string]interface{}) error {
	return nil // Analysis handled separately
}

// ProgressTracker implementations

type DatabaseProgressTracker struct {
	startTime time.Time
	regions   map[string]bool
	workers   map[int]string
}

func (pt *DatabaseProgressTracker) Start(totalRegions int) {
	pt.startTime = time.Now()
	pt.regions = make(map[string]bool)
	pt.workers = make(map[int]string)
	log.Printf("🚀 Starting collection of %d regions", totalRegions)
}

func (pt *DatabaseProgressTracker) Update(region string, itemCount int, status string) {
	log.Printf("📊 %s: %d items (%s)", region, itemCount, status)
}

func (pt *DatabaseProgressTracker) Complete(region string, success bool, itemCount int) {
	pt.regions[region] = success
	if success {
		log.Printf("✅ %s: %d items collected", region, itemCount)
	} else {
		log.Printf("❌ %s: collection failed", region)
	}
}

func (pt *DatabaseProgressTracker) GetStatus() (completed int, failed int, total int, elapsed time.Duration) {
	for _, success := range pt.regions {
		total++
		if success {
			completed++
		} else {
			failed++
		}
	}
	elapsed = time.Since(pt.startTime)
	return
}

func (pt *DatabaseProgressTracker) SetWorking(workerID int, region string) {
	pt.workers[workerID] = region
	log.Printf("🔄 Worker %d: collecting %s", workerID, region)
}

func (pt *DatabaseProgressTracker) ClearWorking(workerID int) {
	delete(pt.workers, workerID)
}

type NoOpProgressTracker struct{}

func (pt *NoOpProgressTracker) Start(totalRegions int)                                        {}
func (pt *NoOpProgressTracker) Update(region string, itemCount int, status string)           {}
func (pt *NoOpProgressTracker) Complete(region string, success bool, itemCount int)          {}
func (pt *NoOpProgressTracker) SetWorking(workerID int, region string)                       {}
func (pt *NoOpProgressTracker) ClearWorking(workerID int)                                     {}
func (pt *NoOpProgressTracker) GetStatus() (int, int, int, time.Duration) { return 0, 0, 0, 0 }