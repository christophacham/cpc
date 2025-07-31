# Getting Started

Welcome to Cloud Price Compare (CPC) - a production-grade API service for aggregating and comparing cloud pricing data.

## Overview

CPC provides a GraphQL API for accessing cloud pricing information from multiple providers. Currently supports Azure with plans to expand to AWS and other cloud providers.

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.19+ (for development)
- PostgreSQL (handled by Docker)

### Running the Services

1. **Start the database:**
   ```bash
   docker-compose up -d postgres
   ```

2. **Run the API server:**
   ```bash
   go run cmd/server/main.go
   ```

3. **Access the GraphQL playground:**
   Open [http://localhost:8080](http://localhost:8080) in your browser

## First Query

Try this basic query in the GraphQL playground:

```graphql
{
  hello
  azureServices {
    serviceName
    serviceFamily
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

## Data Collection

To populate the database with Azure pricing data:

```bash
go run cmd/azure-db-collector/main.go
```

This will fetch pricing data from Azure's public API and store it in the normalized database schema.

## Next Steps

- [API Reference](/api-reference/overview) - Learn about available queries
- [Architecture](/architecture/overview) - Understand the system design
- [Development Setup](/development/setup) - Set up your development environment