package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/raulc0399/cpc/internal/azure"
	"github.com/raulc0399/cpc/internal/database"
)

func main() {
	var (
		profile    = flag.String("profile", "", "Profile name for predefined configuration")
		region     = flag.String("region", "eastus", "Target region for single region collection")
		regions    = flag.String("regions", "single", "Region scope: single, limited, major, all")
		storage    = flag.String("storage", "jsonb", "Storage type: jsonb, database, none")
		output     = flag.String("output", "database", "Output type: console, json-export, database")
		maxItems   = flag.Int("max-items", 0, "Maximum items to collect (0 = unlimited)")
		concurrent = flag.Int("concurrent", 1, "Number of concurrent workers")
		exportFile = flag.String("export-file", "", "File path for JSON export")
		sampling   = flag.Bool("sampling", false, "Enable sampling for console output")
		tracking   = flag.Bool("tracking", true, "Enable progress tracking")
		listProfiles = flag.Bool("list-profiles", false, "List available profiles")
	)
	flag.Parse()

	// List available profiles
	if *listProfiles {
		profiles := azure.CreateProfileConfigs()
		fmt.Println("Available profiles:")
		for name, config := range profiles {
			fmt.Printf("  %s: %s collection, %s storage, %s output\n", 
				name, config.Regions, config.StorageType, config.OutputType)
		}
		os.Exit(0)
	}

	// Initialize database connection
	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	var config azure.CollectionConfig

	// Use profile if specified
	if *profile != "" {
		profiles := azure.CreateProfileConfigs()
		profileConfig, exists := profiles[*profile]
		if !exists {
			log.Fatalf("Profile '%s' not found. Use --list-profiles to see available options.", *profile)
		}
		config = profileConfig
		log.Printf("Using profile: %s", *profile)
	} else {
		// Build config from command line arguments
		config = azure.CollectionConfig{
			Mode:            "manual",
			Regions:         *regions,
			StorageType:     *storage,
			OutputType:      *output,
			TargetRegion:    *region,
			MaxItems:        *maxItems,
			Concurrency:     *concurrent,
			ExportFile:      *exportFile,
			SamplingEnabled: *sampling,
			EnableTracking:  *tracking,
		}

		// Set test regions for limited collection
		if *regions == "limited" || *regions == "test" {
			config.TestRegions = []string{"eastus", "westus", "northeurope", "southeastasia"}
		}
	}

	// Override command line parameters if provided
	if flag.Lookup("region").Changed {
		config.TargetRegion = *region
	}
	if flag.Lookup("max-items").Changed {
		config.MaxItems = *maxItems
	}
	if flag.Lookup("concurrent").Changed {
		config.Concurrency = *concurrent
	}
	if flag.Lookup("export-file").Changed && *exportFile != "" {
		config.ExportFile = *exportFile
	}
	if flag.Lookup("sampling").Changed {
		config.SamplingEnabled = *sampling
	}
	if flag.Lookup("tracking").Changed {
		config.EnableTracking = *tracking
	}

	// Display configuration
	log.Printf("Configuration:")
	log.Printf("  Mode: %s", config.Mode)
	log.Printf("  Regions: %s", config.Regions)
	log.Printf("  Storage: %s", config.StorageType)
	log.Printf("  Output: %s", config.OutputType)
	log.Printf("  Target Region: %s", config.TargetRegion)
	log.Printf("  Max Items: %d", config.MaxItems)
	log.Printf("  Concurrency: %d", config.Concurrency)
	log.Printf("  Tracking: %t", config.EnableTracking)

	// Build collector using factory
	collector, err := azure.BuildCollector(config, db)
	if err != nil {
		log.Fatalf("Failed to build collector: %v", err)
	}

	// Execute collection
	ctx := context.Background()
	
	if config.Concurrency > 1 {
		log.Printf("Starting concurrent collection with %d workers", config.Concurrency)
		err = collector.RunConcurrent(ctx, config.Concurrency)
	} else {
		log.Printf("Starting sequential collection")
		err = collector.Run(ctx)
	}

	if err != nil {
		log.Fatalf("Collection failed: %v", err)
	}

	log.Printf("✅ Collection completed successfully!")
}

func parseTestRegions(regionsStr string) []string {
	if regionsStr == "" {
		return []string{"eastus", "westus", "northeurope", "southeastasia"}
	}
	return strings.Split(regionsStr, ",")
}

func parseIntEnv(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func parseBoolEnv(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultValue
}