package normalizer

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/raulc0399/cpc/internal/database"
)

// ServiceMappingRepositoryImpl implements ServiceMappingRepository using database
type ServiceMappingRepositoryImpl struct {
	db     *database.DB
	cache  map[string]*database.ServiceMapping
	logger Logger
}

// NewServiceMappingRepository creates a new service mapping repository
func NewServiceMappingRepository(db *database.DB, logger Logger) *ServiceMappingRepositoryImpl {
	return &ServiceMappingRepositoryImpl{
		db:     db,
		cache:  make(map[string]*database.ServiceMapping),
		logger: logger,
	}
}

// GetServiceMappingByProvider retrieves service mapping for a specific provider and service
func (r *ServiceMappingRepositoryImpl) GetServiceMappingByProvider(ctx context.Context, provider, serviceName string) (*database.ServiceMapping, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", provider, serviceName)
	if mapping, exists := r.cache[cacheKey]; exists {
		r.logger.Debug("Service mapping cache hit",
			Field{"provider", provider},
			Field{"service", serviceName},
		)
		return mapping, nil
	}

	// Query database with context
	mapping, err := r.getServiceMappingFromDB(ctx, provider, serviceName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if mapping != nil {
		r.cache[cacheKey] = mapping
	}

	return mapping, nil
}

// getServiceMappingFromDB queries the database for service mapping
func (r *ServiceMappingRepositoryImpl) getServiceMappingFromDB(ctx context.Context, provider, serviceName string) (*database.ServiceMapping, error) {
	query := `
		SELECT id, provider, provider_service_name, provider_service_code, 
		       normalized_service_type, service_category, service_family
		FROM service_mappings 
		WHERE provider = $1 AND (provider_service_name = $2 OR provider_service_code = $2)
		LIMIT 1`

	var mapping database.ServiceMapping
	err := r.db.GetConn().QueryRowContext(ctx, query, provider, serviceName).Scan(
		&mapping.ID,
		&mapping.Provider,
		&mapping.ProviderServiceName,
		&mapping.ProviderServiceCode,
		&mapping.NormalizedServiceType,
		&mapping.ServiceCategory,
		&mapping.ServiceFamily,
	)

	if err == sql.ErrNoRows {
		r.logger.Debug("Service mapping not found",
			Field{"provider", provider},
			Field{"service", serviceName},
		)
		return nil, nil // Not found is not an error
	}
	if err != nil {
		r.logger.Error("Failed to query service mapping",
			Field{"provider", provider},
			Field{"service", serviceName},
			Field{"error", err},
		)
		return nil, fmt.Errorf("failed to get service mapping: %w", err)
	}

	return &mapping, nil
}

// GetAllServiceMappings retrieves all service mappings
func (r *ServiceMappingRepositoryImpl) GetAllServiceMappings(ctx context.Context) ([]database.ServiceMapping, error) {
	// This uses the existing database method but adds context support
	mappings, err := r.db.GetServiceMappings()
	if err != nil {
		r.logger.Error("Failed to get all service mappings", Field{"error", err})
		return nil, err
	}

	// Populate cache
	for i := range mappings {
		cacheKey := fmt.Sprintf("%s:%s", mappings[i].Provider, mappings[i].ProviderServiceName)
		r.cache[cacheKey] = &mappings[i]
	}

	return mappings, nil
}

// RegionMappingRepositoryImpl implements RegionMappingRepository using database
type RegionMappingRepositoryImpl struct {
	db     *database.DB
	cache  map[string]*database.NormalizedRegion
	logger Logger
}

// NewRegionMappingRepository creates a new region mapping repository
func NewRegionMappingRepository(db *database.DB, logger Logger) *RegionMappingRepositoryImpl {
	return &RegionMappingRepositoryImpl{
		db:     db,
		cache:  make(map[string]*database.NormalizedRegion),
		logger: logger,
	}
}

// GetNormalizedRegionByProvider finds normalized region by provider-specific region
func (r *RegionMappingRepositoryImpl) GetNormalizedRegionByProvider(ctx context.Context, provider, providerRegion string) (*database.NormalizedRegion, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", provider, providerRegion)
	if region, exists := r.cache[cacheKey]; exists {
		r.logger.Debug("Region mapping cache hit",
			Field{"provider", provider},
			Field{"region", providerRegion},
		)
		return region, nil
	}

	// Query database with context
	region, err := r.getNormalizedRegionFromDB(ctx, provider, providerRegion)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if region != nil {
		r.cache[cacheKey] = region
	}

	return region, nil
}

// getNormalizedRegionFromDB queries the database for normalized region
func (r *RegionMappingRepositoryImpl) getNormalizedRegionFromDB(ctx context.Context, provider, providerRegion string) (*database.NormalizedRegion, error) {
	var query string
	if provider == database.ProviderAWS {
		query = `
			SELECT id, normalized_code, aws_region, azure_region, 
			       display_name, country, continent
			FROM normalized_regions 
			WHERE aws_region = $1
			LIMIT 1`
	} else if provider == database.ProviderAzure {
		query = `
			SELECT id, normalized_code, aws_region, azure_region, 
			       display_name, country, continent
			FROM normalized_regions 
			WHERE azure_region = $1
			LIMIT 1`
	} else {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	var region database.NormalizedRegion
	err := r.db.GetConn().QueryRowContext(ctx, query, providerRegion).Scan(
		&region.ID,
		&region.NormalizedCode,
		&region.AWSRegion,
		&region.AzureRegion,
		&region.DisplayName,
		&region.Country,
		&region.Continent,
	)

	if err == sql.ErrNoRows {
		r.logger.Debug("Normalized region not found",
			Field{"provider", provider},
			Field{"region", providerRegion},
		)
		return nil, nil // Not found is not an error
	}
	if err != nil {
		r.logger.Error("Failed to query normalized region",
			Field{"provider", provider},
			Field{"region", providerRegion},
			Field{"error", err},
		)
		return nil, fmt.Errorf("failed to get normalized region: %w", err)
	}

	return &region, nil
}

// GetAllNormalizedRegions retrieves all normalized regions
func (r *RegionMappingRepositoryImpl) GetAllNormalizedRegions(ctx context.Context) ([]database.NormalizedRegion, error) {
	// This uses the existing database method but adds context support
	regions, err := r.db.GetNormalizedRegions()
	if err != nil {
		r.logger.Error("Failed to get all normalized regions", Field{"error", err})
		return nil, err
	}

	// Populate cache
	for i := range regions {
		if regions[i].AWSRegion != nil {
			cacheKey := fmt.Sprintf("%s:%s", database.ProviderAWS, *regions[i].AWSRegion)
			r.cache[cacheKey] = &regions[i]
		}
		if regions[i].AzureRegion != nil {
			cacheKey := fmt.Sprintf("%s:%s", database.ProviderAzure, *regions[i].AzureRegion)
			r.cache[cacheKey] = &regions[i]
		}
	}

	return regions, nil
}

// NormalizedPricingRepository handles normalized pricing data operations
type NormalizedPricingRepository interface {
	Insert(ctx context.Context, pricing *database.NormalizedPricing) error
	BulkInsert(ctx context.Context, pricings []database.NormalizedPricing) error
	Query(ctx context.Context, filter database.PricingFilter) ([]database.NormalizedPricing, error)
}

// NormalizedPricingRepositoryImpl implements NormalizedPricingRepository
type NormalizedPricingRepositoryImpl struct {
	db     *database.DB
	logger Logger
}

// NewNormalizedPricingRepository creates a new normalized pricing repository
func NewNormalizedPricingRepository(db *database.DB, logger Logger) *NormalizedPricingRepositoryImpl {
	return &NormalizedPricingRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// Insert inserts a single normalized pricing record
func (r *NormalizedPricingRepositoryImpl) Insert(ctx context.Context, pricing *database.NormalizedPricing) error {
	startTime := time.Now()
	err := r.db.InsertNormalizedPricing(pricing)
	
	duration := time.Since(startTime)
	if err != nil {
		r.logger.Error("Failed to insert normalized pricing",
			Field{"provider", pricing.Provider},
			Field{"resource", pricing.ResourceName},
			Field{"duration", duration},
			Field{"error", err},
		)
		return err
	}

	r.logger.Debug("Inserted normalized pricing",
		Field{"provider", pricing.Provider},
		Field{"resource", pricing.ResourceName},
		Field{"duration", duration},
	)
	return nil
}

// BulkInsert inserts multiple normalized pricing records
func (r *NormalizedPricingRepositoryImpl) BulkInsert(ctx context.Context, pricings []database.NormalizedPricing) error {
	if len(pricings) == 0 {
		return nil
	}

	startTime := time.Now()
	err := r.db.BulkInsertNormalizedPricing(pricings)
	
	duration := time.Since(startTime)
	if err != nil {
		r.logger.Error("Failed to bulk insert normalized pricing",
			Field{"count", len(pricings)},
			Field{"duration", duration},
			Field{"error", err},
		)
		return err
	}

	r.logger.Info("Bulk inserted normalized pricing",
		Field{"count", len(pricings)},
		Field{"duration", duration},
		Field{"rate", float64(len(pricings)) / duration.Seconds()},
	)
	return nil
}

// Query queries normalized pricing with filters
func (r *NormalizedPricingRepositoryImpl) Query(ctx context.Context, filter database.PricingFilter) ([]database.NormalizedPricing, error) {
	startTime := time.Now()
	results, err := r.db.QueryNormalizedPricing(filter)
	
	duration := time.Since(startTime)
	if err != nil {
		r.logger.Error("Failed to query normalized pricing",
			Field{"duration", duration},
			Field{"error", err},
		)
		return nil, err
	}

	r.logger.Debug("Queried normalized pricing",
		Field{"resultCount", len(results)},
		Field{"duration", duration},
	)
	return results, nil
}