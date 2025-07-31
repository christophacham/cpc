package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// GetServiceMappings retrieves all service mappings
func (db *DB) GetServiceMappings() ([]ServiceMapping, error) {
	query := `
		SELECT id, provider, provider_service_name, provider_service_code, 
		       normalized_service_type, service_category, service_family
		FROM service_mappings 
		ORDER BY provider, service_category, normalized_service_type`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query service mappings: %w", err)
	}
	defer rows.Close()

	var mappings []ServiceMapping
	for rows.Next() {
		var mapping ServiceMapping
		err := rows.Scan(
			&mapping.ID,
			&mapping.Provider,
			&mapping.ProviderServiceName,
			&mapping.ProviderServiceCode,
			&mapping.NormalizedServiceType,
			&mapping.ServiceCategory,
			&mapping.ServiceFamily,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// GetServiceMappingByProvider retrieves service mapping for a specific provider and service
func (db *DB) GetServiceMappingByProvider(provider, serviceName string) (*ServiceMapping, error) {
	query := `
		SELECT id, provider, provider_service_name, provider_service_code, 
		       normalized_service_type, service_category, service_family
		FROM service_mappings 
		WHERE provider = $1 AND (provider_service_name = $2 OR provider_service_code = $2)
		LIMIT 1`
	
	var mapping ServiceMapping
	err := db.conn.QueryRow(query, provider, serviceName).Scan(
		&mapping.ID,
		&mapping.Provider,
		&mapping.ProviderServiceName,
		&mapping.ProviderServiceCode,
		&mapping.NormalizedServiceType,
		&mapping.ServiceCategory,
		&mapping.ServiceFamily,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get service mapping: %w", err)
	}

	return &mapping, nil
}

// GetNormalizedRegions retrieves all normalized regions
func (db *DB) GetNormalizedRegions() ([]NormalizedRegion, error) {
	query := `
		SELECT id, normalized_code, aws_region, azure_region, 
		       display_name, country, continent
		FROM normalized_regions 
		ORDER BY normalized_code`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query normalized regions: %w", err)
	}
	defer rows.Close()

	var regions []NormalizedRegion
	for rows.Next() {
		var region NormalizedRegion
		err := rows.Scan(
			&region.ID,
			&region.NormalizedCode,
			&region.AWSRegion,
			&region.AzureRegion,
			&region.DisplayName,
			&region.Country,
			&region.Continent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan normalized region: %w", err)
		}
		regions = append(regions, region)
	}

	return regions, nil
}

// GetNormalizedRegionByProvider finds normalized region by provider-specific region
func (db *DB) GetNormalizedRegionByProvider(provider, providerRegion string) (*NormalizedRegion, error) {
	var query string
	if provider == ProviderAWS {
		query = `
			SELECT id, normalized_code, aws_region, azure_region, 
			       display_name, country, continent
			FROM normalized_regions 
			WHERE aws_region = $1
			LIMIT 1`
	} else if provider == ProviderAzure {
		query = `
			SELECT id, normalized_code, aws_region, azure_region, 
			       display_name, country, continent
			FROM normalized_regions 
			WHERE azure_region = $1
			LIMIT 1`
	} else {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	var region NormalizedRegion
	err := db.conn.QueryRow(query, providerRegion).Scan(
		&region.ID,
		&region.NormalizedCode,
		&region.AWSRegion,
		&region.AzureRegion,
		&region.DisplayName,
		&region.Country,
		&region.Continent,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get normalized region: %w", err)
	}

	return &region, nil
}

// InsertNormalizedPricing inserts a single normalized pricing record
func (db *DB) InsertNormalizedPricing(pricing *NormalizedPricing) error {
	// Serialize JSONB fields
	resourceSpecsJSON, err := json.Marshal(pricing.ResourceSpecs)
	if err != nil {
		return fmt.Errorf("failed to marshal resource specs: %w", err)
	}
	
	pricingDetailsJSON, err := json.Marshal(pricing.PricingDetails)
	if err != nil {
		return fmt.Errorf("failed to marshal pricing details: %w", err)
	}

	query := `
		INSERT INTO normalized_pricing (
			provider, provider_service_code, provider_sku, service_mapping_id,
			service_category, service_family, service_type, region_id,
			normalized_region, provider_region, resource_name, resource_description,
			resource_specs, price_per_unit, unit, currency, pricing_model,
			pricing_details, effective_date, expiration_date, minimum_commitment,
			aws_raw_id, azure_raw_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		) RETURNING id, created_at, updated_at`

	err = db.conn.QueryRow(
		query,
		pricing.Provider,
		pricing.ProviderServiceCode,
		pricing.ProviderSKU,
		pricing.ServiceMappingID,
		pricing.ServiceCategory,
		pricing.ServiceFamily,
		pricing.ServiceType,
		pricing.RegionID,
		pricing.NormalizedRegion,
		pricing.ProviderRegion,
		pricing.ResourceName,
		pricing.ResourceDescription,
		string(resourceSpecsJSON),
		pricing.PricePerUnit,
		pricing.Unit,
		pricing.Currency,
		pricing.PricingModel,
		string(pricingDetailsJSON),
		pricing.EffectiveDate,
		pricing.ExpirationDate,
		pricing.MinimumCommitment,
		pricing.AWSRawID,
		pricing.AzureRawID,
	).Scan(&pricing.ID, &pricing.CreatedAt, &pricing.UpdatedAt)

	return err
}

// BulkInsertNormalizedPricing inserts multiple normalized pricing records
func (db *DB) BulkInsertNormalizedPricing(pricings []NormalizedPricing) error {
	if len(pricings) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statement
	query := `
		INSERT INTO normalized_pricing (
			provider, provider_service_code, provider_sku, service_mapping_id,
			service_category, service_family, service_type, region_id,
			normalized_region, provider_region, resource_name, resource_description,
			resource_specs, price_per_unit, unit, currency, pricing_model,
			pricing_details, effective_date, expiration_date, minimum_commitment,
			aws_raw_id, azure_raw_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each record
	for _, pricing := range pricings {
		resourceSpecsJSON, err := json.Marshal(pricing.ResourceSpecs)
		if err != nil {
			return fmt.Errorf("failed to marshal resource specs: %w", err)
		}
		
		pricingDetailsJSON, err := json.Marshal(pricing.PricingDetails)
		if err != nil {
			return fmt.Errorf("failed to marshal pricing details: %w", err)
		}

		_, err = stmt.Exec(
			pricing.Provider,
			pricing.ProviderServiceCode,
			pricing.ProviderSKU,
			pricing.ServiceMappingID,
			pricing.ServiceCategory,
			pricing.ServiceFamily,
			pricing.ServiceType,
			pricing.RegionID,
			pricing.NormalizedRegion,
			pricing.ProviderRegion,
			pricing.ResourceName,
			pricing.ResourceDescription,
			string(resourceSpecsJSON),
			pricing.PricePerUnit,
			pricing.Unit,
			pricing.Currency,
			pricing.PricingModel,
			string(pricingDetailsJSON),
			pricing.EffectiveDate,
			pricing.ExpirationDate,
			pricing.MinimumCommitment,
			pricing.AWSRawID,
			pricing.AzureRawID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert pricing record: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Bulk inserted %d normalized pricing records", len(pricings))
	return nil
}

// QueryNormalizedPricing queries normalized pricing with filters
func (db *DB) QueryNormalizedPricing(filter PricingFilter) ([]NormalizedPricing, error) {
	query := `
		SELECT id, provider, provider_service_code, provider_sku, service_mapping_id,
		       service_category, service_family, service_type, region_id,
		       normalized_region, provider_region, resource_name, resource_description,
		       resource_specs, price_per_unit, unit, currency, pricing_model,
		       pricing_details, effective_date, expiration_date, minimum_commitment,
		       aws_raw_id, azure_raw_id, created_at, updated_at
		FROM normalized_pricing WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	// Build WHERE clauses dynamically
	if filter.Provider != nil {
		argCount++
		query += fmt.Sprintf(" AND provider = $%d", argCount)
		args = append(args, *filter.Provider)
	}

	if filter.ServiceCategory != nil {
		argCount++
		query += fmt.Sprintf(" AND service_category = $%d", argCount)
		args = append(args, *filter.ServiceCategory)
	}

	if filter.ServiceFamily != nil {
		argCount++
		query += fmt.Sprintf(" AND service_family = $%d", argCount)
		args = append(args, *filter.ServiceFamily)
	}

	if filter.ServiceType != nil {
		argCount++
		query += fmt.Sprintf(" AND service_type = $%d", argCount)
		args = append(args, *filter.ServiceType)
	}

	if filter.NormalizedRegion != nil {
		argCount++
		query += fmt.Sprintf(" AND normalized_region = $%d", argCount)
		args = append(args, *filter.NormalizedRegion)
	}

	if filter.PricingModel != nil {
		argCount++
		query += fmt.Sprintf(" AND pricing_model = $%d", argCount)
		args = append(args, *filter.PricingModel)
	}

	if filter.Currency != nil {
		argCount++
		query += fmt.Sprintf(" AND currency = $%d", argCount)
		args = append(args, *filter.Currency)
	}

	if filter.MaxPricePerUnit != nil {
		argCount++
		query += fmt.Sprintf(" AND price_per_unit <= $%d", argCount)
		args = append(args, *filter.MaxPricePerUnit)
	}

	if filter.MinPricePerUnit != nil {
		argCount++
		query += fmt.Sprintf(" AND price_per_unit >= $%d", argCount)
		args = append(args, *filter.MinPricePerUnit)
	}

	// Resource specs filtering (example for vCPU)
	if filter.ResourceSpecs != nil && filter.ResourceSpecs.VCPU != nil {
		argCount++
		query += fmt.Sprintf(" AND resource_specs->>'vcpu' = $%d", argCount)
		args = append(args, fmt.Sprintf("%d", *filter.ResourceSpecs.VCPU))
	}

	// Add ORDER BY
	orderBy := "price_per_unit"
	if filter.OrderBy != nil {
		orderBy = *filter.OrderBy
	}
	
	orderDirection := "ASC"
	if filter.OrderDirection != nil {
		orderDirection = strings.ToUpper(*filter.OrderDirection)
	}
	
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDirection)

	// Add LIMIT and OFFSET
	if filter.Limit != nil {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, *filter.Limit)
		
		if filter.Offset != nil {
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, *filter.Offset)
		}
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query normalized pricing: %w", err)
	}
	defer rows.Close()

	var pricings []NormalizedPricing
	for rows.Next() {
		var pricing NormalizedPricing
		var resourceSpecsJSON, pricingDetailsJSON string

		err := rows.Scan(
			&pricing.ID,
			&pricing.Provider,
			&pricing.ProviderServiceCode,
			&pricing.ProviderSKU,
			&pricing.ServiceMappingID,
			&pricing.ServiceCategory,
			&pricing.ServiceFamily,
			&pricing.ServiceType,
			&pricing.RegionID,
			&pricing.NormalizedRegion,
			&pricing.ProviderRegion,
			&pricing.ResourceName,
			&pricing.ResourceDescription,
			&resourceSpecsJSON,
			&pricing.PricePerUnit,
			&pricing.Unit,
			&pricing.Currency,
			&pricing.PricingModel,
			&pricingDetailsJSON,
			&pricing.EffectiveDate,
			&pricing.ExpirationDate,
			&pricing.MinimumCommitment,
			&pricing.AWSRawID,
			&pricing.AzureRawID,
			&pricing.CreatedAt,
			&pricing.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing record: %w", err)
		}

		// Parse JSONB fields
		if err := json.Unmarshal([]byte(resourceSpecsJSON), &pricing.ResourceSpecs); err != nil {
			log.Printf("Warning: failed to unmarshal resource specs for ID %d: %v", pricing.ID, err)
		}

		if err := json.Unmarshal([]byte(pricingDetailsJSON), &pricing.PricingDetails); err != nil {
			log.Printf("Warning: failed to unmarshal pricing details for ID %d: %v", pricing.ID, err)
		}

		pricings = append(pricings, pricing)
	}

	return pricings, nil
}

// ComparePricing compares equivalent services between providers
func (db *DB) ComparePricing(serviceType, normalizedRegion, pricingModel string) ([]PricingComparison, error) {
	query := `
		SELECT 
			p1.id, p1.provider, p1.resource_name, p1.resource_specs, p1.price_per_unit, p1.unit,
			p2.id, p2.provider, p2.resource_name, p2.resource_specs, p2.price_per_unit, p2.unit
		FROM normalized_pricing p1
		FULL OUTER JOIN normalized_pricing p2 ON (
			p1.service_type = p2.service_type 
			AND p1.normalized_region = p2.normalized_region
			AND p1.pricing_model = p2.pricing_model
			AND p1.provider != p2.provider
			AND p1.resource_specs = p2.resource_specs
		)
		WHERE 
			COALESCE(p1.service_type, p2.service_type) = $1
			AND COALESCE(p1.normalized_region, p2.normalized_region) = $2
			AND COALESCE(p1.pricing_model, p2.pricing_model) = $3
			AND (p1.provider = 'aws' OR p1.provider IS NULL)
		ORDER BY COALESCE(p1.price_per_unit, p2.price_per_unit)`

	rows, err := db.conn.Query(query, serviceType, normalizedRegion, pricingModel)
	if err != nil {
		return nil, fmt.Errorf("failed to compare pricing: %w", err)
	}
	defer rows.Close()

	var comparisons []PricingComparison
	for rows.Next() {
		var comp PricingComparison
		var aws, azure NormalizedPricing
		var awsResourceSpecsJSON, azureResourceSpecsJSON sql.NullString
		var awsID, azureID sql.NullInt64
		var awsProvider, azureProvider, awsResourceName, azureResourceName sql.NullString
		var awsPrice, azurePrice sql.NullFloat64
		var awsUnit, azureUnit sql.NullString

		err := rows.Scan(
			&awsID, &awsProvider, &awsResourceName, &awsResourceSpecsJSON, &awsPrice, &awsUnit,
			&azureID, &azureProvider, &azureResourceName, &azureResourceSpecsJSON, &azurePrice, &azureUnit,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comparison: %w", err)
		}

		comp.ServiceType = serviceType
		comp.NormalizedRegion = normalizedRegion
		comp.PricingModel = pricingModel

		// Parse AWS data if present
		if awsID.Valid {
			aws.ID = int(awsID.Int64)
			aws.Provider = awsProvider.String
			aws.ResourceName = awsResourceName.String
			aws.PricePerUnit = awsPrice.Float64
			aws.Unit = awsUnit.String
			
			if awsResourceSpecsJSON.Valid {
				json.Unmarshal([]byte(awsResourceSpecsJSON.String), &aws.ResourceSpecs)
				comp.ResourceSpecs = aws.ResourceSpecs
			}
			comp.AWS = &aws
		}

		// Parse Azure data if present
		if azureID.Valid {
			azure.ID = int(azureID.Int64)
			azure.Provider = azureProvider.String
			azure.ResourceName = azureResourceName.String
			azure.PricePerUnit = azurePrice.Float64
			azure.Unit = azureUnit.String
			
			if azureResourceSpecsJSON.Valid {
				json.Unmarshal([]byte(azureResourceSpecsJSON.String), &azure.ResourceSpecs)
				if comp.ResourceSpecs.VCPU == nil {
					comp.ResourceSpecs = azure.ResourceSpecs
				}
			}
			comp.Azure = &azure
		}

		// Calculate price difference and savings
		if comp.AWS != nil && comp.Azure != nil {
			priceDiff := comp.AWS.PricePerUnit - comp.Azure.PricePerUnit
			comp.PriceDifference = &priceDiff
			
			if comp.AWS.PricePerUnit < comp.Azure.PricePerUnit {
				cheaperProvider := ProviderAWS
				comp.CheaperProvider = &cheaperProvider
				savingsPercent := (comp.Azure.PricePerUnit - comp.AWS.PricePerUnit) / comp.Azure.PricePerUnit * 100
				comp.SavingsPercent = &savingsPercent
			} else if comp.Azure.PricePerUnit < comp.AWS.PricePerUnit {
				cheaperProvider := ProviderAzure
				comp.CheaperProvider = &cheaperProvider
				savingsPercent := (comp.AWS.PricePerUnit - comp.Azure.PricePerUnit) / comp.AWS.PricePerUnit * 100
				comp.SavingsPercent = &savingsPercent
			}
		}

		comparisons = append(comparisons, comp)
	}

	return comparisons, nil
}