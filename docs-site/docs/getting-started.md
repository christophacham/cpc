#  Getting Started with Cloud Price Compare

Welcome to **Cloud Price Compare (CPC)** - the world's most comprehensive **open-source** cloud pricing platform! This guide will help you get up and running quickly.

##  What is CPC?

Cloud Price Compare extracts, normalizes, and serves **ALL** pricing data from major cloud providers through a modern GraphQL API. Whether you're building cost optimization tools, doing research, or just curious about cloud pricing, CPC gives you complete access to:

- ** 800,000+ pricing records** (500K AWS + 300K Azure)
- ** Real-time ETL pipeline** for cross-provider comparisons
- ** Developer-friendly** GraphQL API with interactive playground
- ** One-command deployment** with Docker

##  Quick Start (2 Minutes)

### Prerequisites

- **Docker & Docker Compose** (easiest setup)
- **5GB+ disk space** (for complete datasets)
- **AWS credentials** (optional, for AWS data collection)

### Setup

```bash
# 1. Clone the repository
git clone https://github.com/your-org/cpc
cd cpc

# 2. Start the entire stack
docker-compose up -d

# 3. That's it! 🎉
# GraphQL API: http://localhost:8080
# Documentation: http://localhost:3000
# Database: localhost:5432 (postgres/password)
```

### Verify Everything Works

```bash
# Test the API
curl http://localhost:8080/query -d '{"query": "{ hello }"}'

# Expected response:
# {"data":{"hello":"Welcome to Cloud Price Compare API!"}}
```

##  Explore the Data

### GraphQL Playground

Visit **http://localhost:8080** to access the interactive GraphQL playground. Try these example queries:

```graphql
# Get system overview
query {
  hello
  providers { name }
  categories { name description }
}

# Check available Azure regions
query {
  azureRegions {
    region
    totalItems
    hasData
  }
}

# View raw pricing data (if collected)
query {
  azurePricing(limit: 5) {
    serviceName
    productName
    skuName
    retailPrice
    unitOfMeasure
    armRegionName
  }
}
```

##  Collect Your First Data

### Azure Data (No Credentials Needed)

Azure data collection requires no authentication:

```bash
# Collect single region (fast, ~5,000 records)
curl -X POST http://localhost:8080/populate \
  -H "Content-Type: application/json" \
  -d '{"region": "eastus"}'

# Monitor progress
curl -s http://localhost:8080/query \
  -d '{"query": "{ azureCollections { region status totalItems progress } }"}' | jq
```

### AWS Data (Requires Credentials)

For AWS data collection, add your credentials:

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add:
# AWS_ACCESS_KEY_ID=your_access_key
# AWS_SECRET_ACCESS_KEY=your_secret_key
# AWS_REGION=us-east-1

# Restart to pick up credentials
docker-compose restart app

# Collect comprehensive AWS data (~200K records)
curl -X POST http://localhost:8080/aws-populate-comprehensive
```

##  Normalize Data for Comparisons

Once you have raw data, use the ETL pipeline to normalize it for cross-provider comparisons:

```graphql
# Start normalization job
mutation {
  startNormalization(config: {
    type: NORMALIZE_ALL
    clearExisting: true
    batchSize: 1000
    concurrentWorkers: 4
  }) {
    id
    status
    progress {
      totalRecords
      currentStage
    }
  }
}

# Monitor progress (use the job ID from above)
query {
  etlJob(id: "your-job-id") {
    status
    progress {
      processedRecords
      normalizedRecords
      skippedRecords
      rate
      currentStage
    }
  }
}
```

##  Real-Time Monitoring

Monitor collection progress in real-time:

```bash
# Watch Azure collection progress
watch -n 2 'curl -s http://localhost:8080/query -d "{\"query\":\"{ azureCollections { region status totalItems progress } }\"}" | jq'

# Watch ETL job progress
watch -n 2 'curl -s http://localhost:8080/query -d "{\"query\":\"{ etlJobs { id status progress { processedRecords rate currentStage } } }\"}" | jq'
```

##  Development Setup

For local development without Docker:

```bash
# Start database only
docker-compose up -d postgres

# Install Go dependencies
go mod download

# Run the API server locally
go run cmd/server/main.go

# Test ETL pipeline
go run cmd/etl-test/main.go

# Run all tests
go test ./...
```

##  What's Next?

###  Explore the API
- **[API Reference](api-reference/overview.md)** - Complete GraphQL documentation
- **[ETL Pipeline Guide](api-reference/etl-pipeline.md)** - Data normalization details

###  Understand the Architecture
- **[Architecture Overview](architecture/overview.md)** - Technical deep-dive
- **[Database Schema](architecture/database.md)** - Data storage design

###  Contribute
- **[Contributing Guide](../CONTRIBUTING.md)** - How to contribute
- **[Development Workflow](development/workflow.md)** - Detailed development guide

###  Use Cases
- **Cost optimization** - Find cheapest services across providers
- **Research projects** - Analyze cloud pricing patterns
- **API integration** - Build your own cost analysis tools
- **Data science** - Machine learning on pricing data

## 🆘 Need Help?

### Common Issues

**Database connection failed:**
```bash
# Check if PostgreSQL is running
docker-compose ps postgres
docker-compose logs postgres
```

**API server won't start:**
```bash
# Check logs
docker-compose logs app

# Restart the service
docker-compose restart app
```

**ETL job stuck:**
```bash
# Check database activity
psql -h localhost -U postgres -d cpc \
  -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"
```

### Get Support

- **📖 Documentation**: Browse the complete docs at http://localhost:3000
- ** Bug Reports**: [Create an issue](https://github.com/your-org/cpc/issues)
- ** Feature Requests**: [Start a discussion](https://github.com/your-org/cpc/discussions)
- **❓ Questions**: Check the [troubleshooting guide](troubleshooting.md)

##  What Makes CPC Special?

### Complete Data Coverage
Unlike other tools that sample or filter data, CPC extracts **everything**:
- Every service from AWS and Azure
- Every pricing model (on-demand, reserved, spot)
- Every region globally
- Every instance type and configuration

### Developer-First Design
- **GraphQL API** with interactive playground
- **Docker deployment** that just works
- **Comprehensive documentation** with examples
- **Open-source** and community-driven

### Production-Ready Scale
- **Proven performance** with 800,000+ records
- **Concurrent processing** with configurable workers
- **Real-time monitoring** of all operations
- **Zero data loss** with raw data preservation

---

**Ready to dive deeper?** Check out the [API Reference](api-reference/overview.md) or start [contributing](../CONTRIBUTING.md) to the project! 