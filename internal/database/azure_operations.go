package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Azure pricing operations

// GetOrCreateAzureService gets or creates a service record
func (db *DB) GetOrCreateAzureService(serviceName, serviceFamily string) (int, error) {
	// Try to get existing service
	var serviceID int
	err := db.conn.QueryRow(
		"SELECT id FROM azure_services WHERE service_name = $1",
		serviceName,
	).Scan(&serviceID)
	
	if err == nil {
		return serviceID, nil
	}
	
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error querying service: %w", err)
	}
	
	// Create new service
	err = db.conn.QueryRow(
		`INSERT INTO azure_services (service_name, service_family) 
		 VALUES ($1, $2) RETURNING id`,
		serviceName, serviceFamily,
	).Scan(&serviceID)
	
	if err != nil {
		return 0, fmt.Errorf("error creating service: %w", err)
	}
	
	return serviceID, nil
}

// GetOrCreateAzureRegion gets or creates a region record
func (db *DB) GetOrCreateAzureRegion(armRegionName, displayName string) (int, error) {
	// Try to get existing region
	var regionID int
	err := db.conn.QueryRow(
		"SELECT id FROM azure_regions WHERE arm_region_name = $1",
		armRegionName,
	).Scan(&regionID)
	
	if err == nil {
		return regionID, nil
	}
	
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error querying region: %w", err)
	}
	
	// Create new region
	err = db.conn.QueryRow(
		`INSERT INTO azure_regions (arm_region_name, display_name) 
		 VALUES ($1, $2) RETURNING id`,
		armRegionName, displayName,
	).Scan(&regionID)
	
	if err != nil {
		return 0, fmt.Errorf("error creating region: %w", err)
	}
	
	return regionID, nil
}

// GetOrCreateAzureProduct gets or creates a product record
func (db *DB) GetOrCreateAzureProduct(serviceID int, productName, productID string) (int, error) {
	// Try to get existing product
	var productIDInt int
	err := db.conn.QueryRow(
		"SELECT id FROM azure_products WHERE service_id = $1 AND product_name = $2",
		serviceID, productName,
	).Scan(&productIDInt)
	
	if err == nil {
		return productIDInt, nil
	}
	
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error querying product: %w", err)
	}
	
	// Create new product
	var productIDPtr *string
	if productID != "" {
		productIDPtr = &productID
	}
	
	err = db.conn.QueryRow(
		`INSERT INTO azure_products (service_id, product_name, product_id) 
		 VALUES ($1, $2, $3) RETURNING id`,
		serviceID, productName, productIDPtr,
	).Scan(&productIDInt)
	
	if err != nil {
		return 0, fmt.Errorf("error creating product: %w", err)
	}
	
	return productIDInt, nil
}

// GetOrCreateAzureSKU gets or creates a SKU record
func (db *DB) GetOrCreateAzureSKU(productID int, skuName, skuID, armSKUName string) (int, error) {
	// Try to get existing SKU
	var skuIDInt int
	err := db.conn.QueryRow(
		"SELECT id FROM azure_skus WHERE product_id = $1 AND sku_name = $2",
		productID, skuName,
	).Scan(&skuIDInt)
	
	if err == nil {
		return skuIDInt, nil
	}
	
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error querying SKU: %w", err)
	}
	
	// Create new SKU
	var skuIDPtr, armSKUPtr *string
	if skuID != "" {
		skuIDPtr = &skuID
	}
	if armSKUName != "" {
		armSKUPtr = &armSKUName
	}
	
	err = db.conn.QueryRow(
		`INSERT INTO azure_skus (product_id, sku_name, sku_id, arm_sku_name) 
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		productID, skuName, skuIDPtr, armSKUPtr,
	).Scan(&skuIDInt)
	
	if err != nil {
		return 0, fmt.Errorf("error creating SKU: %w", err)
	}
	
	return skuIDInt, nil
}

// StartAzureCollection starts a new collection run
func (db *DB) StartAzureCollection() (int, error) {
	var version int
	err := db.conn.QueryRow(
		`INSERT INTO azure_collection_runs (version, started_at, status) 
		 VALUES ((SELECT COALESCE(MAX(version), 0) + 1 FROM azure_collection_runs), $1, 'running') 
		 RETURNING version`,
		time.Now(),
	).Scan(&version)
	
	if err != nil {
		return 0, fmt.Errorf("error starting collection: %w", err)
	}
	
	return version, nil
}

// CompleteAzureCollection completes a collection run
func (db *DB) CompleteAzureCollection(version int, totalItems int, regions []string) error {
	regionsArray := fmt.Sprintf("{%s}", fmt.Sprintf(`"%s"`, regions[0]))
	if len(regions) > 1 {
		for _, region := range regions[1:] {
			regionsArray = regionsArray[:len(regionsArray)-1] + fmt.Sprintf(`,"%s"}`, region)
		}
	}
	
	_, err := db.conn.Exec(
		`UPDATE azure_collection_runs 
		 SET completed_at = $1, status = 'completed', total_items = $2, regions_collected = $3 
		 WHERE version = $4`,
		time.Now(), totalItems, regionsArray, version,
	)
	
	if err != nil {
		return fmt.Errorf("error completing collection: %w", err)
	}
	
	return nil
}

// FailAzureCollection marks a collection run as failed
func (db *DB) FailAzureCollection(version int, errorMsg string) error {
	_, err := db.conn.Exec(
		`UPDATE azure_collection_runs 
		 SET completed_at = $1, status = 'failed', error_message = $2 
		 WHERE version = $3`,
		time.Now(), errorMsg, version,
	)
	
	if err != nil {
		return fmt.Errorf("error failing collection: %w", err)
	}
	
	return nil
}

// BulkInsertAzurePricing inserts multiple pricing records efficiently
func (db *DB) BulkInsertAzurePricing(items []AzurePricingInsert) error {
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
		err = db.insertPricingBatch(tx, batch)
		if err != nil {
			return fmt.Errorf("error inserting batch %d-%d: %w", i, end-1, err)
		}
	}
	
	return tx.Commit()
}

// insertPricingBatch inserts a batch of pricing items
func (db *DB) insertPricingBatch(tx *sql.Tx, items []AzurePricingInsert) error {
	// For each item, resolve IDs and insert pricing
	for _, item := range items {
		// Get or create service
		serviceID, err := db.getOrCreateServiceInTx(tx, item.ServiceName, item.ServiceFamily)
		if err != nil {
			return fmt.Errorf("error with service %s: %w", item.ServiceName, err)
		}
		
		// Get or create region
		regionID, err := db.getOrCreateRegionInTx(tx, item.ARMRegionName, item.DisplayName)
		if err != nil {
			return fmt.Errorf("error with region %s: %w", item.ARMRegionName, err)
		}
		
		// Get or create product
		productID, err := db.getOrCreateProductInTx(tx, serviceID, item.ProductName, item.ProductID)
		if err != nil {
			return fmt.Errorf("error with product %s: %w", item.ProductName, err)
		}
		
		// Get or create SKU
		skuID, err := db.getOrCreateSKUInTx(tx, productID, item.SKUName, item.SKUID, item.ARMSKUName)
		if err != nil {
			return fmt.Errorf("error with SKU %s: %w", item.SKUName, err)
		}
		
		// Insert pricing record
		err = db.insertPricingInTx(tx, AzurePricing{
			ServiceID:            serviceID,
			ProductID:            productID,
			SKUID:                skuID,
			RegionID:             regionID,
			MeterID:              item.MeterID,
			MeterName:            item.MeterName,
			RetailPrice:          item.RetailPrice,
			UnitPrice:            item.UnitPrice,
			TierMinimumUnits:     item.TierMinimumUnits,
			CurrencyCode:         item.CurrencyCode,
			UnitOfMeasure:        item.UnitOfMeasure,
			PriceType:            item.PriceType,
			ReservationTerm:      stringPtr(item.ReservationTerm),
			EffectiveStartDate:   item.EffectiveStartDate,
			IsPrimaryMeterRegion: item.IsPrimaryMeterRegion,
			CollectionVersion:    item.CollectionVersion,
		})
		
		if err != nil {
			return fmt.Errorf("error inserting pricing for meter %s: %w", item.MeterID, err)
		}
	}
	
	return nil
}

// Helper transaction methods (simplified versions of the above)
func (db *DB) getOrCreateServiceInTx(tx *sql.Tx, serviceName, serviceFamily string) (int, error) {
	var id int
	err := tx.QueryRow("SELECT id FROM azure_services WHERE service_name = $1", serviceName).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	
	err = tx.QueryRow(
		"INSERT INTO azure_services (service_name, service_family) VALUES ($1, $2) RETURNING id",
		serviceName, serviceFamily,
	).Scan(&id)
	return id, err
}

func (db *DB) getOrCreateRegionInTx(tx *sql.Tx, armRegionName, displayName string) (int, error) {
	var id int
	err := tx.QueryRow("SELECT id FROM azure_regions WHERE arm_region_name = $1", armRegionName).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	
	err = tx.QueryRow(
		"INSERT INTO azure_regions (arm_region_name, display_name) VALUES ($1, $2) RETURNING id",
		armRegionName, displayName,
	).Scan(&id)
	return id, err
}

func (db *DB) getOrCreateProductInTx(tx *sql.Tx, serviceID int, productName, productID string) (int, error) {
	var id int
	err := tx.QueryRow(
		"SELECT id FROM azure_products WHERE service_id = $1 AND product_name = $2",
		serviceID, productName,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	
	var productIDPtr *string
	if productID != "" {
		productIDPtr = &productID
	}
	
	err = tx.QueryRow(
		"INSERT INTO azure_products (service_id, product_name, product_id) VALUES ($1, $2, $3) RETURNING id",
		serviceID, productName, productIDPtr,
	).Scan(&id)
	return id, err
}

func (db *DB) getOrCreateSKUInTx(tx *sql.Tx, productID int, skuName, skuID, armSKUName string) (int, error) {
	var id int
	err := tx.QueryRow(
		"SELECT id FROM azure_skus WHERE product_id = $1 AND sku_name = $2",
		productID, skuName,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	
	var skuIDPtr, armSKUPtr *string
	if skuID != "" {
		skuIDPtr = &skuID
	}
	if armSKUName != "" {
		armSKUPtr = &armSKUName
	}
	
	err = tx.QueryRow(
		"INSERT INTO azure_skus (product_id, sku_name, sku_id, arm_sku_name) VALUES ($1, $2, $3, $4) RETURNING id",
		productID, skuName, skuIDPtr, armSKUPtr,
	).Scan(&id)
	return id, err
}

func (db *DB) insertPricingInTx(tx *sql.Tx, pricing AzurePricing) error {
	_, err := tx.Exec(`
		INSERT INTO azure_pricing (
			service_id, product_id, sku_id, region_id, meter_id, meter_name,
			retail_price, unit_price, tier_minimum_units, currency_code, unit_of_measure,
			price_type, reservation_term, effective_start_date, is_primary_meter_region, collection_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (service_id, product_id, sku_id, region_id, meter_id, effective_start_date) 
		DO UPDATE SET 
			retail_price = EXCLUDED.retail_price,
			unit_price = EXCLUDED.unit_price, 
			collection_version = EXCLUDED.collection_version,
			created_at = CURRENT_TIMESTAMP`,
		pricing.ServiceID, pricing.ProductID, pricing.SKUID, pricing.RegionID,
		pricing.MeterID, pricing.MeterName, pricing.RetailPrice, pricing.UnitPrice,
		pricing.TierMinimumUnits, pricing.CurrencyCode, pricing.UnitOfMeasure,
		pricing.PriceType, pricing.ReservationTerm, pricing.EffectiveStartDate,
		pricing.IsPrimaryMeterRegion, pricing.CollectionVersion,
	)
	
	return err
}

// GetAzureServices retrieves all Azure services
func (db *DB) GetAzureServices() ([]AzureService, error) {
	rows, err := db.conn.Query(`
		SELECT id, service_name, service_family, category_id, created_at, updated_at
		FROM azure_services
		ORDER BY service_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []AzureService
	for rows.Next() {
		var service AzureService
		if err := rows.Scan(&service.ID, &service.ServiceName, &service.ServiceFamily, 
			&service.CategoryID, &service.CreatedAt, &service.UpdatedAt); err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, rows.Err()
}

// GetAzureRegions retrieves all Azure regions
func (db *DB) GetAzureRegions() ([]AzureRegion, error) {
	rows, err := db.conn.Query(`
		SELECT id, arm_region_name, display_name, created_at
		FROM azure_regions
		ORDER BY display_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []AzureRegion
	for rows.Next() {
		var region AzureRegion
		if err := rows.Scan(&region.ID, &region.ARMRegionName, &region.DisplayName, &region.CreatedAt); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}

	return regions, rows.Err()
}

// GetAzurePricingSample retrieves a sample of pricing data
func (db *DB) GetAzurePricingSample(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT 
			s.service_name,
			s.service_family,
			p.product_name,
			sk.sku_name,
			sk.arm_sku_name,
			r.display_name as region,
			pr.meter_name,
			pr.retail_price,
			pr.unit_of_measure,
			pr.effective_start_date
		FROM azure_pricing pr
		JOIN azure_services s ON pr.service_id = s.id
		JOIN azure_products p ON pr.product_id = p.id
		JOIN azure_skus sk ON pr.sku_id = sk.id
		JOIN azure_regions r ON pr.region_id = r.id
		ORDER BY pr.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var serviceName, serviceFamily, productName, skuName, armSKUName, region, meterName, unitOfMeasure string
		var retailPrice float64
		var effectiveDate time.Time

		err := rows.Scan(&serviceName, &serviceFamily, &productName, &skuName, &armSKUName, 
			&region, &meterName, &retailPrice, &unitOfMeasure, &effectiveDate)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"serviceName":      serviceName,
			"serviceFamily":    serviceFamily,
			"productName":      productName,
			"skuName":          skuName,
			"armSkuName":       armSKUName,
			"region":           region,
			"meterName":        meterName,
			"retailPrice":      retailPrice,
			"unitOfMeasure":    unitOfMeasure,
			"effectiveDate":    effectiveDate.Format("2006-01-02"),
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// GetAzureServicePricing retrieves pricing for a specific service
func (db *DB) GetAzureServicePricing(serviceName string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.conn.Query(`
		SELECT 
			s.service_name,
			p.product_name,
			sk.sku_name,
			r.display_name as region,
			pr.meter_name,
			pr.retail_price,
			pr.unit_of_measure
		FROM azure_pricing pr
		JOIN azure_services s ON pr.service_id = s.id
		JOIN azure_products p ON pr.product_id = p.id
		JOIN azure_skus sk ON pr.sku_id = sk.id
		JOIN azure_regions r ON pr.region_id = r.id
		WHERE s.service_name = $1
		ORDER BY pr.retail_price ASC
		LIMIT $2
	`, serviceName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var serviceName, productName, skuName, region, meterName, unitOfMeasure string
		var retailPrice float64

		err := rows.Scan(&serviceName, &productName, &skuName, &region, &meterName, &retailPrice, &unitOfMeasure)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"serviceName":   serviceName,
			"productName":   productName,
			"skuName":       skuName,
			"region":        region,
			"meterName":     meterName,
			"retailPrice":   retailPrice,
			"unitOfMeasure": unitOfMeasure,
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// Helper function
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}