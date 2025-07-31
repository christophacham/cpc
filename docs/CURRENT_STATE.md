# CPC - Current State Documentation (Raw JSON Approach)

## What We Have

### 1. GraphQL API Server
- **Location**: `cmd/server/main.go`
- **Features**: GraphQL endpoint with raw JSON storage and population endpoints
- **Queries Available**:
  - `hello` - Returns greeting message
  - `messages` - Lists all messages from database
  - `providers` - Shows AWS and Azure
  - `categories` - Shows 13 service categories
  - `azureRegions` - Lists regions with collected data
  - `azureServices` - Lists services with collected data
  - `azurePricing` - Raw Azure pricing data
  - `azureCollections` - Collection run tracking
  - `createMessage` mutation - Adds a message

### 2. Population Endpoints
- **Single Region**: `POST /populate` - Collect data for one region
- **All Regions**: `POST /populate-all` - Collect data from all 70+ Azure regions
- **Interactive UI**: Web playground with region buttons

### 3. Azure Data Collection Tools
- **Raw Collector** (`cmd/azure-raw-collector/main.go`): Collects raw JSON data for single region
- **All Regions Collector** (`cmd/azure-all-regions/main.go`): Concurrent collection from all regions
- **Legacy Tools**: Explorer and normalized collectors (deprecated)

### 4. Database (Simplified Raw JSON Approach)
- PostgreSQL running in Docker
- **Raw storage**: `azure_pricing_raw` table with JSONB data
- **Collection tracking**: `azure_collections` table
- **Core tables**: messages, providers, service_categories

### 5. Docker Services
- **PostgreSQL**: Database with health checks
- **API Server**: Go application with population endpoints
- **Documentation**: Docusaurus site with API guides
- **Complete Stack**: All services orchestrated with docker-compose

## How to Test

### Start the Services

**Option 1: Full Docker Stack (Recommended)**
```bash
# Start all services (PostgreSQL + API + Documentation)
docker-compose up -d

# Or start specific services
docker-compose up -d postgres api
```

**Available Services:**
- **GraphQL API**: http://localhost:8080
- **Documentation**: http://localhost:3000  
- **PostgreSQL**: localhost:5432

**Option 2: Local Development**
```bash
# Start PostgreSQL only
docker-compose up -d postgres

# Run the API server locally
go run cmd/server/main.go
```

### Test GraphQL API
Open browser at http://localhost:8080 or use curl:

```bash
# Test hello query
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{hello}"}'

# Get basic system info
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{hello providers{name} categories{name}}"}'

# Get Azure data overview
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{azureRegions{name} azureServices{name} azureCollections{region status}}"}'

# Get raw pricing data
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{azurePricing{serviceName retailPrice unitOfMeasure armRegionName}}"}'
```

### Test Population Endpoints
```bash
# Populate single region
curl -X POST http://localhost:8080/populate \
  -H "Content-Type: application/json" \
  -d '{"region": "eastus"}'

# Populate all regions (concurrent)
curl -X POST http://localhost:8080/populate-all \
  -H "Content-Type: application/json" \
  -d '{"concurrency": 3}'
```

### Test Raw Data Collection Tools
```bash
# Collect single region
go run cmd/azure-raw-collector/main.go eastus

# Collect all regions (with 3 concurrent workers)
go run cmd/azure-all-regions/main.go 3
```

### Docker Management
```bash
# View running containers
docker-compose ps

# View logs
docker-compose logs api
docker-compose logs postgres
docker-compose logs docs

# Stop services
docker-compose down

# Rebuild and restart
docker-compose up -d --build

# Scale specific services (if needed)
docker-compose up -d --scale api=2
```

## Key Findings

### Azure API
- **No auth required** - It's a public API
- **70+ regions** available worldwide
- **100+ services** across different categories
- **2000+ pricing items** per region (varies by region)
- Returns prices in various units (hour, GB/month, transactions, etc.)

### Raw Data Structure
Every pricing item from Azure API contains:
- ServiceName (e.g., "Virtual Machines")
- ProductName (e.g., "Virtual Machines BS Series")
- SKU & MeterName (specific configurations)
- RetailPrice, UnitPrice, Currency, UnitOfMeasure
- ArmRegionName, Location (region info)
- ServiceFamily (category)
- EffectiveStartDate, PriceType
- Complete metadata preserved as JSON

## Architecture Summary (Raw JSON Approach)

### Database Schema (Simplified)
- `azure_pricing_raw` - Raw Azure API responses stored as JSONB
- `azure_collections` - Collection run tracking and metadata
- `providers` - Cloud providers (AWS, Azure)
- `service_categories` - Service categorization
- `messages` - System messages

### Data Collection Pipeline
1. **Fetch** from Azure Retail Pricing API (no auth required)
2. **Store** raw JSON responses in JSONB column
3. **Index** key fields for fast queries
4. **Track** collection runs and status

### Benefits of Raw JSON Approach
- **Simpler**: No complex normalization logic
- **Flexible**: Preserves all original data
- **Fast**: Direct JSON inserts
- **Scalable**: Easy to add new providers/regions
- **Future-proof**: Can normalize later if needed

## Population Capabilities

### Single Region Collection
- Collect pricing data for any specific Azure region
- ~2000-5000 items per region
- Takes 30-60 seconds per region

### All Regions Collection
- Concurrent collection from 70+ Azure regions
- Configurable concurrency (3-10 workers recommended)
- Total time: 30-60 minutes for complete global dataset
- Estimated total records: 150,000-300,000 pricing items

## Next Steps
1. ✅ Collect data from all Azure regions
2. Add intelligent region selection (active regions only)
3. Implement pricing comparison queries across regions
4. Add automated data refresh scheduling
5. Expand to AWS pricing data
6. Add cost calculation and comparison features