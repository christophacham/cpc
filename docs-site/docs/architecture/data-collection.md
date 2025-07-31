# Data Collection Pipeline

CPC collects pricing data from cloud provider APIs and stores it in a normalized database structure.

## Azure Data Collection

### Data Source
- **API:** Azure Retail Pricing API
- **URL:** `https://prices.azure.com/api/retail/prices`
- **Authentication:** None required (public API)
- **Format:** JSON with OData query support

### Collection Tools

#### 1. Azure Explorer (`cmd/azure-explorer/main.go`)
- **Purpose:** Test API and explore data structure
- **Scope:** One service per category
- **Usage:** `go run cmd/azure-explorer/main.go`

#### 2. Azure Collector (`cmd/azure-collector/main.go`)  
- **Purpose:** Collect from multiple regions
- **Scope:** Limited sample data
- **Usage:** `go run cmd/azure-collector/main.go`

#### 3. Azure Full Collector (`cmd/azure-full-collector/main.go`)
- **Purpose:** Complete data collection from one region
- **Scope:** All services and pricing from East US
- **Usage:** `go run cmd/azure-full-collector/main.go`

#### 4. Azure DB Collector (`cmd/azure-db-collector/main.go`)
- **Purpose:** Production data collection and storage
- **Scope:** Complete data with database storage
- **Usage:** `go run cmd/azure-db-collector/main.go`

## Collection Process

### 1. Data Fetching
```go
// API query with pagination
url := "https://prices.azure.com/api/retail/prices"
params := "$filter=armRegionName eq 'eastus'"
```

### 2. Data Transformation  
Raw API data is transformed into normalized structure:
```go
type AzurePricingInsert struct {
    ServiceName      string
    ServiceFamily    string
    ProductName      string
    SKUName          string
    ARMRegionName    string
    DisplayName      string
    RetailPrice      float64
    UnitOfMeasure    string
    // ... other fields
}
```

### 3. Database Storage
- **Batch Processing:** 100 records per transaction
- **Deduplication:** Unique constraints prevent duplicates  
- **Versioning:** Each collection run gets a version number
- **Error Handling:** Failed collections are logged

### 4. Collection Metadata
```sql
CREATE TABLE azure_collection_runs (
    version INTEGER PRIMARY KEY,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'running',
    total_items INTEGER,
    regions_collected TEXT[],
    error_message TEXT
);
```

## Data Quality

### Validation
- Required fields checked before insertion
- Price values validated as positive numbers
- Date formats standardized
- Region names validated against known list

### Error Handling
- API rate limiting respected
- Network timeout handling
- Partial collection recovery
- Failed record logging

### Data Freshness
- Collections tracked by timestamp
- API provides real-time pricing
- Update mechanism for price changes
- Historical data preserved with versioning

## Performance Metrics

### Collection Speed
- ~1000 records in 5 seconds
- Bulk operations with transactions
- Efficient deduplication
- Minimal memory footprint

### API Limits
- No explicit rate limits observed
- Pagination at 2000 records
- Some queries return errors after 2000 items
- Regional filtering recommended

## Monitoring

### Collection Status
```bash
# Check last collection
SELECT * FROM azure_collection_runs ORDER BY version DESC LIMIT 1;

# Count pricing records
SELECT COUNT(*) FROM azure_pricing;

# Services coverage
SELECT COUNT(DISTINCT service_name) FROM azure_services;
```

### Data Quality Checks
```sql
-- Find missing prices
SELECT COUNT(*) FROM azure_pricing WHERE retail_price IS NULL;

-- Check regional coverage  
SELECT r.display_name, COUNT(p.id) as price_count
FROM azure_regions r
LEFT JOIN azure_pricing p ON r.id = p.region_id
GROUP BY r.id, r.display_name;
```

## Future Enhancements

### Automation
- Scheduled collections
- Incremental updates
- Change detection
- Alert system

### Multi-Cloud
- AWS pricing integration
- GCP pricing support
- Cross-cloud comparison
- Unified data model