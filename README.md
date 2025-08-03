# 🌩️ Cloud Price Compare (CPC)

**The world's most comprehensive cloud pricing data platform - built for developers, by developers.**

> Extract, normalize, and compare pricing data from AWS and Azure. No limits, no samples - everything.

## 🚀 What is CPC?

Cloud Price Compare is an **open-source platform** that extracts **ALL** pricing data from major cloud providers and makes it queryable through a modern GraphQL API. Whether you're building cost optimization tools, doing research, or just curious about cloud pricing, CPC gives you complete access to:

- **📊 800,000+ pricing records** (500K AWS + 300K Azure)
- **🔄 Real-time data collection** with progress monitoring
- **🎯 Unified normalization** for cross-provider comparisons
- **⚡ Production-grade performance** with concurrent processing
- **🛠️ Developer-friendly** GraphQL API and Docker deployment

## ✨ Why Choose CPC?

### 🎯 **Complete Data Coverage**
Unlike other tools that sample or filter data, CPC extracts **everything**:

| Provider | Services | Records | Regions | Coverage |
|----------|----------|---------|---------|----------|
| **AWS** | 60+ services | 500,000+ records | All major regions | Every pricing model |
| **Azure** | All services | 300,000+ records | 70+ regions | Global coverage |

### 🏗️ **Developer-First Architecture**
- **🐳 One-Command Deploy**: `docker-compose up -d`
- **📊 GraphQL API**: Modern, flexible queries
- **🔄 ETL Pipeline**: Normalize data for comparisons
- **📝 Comprehensive Docs**: Everything you need to contribute
- **🧪 Test Coverage**: Reliable and maintainable

### 📈 **Production-Proven Performance**
- ✅ **40,000+ EC2 records** collected and verified
- ✅ **Concurrent processing** with configurable workers
- ✅ **Real-time monitoring** with progress tracking
- ✅ **Zero data loss** with raw JSON preservation

## 🚀 Getting Started

### 📋 Prerequisites

- **Docker & Docker Compose** (easiest setup)
- **Go 1.24+** (for local development)
- **AWS credentials** (optional, for AWS data collection)
- **5-10GB disk space** (for complete datasets)

### ⚡ 2-Minute Setup

```bash
# 1. Clone the repository
git clone <repository-url>
cd cpc

# 2. Start the entire stack
docker-compose up -d

# 3. That's it! 🎉
# GraphQL Playground: http://localhost:8080
# Documentation: http://localhost:3000
# Database: localhost:5432 (postgres/password)
```

## 🔌 Port Configuration & Funky Integration

### **CPC Port Assignments**

| Service | Port | Purpose | Integration |
|---------|------|---------|-------------|
| **GraphQL API** | 8080 | Main CPC backend API | ← Funky connects here |
| **Documentation** | 3000 | Docusaurus docs site | Independent |
| **PostgreSQL** | 5432 | Pricing database | ← Funky pricing API connects |

### **Funky Integration Points**

```
┌─────────────────┐    ┌─────────────────┐
│   Funky App     │    │   CPC Backend   │
│                 │    │                 │
│ :8501/8502/8503 │    │ GraphQL API     │
│ (Streamlit)     │◄──►│ :8080          │
│                 │    │                 │
│ Pricing API     │    │ PostgreSQL      │
│ :8082           │◄──►│ :5432          │
└─────────────────┘    └─────────────────┘
```

**✅ Clean Separation**: CPC uses 8080, 3000, 5432 / Funky uses 8501-8503, 8082, 8888

### 🔑 AWS Setup (Optional)

For AWS data collection, add your credentials:

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add:
# AWS_ACCESS_KEY_ID=your_access_key
# AWS_SECRET_ACCESS_KEY=your_secret_key
# AWS_REGION=us-east-1
```

> 💡 **Pro Tip**: You can start with Azure data (no credentials needed) and add AWS later!

## Comprehensive Data Collection

### **AWS - Complete Extraction (Recommended)**
```bash
# Major Services (80/20 rule) - ~200K pricing records
curl -X POST http://localhost:8080/aws-populate-comprehensive

# EVERYTHING (All 60+ services) - ~500K+ pricing records  
curl -X POST http://localhost:8080/aws-populate-everything
```

### **Azure - All Regions**
```bash
# Single region (fast testing)
curl -X POST http://localhost:8080/populate -d '{"region": "eastus"}'

# All 70+ regions (complete dataset)
curl -X POST http://localhost:8080/populate-all -d '{"concurrency": 5}'
```

## Data Analysis & Querying

### **Real-Time Collection Monitoring**
```graphql
{
  # Monitor AWS comprehensive collection
  awsCollections {
    collectionId
    serviceCodes    # ["AmazonEC2", "AmazonS3", "AmazonRDS"...]
    regions         # ["us-east-1", "eu-west-1", "ap-southeast-1"...]
    status          # "running", "completed", "failed"
    totalItems      # Real count: 40,000+ for EC2 alone
    startedAt
    completedAt
    duration
  }
  
  # Monitor Azure regional collection
  azureCollections {
    region
    status
    totalItems
    progress
  }
}
```

### **Query Collected Pricing Data**
```graphql
{
  # System overview
  hello
  providers { name }
  categories { name description }
  
  # Raw AWS pricing data (hundreds of thousands of records)
  awsPricing {
    serviceCode
    serviceName
    location
    instanceType
    pricePerUnit
    unit
    currency
    termType
  }
  
  # Raw Azure pricing data
  azurePricing {
    serviceName
    productName
    skuName
    retailPrice
    unitOfMeasure
    armRegionName
  }
}
```

## Production Endpoints

### **AWS - Comprehensive Collection**
```bash
# RECOMMENDED: Major services (14 services, 4 regions)
# Expected: ~200,000 pricing records, ~30 minutes
curl -X POST http://localhost:8080/aws-populate-comprehensive

# MAXIMUM: Everything (60+ services, 3 regions) 
# Expected: ~500,000+ pricing records, ~2-6 hours
curl -X POST http://localhost:8080/aws-populate-everything

# Custom: Specific services and regions
curl -X POST http://localhost:8080/aws-populate \
  -H "Content-Type: application/json" \
  -d '{
    "serviceCodes": ["AmazonEC2", "AmazonRDS", "AmazonS3"],
    "regions": ["us-east-1", "eu-west-1", "ap-southeast-1"]
  }'
```

### **Azure - Regional Extraction**
```bash
# Single region (testing/development)
curl -X POST http://localhost:8080/populate \
  -H "Content-Type: application/json" \
  -d '{"region": "eastus"}'

# Complete dataset (all 70+ regions, ~300K records)
curl -X POST http://localhost:8080/populate-all \
  -H "Content-Type: application/json" \
  -d '{"concurrency": 5}'
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

## Complete API Reference

### **GraphQL API** (`http://localhost:8080/query`)
**Data Queries:**
- `awsPricing` - Raw AWS pricing data (500K+ records when populated)
- `azurePricing` - Raw Azure pricing data (300K+ records when populated)
- `providers` - Cloud providers (AWS, Azure)
- `categories` - Service categories (13 standardized types)

**Collection Monitoring:**
- `awsCollections` - Real-time AWS collection progress
- `azureCollections` - Real-time Azure collection progress
- `azureRegions` - Azure regions with collected data
- `azureServices` - Azure services with collected data

### **Production Collection Endpoints**
**AWS Comprehensive (NEW):**
- `POST /aws-populate-comprehensive` - Major services (14 services, recommended)
- `POST /aws-populate-everything` - ALL services (60+ services, maximum extraction)
- `POST /aws-populate` - Custom services/regions
- `POST /aws-populate-all` - Multi-region concurrent collection

**Azure Complete:**
- `POST /populate` - Single Azure region
- `POST /populate-all` - All 70+ Azure regions concurrently

### **Interactive Web Interface** (`http://localhost:8080`)
- **GraphQL Playground** with comprehensive query examples
- **One-click collection buttons** for major regions and services
- **Real-time progress monitoring** with auto-refresh every 10 seconds
- **Pre-built query templates** for common use cases

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

## Enterprise Architecture

### **Dual-Provider Raw JSON Storage**
- **aws_pricing_raw** - Complete AWS pricing data with full attribute preservation
- **azure_pricing_raw** - Complete Azure pricing data with regional metadata
- **Collection tracking** - Real-time progress monitoring for both providers
- **JSONB indexing** - High-performance querying on massive datasets
- **No data loss** - Preserve all vendor-specific metadata for future analysis

### **Production-Grade Features**
- **Concurrent processing** - Multiple services/regions collected simultaneously
- **Automatic pagination** - Handle 100+ pages per service seamlessly  
- **Error resilience** - Comprehensive retry logic and graceful degradation
- **Progress tracking** - Real-time status updates with detailed metrics
- **Container orchestration** - Complete Docker stack ready for any environment

## Real-World Data Scale

### **AWS - Comprehensive Coverage**
- **✅ 40,000+ EC2 pricing items** (proven in production)
- **✅ 16,000+ RDS pricing items** (proven in production)
- **Expected: 500,000+ total items** from complete collection
- **60+ services supported**: Compute, Storage, Database, AI/ML, Analytics, Networking

### **Azure - Global Coverage**  
- **70+ regions supported** worldwide
- **~5,000 pricing items per region** average
- **Expected: 300,000+ total items** from complete collection
- **All service families**: Compute, Storage, Database, AI, Analytics

### **Performance Benchmarks**
- **Concurrent collection**: 3-5 workers optimal
- **Collection speed**: ~100 items/second sustained
- **Time to complete**: 30-60 minutes for comprehensive datasets
- **Storage efficiency**: Raw JSON with JSONB compression
