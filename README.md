# CPC - Cloud Price Compare

A production-grade API service that aggregates and serves Azure pricing data through a unified GraphQL API with raw JSON storage and on-demand population endpoints.

## Quick Start

### Prerequisites

- Go 1.24+
- Docker and Docker Compose

### Running with Docker Compose (Recommended)

1. Start the complete stack (PostgreSQL + API + Documentation):
```bash
docker-compose up -d
```

2. Access the services:
- **GraphQL API & Playground**: http://localhost:8080
- **Documentation**: http://localhost:3000
- **PostgreSQL**: localhost:5432

### Basic GraphQL Queries

**System Information:**
```graphql
query {
  hello
  providers { name }
  categories { name description }
}
```

**Azure Data Overview:**
```graphql
query {
  azureRegions { name }
  azureServices { name }
  azureCollections {
    collectionId
    region
    status
    totalItems
    startedAt
  }
}
```

**Raw Azure Pricing Data:**
```graphql
query {
  azurePricing {
    serviceName
    productName
    retailPrice
    unitOfMeasure
    armRegionName
  }
}
```

## Population Endpoints

### Single Region Collection
```bash
# Collect pricing data for East US
curl -X POST http://localhost:8080/populate \
  -H "Content-Type: application/json" \
  -d '{"region": "eastus"}'
```

### All Regions Collection
```bash
# Collect data from all 70+ Azure regions (concurrent)
curl -X POST http://localhost:8080/populate-all \
  -H "Content-Type: application/json" \
  -d '{"concurrency": 3}'
```

### Progress Monitoring
```graphql
query {
  azureCollections {
    region
    status
    totalItems
    progress
    duration
    errorMessage
  }
}
```

## Available Endpoints

### GraphQL API (`http://localhost:8080/query`)
- **hello** - System status and greeting
- **providers** - Cloud providers (AWS, Azure)
- **categories** - Service categories (13 types)
- **azureRegions** - Azure regions with collected data
- **azureServices** - Azure services with collected data
- **azurePricing** - Raw Azure pricing data with filters
- **azureCollections** - Collection run tracking and progress

### Population Endpoints
- **POST /populate** - Collect data for single Azure region
- **POST /populate-all** - Collect data from all Azure regions concurrently

### Web Playground (`http://localhost:8080`)
- Interactive GraphQL playground with sample queries
- Region-specific population buttons
- Real-time progress monitoring with auto-refresh
- Pre-built query templates

## Development Setup

### Local Development (Alternative)
```bash
# Start PostgreSQL only
docker-compose up -d postgres

# Install Go dependencies
go mod download

# Run the API server locally
go run cmd/server/main.go
```

### Direct Data Collection Tools
```bash
# Collect single region data
go run cmd/azure-raw-collector/main.go eastus

# Collect all regions with 3 concurrent workers
go run cmd/azure-all-regions/main.go 3
```

## Project Structure

```
cpc/
├── cmd/
│   ├── server/main.go              # GraphQL API server with population endpoints
│   ├── azure-raw-collector/        # Single region data collector
│   └── azure-all-regions/          # Multi-region concurrent collector
├── internal/database/
│   ├── database.go                 # Core database operations
│   └── azure_raw.go               # Azure raw data operations & progress tracking
├── docs-site/                     # Docusaurus documentation site
├── docker-compose.yml             # Complete Docker stack
├── Dockerfile                     # Go application container
├── init.sql                       # Database schema (raw JSON approach)
└── .dockerignore                  # Docker build optimization
```

## Architecture

### Raw JSON Storage Approach
- **azure_pricing_raw** - Raw Azure API responses stored as JSONB
- **azure_collections** - Collection run tracking with progress metadata
- **Simplified pipeline** - Direct JSON storage, query on-demand
- **Preserved metadata** - Complete Azure API response structure maintained

### Key Features
- **No authentication required** - Uses Azure's public pricing API
- **70+ regions supported** - Global Azure region coverage
- **Concurrent collection** - Configurable worker pools for faster data gathering
- **Progress tracking** - Real-time status updates with collection metadata
- **Docker orchestration** - Complete containerized stack ready for deployment

## Data Scale
- **~2,000-5,000 pricing items** per Azure region
- **150,000-300,000 total records** for complete global dataset
- **30-60 minutes** for full all-regions collection
- **30-60 seconds** per single region collection
