# CPC - Current State Documentation

## What We Have

### 1. GraphQL API Server
- **Location**: `cmd/server/main.go`
- **Features**: Basic GraphQL endpoint with database connection
- **Queries Available**:
  - `hello` - Returns greeting message
  - `messages` - Lists all messages from database
  - `providers` - Shows AWS and Azure
  - `categories` - Shows 13 service categories
  - `createMessage` mutation - Adds a message

### 2. Azure Pricing Tools
- **Explorer** (`cmd/azure-explorer/main.go`): Tests API queries for one service per category
- **Collector** (`cmd/azure-collector/main.go`): Fetches pricing from multiple regions
- **Full Collector** (`cmd/azure-full-collector/main.go`): Gets ALL pricing from one region

### 3. Database
- PostgreSQL running in Docker
- Initial tables: messages, providers, service_categories
- Pre-populated with AWS/Azure providers and 13 categories

## How to Test

### Start the Services
```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run the API server
go run cmd/server/main.go
```

### Test GraphQL API
Open browser at http://localhost:8080 or use curl:

```bash
# Test hello query
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{hello}"}'

# Get all data
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{hello messages{id content createdAt} providers{id name} categories{id name description}}"}'

# Create a message
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation {createMessage(content: \"Testing!\") {id content createdAt}}"}'
```

### Test Azure Pricing Tools
```bash
# Explore one service from each category
go run cmd/azure-explorer/main.go

# Collect from multiple regions (limited data)
go run cmd/azure-collector/main.go

# Get ALL pricing from East US (takes ~3 seconds)
go run cmd/azure-full-collector/main.go
```

## Key Findings

### Azure API
- **No auth required** - It's a public API
- **71 regions** available
- **83 services** in our test
- **2000+ pricing items** per region
- Returns prices in various units (hour, GB/month, transactions, etc.)

### Data Structure
Every pricing item has:
- ServiceName (e.g., "Virtual Machines")
- ProductName (e.g., "Virtual Machines BS Series")
- SKU & MeterName (specific configurations)
- Price, Currency, Unit
- Region info
- ServiceFamily (category)

### 4. Azure Pricing Database
- **Schema**: Normalized tables for services, products, SKUs, regions, and pricing
- **Population**: `cmd/azure-db-collector/main.go` - Fetches and stores pricing data
- **Data**: Successfully stored 1000+ Azure pricing records with proper relationships
- **GraphQL Queries**:
  - `azureServices` - Lists all Azure services
  - `azureRegions` - Lists all Azure regions  
  - `azurePricing` - Sample pricing data with full details

### Test Azure Database Features
```bash
# Populate database with Azure pricing
go run cmd/azure-db-collector/main.go

# Query Azure data via GraphQL
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{azureServices{serviceName serviceFamily} azurePricing{serviceName productName retailPrice unitOfMeasure region}}"}'
```

## Architecture Summary

### Database Schema (Normalized)
- `azure_services` - Service catalog (62 unique services)
- `azure_regions` - Region catalog (1 region so far)  
- `azure_products` - Products within services (397 unique)
- `azure_skus` - SKUs within products (802 unique)
- `azure_pricing` - Main pricing table with foreign keys
- `azure_collection_runs` - Tracks data collection versions

### Data Collection Pipeline
1. **Fetch** from Azure Retail Pricing API (no auth required)
2. **Normalize** into relational structure
3. **Store** with proper relationships and deduplication
4. **Track** collection versions and metadata

## Next Steps
1. Expand to collect from all Azure regions
2. Add service-to-category mapping logic
3. Implement pricing comparison queries
4. Add data refresh/update mechanisms