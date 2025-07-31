# API Reference Overview

CPC provides a GraphQL API for querying cloud pricing data. The API is designed to be simple, efficient, and scalable.

## Endpoint

- **GraphQL API:** `http://localhost:8080/query`
- **Playground:** `http://localhost:8080/`

## Authentication

Currently, the API does not require authentication. This may change in future versions for production deployments.

## Core Concepts

### Services
Cloud services like "Virtual Machines", "Storage", etc.

### Products
Specific products within a service, e.g., "Virtual Machines BS Series"

### SKUs
Stock Keeping Units - specific configurations within products

### Regions
Geographic locations where services are available

### Pricing
Actual pricing data with rates, units, and effective dates

## Available Queries

| Query | Description |
|-------|-------------|
| `hello` | Simple health check |
| `azureServices` | List all Azure services |
| `azureRegions` | List all Azure regions |
| `azurePricing` | Get pricing data with full details |
| `providers` | List cloud providers (AWS, Azure) |
| `categories` | List service categories |

## Response Format

All responses follow the GraphQL standard format:

```json
{
  "data": {
    // Your requested data
  },
  "errors": [
    // Any errors that occurred
  ]
}
```