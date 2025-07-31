# ETL Pipeline Specification

## Overview
Complete ETL (Extract, Transform, Load) pipeline for normalizing raw cloud pricing data from AWS and Azure into a unified format suitable for cross-provider comparisons and cost optimization.

## Architecture Components

### 1. Pipeline Core (`internal/etl/pipeline.go`)
```
Pipeline
├── Job Management
│   ├── StartJob() - Create new ETL jobs
│   ├── GetJob() - Retrieve job status
│   ├── CancelJob() - Stop running jobs
│   └── GetAllJobs() - List all jobs
├── Normalizers
│   ├── AWSNormalizerV2 - AWS-specific normalization
│   └── AzureNormalizerV2 - Azure-specific normalization
└── Repositories
    ├── ServiceMappingRepository - Service name mappings
    ├── RegionMappingRepository - Region mappings
    ├── UnitNormalizer - Price unit standardization
    └── NormalizedPricingRepository - Output storage
```

### 2. Job Types & Configurations
```
JobTypes:
├── NORMALIZE_ALL - Process all raw data
├── NORMALIZE_PROVIDER - Process specific provider (AWS/Azure)
├── NORMALIZE_REGION - Process specific regions
├── NORMALIZE_SERVICE - Process specific services
└── CLEANUP_NORMALIZED - Remove orphaned records

JobConfiguration:
├── providers: []string - Filter by AWS/Azure
├── regions: []string - Filter by specific regions
├── services: []string - Filter by specific services
├── batchSize: int - Records per batch (default: 1000)
├── concurrentWorkers: int - Parallel workers (default: 4)
├── clearExisting: bool - Clear old normalized data
└── dryRun: bool - Test mode without inserting
```

### 3. Processing Flow
```
1. Input Validation
   ├── Validate job configuration
   ├── Check provider/region/service filters
   └── Verify normalizer availability

2. Data Retrieval
   ├── Count total records for progress tracking
   ├── Fetch raw data in configurable batches
   └── Apply filters (provider/region/service)

3. Concurrent Processing
   ├── Worker Pool (configurable size)
   ├── Batch Processing (configurable batch size)
   ├── Provider-Specific Normalization
   │   ├── AWS: Parse pricing terms, extract specs
   │   └── Azure: Parse pricing items, extract VM specs
   └── Result Aggregation

4. Output & Storage
   ├── Bulk Insert normalized records
   ├── Progress tracking updates
   └── Error collection and reporting
```

## Data Structures

### Input: Raw Pricing Data
```
AWS Raw Data:
├── service_code: string (e.g., "AmazonEC2")
├── region: string (e.g., "us-east-1")
├── data: JSON - Complex AWS pricing structure
│   ├── product.attributes - Resource specifications
│   └── terms.OnDemand/Reserved - Pricing models
└── collection_id: string - Batch identifier

Azure Raw Data:
├── service_name: string (e.g., "Virtual Machines")
├── region: string (e.g., "eastus")  
├── data: JSON - Azure pricing structure
│   ├── retailPrice/unitPrice - Pricing amounts
│   ├── armSkuName - VM specifications
│   └── meterName - Usage metrics
└── collection_id: string - Batch identifier
```

### Output: Normalized Pricing
```
NormalizedPricing:
├── provider: string (aws/azure)
├── service_category: string (Compute & Web, Storage, etc.)
├── service_type: string (Virtual Machines, etc.)
├── normalized_region: string (us-east, etc.)
├── resource_name: string (t3.medium, Standard_D2s_v3)
├── resource_specs: JSON
│   ├── vcpu: int
│   ├── memory_gb: float64
│   ├── storage_gb: float64
│   └── storage_type: string
├── price_per_unit: float64
├── unit: string (hour, gb-month, request)
├── currency: string (USD)
├── pricing_model: string (on_demand, reserved_1yr, etc.)
├── pricing_details: JSON
│   ├── term_length: string (1yr, 3yr)
│   └── payment_option: string (All Upfront, etc.)
└── raw_data_id: int - Traceability to source
```

## GraphQL API Interface

### Mutations
```graphql
# Start new normalization job
startNormalization(config: NormalizationConfigInput!): ETLJob!

# Cancel running job  
cancelETLJob(id: ID!): Boolean!
```

### Queries
```graphql
# Get specific job
etlJob(id: ID!): ETLJob

# List all jobs
etlJobs: [ETLJob!]!
```

### Types
```graphql
type ETLJob {
  id: ID!
  type: ETLJobType!
  status: ETLJobStatus!
  provider: String
  progress: ETLJobProgress
  startedAt: String!
  completedAt: String
  error: String
  configuration: ETLJobConfiguration!
}

type ETLJobProgress {
  totalRecords: Int!
  processedRecords: Int!
  normalizedRecords: Int!
  skippedRecords: Int!
  errorRecords: Int!
  currentStage: String!
  lastUpdated: String!
  rate: Float! # records per second
}
```

## Processing Capabilities

### Scale & Performance
```
Current Capacity:
├── Azure: ~300,000 pricing records
├── AWS: ~500,000 pricing records  
├── Total: ~800,000 records to process
├── Batch Size: Configurable (default: 1000)
├── Workers: Configurable (default: 4)
└── Estimated Time: ~30-60 minutes for full normalization

Performance Tuning:
├── Increase batch size for faster processing
├── Increase workers for more concurrency
├── Filter by provider/region for targeted processing
└── Use dry-run mode for testing
```

### Resource Extraction Capabilities
```
AWS Resource Extraction:
├── EC2: Instance types, vCPU, memory, storage
├── RDS: Database engine, instance class, storage
├── Lambda: Architecture, memory allocation
├── S3: Storage class, transfer types
└── General: All AWS service attributes

Azure Resource Extraction:
├── Virtual Machines: ARM SKU parsing (D/F/E/B series)
├── Memory Ratios: D=4GB/vCPU, F=2GB/vCPU, E=8GB/vCPU
├── Storage Detection: Premium SSD identification
├── Functions: Azure Functions specifications
└── Storage: Hot/Cool/Archive tier detection
```

## Error Handling & Monitoring

### Error Categories
```
Validation Errors:
├── Invalid provider/region/service
├── Invalid configuration parameters
└── Missing required fields

Processing Errors:
├── JSON parsing failures
├── Service mapping not found
├── Region mapping not found
├── Resource extraction failures
└── Database insertion failures

System Errors:
├── Database connection issues
├── Memory/resource constraints
└── Job cancellation
```

### Progress Monitoring
```
Real-time Metrics:
├── Records processed per second
├── Success/skip/error counts
├── Current processing stage
├── Estimated completion time
└── Memory/resource usage

Job States:
├── PENDING - Job queued
├── RUNNING - Currently processing
├── COMPLETED - Successfully finished
├── FAILED - Encountered errors
└── CANCELLED - Manually stopped
```

## Integration Points

### Database Dependencies
```
Required Tables:
├── aws_pricing_raw - Source AWS data
├── azure_pricing_raw - Source Azure data
├── service_mappings - Provider service mappings
├── normalized_regions - Region mappings
└── normalized_pricing - Output table

Required Functions:
├── InsertNormalizedPricing()
├── BulkInsertNormalizedPricing()
├── GetServiceMappings()
└── GetNormalizedRegions()
```

### External Dependencies
```
Go Packages:
├── github.com/lib/pq - PostgreSQL driver
├── github.com/google/uuid - Job ID generation
├── sync - Concurrent processing
└── context - Cancellation support

Internal Dependencies:
├── internal/database - Database operations
├── internal/normalizer - Normalization logic
└── internal/graph - GraphQL integration
```

## Usage Examples

### Basic Full Normalization
```graphql
mutation {
  startNormalization(config: {
    type: NORMALIZE_ALL
    clearExisting: true
  }) {
    id
    status
  }
}
```

### Azure-Only Processing
```graphql
mutation {
  startNormalization(config: {
    type: NORMALIZE_PROVIDER
    providers: ["azure"]
    batchSize: 500
    concurrentWorkers: 2
  }) {
    id
    progress {
      totalRecords
      currentStage
    }
  }
}
```

### Dry Run Testing
```graphql
mutation {
  startNormalization(config: {
    type: NORMALIZE_ALL
    dryRun: true
    batchSize: 100
  }) {
    id
    configuration {
      dryRun
    }
  }
}
```

### Progress Monitoring
```graphql
query {
  etlJob(id: "normalize_all-1722455234") {
    status
    progress {
      processedRecords
      totalRecords
      normalizedRecords
      skippedRecords
      errorRecords
      currentStage
      rate
    }
  }
}
```

## Testing & Validation

### Test Command
```bash
go run cmd/etl-test/main.go
```

### Test Features
```
Test Capabilities:
├── Database connection validation
├── Pipeline initialization testing
├── Job creation and monitoring
├── Progress tracking validation
├── Automatic timeout and cleanup
└── Real-time status reporting
```

This specification provides a complete view of the ETL pipeline architecture, capabilities, and interfaces, making it easy to understand what's inside and how to use it effectively.