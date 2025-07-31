package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AzureRawPricing represents raw Azure pricing data
type AzureRawPricing struct {
	ID           int                    `json:"id"`
	Region       string                 `json:"region"`
	ServiceName  *string                `json:"serviceName"`
	ServiceFamily *string               `json:"serviceFamily"`
	Data         map[string]interface{} `json:"data"`
	CollectedAt  time.Time              `json:"collectedAt"`
	CollectionID string                 `json:"collectionId"`
	TotalItems   int                    `json:"totalItems"`
}

// AzureCollection represents a collection run
type AzureCollection struct {
	ID           int                    `json:"id"`
	CollectionID string                 `json:"collectionId"`
	Region       string                 `json:"region"`
	Status       string                 `json:"status"`
	StartedAt    time.Time              `json:"startedAt"`
	CompletedAt  *time.Time             `json:"completedAt"`
	TotalItems   int                    `json:"totalItems"`
	ErrorMessage *string                `json:"errorMessage"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// StartAzureRawCollection starts a new raw data collection
func (db *DB) StartAzureRawCollection(region string) (string, error) {
	collectionID := uuid.New().String()
	
	metadata := map[string]interface{}{
		"approach": "raw_json",
		"progress": map[string]interface{}{
			"current_page": 0,
			"total_pages": 0,
			"items_collected": 0,
			"last_update": time.Now().Format(time.RFC3339),
		},
	}
	
	metadataJSON, _ := json.Marshal(metadata)
	
	_, err := db.conn.Exec(`
		INSERT INTO azure_collections (collection_id, region, status, started_at, metadata)
		VALUES ($1, $2, 'running', $3, $4)`,
		collectionID, region, time.Now(), metadataJSON)
	
	if err != nil {
		return "", fmt.Errorf("error starting collection: %w", err)
	}
	
	return collectionID, nil
}

// UpdateAzureRawCollectionProgress updates the progress of a collection
func (db *DB) UpdateAzureRawCollectionProgress(collectionID string, currentPage, totalPages, itemsCollected int, statusMessage string) error {
	// Get current metadata
	var metadataJSON []byte
	err := db.conn.QueryRow(`
		SELECT metadata FROM azure_collections WHERE collection_id = $1`,
		collectionID).Scan(&metadataJSON)
	
	if err != nil {
		return fmt.Errorf("error getting current metadata: %w", err)
	}
	
	// Parse current metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return fmt.Errorf("error parsing metadata: %w", err)
	}
	
	// Update progress
	if metadata["progress"] == nil {
		metadata["progress"] = make(map[string]interface{})
	}
	
	progress := metadata["progress"].(map[string]interface{})
	progress["current_page"] = currentPage
	progress["total_pages"] = totalPages
	progress["items_collected"] = itemsCollected
	progress["last_update"] = time.Now().Format(time.RFC3339)
	progress["status_message"] = statusMessage
	
	// Marshal back to JSON
	updatedMetadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}
	
	// Update database
	_, err = db.conn.Exec(`
		UPDATE azure_collections 
		SET metadata = $1, total_items = $2
		WHERE collection_id = $3`,
		updatedMetadataJSON, itemsCollected, collectionID)
	
	if err != nil {
		return fmt.Errorf("error updating progress: %w", err)
	}
	
	return nil
}

// CompleteAzureRawCollection marks a collection as completed
func (db *DB) CompleteAzureRawCollection(collectionID string, totalItems int) error {
	// Update final progress
	db.UpdateAzureRawCollectionProgress(collectionID, -1, -1, totalItems, "Collection completed successfully")
	
	_, err := db.conn.Exec(`
		UPDATE azure_collections 
		SET completed_at = $1, status = 'completed', total_items = $2
		WHERE collection_id = $3`,
		time.Now(), totalItems, collectionID)
	
	if err != nil {
		return fmt.Errorf("error completing collection: %w", err)
	}
	
	return nil
}

// FailAzureRawCollection marks a collection as failed
func (db *DB) FailAzureRawCollection(collectionID string, errorMsg string) error {
	_, err := db.conn.Exec(`
		UPDATE azure_collections 
		SET completed_at = $1, status = 'failed', error_message = $2
		WHERE collection_id = $3`,
		time.Now(), errorMsg, collectionID)
	
	if err != nil {
		return fmt.Errorf("error failing collection: %w", err)
	}
	
	return nil
}

// BulkInsertAzureRawPricing inserts raw Azure pricing data efficiently
func (db *DB) BulkInsertAzureRawPricing(collectionID string, region string, items []map[string]interface{}) error {
	if len(items) == 0 {
		return nil
	}
	
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Process items in batches
	batchSize := 100
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		
		batch := items[i:end]
		err = db.insertRawPricingBatch(tx, collectionID, region, batch)
		if err != nil {
			return fmt.Errorf("error inserting batch %d-%d: %w", i, end-1, err)
		}
	}
	
	return tx.Commit()
}

// insertRawPricingBatch inserts a batch of raw pricing items
func (db *DB) insertRawPricingBatch(tx *sql.Tx, collectionID string, region string, items []map[string]interface{}) error {
	for _, item := range items {
		// Extract service info for indexing
		var serviceName, serviceFamily *string
		if sn, ok := item["serviceName"].(string); ok {
			serviceName = &sn
		}
		if sf, ok := item["serviceFamily"].(string); ok {
			serviceFamily = &sf
		}
		
		// Convert item to JSON
		dataJSON, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("error marshaling item: %w", err)
		}
		
		// Insert raw data
		_, err = tx.Exec(`
			INSERT INTO azure_pricing_raw (region, service_name, service_family, data, collection_id, total_items)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			region, serviceName, serviceFamily, dataJSON, collectionID, len(items))
		
		if err != nil {
			return fmt.Errorf("error inserting raw pricing: %w", err)
		}
	}
	
	return nil
}

// GetAzureRawPricing retrieves raw pricing data with pagination
func (db *DB) GetAzureRawPricing(region string, limit int, offset int) ([]AzureRawPricing, error) {
	if limit <= 0 {
		limit = 100
	}
	
	var query string
	var args []interface{}
	
	if region != "" {
		query = `
			SELECT id, region, service_name, service_family, data, collected_at, collection_id, total_items
			FROM azure_pricing_raw 
			WHERE region = $1
			ORDER BY collected_at DESC, id DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{region, limit, offset}
	} else {
		query = `
			SELECT id, region, service_name, service_family, data, collected_at, collection_id, total_items
			FROM azure_pricing_raw 
			ORDER BY collected_at DESC, id DESC
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}
	
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []AzureRawPricing
	for rows.Next() {
		var item AzureRawPricing
		var dataJSON []byte
		
		err := rows.Scan(&item.ID, &item.Region, &item.ServiceName, &item.ServiceFamily, 
			&dataJSON, &item.CollectedAt, &item.CollectionID, &item.TotalItems)
		if err != nil {
			return nil, err
		}
		
		// Parse JSON data
		err = json.Unmarshal(dataJSON, &item.Data)
		if err != nil {
			return nil, err
		}
		
		results = append(results, item)
	}
	
	return results, rows.Err()
}

// GetAzureCollections retrieves collection runs
func (db *DB) GetAzureCollections(limit int) ([]AzureCollection, error) {
	if limit <= 0 {
		limit = 50
	}
	
	rows, err := db.conn.Query(`
		SELECT id, collection_id, region, status, started_at, completed_at, total_items, error_message, metadata
		FROM azure_collections 
		ORDER BY started_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []AzureCollection
	for rows.Next() {
		var collection AzureCollection
		var metadataJSON []byte
		
		err := rows.Scan(&collection.ID, &collection.CollectionID, &collection.Region, 
			&collection.Status, &collection.StartedAt, &collection.CompletedAt, 
			&collection.TotalItems, &collection.ErrorMessage, &metadataJSON)
		if err != nil {
			return nil, err
		}
		
		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			err = json.Unmarshal(metadataJSON, &collection.Metadata)
			if err != nil {
				return nil, err
			}
		}
		
		results = append(results, collection)
	}
	
	return results, rows.Err()
}

// GetAzureRegionsAvailable returns list of regions with data
func (db *DB) GetAzureRegionsAvailable() ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT region 
		FROM azure_pricing_raw 
		ORDER BY region`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var regions []string
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}
	
	return regions, rows.Err()
}

// GetAzureServicesAvailable returns list of services with data
func (db *DB) GetAzureServicesAvailable(region string) ([]string, error) {
	var query string
	var args []interface{}
	
	if region != "" {
		query = `
			SELECT DISTINCT service_name 
			FROM azure_pricing_raw 
			WHERE region = $1 AND service_name IS NOT NULL
			ORDER BY service_name`
		args = []interface{}{region}
	} else {
		query = `
			SELECT DISTINCT service_name 
			FROM azure_pricing_raw 
			WHERE service_name IS NOT NULL
			ORDER BY service_name`
	}
	
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var services []string
	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	
	return services, rows.Err()
}

// QueryAzurePricingByService queries pricing data by service name
func (db *DB) QueryAzurePricingByService(serviceName string, region string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	
	var query string
	var args []interface{}
	
	if region != "" {
		query = `
			SELECT data 
			FROM azure_pricing_raw 
			WHERE service_name = $1 AND region = $2
			ORDER BY collected_at DESC
			LIMIT $3`
		args = []interface{}{serviceName, region, limit}
	} else {
		query = `
			SELECT data 
			FROM azure_pricing_raw 
			WHERE service_name = $1
			ORDER BY collected_at DESC
			LIMIT $2`
		args = []interface{}{serviceName, limit}
	}
	
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []map[string]interface{}
	for rows.Next() {
		var dataJSON []byte
		if err := rows.Scan(&dataJSON); err != nil {
			return nil, err
		}
		
		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, err
		}
		
		results = append(results, data)
	}
	
	return results, rows.Err()
}