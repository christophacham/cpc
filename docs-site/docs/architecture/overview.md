# 🏗️ Architecture Overview

This document provides a comprehensive overview of the Cloud Price Compare (CPC) architecture, designed for developers who want to understand how the system works.

## 🎯 High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Data Sources  │    │   CPC Platform   │    │   Consumers     │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ AWS Pricing │ │───►│ │ Data Ingestion│ │    │ │ GraphQL API │ │
│ │ API         │ │    │ │              │ │    │ │ Playground  │ │
│ └─────────────┘ │    │ └──────────────┘ │    │ └─────────────┘ │
│                 │    │         │        │    │                 │
│ ┌─────────────┐ │    │         ▼        │    │ ┌─────────────┐ │
│ │ Azure Retail│ │───►│ ┌──────────────┐ │───►│ │ Custom Apps │ │
│ │ Pricing API │ │    │ │ Raw Storage  │ │    │ │ & Tools     │ │
│ └─────────────┘ │    │ │ (PostgreSQL) │ │    │ └─────────────┘ │
│                 │    │ └──────────────┘ │    │                 │
└─────────────────┘    │         │        │    │ ┌─────────────┐ │
                       │         ▼        │    │ │ Research &  │ │
                       │ ┌──────────────┐ │    │ │ Analytics   │ │
                       │ │ ETL Pipeline │ │    │ └─────────────┘ │
                       │ │ (Normalize)  │ │    │                 │
                       │ └──────────────┘ │    └─────────────────┘
                       │         │        │
                       │         ▼        │
                       │ ┌──────────────┐ │
                       │ │ Normalized   │ │
                       │ │ Data Store   │ │
                       │ └──────────────┘ │
                       │                  │
                       └──────────────────┘
```

## 📊 Core Components

### 1. Data Ingestion Layer

**Purpose**: Collect pricing data from cloud provider APIs

**Components**:
- **AWS Collector** (`cmd/aws-collector/`)
  - Uses AWS Price List Query API
  - Handles authentication via AWS SDK
  - Supports 60+ AWS services
  - Automatic pagination for large datasets

- **Azure Collector** (`cmd/azure-collector/`)
  - Uses Azure Retail Pricing API (public)
  - No authentication required
  - Covers 70+ global regions
  - OData filtering for targeted collection

**Key Features**:
- **Concurrent processing** with configurable worker pools
- **Rate limiting** with exponential backoff
- **Progress tracking** with real-time updates
- **Error handling** with retry logic

### 2. Raw Data Storage

**Purpose**: Preserve complete API responses without data loss

**Database Schema**:
```sql
-- AWS raw data
CREATE TABLE aws_pricing_raw (
    id SERIAL PRIMARY KEY,
    service_code TEXT NOT NULL,
    region TEXT NOT NULL,
    data JSONB NOT NULL,                    -- Complete API response
    collection_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Azure raw data
CREATE TABLE azure_pricing_raw (
    id SERIAL PRIMARY KEY,
    service_name TEXT NOT NULL,
    region TEXT NOT NULL,
    data JSONB NOT NULL,                    -- Complete API response
    collection_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Collection tracking
CREATE TABLE aws_collections (
    collection_id TEXT PRIMARY KEY,
    service_codes TEXT[],
    regions TEXT[],
    status TEXT,
    total_items INTEGER,
    progress JSONB,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);
```

**Design Principles**:
- **JSONB for flexibility** - enables any future query pattern
- **Complete preservation** - never lose original data
- **Efficient indexing** - GIN indexes on JSONB columns
- **Collection metadata** - track progress and status

### 3. ETL Pipeline

**Purpose**: Transform raw data into normalized format for cross-provider comparisons

**Architecture**:
```
Raw Data → Job Queue → Worker Pool → Normalizers → Unified Storage
    ↓           ↓           ↓            ↓             ↓
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌─────────┐
│ AWS Raw │ │ ETL Job │ │ Worker  │ │ AWS      │ │ Normal- │
│ Tables  │ │ Manager │ │ Pool    │ │ Normal-  │ │ ized    │
│         │ │         │ │ (Go)    │ │ izer     │ │ Records │
├─────────┤ ├─────────┤ ├─────────┤ ├──────────┤ ├─────────┤
│ Azure   │ │Progress │ │ Batch   │ │ Azure    │ │ Cross   │
│ Raw     │ │Tracking │ │Process  │ │ Normal-  │ │Provider │
│ Tables  │ │         │ │         │ │ izer     │ │ Queries │
└─────────┘ └─────────┘ └─────────┘ └──────────┘ └─────────┘
```

**Key Components**:
- **Job Manager** (`internal/etl/pipeline.go`)
  - Manages concurrent ETL jobs
  - Tracks progress and status
  - Handles job cancellation

- **Normalizers** (`internal/normalizer/`)
  - Provider-specific data transformation
  - Extract resource specifications
  - Standardize pricing units and currencies

- **Worker Pools**
  - Configurable concurrency
  - Batch processing for performance
  - Error isolation and recovery

### 4. GraphQL API Layer

**Purpose**: Provide flexible, developer-friendly access to pricing data

**Schema Design**:
```graphql
type Query {
    # System information
    hello: String!
    providers: [Provider!]!
    categories: [Category!]!
    
    # Raw data access
    awsPricing: [AWSPricing!]!
    azurePricing: [AzurePricing!]!
    
    # Collection monitoring
    awsCollections: [AWSCollection!]!
    azureCollections: [AzureCollection!]!
    
    # ETL pipeline
    etlJob(id: ID!): ETLJob
    etlJobs: [ETLJob!]!
    
    # Normalized data (future)
    normalizedPricing: [NormalizedPricing!]!
}

type Mutation {
    # ETL operations
    startNormalization(config: NormalizationConfigInput!): ETLJob!
    cancelETLJob(id: ID!): Boolean!
}
```

**Resolver Architecture**:
```
GraphQL Request → Resolver → Database Query → Response
       ↓              ↓            ↓           ↓
┌─────────────┐ ┌─────────────┐ ┌─────────┐ ┌─────────┐
│ Client      │ │ GraphQL     │ │ Postgres│ │ JSON    │
│ (Playground)│ │ Resolvers   │ │ JSONB   │ │ Response│
│             │ │             │ │ Queries │ │         │
├─────────────┤ ├─────────────┤ ├─────────┤ ├─────────┤
│ Custom Apps │ │ Validation  │ │ Indexes │ │ Typed   │
│ & Tools     │ │ & Auth      │ │ & Cache │ │ Data    │
└─────────────┘ └─────────────┘ └─────────┘ └─────────┘
```

### 5. Service Mapping System

**Purpose**: Categorize and normalize service names across providers

**Mapping Tables**:
```sql
CREATE TABLE service_mappings (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,              -- 'aws' or 'azure'
    service_name TEXT NOT NULL,          -- Original service name
    category TEXT NOT NULL,              -- Standardized category
    service_type TEXT NOT NULL,          -- Normalized service type
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE normalized_regions (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    original_region TEXT NOT NULL,       -- us-east-1, eastus
    normalized_region TEXT NOT NULL,     -- us-east, us-east
    display_name TEXT NOT NULL           -- US East
);
```

**Categories** (13 standardized):
- General, Networking, Compute & Web, Containers
- Databases, Storage, AI & ML, Analytics & IoT
- Virtual Desktop, Dev Tools, Integration, Migration, Management

## 🔄 Data Flow

### Collection Flow

```
1. API Request → 2. Data Collection → 3. Raw Storage → 4. Progress Update
      ↓                  ↓                 ↓               ↓
┌─────────────┐    ┌─────────────┐   ┌─────────────┐  ┌─────────────┐
│POST /populate│    │HTTP Request │   │INSERT INTO  │  │UPDATE       │
│{region: ".."}│ → │to Azure API │ → │raw_table    │ →│collection   │
│             │    │             │   │with JSONB   │  │progress     │
└─────────────┘    └─────────────┘   └─────────────┘  └─────────────┘
                           ↓                               ↑
                   ┌─────────────┐                  ┌─────────────┐
                   │Pagination   │                  │Real-time    │
                   │NextPageLink │ ←─────────────── │monitoring   │
                   │handling     │                  │via GraphQL  │
                   └─────────────┘                  └─────────────┘
```

### ETL Flow

```
1. Raw Data → 2. Job Creation → 3. Worker Pool → 4. Normalization → 5. Storage
      ↓              ↓              ↓               ↓               ↓
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│JSONB records│ │ETL job with │ │Concurrent   │ │Parse JSON   │ │INSERT       │
│from AWS/    │ │batch config │ │workers      │ │Extract specs│ │normalized   │
│Azure APIs   │ │and progress │ │processing   │ │Map services │ │records      │
│             │ │tracking     │ │in parallel  │ │Standardize  │ │             │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
                                        ↓               ↓
                                ┌─────────────┐ ┌─────────────┐
                                │Progress     │ │Error        │
                                │updates     │ │handling     │
                                │(rec/sec)   │ │& retry      │
                                └─────────────┘ └─────────────┘
```

### Query Flow

```
1. GraphQL → 2. Resolver → 3. Database → 4. Transform → 5. Response
      ↓            ↓            ↓             ↓            ↓
┌─────────────┐ ┌─────────┐ ┌─────────────┐ ┌─────────┐ ┌─────────────┐
│query {      │ │Validate │ │SELECT data  │ │Convert  │ │JSON response│
│ azurePricing│ │& parse  │ │FROM azure_  │ │JSONB to │ │with typed   │
│ { service..}│ │GraphQL  │ │pricing_raw  │ │Go struct│ │data         │
│}            │ │schema   │ │WHERE ...    │ │Apply    │ │             │
└─────────────┘ └─────────┘ └─────────────┘ └─────────┘ └─────────────┘
                                    ↓           ↓
                            ┌─────────────┐ ┌─────────┐
                            │JSONB        │ │Field    │
                            │operators    │ │selection│
                            │GIN indexes  │ │& limits │
                            └─────────────┘ └─────────┘
```

## 🛠️ Technology Stack

### Backend
- **Language**: Go 1.24+
  - Excellent concurrency with goroutines
  - Strong typing for data integrity
  - Great performance for high-throughput operations

- **Database**: PostgreSQL 13+ with JSONB
  - Flexible schema for raw data storage
  - Advanced indexing (GIN indexes on JSON)
  - ACID compliance for data integrity

- **API**: GraphQL with gqlgen
  - Flexible queries (clients get exactly what they need)
  - Strong typing with schema-first development
  - Interactive playground for development

### Infrastructure
- **Containerization**: Docker + Docker Compose
  - Reproducible deployments
  - Environment isolation
  - Easy local development

- **Monitoring**: Built-in progress tracking
  - Real-time job status updates
  - Performance metrics (records/second)
  - Error tracking and reporting

### External APIs
- **AWS**: Price List Query API
  - Requires AWS credentials
  - Comprehensive service coverage
  - Automatic pagination support

- **Azure**: Retail Pricing API
  - Public API (no authentication)
  - Global region coverage
  - OData filtering support

## 📊 Performance Characteristics

### Scalability

| Component | Current Capacity | Performance | Bottlenecks |
|-----------|------------------|-------------|-------------|
| **Data Collection** | 800K+ records | 100-500 rec/sec | API rate limits |
| **Raw Storage** | Millions of records | Sub-second inserts | Disk I/O |
| **ETL Pipeline** | 1K-2K rec/sec | Configurable workers | CPU & memory |
| **GraphQL API** | <500ms queries | Concurrent requests | Database queries |

### Optimizations

**Database**:
- JSONB with GIN indexes for fast JSON queries
- Bulk insert operations for data collection
- Connection pooling for concurrent access
- Vacuum and analyze for maintenance

**Concurrency**:
- Worker pools for parallel processing
- Configurable batch sizes for memory management
- Rate limiting with exponential backoff
- Circuit breakers for external APIs

**Memory Management**:
- Streaming processing for large datasets
- Configurable batch sizes to control memory usage
- Garbage collection optimization in Go
- Connection pooling to limit resource usage

## 🔒 Security & Reliability

### Data Security
- **No sensitive data storage** - only public pricing information
- **AWS credentials** handled via environment variables
- **Database access** restricted to application containers
- **No user authentication** required (read-only public data)

### Reliability
- **Graceful degradation** when APIs are unavailable
- **Retry logic** with exponential backoff
- **Circuit breakers** to prevent cascade failures
- **Health checks** for all services
- **Complete data preservation** - never lose raw data

### Monitoring
- **Real-time progress tracking** for all operations
- **Error logging** with structured logging
- **Performance metrics** (throughput, latency)
- **Resource monitoring** (CPU, memory, disk)

## 🔄 Extensibility

### Adding New Providers

1. **Create Collector** (`cmd/new-provider-collector/`)
2. **Add Database Tables** (raw storage schema)
3. **Implement Normalizer** (`internal/normalizer/new_provider.go`)
4. **Update GraphQL Schema** (queries and types)
5. **Add Service Mappings** (categorization)

### Adding New Features

1. **Extend GraphQL Schema** (`internal/graph/schema.graphql`)
2. **Implement Resolvers** (`internal/graph/*_resolver.go`)
3. **Add Database Methods** (`internal/database/*.go`)
4. **Write Tests** (`*_test.go`)
5. **Update Documentation**

### Performance Scaling

- **Horizontal scaling**: Stateless Go services
- **Database scaling**: Read replicas, partitioning
- **Caching**: Redis for frequently accessed data
- **Load balancing**: Multiple API server instances
- **Distributed ETL**: Kubernetes job processing

## 📈 Future Architecture

### Microservices Evolution

```
Current Monolith → Target Microservices Architecture

┌─────────────────┐    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│                 │    │   API        │ │   ETL        │ │   Collection │
│   Single Go     │ →  │   Gateway    │ │   Service    │ │   Service    │
│   Application   │    │              │ │              │ │              │
│                 │    └──────────────┘ └──────────────┘ └──────────────┘
└─────────────────┘                           │                  │
                                       ┌──────────────┐ ┌──────────────┐
                                       │   ML/        │ │   Notification│
                                       │   Analytics  │ │   Service     │
                                       │   Service    │ │              │
                                       └──────────────┘ └──────────────┘
```

### Planned Enhancements

- **Machine Learning**: Price prediction and anomaly detection
- **Real-time Streaming**: WebSocket subscriptions for live updates
- **Caching Layer**: Redis for high-frequency queries
- **Event Sourcing**: Complete audit trail of data changes
- **Multi-tenant**: Support for enterprise deployments

---

This architecture provides a solid foundation for comprehensive cloud pricing data management while maintaining flexibility for future enhancements. The design prioritizes **data integrity**, **performance**, and **developer experience**. 🚀