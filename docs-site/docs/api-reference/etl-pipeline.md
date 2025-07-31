# ETL Pipeline API Reference

The ETL (Extract, Transform, Load) Pipeline provides on-demand normalization of raw cloud pricing data from AWS and Azure into a unified format for cross-provider comparisons.

## Overview

The ETL Pipeline exposes a GraphQL API that allows you to:
- **Start normalization jobs** with configurable parameters
- **Monitor real-time progress** of running jobs
- **Cancel jobs** if needed
- **Query job history** and results

## GraphQL Endpoint

```
POST http://localhost:8080/query
```

## Mutations

### Start Normalization Job

Start a new ETL job to normalize raw pricing data.

```graphql
mutation StartNormalization($config: NormalizationConfigInput!) {
  startNormalization(config: $config) {
    id
    type
    status
    provider
    startedAt
    configuration {
      providers
      regions
      services
      batchSize
      concurrentWorkers
      clearExisting
      dryRun
    }
    progress {
      totalRecords
      processedRecords
      normalizedRecords
      currentStage
      rate
    }
  }
}
```

**Variables:**
```json
{
  "config": {
    "type": "NORMALIZE_ALL",
    "clearExisting": true,
    "batchSize": 1000,
    "concurrentWorkers": 4
  }
}
```

**Response:**
```json
{
  "data": {
    "startNormalization": {
      "id": "normalize_all-1722455234",
      "type": "NORMALIZE_ALL",
      "status": "PENDING",
      "provider": null,
      "startedAt": "2025-07-31T22:30:34Z",
      "configuration": {
        "providers": [],
        "regions": [],
        "services": [],
        "batchSize": 1000,
        "concurrentWorkers": 4,
        "clearExisting": true,
        "dryRun": false
      },
      "progress": {
        "totalRecords": 0,
        "processedRecords": 0,
        "normalizedRecords": 0,
        "currentStage": "Initializing",
        "rate": 0.0
      }
    }
  }
}
```

### Cancel ETL Job

Cancel a running or pending ETL job.

```graphql
mutation CancelJob($id: ID!) {
  cancelETLJob(id: $id)
}
```

**Variables:**
```json
{
  "id": "normalize_all-1722455234"
}
```

**Response:**
```json
{
  "data": {
    "cancelETLJob": true
  }
}
```

## Queries

### Get Specific Job

Retrieve details and progress of a specific ETL job.

```graphql
query GetETLJob($id: ID!) {
  etlJob(id: $id) {
    id
    type
    status
    provider
    startedAt
    completedAt
    error
    progress {
      totalRecords
      processedRecords
      normalizedRecords
      skippedRecords
      errorRecords
      currentStage
      lastUpdated
      rate
    }
    configuration {
      providers
      regions
      services
      batchSize
      concurrentWorkers
      clearExisting
      dryRun
    }
  }
}
```

**Variables:**
```json
{
  "id": "normalize_all-1722455234"
}
```

**Response (Running Job):**
```json
{
  "data": {
    "etlJob": {
      "id": "normalize_all-1722455234",
      "type": "NORMALIZE_ALL",
      "status": "RUNNING",
      "provider": "azure",
      "startedAt": "2025-07-31T22:30:34Z",
      "completedAt": null,
      "error": null,
      "progress": {
        "totalRecords": 287450,
        "processedRecords": 45600,
        "normalizedRecords": 42100,
        "skippedRecords": 2800,
        "errorRecords": 700,
        "currentStage": "Processing Azure raw data",
        "lastUpdated": "2025-07-31T22:32:15Z",
        "rate": 1250.5
      },
      "configuration": {
        "providers": [],
        "regions": [],
        "services": [],
        "batchSize": 1000,
        "concurrentWorkers": 4,
        "clearExisting": true,
        "dryRun": false
      }
    }
  }
}
```

**Response (Completed Job):**
```json
{
  "data": {
    "etlJob": {
      "id": "normalize_all-1722455234",
      "type": "NORMALIZE_ALL", 
      "status": "COMPLETED",
      "provider": "azure",
      "startedAt": "2025-07-31T22:30:34Z",
      "completedAt": "2025-07-31T22:45:22Z",
      "error": null,
      "progress": {
        "totalRecords": 287450,
        "processedRecords": 287450,
        "normalizedRecords": 264800,
        "skippedRecords": 18200,
        "errorRecords": 4450,
        "currentStage": "Completed",
        "lastUpdated": "2025-07-31T22:45:22Z",
        "rate": 1180.2
      },
      "configuration": {
        "providers": [],
        "regions": [],
        "services": [],
        "batchSize": 1000,
        "concurrentWorkers": 4,
        "clearExisting": true,
        "dryRun": false
      }
    }
  }
}
```

### Get All Jobs

List all ETL jobs (running, completed, and failed).

```graphql
query GetAllETLJobs {
  etlJobs {
    id
    type
    status
    provider
    startedAt
    completedAt
    progress {
      totalRecords
      processedRecords
      normalizedRecords
      currentStage
      rate
    }
  }
}
```

**Response:**
```json
{
  "data": {
    "etlJobs": [
      {
        "id": "normalize_all-1722455234",
        "type": "NORMALIZE_ALL",
        "status": "COMPLETED",
        "provider": "azure",
        "startedAt": "2025-07-31T22:30:34Z",
        "completedAt": "2025-07-31T22:45:22Z",
        "progress": {
          "totalRecords": 287450,
          "processedRecords": 287450,
          "normalizedRecords": 264800,
          "currentStage": "Completed",
          "rate": 1180.2
        }
      },
      {
        "id": "normalize_provider-1722453567",
        "type": "NORMALIZE_PROVIDER",
        "status": "RUNNING",
        "provider": "aws",
        "startedAt": "2025-07-31T22:26:07Z",
        "completedAt": null,
        "progress": {
          "totalRecords": 487200,
          "processedRecords": 156700,
          "normalizedRecords": 145300,
          "currentStage": "Processing AWS raw data",
          "rate": 892.4
        }
      }
    ]
  }
}
```

## Types & Enums

### ETLJob Type

```graphql
type ETLJob {
  id: ID!                           # Unique job identifier
  type: ETLJobType!                 # Type of normalization job
  provider: String                  # Current provider being processed
  status: ETLJobStatus!             # Current job status
  progress: ETLJobProgress          # Real-time progress information
  startedAt: String!                # ISO 8601 timestamp
  completedAt: String               # ISO 8601 timestamp (when completed)
  error: String                     # Error message (if failed)
  configuration: ETLJobConfiguration! # Job configuration used
}
```

### ETLJobProgress Type

```graphql
type ETLJobProgress {
  totalRecords: Int!      # Total records to process
  processedRecords: Int!  # Records processed so far
  normalizedRecords: Int! # Successfully normalized records
  skippedRecords: Int!    # Records skipped (zero price, unmapped)
  errorRecords: Int!      # Records that failed processing
  currentStage: String!   # Human-readable current stage
  lastUpdated: String!    # ISO 8601 timestamp of last update
  rate: Float!            # Processing rate (records per second)
}
```

### ETLJobConfiguration Type

```graphql
type ETLJobConfiguration {
  providers: [String!]    # Filter by providers (aws, azure)
  regions: [String!]      # Filter by regions (us-east-1, eastus)
  services: [String!]     # Filter by services (AmazonEC2, Virtual Machines)
  batchSize: Int!         # Records processed per batch
  concurrentWorkers: Int! # Number of concurrent workers
  clearExisting: Boolean! # Whether to clear existing normalized data
  dryRun: Boolean!        # Test mode without inserting data
}
```

### ETLJobType Enum

```graphql
enum ETLJobType {
  NORMALIZE_ALL        # Process all raw data from all providers
  NORMALIZE_PROVIDER   # Process data from specific provider(s)
  NORMALIZE_REGION     # Process data from specific region(s)
  NORMALIZE_SERVICE    # Process data from specific service(s)
  CLEANUP_NORMALIZED   # Remove orphaned normalized records
}
```

### ETLJobStatus Enum

```graphql
enum ETLJobStatus {
  PENDING     # Job queued for processing
  RUNNING     # Currently processing
  COMPLETED   # Successfully finished
  FAILED      # Encountered fatal errors
  CANCELLED   # Manually cancelled
}
```

## Configuration Examples

### Full Normalization (All Data)

```json
{
  "type": "NORMALIZE_ALL",
  "clearExisting": true,
  "batchSize": 1000,
  "concurrentWorkers": 4
}
```

### Azure-Only Processing

```json
{
  "type": "NORMALIZE_PROVIDER",
  "providers": ["azure"],
  "batchSize": 500,
  "concurrentWorkers": 2
}
```

### Specific Regions

```json
{
  "type": "NORMALIZE_REGION",
  "regions": ["us-east-1", "us-west-2", "eastus", "westus2"],
  "batchSize": 1500,
  "concurrentWorkers": 6
}
```

### Specific Services

```json
{
  "type": "NORMALIZE_SERVICE",
  "services": ["AmazonEC2", "Virtual Machines"],
  "providers": ["aws", "azure"]
}
```

### Dry Run (Testing)

```json
{
  "type": "NORMALIZE_ALL",
  "dryRun": true,
  "batchSize": 100,
  "concurrentWorkers": 1
}
```

### High Performance Processing

```json
{
  "type": "NORMALIZE_ALL",
  "batchSize": 2000,
  "concurrentWorkers": 8,
  "clearExisting": true
}
```

## Normalized Data Output

After ETL processing completes, the normalized data is available through standard pricing queries with enhanced cross-provider comparison capabilities.

### Normalized Data Structure

The ETL pipeline produces records with this structure:

```json
{
  "provider": "aws",                    # aws or azure
  "providerServiceCode": "AmazonEC2",   # Original service identifier
  "serviceCategory": "Compute & Web",   # Standardized category
  "serviceFamily": "Virtual Machines",  # Service family
  "serviceType": "Virtual Machines",    # Normalized service type
  "normalizedRegion": "us-east",        # Standardized region code
  "providerRegion": "us-east-1",        # Original region code
  "resourceName": "t3.medium",          # Resource identifier
  "resourceSpecs": {                    # Standardized specifications
    "vcpu": 2,
    "memoryGB": 4.0,
    "storageType": "EBS-Optimized"
  },
  "pricePerUnit": 0.0416,              # Standardized price
  "unit": "hour",                       # Standardized unit
  "currency": "USD",                    # Currency
  "pricingModel": "on_demand",          # on_demand, reserved_1yr, etc.
  "pricingDetails": {                   # Additional pricing info
    "termLength": "1yr",
    "paymentOption": "All Upfront"
  }
}
```

### Cross-Provider Queries

Once normalized, you can perform cross-provider comparisons:

```graphql
query CompareVirtualMachines {
  # This would be a new query type enabled by normalization
  normalizedPricing(
    serviceType: "Virtual Machines"
    resourceSpecs: { vcpu: 2, memoryGB: 4 }
    pricingModel: "on_demand"
  ) {
    provider
    resourceName
    pricePerUnit
    unit
    normalizedRegion
    resourceSpecs {
      vcpu
      memoryGB
      storageType
    }
  }
}
```

## Error Handling

### Common Error Responses

**Invalid Job Type:**
```json
{
  "errors": [
    "Variable '$config' got invalid value 'INVALID_TYPE' at 'config.type'; Expected type 'ETLJobType'."
  ]
}
```

**Job Not Found:**
```json
{
  "data": {
    "etlJob": null
  }
}
```

**ETL Pipeline Not Initialized:**
```json
{
  "errors": [
    "ETL pipeline not initialized"
  ]
}
```

**Failed to Start Job:**
```json
{
  "errors": [
    "failed to start normalization job: database connection failed"
  ]
}
```

## Performance Characteristics

### Processing Rates

| Configuration | Records/Second | Use Case |
|---|---|---|
| `batchSize: 500, workers: 2` | ~600-800 | Conservative processing |
| `batchSize: 1000, workers: 4` | ~1000-1200 | Balanced performance |
| `batchSize: 2000, workers: 8` | ~1500-2000 | High-performance processing |

### Expected Processing Times

| Dataset | Records | Time (Conservative) | Time (High-Performance) |
|---|---|---|---|
| Azure Only | ~300,000 | ~6-8 minutes | ~3-4 minutes |
| AWS Only | ~500,000 | ~10-12 minutes | ~5-6 minutes |
| Full Dataset | ~800,000 | ~15-20 minutes | ~8-10 minutes |

### Resource Usage

- **Memory**: ~100-500MB depending on batch size and concurrency
- **CPU**: Scales with concurrent workers
- **Database**: Bulk insert operations for optimal performance
- **Network**: Minimal - processes existing database data

## Monitoring & Troubleshooting

### Progress Monitoring

Monitor job progress by polling the `etlJob` query every 2-5 seconds:

```graphql
query MonitorProgress($id: ID!) {
  etlJob(id: $id) {
    status
    progress {
      processedRecords
      totalRecords
      rate
      currentStage
      errorRecords
    }
  }
}
```

### Troubleshooting Common Issues

**Job Stuck in PENDING:**
- Check database connectivity
- Verify service mappings exist
- Check for sufficient database permissions

**High Error Count:**
- Review raw data quality
- Check service mapping completeness
- Verify region mappings

**Slow Processing:**
- Increase batch size
- Increase concurrent workers
- Check database performance

**Memory Issues:**
- Reduce batch size
- Reduce concurrent workers
- Monitor system resources

This API provides powerful, flexible normalization capabilities with real-time monitoring and comprehensive error handling, enabling seamless transformation of raw cloud pricing data into normalized, comparable formats.