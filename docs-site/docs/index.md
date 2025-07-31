---
slug: /
---

# Cloud Price Compare (CPC)

Welcome to the official documentation for Cloud Price Compare - a production-grade API service for aggregating and comparing cloud pricing data.

## What is CPC?

CPC provides a unified GraphQL API for accessing and comparing cloud pricing information across multiple providers. Currently supporting Azure with plans for AWS and other providers.

## Key Features

- **GraphQL API** - Flexible queries for cloud pricing data
- **Normalized Data** - Consistent structure across providers
- **Real-time Pricing** - Direct from provider APIs
- **Docker Ready** - Easy deployment with containers
- **Comprehensive Docs** - Complete API and development guides

## Quick Links

- [Getting Started](getting-started) - Set up and run CPC
- [API Reference](api-reference/overview) - GraphQL queries and endpoints
- [Architecture](architecture/overview) - System design and components
- [Development](development/setup) - Local development setup

## Sample Query

```graphql
{
  azureServices {
    serviceName
    serviceFamily
  }
  azurePricing {
    serviceName
    retailPrice
    unitOfMeasure
    region
  }
}
```

## Get Started

1. Clone the repository
2. Start the database: `docker-compose up -d postgres`
3. Run the API: `go run cmd/server/main.go`
4. Open GraphQL playground: http://localhost:8080

Ready to dive in? Head to our [Getting Started](getting-started) guide!