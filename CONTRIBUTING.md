# 🤝 Contributing to Cloud Price Compare

Welcome! We're thrilled you want to contribute to CPC. This guide will help you get started quickly and make meaningful contributions.

## 🚀 Quick Start for Contributors

### 🛠️ Development Setup

1. **Fork & Clone**
   ```bash
   git fork https://github.com/your-org/cpc
   git clone https://github.com/YOUR-USERNAME/cpc
   cd cpc
   ```

2. **Start Development Environment**
   ```bash
   # Start database only (for local development)
   docker-compose up -d postgres
   
   # Install Go dependencies
   go mod download
   
   # Run the server locally
   go run cmd/server/main.go
   ```

3. **Verify Setup**
   ```bash
   # Test the API
   curl http://localhost:8080/query -d '{"query": "{ hello }"}'
   
   # Test ETL pipeline
   go run cmd/etl-test/main.go
   
   # Run all tests
   go test ./...
   ```

## 🎯 Ways to Contribute

### 🐛 Found a Bug?
1. Check [existing issues](https://github.com/your-org/cpc/issues)
2. Create a new issue with:
   - Clear description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Your environment details

### 💡 Have a Feature Idea?
1. Check [discussions](https://github.com/your-org/cpc/discussions)
2. Start a new discussion to gather feedback
3. Create an issue if the community agrees

### 📚 Improve Documentation?
- Fix typos or unclear explanations
- Add missing examples
- Improve API documentation
- Update setup instructions

### 🧪 Add Tests?
We especially welcome:
- Unit tests for normalizers
- Integration tests for data collection
- ETL pipeline tests
- GraphQL resolver tests

## 🏗️ Development Guidelines

### 📁 Project Structure Overview

```
cpc/
├── 🛠️  cmd/                    # Executable commands
│   ├── server/             # Main GraphQL API server
│   ├── etl-test/           # ETL pipeline testing
│   └── collectors/         # Data collection utilities
├── 📊  internal/               # Core application logic
│   ├── database/           # Database operations & models
│   ├── etl/                # ETL pipeline for normalization
│   ├── graph/              # GraphQL resolvers & schema
│   └── normalizer/         # Data normalization logic
├── 📝  docs-site/              # Documentation website
├── 🐳  docker-compose.yml      # Complete stack deployment
└── 📋  init.sql               # Database schema
```

### 🎨 Code Style Guidelines

**Go Code Standards:**
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names
- Add comments for exported functions
- Keep functions small and focused
- Handle errors explicitly

**Example:**
```go
// ✅ Good
func NormalizePrice(rawPrice string, currency string) (float64, error) {
    if rawPrice == "" {
        return 0, fmt.Errorf("raw price cannot be empty")
    }
    
    price, err := strconv.ParseFloat(rawPrice, 64)
    if err != nil {
        return 0, fmt.Errorf("failed to parse price %q: %w", rawPrice, err)
    }
    
    return price, nil
}

// ❌ Avoid
func np(p string, c string) float64 {
    price, _ := strconv.ParseFloat(p, 64) // Never ignore errors
    return price
}
```

### 🧪 Testing Guidelines

**Writing Good Tests:**
- Test both success and error cases
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Keep tests fast and isolated

**Example:**
```go
func TestNormalizePrice(t *testing.T) {
    tests := []struct {
        name        string
        rawPrice    string
        currency    string
        expected    float64
        expectError bool
    }{
        {
            name:     "valid price",
            rawPrice: "1.23",
            currency: "USD",
            expected: 1.23,
        },
        {
            name:        "empty price",
            rawPrice:    "",
            currency:    "USD",
            expectError: true,
        },
        {
            name:        "invalid price format",
            rawPrice:    "not-a-number",
            currency:    "USD",
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NormalizePrice(tt.rawPrice, tt.currency)
            
            if tt.expectError {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 📊 Database Guidelines

**Working with Raw Data:**
- Always preserve original JSON data
- Use JSONB for flexible querying
- Add proper indexes for common queries
- Handle migration scripts carefully

**Example:**
```sql
-- ✅ Good: Preserve original data
CREATE TABLE aws_pricing_raw (
    id SERIAL PRIMARY KEY,
    service_code TEXT NOT NULL,
    region TEXT NOT NULL,
    data JSONB NOT NULL,  -- Original AWS response
    created_at TIMESTAMP DEFAULT NOW()
);

-- Add useful indexes
CREATE INDEX idx_aws_pricing_service_region 
ON aws_pricing_raw(service_code, region);

CREATE INDEX idx_aws_pricing_data_gin 
ON aws_pricing_raw USING GIN(data);
```

## 🔄 Development Workflow

### 🌟 Creating a Pull Request

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write code
   - Add tests
   - Update documentation
   - Test locally

3. **Commit with clear messages**
   ```bash
   git commit -m "feat: add Azure VM size normalization
   
   - Extract CPU and memory from ARM SKU names
   - Handle D, F, E, and B series VMs
   - Add comprehensive tests for all VM types"
   ```

4. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   # Create PR via GitHub UI
   ```

### ✅ PR Checklist

Before submitting your PR, ensure:

- [ ] **Tests pass**: `go test ./...`
- [ ] **Code is formatted**: `gofmt -w .`
- [ ] **No linting errors**: `golangci-lint run`
- [ ] **Documentation updated** (if needed)
- [ ] **ETL tests pass**: `go run cmd/etl-test/main.go`
- [ ] **Clear commit messages**
- [ ] **PR description explains the change**

### 🎯 Common Contribution Areas

#### 🌐 Adding New Cloud Providers

1. **Create normalizer**:
   ```go
   // internal/normalizer/gcp_normalizer.go
   type GCPNormalizer struct {
       // implement NormalizerInterface
   }
   ```

2. **Add data collection**:
   ```go
   // cmd/gcp-collector/main.go
   // Implement GCP pricing API client
   ```

3. **Update GraphQL schema**:
   ```graphql
   # internal/graph/schema.graphql
   extend type Query {
       gcpPricing: [GCPPricing!]!
   }
   ```

#### 🔧 Improving Data Normalization

Focus areas:
- **Service mapping accuracy**
- **Resource specification extraction**
- **Pricing model standardization**
- **Cross-provider comparison logic**

#### 📊 Enhancing ETL Pipeline

Ideas for improvement:
- **Performance optimization**
- **Better error handling**
- **Progress reporting enhancements**
- **Incremental processing**

#### 🎨 GraphQL API Enhancements

Common requests:
- **New query types**
- **Better filtering options**
- **Aggregation queries**
- **Performance optimizations**

## 🧪 Testing Your Changes

### 🔍 Local Testing

```bash
# Unit tests
go test ./internal/normalizer/
go test ./internal/etl/

# Integration tests
go test ./internal/database/
go test ./internal/graph/

# ETL pipeline test
go run cmd/etl-test/main.go

# Manual API testing
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{ hello }"}'
```

### 🐳 Docker Testing

```bash
# Test full stack
docker-compose up -d
docker-compose logs app

# Test specific services
docker-compose up -d postgres
go run cmd/server/main.go
```

### 📊 Performance Testing

```bash
# Load test collection endpoints
go run cmd/azure-collector/main.go eastus

# Monitor resource usage
docker stats

# Test ETL performance
time go run cmd/etl-test/main.go
```

## 🎯 Specific Contribution Guides

### 🤖 Adding New Normalizers

1. **Implement the interface**:
   ```go
   type YourNormalizer struct {
       serviceMappingRepo ServiceMappingRepository
       regionMappingRepo  RegionMappingRepository
       unitNormalizer     UnitNormalizer
       logger            Logger
   }

   func (n *YourNormalizer) NormalizeRecord(ctx context.Context, rawData []byte) (*NormalizedPricing, error) {
       // Your implementation
   }
   ```

2. **Add comprehensive tests**
3. **Update ETL pipeline**
4. **Add GraphQL resolvers**

### 📊 Improving Database Schema

1. **Create migration script**:
   ```sql
   -- migrations/001_add_your_feature.sql
   ALTER TABLE pricing_records ADD COLUMN new_field TEXT;
   CREATE INDEX idx_new_field ON pricing_records(new_field);
   ```

2. **Update Go models**
3. **Test with existing data**

### 🎨 Enhancing GraphQL API

1. **Update schema**:
   ```graphql
   extend type Query {
       newQuery(filter: FilterInput): [Result!]!
   }
   ```

2. **Implement resolver**:
   ```go
   func (r *queryResolver) NewQuery(ctx context.Context, filter *FilterInput) ([]*Result, error) {
       // Implementation
   }
   ```

3. **Add tests and documentation**

## 🚨 Common Gotchas

### ⚠️ Things to Avoid

- **Don't ignore errors** - Always handle them explicitly
- **Don't break existing APIs** - Maintain backward compatibility
- **Don't commit secrets** - Use environment variables
- **Don't skip tests** - They catch regressions
- **Don't modify raw data** - Always preserve original responses

### 💡 Pro Tips

- **Use contexts** for cancellation and timeouts
- **Log structured data** for better debugging
- **Cache expensive operations** where appropriate
- **Use bulk operations** for better database performance
- **Follow Go idioms** and standard library patterns

## 🆘 Getting Help

### 💬 Communication Channels

- **GitHub Issues** - Bug reports and feature requests
- **GitHub Discussions** - Questions and ideas
- **Code Reviews** - PR feedback and learning

### 📚 Learning Resources

- **[Go Documentation](https://golang.org/doc/)** - Official Go docs
- **[GraphQL Guide](https://graphql.org/learn/)** - GraphQL fundamentals
- **[PostgreSQL JSONB](https://www.postgresql.org/docs/current/datatype-json.html)** - JSONB operations
- **[Docker Compose](https://docs.docker.com/compose/)** - Container orchestration

### 🔍 Code Examples

Check these files for reference implementations:
- **Normalizers**: `internal/normalizer/`
- **ETL Pipeline**: `internal/etl/`
- **GraphQL Resolvers**: `internal/graph/`
- **Database Operations**: `internal/database/`

## 🎉 Recognition

Contributors are the heart of this project! All contributors will be:
- **Listed in CONTRIBUTORS.md**
- **Mentioned in release notes** for significant contributions
- **Invited to special contributor discussions**

## 📜 Code of Conduct

We follow the [Contributor Covenant](https://www.contributor-covenant.org/). Be kind, respectful, and inclusive. We're all here to learn and build something amazing together.

---

**Ready to contribute?** Start by exploring the codebase, pick an issue, or suggest an improvement. Every contribution, no matter how small, makes CPC better for everyone! 🚀

**Questions?** Don't hesitate to ask in [GitHub Discussions](https://github.com/your-org/cpc/discussions) - we're here to help! 💪