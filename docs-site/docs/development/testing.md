# Testing Guide

How to test the CPC API and verify functionality.

## API Testing

### GraphQL Playground
The easiest way to test queries is through the built-in playground:

1. Start the API server: `go run cmd/server/main.go`
2. Open [http://localhost:8080](http://localhost:8080)
3. Use the interactive query editor

### Sample Queries

#### Basic Health Check
```graphql
{
  hello
}
```

#### Complete Data Overview
```graphql
{
  hello
  providers {
    id
    name
  }
  categories {
    name
    description
  }
  azureServices {
    serviceName
    serviceFamily
  }
  azureRegions {
    armRegionName
    displayName
  }
  azurePricing {
    serviceName
    productName
    retailPrice
    unitOfMeasure
    region
  }
}
```

## cURL Testing

### Basic Query
```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{hello}"}'
```

### Azure Services Query
```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{azureServices{serviceName serviceFamily}}"}'
```

### Complex Query
```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{azurePricing{serviceName productName retailPrice unitOfMeasure region}}"}'
```

## Database Testing

### Direct Database Queries
```bash
# Connect to database
docker exec -it cpc-postgres-1 psql -U cpc_user -d cpc_db
```

### Verification Queries
```sql
-- Check data counts
SELECT 
  (SELECT COUNT(*) FROM azure_services) as services,
  (SELECT COUNT(*) FROM azure_regions) as regions,
  (SELECT COUNT(*) FROM azure_products) as products,
  (SELECT COUNT(*) FROM azure_skus) as skus,
  (SELECT COUNT(*) FROM azure_pricing) as pricing_records;

-- Sample pricing data
SELECT 
  s.service_name,
  p.product_name,
  sk.sku_name,
  r.display_name,
  pr.retail_price,
  pr.unit_of_measure
FROM azure_pricing pr
JOIN azure_services s ON pr.service_id = s.id
JOIN azure_products p ON pr.product_id = p.id
JOIN azure_skus sk ON pr.sku_id = sk.id
JOIN azure_regions r ON pr.region_id = r.id
LIMIT 10;

-- Check collection status
SELECT * FROM azure_collection_runs ORDER BY version DESC;
```

## Data Collection Testing

### Explore Azure API
```bash
# Test API connectivity and explore data structure
go run cmd/azure-explorer/main.go
```

Expected output:
- List of service categories
- Sample services from each category
- Data structure examples

### Collect Sample Data
```bash
# Collect from multiple regions (limited data)
go run cmd/azure-collector/main.go
```

### Full Data Collection
```bash
# Collect complete data from East US
go run cmd/azure-full-collector/main.go
```

### Database Population
```bash
# Collect and store in database
go run cmd/azure-db-collector/main.go
```

Expected results:
- New collection run entry
- Services, products, SKUs populated
- Pricing records inserted
- Collection marked as completed

## Performance Testing

### Response Time Testing
```bash
# Time a complex query
time curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{azurePricing{serviceName productName retailPrice}}"}'
```

### Database Performance
```sql
-- Query execution time
EXPLAIN ANALYZE SELECT COUNT(*) FROM azure_pricing 
JOIN azure_services s ON azure_pricing.service_id = s.id
WHERE s.service_name = 'Virtual Machines';
```

## Error Testing

### Invalid Queries
```bash
# Test malformed GraphQL
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{invalid_field}"}'
```

### Database Connection
```bash
# Test with database down
docker-compose stop postgres
go run cmd/server/main.go
# Should show connection error
```

## Integration Testing

### End-to-End Workflow
1. Start fresh database: `docker-compose down -v && docker-compose up -d postgres`
2. Collect data: `go run cmd/azure-db-collector/main.go`
3. Start API: `go run cmd/server/main.go`
4. Test queries via playground or cURL
5. Verify data in database

### Expected Results Checklist
- [ ] Database tables created
- [ ] Azure services populated (~60+ services)
- [ ] Regions populated (1+ regions)
- [ ] Products populated (400+ products)
- [ ] SKUs populated (800+ SKUs)
- [ ] Pricing records populated (1000+ records)
- [ ] GraphQL queries return data
- [ ] No errors in server logs

## Test Data Validation

### Data Quality Checks
```sql
-- Check for missing required fields
SELECT COUNT(*) FROM azure_pricing WHERE retail_price IS NULL;
SELECT COUNT(*) FROM azure_services WHERE service_name IS NULL;

-- Verify foreign key relationships
SELECT COUNT(*) FROM azure_pricing p 
LEFT JOIN azure_services s ON p.service_id = s.id 
WHERE s.id IS NULL;

-- Check for duplicate entries
SELECT service_name, COUNT(*) 
FROM azure_services 
GROUP BY service_name 
HAVING COUNT(*) > 1;
```

## Automated Testing

### Future Test Implementation
```go
// Example test structure
func TestGraphQLQueries(t *testing.T) {
    // Setup test database
    // Run sample queries
    // Verify responses
}

func TestDataCollection(t *testing.T) {
    // Mock Azure API
    // Test collection process
    // Verify data integrity
}
```