# Database Architecture

Cloud Price Compare uses PostgreSQL with JSONB for flexible raw data storage and normalized tables for cross-provider comparisons.

## Core Design Principles

### 1. Raw Data Preservation
- **Never lose original API responses** - Complete provider data stored as JSONB
- **Immutable raw storage** - Raw data tables are append-only
- **Flexible querying** - JSONB enables complex queries without schema changes

### 2. Normalized Layer for Comparisons
- **Cross-provider standardization** - Unified pricing format
- **Service categorization** - 13 standard categories across all providers
- **Performance optimization** - Indexed normalized data for fast queries

## Database Schema

### Raw Data Tables

#### `aws_pricing_raw`
```sql
CREATE TABLE aws_pricing_raw (
    id SERIAL PRIMARY KEY,
    service_code TEXT NOT NULL,
    region TEXT,
    data JSONB NOT NULL,  -- Complete AWS API response
    collection_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_aws_pricing_service ON aws_pricing_raw (service_code);
CREATE INDEX idx_aws_pricing_region ON aws_pricing_raw (region);
CREATE GIN INDEX idx_aws_pricing_data ON aws_pricing_raw USING gin (data);
```

#### `azure_pricing_raw`
```sql
CREATE TABLE azure_pricing_raw (
    id SERIAL PRIMARY KEY,
    service_name TEXT,
    sku_name TEXT,
    product_name TEXT,
    arm_region_name TEXT,
    data JSONB NOT NULL,  -- Complete Azure API response
    collection_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_azure_pricing_service ON azure_pricing_raw (service_name);
CREATE INDEX idx_azure_pricing_region ON azure_pricing_raw (arm_region_name);
CREATE GIN INDEX idx_azure_pricing_data ON azure_pricing_raw USING gin (data);
```

### Normalized Data Tables

#### `normalized_pricing`
```sql
CREATE TABLE normalized_pricing (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,                    -- 'aws' or 'azure'
    service_category TEXT NOT NULL,            -- Standardized category
    service_name TEXT NOT NULL,
    resource_name TEXT,
    instance_type TEXT,
    region TEXT,
    pricing_model TEXT,                        -- 'on-demand', 'reserved', etc.
    price_per_unit DECIMAL(20,10),
    unit_of_measure TEXT,
    currency TEXT DEFAULT 'USD',
    effective_date TIMESTAMP WITH TIME ZONE,
    raw_data_id INTEGER,                       -- Reference to raw data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Performance indexes for common queries
CREATE INDEX idx_normalized_provider ON normalized_pricing (provider);
CREATE INDEX idx_normalized_category ON normalized_pricing (service_category);
CREATE INDEX idx_normalized_region ON normalized_pricing (region);
CREATE INDEX idx_normalized_price ON normalized_pricing (price_per_unit);
CREATE INDEX idx_normalized_composite ON normalized_pricing (provider, service_category, region);
```

#### `service_mappings`
```sql
CREATE TABLE service_mappings (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    service_name TEXT NOT NULL,
    category TEXT NOT NULL,
    service_type TEXT,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, service_name)
);
```

### Collection Tracking Tables

#### `aws_collections`
```sql
CREATE TABLE aws_collections (
    id TEXT PRIMARY KEY,
    service_code TEXT NOT NULL,
    region TEXT,
    status TEXT NOT NULL,          -- 'pending', 'running', 'completed', 'failed'
    total_items INTEGER DEFAULT 0,
    processed_items INTEGER DEFAULT 0,
    progress DECIMAL(5,2) DEFAULT 0.0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    error_message TEXT
);
```

#### `azure_collections`
```sql
CREATE TABLE azure_collections (
    id TEXT PRIMARY KEY,
    region TEXT NOT NULL,
    status TEXT NOT NULL,          -- 'pending', 'running', 'completed', 'failed'
    total_items INTEGER DEFAULT 0,
    processed_items INTEGER DEFAULT 0,
    progress DECIMAL(5,2) DEFAULT 0.0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    next_page_link TEXT,
    error_message TEXT
);
```

#### `etl_jobs`
```sql
CREATE TABLE etl_jobs (
    id TEXT PRIMARY KEY,
    job_type TEXT NOT NULL,        -- 'normalize_aws', 'normalize_azure', 'normalize_all'
    status TEXT NOT NULL,          -- 'pending', 'running', 'completed', 'failed'
    config JSONB,                  -- Job configuration
    progress JSONB,                -- Detailed progress tracking
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT
);
```

## Data Flow

### 1. Collection Phase
```
AWS/Azure APIs → Raw Data Tables (JSONB preservation)
                ↓
           Collection Tracking Tables (Progress monitoring)
```

### 2. ETL Phase
```
Raw Data Tables → ETL Jobs → Normalized Pricing Table
                     ↓
              Service Mappings (Category standardization)
```

### 3. Query Phase
```
GraphQL API → Normalized Data (Fast comparisons)
            → Raw Data (Complete original information)
```

## Performance Considerations

### Indexing Strategy
- **GIN indexes** on JSONB columns for flexible queries
- **Composite indexes** for common query patterns
- **Partial indexes** for frequently filtered data

### Query Optimization
```sql
-- Efficient cross-provider comparison
SELECT provider, service_category, AVG(price_per_unit) as avg_price
FROM normalized_pricing 
WHERE service_category = 'Compute & Web' 
  AND region IN ('us-east-1', 'eastus')
GROUP BY provider, service_category;

-- Raw data drill-down with JSONB queries
SELECT data->>'instanceType', data->>'vcpu', data->>'memory'
FROM aws_pricing_raw 
WHERE service_code = 'AmazonEC2' 
  AND data->>'location' = 'US East (N. Virginia)';
```

### Scaling Recommendations
- **Connection pooling** for high-concurrency workloads
- **Read replicas** for analytics workloads
- **Partitioning** by provider/date for very large datasets
- **VACUUM** and **ANALYZE** scheduling for optimal performance

## Data Integrity

### Raw Data Protection
- Raw data tables are **append-only**
- Original API responses preserved completely
- No data transformation during collection

### Referential Integrity
- Normalized records reference raw data via `raw_data_id`
- Service mappings ensure consistent categorization
- ETL jobs track transformation lineage

### Backup Strategy
- **Point-in-time recovery** for operational data
- **Archive storage** for historical raw data
- **Schema versioning** for migration safety

## Monitoring Queries

### Collection Status
```sql
-- Azure collection progress
SELECT region, status, progress, 
       processed_items || '/' || total_items as items
FROM azure_collections 
ORDER BY started_at DESC;

-- AWS collection status
SELECT service_code, status, progress,
       duration_seconds, error_message
FROM aws_collections 
ORDER BY started_at DESC;
```

### Data Quality
```sql
-- Normalized data coverage
SELECT provider, service_category, COUNT(*) as records
FROM normalized_pricing 
GROUP BY provider, service_category 
ORDER BY provider, records DESC;

-- ETL job monitoring
SELECT id, job_type, status, 
       progress->>'processedRecords' as processed,
       progress->>'rate' as rate
FROM etl_jobs 
ORDER BY started_at DESC;
```

## Common Patterns

### Adding New Providers
1. Create raw data table following naming convention
2. Add collection tracking table
3. Implement normalizer with service mappings
4. Add GraphQL resolvers and mutations

### Querying Patterns
```sql
-- Price comparison across providers
SELECT provider, MIN(price_per_unit) as min_price, 
       MAX(price_per_unit) as max_price
FROM normalized_pricing 
WHERE service_category = 'Storage' 
GROUP BY provider;

-- Service availability by region
SELECT region, COUNT(DISTINCT service_name) as services
FROM normalized_pricing 
WHERE provider = 'aws' 
GROUP BY region 
ORDER BY services DESC;
```

This architecture provides the foundation for CPC's comprehensive cloud pricing analysis while maintaining flexibility for future enhancements.