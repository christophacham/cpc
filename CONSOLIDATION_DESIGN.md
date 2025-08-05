# CPC Consolidated Collector Architecture Design

## Architecture Overview

Consolidate 6 Azure collectors into 3 unified tools using configuration-driven, interface-based design with dependency injection.

### Target Structure
```
cmd/
├── azure-collector/     # Production data collection (replaces 4 collectors)
├── azure-explorer/      # Development & analysis (replaces 2 collectors)  
└── azure-admin/         # Administrative operations (new)
```

## Core Architecture Components

### 1. Core Collector Engine
```go
package collector

type Collector struct {
    RegionHandler   RegionHandler
    DataStore       DataStore
    OutputHandler   OutputHandler
    ProgressTracker ProgressTracker
}

func NewCollector(regionHandler RegionHandler, dataStore DataStore, 
                 outputHandler OutputHandler, tracker ProgressTracker) *Collector {
    return &Collector{
        RegionHandler:   regionHandler,
        DataStore:       dataStore,
        OutputHandler:   outputHandler,
        ProgressTracker: tracker,
    }
}

func (c *Collector) Run(ctx context.Context) error {
    regions := c.RegionHandler.GetRegions()
    for _, region := range regions {
        data, err := c.RegionHandler.Collect(region)
        if err != nil {
            return err
        }
        if err := c.DataStore.Store(data); err != nil {
            return err
        }
        c.OutputHandler.Write(data)
        c.ProgressTracker.Update(region)
    }
    return nil
}
```

### 2. Modular Interfaces
```go
type RegionHandler interface {
    GetRegions() []string
    Collect(region string) (interface{}, error)
}

type DataStore interface {
    Store(data interface{}) error
}

type OutputHandler interface {
    Write(data interface{}) error
}

type ProgressTracker interface {
    Update(region string)
}
```

### 3. Configuration Structure
```go
type Config struct {
    Mode            string // "production", "explorer", "admin"
    Regions         string // "single", "all", "limited"
    StorageType     string // "jsonb", "database", "none"
    OutputType      string // "console", "json-export", "database"
    Concurrency     int    // for multi-region concurrent workers
    EnableTracking  bool   // progress tracking toggle
    SamplingEnabled bool   // for explorer mode sampling
    MaxItems        int    // limit for testing/exploration
    ExportFile      string // JSON export file path
    TargetRegion    string // specific region for single mode
}
```

### 4. Factory with Dependency Injection
```go
func BuildCollector(cfg Config) (*Collector, error) {
    var regionHandler RegionHandler
    switch cfg.Regions {
    case "all":
        regionHandler = &MultiRegionHandler{Concurrency: cfg.Concurrency}
    case "limited":
        regionHandler = &LimitedRegionHandler{TestRegions: []string{"eastus", "westus", "northeurope", "southeastasia"}}
    default:
        regionHandler = &SingleRegionHandler{Region: cfg.TargetRegion}
    }

    var dataStore DataStore
    switch cfg.StorageType {
    case "jsonb":
        dataStore = &JSONBStore{}
    case "database":
        dataStore = &DatabaseStore{}
    default:
        dataStore = &NoOpStore{}
    }

    var outputHandler OutputHandler
    switch cfg.OutputType {
    case "console":
        outputHandler = &ConsoleOutput{SamplingEnabled: cfg.SamplingEnabled}
    case "json-export":
        outputHandler = &JSONExportOutput{FilePath: cfg.ExportFile}
    default:
        outputHandler = &DatabaseOutput{}
    }

    tracker := &NoOpTracker{}
    if cfg.EnableTracking {
        tracker = &ProgressTrackerImpl{}
    }

    return NewCollector(regionHandler, dataStore, outputHandler, tracker), nil
}
```

## Configuration Profiles for Legacy Compatibility

### Production Collectors

**azure-raw-collector** → azure-collector
```yaml
mode: production
regions: single
storage_type: jsonb
enable_tracking: true
output_type: database
target_region: ${ARG1:-eastus}
```

**azure-all-regions** → azure-collector
```yaml
mode: production
regions: all
storage_type: jsonb
enable_tracking: true
concurrency: ${ARG1:-3}
output_type: database
```

**azure-db-collector** → azure-collector
```yaml
mode: production
regions: single
storage_type: database
enable_tracking: true
max_items: 1000
target_region: eastus
```

### Analysis/Exploration Collectors

**azure-collector** → azure-explorer
```yaml
mode: explorer
regions: limited
output_type: console
sampling_enabled: true
max_items: 2000
```

**azure-explorer** → azure-explorer
```yaml
mode: explorer
regions: single
output_type: console
sampling_enabled: true
target_region: eastus
categories: ["General", "Compute", "Storage", "Databases"]
```

**azure-full-collector** → azure-explorer
```yaml
mode: explorer
regions: single
output_type: json-export
export_file: azure_pricing_sample.json
max_items: 10000
target_region: eastus
```

## Interface Implementations

### RegionHandler Implementations
1. **SingleRegionHandler** - Single region collection
2. **MultiRegionHandler** - Concurrent multi-region with worker pools
3. **LimitedRegionHandler** - Test regions only (eastus, westus, northeurope, southeastasia)

### DataStore Implementations
1. **JSONBStore** - Raw JSONB storage (azure_pricing_raw table)
2. **DatabaseStore** - Structured storage (azure_pricing table)
3. **NoOpStore** - No storage (console-only modes)

### OutputHandler Implementations
1. **ConsoleOutput** - Statistics, analysis display, data shape analysis
2. **JSONExportOutput** - File export functionality
3. **DatabaseOutput** - Silent database insertion

### ProgressTracker Implementations
1. **ProgressTrackerImpl** - Real-time progress tracking with database updates
2. **NoOpTracker** - No tracking for simple modes

## Migration Strategy

### Phase 1: Shared Package Creation
```
internal/
├── azure/
│   ├── client.go          # Reusable Azure API client
│   ├── types.go           # Common structures (AzureAPIResponse, PriceItem)
│   ├── regions.go         # Region definitions and helpers
│   └── collector.go       # Core collector engine
```

### Phase 2: Legacy Wrapper Scripts
```bash
#!/bin/bash
# cmd/azure-raw-collector/legacy-wrapper.sh
echo "DEPRECATED: Use 'go run cmd/azure-collector/main.go --mode=production --regions=single --target-region=$1'"
go run cmd/azure-collector/main.go \
  --mode=production \
  --regions=single \
  --storage-type=jsonb \
  --enable-tracking \
  --target-region="${1:-eastus}"
```

### Phase 3: CLI Interface
```go
func main() {
    var cfg Config
    
    flag.StringVar(&cfg.Mode, "mode", "production", "Operation mode: production, explorer, admin")
    flag.StringVar(&cfg.Regions, "regions", "single", "Region scope: single, all, limited")
    flag.StringVar(&cfg.StorageType, "storage-type", "jsonb", "Storage type: jsonb, database, none")
    flag.StringVar(&cfg.OutputType, "output-type", "database", "Output type: console, json-export, database")
    flag.StringVar(&cfg.TargetRegion, "target-region", "eastus", "Specific region for single mode")
    flag.IntVar(&cfg.Concurrency, "concurrency", 1, "Concurrent workers")
    flag.BoolVar(&cfg.EnableTracking, "enable-tracking", true, "Enable progress tracking")
    flag.BoolVar(&cfg.SamplingEnabled, "sampling", false, "Enable sampling mode")
    flag.IntVar(&cfg.MaxItems, "max-items", 0, "Maximum items to collect (0 = unlimited)")
    flag.StringVar(&cfg.ExportFile, "export-file", "", "JSON export file path")
    flag.Parse()
    
    collector, err := BuildCollector(cfg)
    if err != nil {
        log.Fatalf("Failed to build collector: %v", err)
    }
    
    if err := collector.Run(context.Background()); err != nil {
        log.Fatalf("Collection failed: %v", err)
    }
}
```

## Benefits

### Complexity Reduction
- 6 collectors → 3 unified tools (50% reduction)
- Single Azure API client implementation
- Shared error handling and retry logic
- Unified progress tracking system

### Flexibility Enhancement
- Runtime behavior modification via configuration
- Easy A/B testing of collection strategies
- Environment-specific deployments
- Simplified Docker orchestration

### Maintainability Improvement
- Interface-driven design for testability
- Clear separation of concerns
- Dependency injection for component isolation
- Focused implementations per responsibility

### Migration Safety
- Legacy commands work through wrappers
- Gradual transition capability
- No immediate breaking changes
- Rollback strategy maintained

## Next Steps

1. **Create shared Azure package** with common API client
2. **Implement core collector engine** with interface abstractions
3. **Build factory function** with dependency injection
4. **Create wrapper scripts** for backward compatibility
5. **Update documentation** with new unified commands
6. **Deploy with parallel validation** to ensure feature parity