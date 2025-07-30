# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Cloud Price Compare (cpc) project.

## Project Overview

Cloud Price Compare is a production-grade API service that aggregates, normalizes, and serves pricing data from AWS and Azure through a unified GraphQL API. The service provides both raw provider-specific pricing data and standardized formats for easy comparison across cloud providers.

## Project Goals

- Build a centralized pricing database with ALL services from AWS and Azure
- Support ALL pricing models (on-demand, reserved, spot, savings plans)
- Cover ALL regions for both providers
- Provide GraphQL API for flexible querying
- Maintain both raw and standardized data formats
- Enable monthly updates with manual trigger capability
- Host on AWS with managed services (no authentication required)

## Service Categories

All cloud services must be mapped to these standardized categories:

- **General** - Core infrastructure and foundational services
- **Networking** - Network connectivity, load balancing, CDN
- **Compute & Web** - Virtual machines, containers, serverless compute
- **Containers** - Container orchestration and management
- **Databases** - Relational, NoSQL, and specialized databases
- **Storage** - Object storage, file systems, backup solutions
- **AI & ML** - Machine learning, cognitive services, AI tools
- **Analytics & IoT** - Data analytics, streaming, IoT platforms
- **Virtual Desktop** - Desktop virtualization and workspace solutions
- **Dev Tools** - Development, CI/CD, and testing tools
- **Integration** - API management, messaging, event services
- **Migration** - Data migration and transfer services
- **Management** - Monitoring, governance, security tools

## Architecture Guidelines

### Data Collection Strategy

**AWS Pricing Collection:**
- Use AWS Price List Query API for real-time pricing
- Handle ALL pricing models (on-demand, reserved, spot, savings plans)
- Collect data for 200+ AWS services across all regions
- Implement exponential backoff for API rate limits
- Consider bulk API for large services like EC2, RDS, S3

**Azure Pricing Collection:**
- Use Azure Retail Pricing API (no authentication required)
- Collect all pricing tiers (consumption, reservation, spot)
- Query all Azure services across all regions
- Handle pagination with NextPageLink
- API is faster than AWS, can update more frequently

### Database Design Principles

**Core Tables:**
- `pricing_records` - Standardized pricing data
- `raw_pricing_data` - Original API responses
- `service_mappings` - Provider service to category mappings
- `regions` - Region metadata and mappings
- `collection_versions` - Track data collection runs

**Key Design Decisions:**
- Store both raw and normalized data
- Use PostgreSQL with JSONB for flexibility
- Implement versioning for historical tracking
- Create indexes for common query patterns
- Consider partitioning for large datasets

### GraphQL API Design

**Query Capabilities:**
- Query by provider, service, region, category
- Compare equivalent services across providers
- Filter by pricing model and specifications
- Support both raw and standardized responses
- Enable complex cross-provider comparisons

**Schema Approach:**
- Flexible attribute system for provider-specific fields
- Standardized fields for common properties
- Support nested queries for detailed pricing
- Include metadata about data freshness

### Infrastructure Requirements

**AWS Services to Use:**
- **API**: ECS Fargate or Lambda (consider request patterns)
- **Database**: RDS PostgreSQL or Aurora Serverless
- **Scheduling**: EventBridge for monthly updates
- **Storage**: S3 for backup and raw data archives
- **Monitoring**: CloudWatch for logs and metrics

**Performance Targets:**
- Sub-second query response times
- Handle concurrent data collection jobs
- Support future growth without major refactoring

## Implementation Priorities

1. **Data Models First** - Design database schema and GraphQL types
2. **Collection Scripts** - Build robust data collectors for both providers
3. **Normalization Logic** - Create service mapping and categorization
4. **API Layer** - Implement GraphQL with efficient resolvers
5. **Infrastructure** - Deploy on AWS with automation
6. **Update Pipeline** - Set up monthly collection workflow

## Technical Decisions

**Language**: Go (specified requirement)
- Use for API server and data collectors
- Leverage concurrency for parallel processing
- Strong typing for data integrity

**GraphQL Framework**: Consider performance-focused options
- Focus on query efficiency over features
- Implement DataLoader pattern for batching
- Add query complexity limits

**Data Collection**:
- Separate collectors for each provider
- Store raw responses for debugging
- Implement comprehensive error handling
- Log all collection metrics

## Key Challenges to Address

1. **Service Mapping** - Many services don't fit neatly into one category
2. **Regional Variations** - Pricing differs significantly by region
3. **API Rate Limits** - Especially challenging with AWS
4. **Data Volume** - Millions of pricing records to manage
5. **Schema Evolution** - Providers add new services regularly

## Development Workflow

1. Start with data collection scripts to understand API responses
2. Design database schema based on actual data structure
3. Build normalization layer with service mappings
4. Implement GraphQL API with basic queries
5. Add infrastructure as code for deployment
6. Create update automation with proper monitoring

## Success Criteria

- Complete pricing data for all AWS and Azure services
- Accurate service categorization with confidence scores
- Fast query performance for common use cases
- Reliable monthly updates without manual intervention
- Clear documentation for API consumers

## Anti-Patterns to Avoid

- Don't over-normalize the database schema
- Don't cache aggressively (data updates monthly)
- Don't build complex authentication (not required)
- Don't create a UI (API only)
- Don't ignore rate limits or API costs

## Future Considerations

- Service equivalency mappings between providers
- Cost calculation endpoints
- Historical pricing trends
- Pricing change notifications
- Multi-currency support

## API Implementation Details (Go)

### AWS Price List Query API Integration

**Key Implementation Points from Python Reference:**

```go
// AWS Pricing Client Configuration
type AWSPricingClient struct {
    client        *pricing.Client
    region        string
    cacheFile     string
    cacheDuration time.Duration // 6 hours default
}

// Authentication: Uses standard AWS SDK credential chain
// - IAM roles, environment variables, shared credentials
// - Pricing API only works in us-east-1 region
// - No special permissions needed beyond pricing:DescribeServices, pricing:GetProducts

// Region to Location mapping for API queries
var AWSRegionLocationMap = map[string]string{
    "us-east-1": "US East (N. Virginia)",
    "us-west-2": "US West (Oregon)",
    "eu-west-1": "Europe (Ireland)",
    // ... complete mapping needed
}

// API Rate Limiting
// - Implement exponential backoff
// - Max 10 retries with adaptive mode
// - Handle throttling gracefully
```

**Essential API Queries:**

1. **EC2 Pricing Query:**
```go
// Filters for EC2 pricing
filters := []types.Filter{
    {Type: "TERM_MATCH", Field: "instanceType", Value: "t3.medium"},
    {Type: "TERM_MATCH", Field: "location", Value: locationName},
    {Type: "TERM_MATCH", Field: "tenancy", Value: "Shared"},
    {Type: "TERM_MATCH", Field: "operatingSystem", Value: "Linux"},
}
```

2. **S3 Storage Pricing:**
```go
// Storage class mapping
storageClasses := map[string]string{
    "General Purpose": "standard",
    "Infrequent Access": "standard_ia",
    "Glacier Instant Retrieval": "glacier",
}
```

3. **Data Transfer Pricing:**
```go
// Query for AWS Outbound data transfer
filters := []types.Filter{
    {Type: "TERM_MATCH", Field: "transferType", Value: "AWS Outbound"},
}
```

### Azure Retail Pricing API Integration

**Key Implementation Points:**

```go
// Azure Pricing Client - No authentication required!
type AzurePricingClient struct {
    baseURL       string // "https://prices.azure.com/api/retail/prices"
    httpClient    *http.Client
    location      string
    currency      string
    cacheDuration time.Duration // 6 hours default
}

// No authentication needed - public API
// Supports filtering and pagination
// Much faster than AWS API - can update more frequently

// API Query Parameters
type AzureAPIParams struct {
    CurrencyCode string
    Filter       string // OData filter syntax
    Top          int    // Results per page
}
```

**Essential Filters:**

1. **Virtual Machines:**
```go
filter := fmt.Sprintf(
    "serviceName eq 'Virtual Machines' and "+
    "armRegionName eq '%s' and "+
    "armSkuName eq '%s' and "+
    "priceType eq 'Consumption'",
    region, vmSize
)
```

2. **Storage:**
```go
filter := fmt.Sprintf(
    "serviceName eq 'Storage' and "+
    "productName contains '%s' and "+
    "meterName contains 'Data Stored'",
    storageType
)
```

### Data Structures and Normalization

**Unified Pricing Structure:**

```go
type UnifiedPricing struct {
    // Provider info
    Provider     string
    ServiceName  string
    Category     string
    Region       string
    
    // Pricing details
    PricePerUnit float64
    Unit         string
    Currency     string
    PricingModel string // "on_demand", "reserved_1yr", etc.
    
    // Resource specs (optional)
    CPUCores     int
    MemoryGB     float64
    StorageGB    float64
    
    // Metadata
    LastUpdated  time.Time
    EffectiveDate time.Time
    RawData      json.RawMessage
}
```

### Caching Strategy

```go
type CacheManager struct {
    cacheDir      string
    defaultTTL    time.Duration
}

// Cache file naming convention
// aws_pricing_cache_us-east-1.json
// azure_pricing_cache_eastus_usd.json

// Cache validation
func (c *CacheManager) IsValid(cacheFile string) bool {
    // Check file exists
    // Check timestamp < TTL
    // Validate JSON structure
}
```

### Error Handling Patterns

```go
// Custom error types for better handling
type PricingAPIError struct {
    Provider string
    Message  string
    Retry    bool
}

// Retry logic with exponential backoff
func RetryWithBackoff(fn func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        
        // Check if error is retryable
        if !IsRetryable(err) {
            return err
        }
        
        // Exponential backoff: 2^i seconds
        time.Sleep(time.Second * time.Duration(math.Pow(2, float64(i))))
    }
    return fmt.Errorf("max retries exceeded")
}
```

### Concurrent Data Collection

```go
// Collect pricing from multiple regions concurrently
func CollectAllRegions(regions []string) ([]UnifiedPricing, error) {
    var wg sync.WaitGroup
    results := make(chan RegionPricing, len(regions))
    errors := make(chan error, len(regions))
    
    // Use semaphore to limit concurrent API calls
    sem := make(chan struct{}, 10) // Max 10 concurrent
    
    for _, region := range regions {
        wg.Add(1)
        go func(r string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            
            pricing, err := CollectRegionPricing(r)
            if err != nil {
                errors <- err
                return
            }
            results <- pricing
        }(region)
    }
    
    wg.Wait()
    close(results)
    close(errors)
    
    // Aggregate results and handle errors
}
```

### Service Mapping Logic

```go
// Map provider-specific services to categories
var AWSServiceCategoryMap = map[string]string{
    "AmazonEC2": "Compute & Web",
    "AmazonS3": "Storage",
    "AmazonRDS": "Databases",
    "AWSLambda": "Compute & Web",
    // ... complete mapping
}

var AzureServiceCategoryMap = map[string]string{
    "Virtual Machines": "Compute & Web",
    "Storage": "Storage",
    "SQL Database": "Databases",
    "Functions": "Compute & Web",
    // ... complete mapping
}

// Instance type normalization
func NormalizeInstanceType(provider, instanceType string) string {
    // Map provider-specific instance types to standard sizes
    // e.g., aws:t3.medium -> standard_2cpu_4gb
    // azure:D2s_v3 -> standard_2cpu_8gb
}
```

### GraphQL Schema Considerations

```graphql
type Query {
  # Get pricing for specific criteria
  pricing(
    provider: Provider!
    category: Category
    region: String
    service: String
    pricingModel: PricingModel
  ): [Pricing!]!
  
  # Compare services across providers
  compareServices(
    category: Category!
    regions: [String!]
    specifications: ResourceSpecs
  ): ComparisonResult!
  
  # Find cheapest option
  cheapestOption(
    category: Category!
    requirements: ResourceRequirements!
  ): [Pricing!]!
}

enum Provider {
  AWS
  AZURE
}

enum Category {
  GENERAL
  NETWORKING
  COMPUTE_WEB
  CONTAINERS
  DATABASES
  STORAGE
  AI_ML
  ANALYTICS_IOT
  VIRTUAL_DESKTOP
  DEV_TOOLS
  INTEGRATION
  MIGRATION
  MANAGEMENT
}

enum PricingModel {
  ON_DEMAND
  RESERVED_1YR
  RESERVED_3YR
  SPOT
  SAVINGS_PLAN
}
```