# GraphQL Queries

This page documents all available GraphQL queries and mutations in the CPC API.

## Basic Queries

### Hello Query
Simple health check query.

```graphql
{
  hello
}
```

**Response:**
```json
{
  "data": {
    "hello": "Hello from Cloud Price Compare GraphQL API!"
  }
}
```

## Azure Pricing Queries

### Azure Services
Get all available Azure services.

```graphql
{
  azureServices {
    id
    serviceName
    serviceFamily
    createdAt
  }
}
```

### Azure Regions
Get all Azure regions.

```graphql
{
  azureRegions {
    id
    armRegionName
    displayName
    createdAt
  }
}
```

### Azure Pricing Data
Get pricing information with full details.

```graphql
{
  azurePricing {
    serviceName
    serviceFamily
    productName
    skuName
    armSkuName
    region
    meterName
    retailPrice
    unitOfMeasure
    effectiveDate
  }
}
```

## System Queries

### Providers
Get all cloud providers.

```graphql
{
  providers {
    id
    name
    createdAt
  }
}
```

### Categories
Get service categories.

```graphql
{
  categories {
    id
    name
    description
    createdAt
  }
}
```

## Combined Queries

You can combine multiple queries in a single request:

```graphql
{
  hello
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

## Using cURL

You can also query the API using cURL:

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{hello azureServices{serviceName}}"}'
```