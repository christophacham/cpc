package azure

import (
	"fmt"
	"github.com/raulc0399/cpc/internal/database"
)

// BuildCollector creates a configured collector using dependency injection
func BuildCollector(cfg CollectionConfig, db *database.DB) (*Collector, error) {
	// Build region handler
	regionHandler, err := buildRegionHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build region handler: %w", err)
	}

	// Build data store
	dataStore, err := buildDataStore(cfg, db)
	if err != nil {
		return nil, fmt.Errorf("failed to build data store: %w", err)
	}

	// Build output handler
	outputHandler, err := buildOutputHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build output handler: %w", err)
	}

	// Build progress tracker
	progressTracker := buildProgressTracker(cfg)

	return NewCollector(regionHandler, dataStore, outputHandler, progressTracker), nil
}

func buildRegionHandler(cfg CollectionConfig) (RegionHandler, error) {
	regions := GetRegionsByScope(cfg.Regions, cfg.TargetRegion)
	
	switch cfg.Regions {
	case "all":
		return &MultiRegionHandler{
			regions:  regions,
			maxItems: cfg.MaxItems,
		}, nil
	case "limited", "test":
		return &LimitedRegionHandler{
			regions:  cfg.TestRegions,
			maxItems: cfg.MaxItems,
		}, nil
	case "single":
		return &SingleRegionHandler{
			region:   cfg.TargetRegion,
			maxItems: cfg.MaxItems,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported regions mode: %s", cfg.Regions)
	}
}

func buildDataStore(cfg CollectionConfig, db *database.DB) (DataStore, error) {
	switch cfg.StorageType {
	case "jsonb":
		return &JSONBDataStore{db: db}, nil
	case "database":
		return &StructuredDataStore{db: db}, nil
	case "none":
		return &NoOpDataStore{}, nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.StorageType)
	}
}

func buildOutputHandler(cfg CollectionConfig) (OutputHandler, error) {
	switch cfg.OutputType {
	case "console":
		return &ConsoleOutputHandler{
			samplingEnabled: cfg.SamplingEnabled,
		}, nil
	case "json-export":
		if cfg.ExportFile == "" {
			cfg.ExportFile = "azure_pricing_export.json"
		}
		return &JSONExportOutputHandler{
			filePath: cfg.ExportFile,
		}, nil
	case "database":
		return &DatabaseOutputHandler{}, nil
	default:
		return nil, fmt.Errorf("unsupported output type: %s", cfg.OutputType)
	}
}

func buildProgressTracker(cfg CollectionConfig) ProgressTracker {
	if cfg.EnableTracking {
		return &DatabaseProgressTracker{}
	}
	return &NoOpProgressTracker{}
}

// CreateProfileConfigs returns predefined configurations for legacy compatibility
func CreateProfileConfigs() map[string]CollectionConfig {
	return map[string]CollectionConfig{
		"azure-raw-collector": {
			Mode:           "production",
			Regions:        "single",
			StorageType:    "jsonb",
			OutputType:     "database",
			EnableTracking: true,
			TargetRegion:   "eastus",
		},
		"azure-all-regions": {
			Mode:           "production",
			Regions:        "all",
			StorageType:    "jsonb",
			OutputType:     "database",
			EnableTracking: true,
			Concurrency:    3,
		},
		"azure-collector": {
			Mode:            "explorer",
			Regions:         "limited",
			StorageType:     "none",
			OutputType:      "console",
			SamplingEnabled: true,
			MaxItems:        2000,
		},
		"azure-db-collector": {
			Mode:           "production",
			Regions:        "single",
			StorageType:    "database",
			OutputType:     "database",
			EnableTracking: true,
			MaxItems:       1000,
			TargetRegion:   "eastus",
		},
		"azure-explorer": {
			Mode:            "explorer",
			Regions:         "single",
			StorageType:     "none",
			OutputType:      "console",
			SamplingEnabled: true,
			TargetRegion:    "eastus",
		},
		"azure-full-collector": {
			Mode:        "explorer",
			Regions:     "single",
			StorageType: "none",
			OutputType:  "json-export",
			ExportFile:  "azure_pricing_sample.json",
			MaxItems:    10000,
			TargetRegion: "eastus",
		},
	}
}