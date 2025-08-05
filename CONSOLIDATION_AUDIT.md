# CPC Azure Collectors Consolidation Audit

## Functionality Matrix

| Collector | Purpose | Scope | Output | Database | Concurrency | Unique Features |
|-----------|---------|-------|--------|----------|-------------|-----------------|
| **azure-raw-collector** | Single region raw data collection | Single region (arg/default eastus) | Database (azure_pricing_raw) | Yes | No | Progress tracking, raw JSONB storage |
| **azure-all-regions** | Multi-region concurrent collection | All 71 Azure regions | Database (azure_pricing_raw) | Yes | Yes (3 workers) | Worker pools, progress reporter, comprehensive coverage |
| **azure-collector** | Exploration & analysis | 4 test regions (eastus, westus, northeurope, southeastasia) | Console output only | No | No | Data shape analysis, field usage stats, discovery mode |
| **azure-db-collector** | Database-focused collection | Single region (eastus), limited to 1000 items | Database (azure_pricing table) | Yes | No | Version tracking, structured DB format, statistics display |
| **azure-explorer** | Service category exploration | Single region (eastus), 13 service categories | Console output only | No | No | Category-based queries, representative samples per service |
| **azure-full-collector** | Complete region analysis | Single region (eastus), max 10 pages | JSON file export | No | No | Service analysis, price range analysis, JSON export |

## Consolidation Analysis

### CLEAR CONSOLIDATION OPPORTUNITIES

**1. Database Storage Collectors (3 variants → 1 unified)**
- `azure-raw-collector` (single region, JSONB)
- `azure-all-regions` (multi-region, JSONB) 
- `azure-db-collector` (single region, structured format)

**Consolidation Plan**: Create unified `azure-collector` with configuration modes:
```go
type CollectorConfig struct {
    Regions      []string `json:"regions"`     // Single or multiple
    Concurrency  int      `json:"concurrency"` // 1 for single, N for multi
    OutputFormat string   `json:"format"`      // "raw", "structured"
    MaxItems     int      `json:"maxItems"`    // Limit for testing
}
```

**2. Analysis/Exploration Tools (3 variants → 1 unified)**
- `azure-collector` (data shape analysis)
- `azure-explorer` (service categories) 
- `azure-full-collector` (service analysis + export)

**Consolidation Plan**: Create unified `azure-explorer` with analysis modes:
```go
type ExplorerConfig struct {
    AnalysisType string   `json:"type"`        // "shape", "services", "full"
    Regions      []string `json:"regions"`     // Target regions
    Export       bool     `json:"export"`      // JSON file export
    Categories   []string `json:"categories"`  // Service category filter
}
```

### RECOMMENDED FINAL STRUCTURE

**3 Consolidated Collectors:**

1. **`azure-collector`** - Production data collection
   - Replaces: azure-raw-collector, azure-all-regions, azure-db-collector
   - Features: Single/multi-region, concurrent workers, progress tracking, multiple output formats

2. **`azure-explorer`** - Development & analysis
   - Replaces: azure-collector, azure-explorer, azure-full-collector  
   - Features: Data shape analysis, service exploration, JSON export, category queries

3. **`azure-admin`** - Administrative operations (new)
   - Features: Collection monitoring, data cleanup, health checks
   - Purpose: Operational tasks not covered by existing collectors

## Normalizer Version Audit

### PRODUCTION STATUS
**Current ETL Pipeline Uses:** V2 normalizers only
- `NewAWSNormalizerV2()` - /internal/etl/pipeline.go:99
- `NewAzureNormalizerV2()` - /internal/etl/pipeline.go:107

**Legacy Normalizers Status:**
- `aws_normalizer.go` - **UNUSED** (only self-referencing constructor)
- `aws_normalizer_refactored.go` - **UNUSED** (no references found)
- `azure_normalizer.go` - **UNUSED** (no references found)

### ARCHITECTURAL DIFFERENCES

**Legacy vs V2 Design:**
- **Legacy**: Direct database dependency, monolithic structure
- **V2**: Interface-driven with BaseNormalizer, repository pattern, better separation of concerns

**V2 Improvements:**
- Repository pattern for data access
- Input validation layer
- Unit normalization abstraction
- Logger interface for testing
- Resource spec extraction pattern

### CONSOLIDATION RECOMMENDATION
**Safe to Remove:** All legacy normalizer files
- No production code references found
- V2 implementations have complete feature parity
- ETL pipeline already standardized on V2

## Documentation Dependencies Audit

### FILES WITH COLLECTOR REFERENCES
**High Impact Documentation:**
- `/README.md` - References azure-raw-collector and azure-all-regions
- `/CONTRIBUTING.md` - References azure-collector for examples
- `/docs-site/docs/architecture/data-collection.md` - Documents all collector usage
- `/docs-site/docs/development/testing.md` - Includes collector test commands
- `/docs-site/docs/development/setup.md` - Setup instructions with collectors

**Referenced Commands in Documentation:**
- `go run cmd/azure-raw-collector/main.go eastus`
- `go run cmd/azure-all-regions/main.go 3`
- `go run cmd/azure-collector/main.go`
- `go run cmd/azure-explorer/main.go`
- `go run cmd/azure-full-collector/main.go`
- `go run cmd/azure-db-collector/main.go`

### DOCUMENTATION UPDATE REQUIREMENTS
**Critical Updates Needed:**
- Update all README examples to use consolidated collectors
- Revise data-collection.md architecture documentation
- Update testing.md with new collector commands
- Modify setup.md to reflect simplified structure
- Update CONTRIBUTING.md examples

## Impact Assessment

**Complexity Reduction:**
- From 6 Azure collectors to 3 unified tools (50% reduction)
- Remove 3 legacy normalizer files (aws_normalizer.go, aws_normalizer_refactored.go, azure_normalizer.go)
- Eliminate code duplication across collectors
- Simplified command structure for contributors

**Functionality Preservation:**
- All current capabilities maintained through configuration
- No loss of data collection features
- Enhanced flexibility through unified interfaces
- V2 normalizers already in production use

**Migration Effort:**
- Low risk - existing collectors remain functional during transition
- Zero risk for normalizer cleanup - no production references
- Clear migration path through configuration mapping
- Backward compatibility maintained