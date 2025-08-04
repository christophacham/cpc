# Development Workflow

This guide covers the complete development workflow for contributing to Cloud Price Compare (CPC).

## Quick Setup

### Prerequisites
- **Docker & Docker Compose** (recommended)
- **Go 1.19+** (for local development)
- **PostgreSQL 13+** (if running without Docker)
- **Git** for version control

### 5-Minute Development Setup
```bash
# 1. Fork and clone the repository
git clone https://github.com/YOUR-USERNAME/cpc
cd cpc

# 2. Start development environment
docker-compose up -d postgres  # Database only for local Go development
# OR
docker-compose up -d           # Full stack including API server

# 3. Install Go dependencies
go mod download

# 4. Run API server locally (if using database-only mode)
go run cmd/server/main.go

# 5. Verify setup
curl http://localhost:8080/query -d '{"query": "{ hello }"}'
```

## Development Modes

### Mode 1: Full Docker Stack
**Best for:** Testing complete system, documentation, frontend work
```bash
# Start everything
docker-compose up -d

# Services available:
# - GraphQL API: http://localhost:8080
# - Documentation: http://localhost:3000
# - Database: localhost:5432

# View logs
docker-compose logs -f app
docker-compose logs -f postgres
```

### Mode 2: Database + Local Go Development
**Best for:** Backend development, debugging, testing
```bash
# Start database only
docker-compose up -d postgres

# Run API server locally for development
go run cmd/server/main.go

# Run tests
go test ./...

# Test ETL pipeline
go run cmd/etl-test/main.go
```

### Mode 3: Local Development (No Docker)
**Best for:** Advanced Go development, performance testing
```bash
# Start PostgreSQL locally (varies by OS)
# macOS: brew services start postgresql
# Ubuntu: sudo systemctl start postgresql

# Set environment variables
export DATABASE_URL="postgres://postgres:password@localhost:5432/cpc?sslmode=disable"

# Run migrations (if needed)
psql -h localhost -U postgres -f schema.sql

# Run server
go run cmd/server/main.go
```

## Common Development Tasks

### Adding New GraphQL Queries

1. **Update GraphQL Schema**
```graphql
# internal/graph/schema.graphql
extend type Query {
    newPricingQuery(filter: PricingFilterInput): [PricingResult!]!
}
```

2. **Implement Resolver**
```go
// internal/graph/query_resolver.go
func (r *queryResolver) NewPricingQuery(ctx context.Context, filter *PricingFilterInput) ([]*PricingResult, error) {
    // Implementation here
    return r.db.GetPricingData(filter)
}
```

3. **Add Database Method**
```go
// internal/database/pricing.go
func (db *DB) GetPricingData(filter *PricingFilterInput) ([]*PricingResult, error) {
    // Database query implementation
}
```

4. **Test in GraphQL Playground**
```bash
# Start server and visit http://localhost:8080
# Test your new query
```

### Adding New Service Mappings

1. **Add to Database**
```sql
INSERT INTO service_mappings (provider, service_name, category, service_type)
VALUES ('aws', 'AmazonNewService', 'Compute & Web', 'Container Service');
```

2. **Update Normalizer**
```go
// internal/normalizer/aws_normalizer.go
func (n *AWSNormalizer) categorizeService(serviceCode string) string {
    switch serviceCode {
    case "AmazonNewService":
        return "Compute & Web"
    // ... existing cases
    }
}
```

3. **Test with ETL Pipeline**
```bash
go run cmd/etl-test/main.go
```

### Improving ETL Performance

1. **Profile Current Performance**
```go
// Add to normalizer
import _ "net/http/pprof"

// Run with profiling
go run cmd/etl-test/main.go &
go tool pprof http://localhost:6060/debug/pprof/profile
```

2. **Optimize Database Operations**
```go
// Use bulk operations instead of individual inserts
func (db *DB) BulkInsertNormalized(records []*NormalizedPricing) error {
    // Batch insert implementation
}
```

3. **Tune Worker Pool Size**
```go
// internal/etl/worker.go
const DefaultWorkerCount = 8  // Adjust based on system resources
```

### Adding New Cloud Provider

1. **Create Raw Data Table**
```sql
CREATE TABLE gcp_pricing_raw (
    id SERIAL PRIMARY KEY,
    service_name TEXT,
    sku_name TEXT,
    region TEXT,
    data JSONB NOT NULL,
    collection_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

2. **Implement Collector**
```go
// cmd/gcp-collector/main.go
type GCPCollector struct {
    client *http.Client
    apiKey string
}

func (c *GCPCollector) CollectPricing() error {
    // GCP Cloud Pricing API implementation
}
```

3. **Create Normalizer**
```go
// internal/normalizer/gcp_normalizer.go
type GCPNormalizer struct {
    db *database.DB
}

func (n *GCPNormalizer) NormalizeRecord(ctx context.Context, rawData []byte) (*NormalizedPricing, error) {
    // GCP-specific normalization logic
}
```

4. **Add GraphQL Support**
```graphql
# internal/graph/schema.graphql
extend type Query {
    gcpPricing(limit: Int, offset: Int): [GCPPricing!]!
    gcpCollections: [GCPCollection!]!
}
```

## Testing Strategy

### Unit Tests
```bash
# Test specific packages
go test ./internal/normalizer/
go test ./internal/database/
go test ./internal/etl/

# Test with coverage
go test -cover ./...

# Test with race detection
go test -race ./...
```

### Integration Tests
```bash
# Test with real database
docker-compose up -d postgres
go test ./internal/database/ -tags=integration

# Test ETL pipeline end-to-end
go run cmd/etl-test/main.go
```

### Manual API Testing
```bash
# Test GraphQL queries
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{ providers { name } }"}'

# Test collection endpoints
curl -X POST http://localhost:8080/populate \
  -H "Content-Type: application/json" \
  -d '{"region": "eastus"}'

# Monitor collection progress
curl -s http://localhost:8080/query \
  -d '{"query": "{ azureCollections { region status progress } }"}' | jq
```

### Load Testing
```bash
# Install hey for load testing
go install github.com/rakyll/hey@latest

# Test API performance
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"query": "{ hello }"}' \
  http://localhost:8080/query
```

## Code Quality Standards

### Go Code Style
```bash
# Format code
gofmt -w .

# Run linter (install golangci-lint first)
golangci-lint run

# Check for common issues
go vet ./...
```

### Git Workflow
```bash
# Create feature branch
git checkout -b feature/new-gcp-support

# Make changes and commit
git add .
git commit -m "feat: add GCP pricing collector

- Implement GCP Cloud Pricing API client
- Add GCP normalizer with service mappings
- Update GraphQL schema for GCP queries
- Add integration tests

Closes #123"

# Push and create PR
git push origin feature/new-gcp-support
```

### Commit Message Format
```
type(scope): brief description

Detailed explanation of what changed and why.

- List specific changes made
- Reference any issues closed
- Include breaking changes if any

Closes #issue-number
```

**Commit Types:**
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding tests
- `perf`: Performance improvements

## Performance Monitoring

### Database Performance
```sql
-- Monitor slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Check index usage
SELECT schemaname, tablename, attname, n_distinct, correlation
FROM pg_stats 
WHERE tablename IN ('aws_pricing_raw', 'azure_pricing_raw');
```

### Application Metrics
```go
// Add metrics to critical paths
import "github.com/prometheus/client_golang/prometheus"

var (
    normalizationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cpc_normalization_duration_seconds",
            Help: "Time spent normalizing records",
        },
        []string{"provider"},
    )
)
```

### System Resources
```bash
# Monitor Docker resource usage
docker stats

# Monitor Go application
go tool pprof http://localhost:8080/debug/pprof/heap
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

## Debugging Tips

### Common Issues

**Database Connection Problems:**
```bash
# Check PostgreSQL status
docker-compose ps postgres
docker-compose logs postgres

# Test connection manually
psql -h localhost -U postgres -d cpc
```

**ETL Job Stuck:**
```sql
-- Check for database locks
SELECT pid, state, query, query_start 
FROM pg_stat_activity 
WHERE state != 'idle';

-- Kill stuck queries if needed
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'active';
```

**Memory Issues:**
```bash
# Enable Go memory profiling
go run cmd/server/main.go -memprofile=mem.prof

# Analyze memory usage
go tool pprof mem.prof
```

### Debugging GraphQL
```go
// Enable debug logging in resolvers
import "log"

func (r *queryResolver) AzurePricing(ctx context.Context, limit *int, offset *int) ([]*model.AzurePricing, error) {
    log.Printf("AzurePricing query: limit=%v, offset=%v", limit, offset)
    // ... implementation
}
```

## Release Process

### Pre-Release Checklist
- [ ] All tests passing: `go test ./...`
- [ ] Documentation updated
- [ ] ETL pipeline tested: `go run cmd/etl-test/main.go`
- [ ] Docker builds successfully: `docker-compose build`
- [ ] Performance regression testing completed
- [ ] Security review (if applicable)

### Version Tagging
```bash
# Create release tag
git tag -a v1.2.0 -m "Release v1.2.0: Add GCP support"
git push origin v1.2.0

# Update changelog
echo "## v1.2.0 - $(date +%Y-%m-%d)" >> CHANGELOG.md
echo "- Add GCP pricing collector" >> CHANGELOG.md
echo "- Improve ETL performance by 50%" >> CHANGELOG.md
```

### Deployment
```bash
# Build production images
docker-compose -f docker-compose.prod.yml build

# Deploy to staging
docker-compose -f docker-compose.staging.yml up -d

# Deploy to production (after validation)
docker-compose -f docker-compose.prod.yml up -d
```

## Contributing Guidelines

### Before You Start
1. **Check existing issues** for similar work
2. **Create an issue** to discuss significant changes
3. **Fork the repository** and create a feature branch
4. **Follow the coding standards** and test requirements

### Pull Request Process
1. **Write clear description** of changes made
2. **Include tests** for new functionality
3. **Update documentation** as needed
4. **Ensure CI passes** (tests, linting, builds)
5. **Request review** from maintainers

### Getting Help
- **Documentation**: Browse the complete docs
- **Issues**: Create issue for bugs or feature requests
- **Discussions**: Ask questions in GitHub discussions
- **Code Review**: Tag maintainers for review feedback

This workflow ensures high-quality contributions while maintaining the project's technical excellence and community standards.