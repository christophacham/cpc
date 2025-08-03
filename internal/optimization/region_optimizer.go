package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"

	_ "github.com/lib/pq"
)

// WorkloadProfile represents compute requirements for optimization
type WorkloadProfile struct {
	ComputeHours float64  `json:"compute_hours"`
	StorageGB    float64  `json:"storage_gb"`
	EgressGB     float64  `json:"egress_gb"`
	CPUCount     int      `json:"cpu_count"`
	GPUCount     int      `json:"gpu_count"`
	Providers    []string `json:"providers"`
	Geography    string   `json:"geography"`
}

// RegionOptimization represents the result of region optimization
type RegionOptimization struct {
	Provider        string   `json:"provider"`
	Region          string   `json:"region"`
	RegionName      string   `json:"region_name"`
	MonthlyCost     float64  `json:"monthly_cost"`
	ComputeCost     float64  `json:"compute_cost"`
	StorageCost     float64  `json:"storage_cost"`
	EgressCost      float64  `json:"egress_cost"`
	Recommendations []string `json:"recommendations"`
}

// RegionComparison represents comparison between regions
type RegionComparison struct {
	Provider       string   `json:"provider"`
	Region         string   `json:"region"`
	RegionName     string   `json:"region_name"`
	MonthlyCost    float64  `json:"monthly_cost"`
	ComputeCost    float64  `json:"compute_cost"`
	StorageCost    float64  `json:"storage_cost"`
	EgressCost     float64  `json:"egress_cost"`
	SavingsPercent float64  `json:"savings_percent"`
	BestFor        []string `json:"best_for"`
}

// RegionOptimizer provides region optimization functionality using real CPC data
type RegionOptimizer struct {
	db *sql.DB
}

// NewRegionOptimizer creates a new region optimizer
func NewRegionOptimizer(db *sql.DB) *RegionOptimizer {
	return &RegionOptimizer{db: db}
}

// OptimizeRegions finds the best regions for a given workload using real pricing data
func (ro *RegionOptimizer) OptimizeRegions(ctx context.Context, workload WorkloadProfile) ([]RegionOptimization, error) {
	// Get normalized pricing data for compute, storage, and egress
	computePrices, err := ro.getComputePrices(ctx, workload.Providers)
	if err != nil {
		log.Printf("Error getting compute prices: %v", err)
		// Fall back to sample data if database query fails
		return ro.getFallbackOptimizations(workload), nil
	}

	storagePrices, err := ro.getStoragePrices(ctx, workload.Providers)
	if err != nil {
		log.Printf("Error getting storage prices: %v", err)
		return ro.getFallbackOptimizations(workload), nil
	}

	egressPrices, err := ro.getEgressPrices(ctx, workload.Providers)
	if err != nil {
		log.Printf("Error getting egress prices: %v", err)
		return ro.getFallbackOptimizations(workload), nil
	}

	// Calculate total cost for each region
	var results []RegionOptimization

	for _, cp := range computePrices {
		// Find matching storage and egress prices for this region
		storagePrice := ro.findPriceForRegion(storagePrices, cp.Provider, cp.Region)
		egressPrice := ro.findPriceForRegion(egressPrices, cp.Provider, cp.Region)

		// Calculate costs
		computeCost := workload.ComputeHours * cp.Price * 30 // Monthly
		storageCost := workload.StorageGB * storagePrice
		egressCost := workload.EgressGB * egressPrice

		totalCost := computeCost + storageCost + egressCost

		recommendations := ro.generateRecommendations(cp, storagePrice, egressPrice)

		results = append(results, RegionOptimization{
			Provider:        cp.Provider,
			Region:          cp.Region,
			RegionName:      cp.RegionName,
			MonthlyCost:     totalCost,
			ComputeCost:     computeCost,
			StorageCost:     storageCost,
			EgressCost:      egressCost,
			Recommendations: recommendations,
		})
	}

	// Sort by total cost
	sort.Slice(results, func(i, j int) bool {
		return results[i].MonthlyCost < results[j].MonthlyCost
	})

	// Return top 10 results
	if len(results) > 10 {
		results = results[:10]
	}

	return results, nil
}

// CompareRegions compares specific regions for a workload
func (ro *RegionOptimizer) CompareRegions(ctx context.Context, workload WorkloadProfile, regions []RegionInput) ([]RegionComparison, error) {
	var results []RegionComparison
	var lowestCost float64 = math.MaxFloat64

	// First pass: calculate costs for all regions
	for _, region := range regions {
		computePrice, err := ro.getSpecificComputePrice(ctx, region.Provider, region.Region)
		if err != nil {
			log.Printf("Error getting compute price for %s/%s: %v", region.Provider, region.Region, err)
			continue
		}

		storagePrice, err := ro.getSpecificStoragePrice(ctx, region.Provider, region.Region)
		if err != nil {
			log.Printf("Error getting storage price for %s/%s: %v", region.Provider, region.Region, err)
			storagePrice = 0.025 // Default fallback
		}

		egressPrice, err := ro.getSpecificEgressPrice(ctx, region.Provider, region.Region)
		if err != nil {
			log.Printf("Error getting egress price for %s/%s: %v", region.Provider, region.Region, err)
			egressPrice = 0.09 // Default fallback
		}

		computeCost := workload.ComputeHours * computePrice * 30
		storageCost := workload.StorageGB * storagePrice
		egressCost := workload.EgressGB * egressPrice
		totalCost := computeCost + storageCost + egressCost

		if totalCost < lowestCost {
			lowestCost = totalCost
		}

		results = append(results, RegionComparison{
			Provider:    region.Provider,
			Region:      region.Region,
			RegionName:  ro.getRegionDisplayName(region.Provider, region.Region),
			MonthlyCost: totalCost,
			ComputeCost: computeCost,
			StorageCost: storageCost,
			EgressCost:  egressCost,
			BestFor:     ro.determineBestFor(computePrice, storagePrice, egressPrice),
		})
	}

	// Second pass: calculate savings percentages
	for i := range results {
		if lowestCost > 0 {
			results[i].SavingsPercent = ((results[i].MonthlyCost - lowestCost) / lowestCost) * 100
		}
	}

	return results, nil
}

// RegionInput represents a specific region for comparison
type RegionInput struct {
	Provider string `json:"provider"`
	Region   string `json:"region"`
}

// PriceData represents pricing information for a region
type PriceData struct {
	Provider   string
	Region     string
	RegionName string
	Price      float64
}

// Database query methods

func (ro *RegionOptimizer) getComputePrices(ctx context.Context, providers []string) ([]PriceData, error) {
	query := `
		SELECT DISTINCT 
			provider,
			region,
			region as region_name,
			AVG(price_per_unit) as avg_price
		FROM normalized_pricing 
		WHERE service_category = 'Compute & Web' 
		AND price_per_unit > 0
		AND ($1 = '' OR provider = ANY($2))
		GROUP BY provider, region
		ORDER BY avg_price ASC
		LIMIT 50
	`

	var args []interface{}
	if len(providers) == 0 {
		args = []interface{}{"", nil}
	} else {
		// Convert to PostgreSQL array format
		args = []interface{}{"has_providers", providers}
	}

	rows, err := ro.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query compute prices: %w", err)
	}
	defer rows.Close()

	var results []PriceData
	for rows.Next() {
		var pd PriceData
		err := rows.Scan(&pd.Provider, &pd.Region, &pd.RegionName, &pd.Price)
		if err != nil {
			log.Printf("Error scanning compute price row: %v", err)
			continue
		}
		results = append(results, pd)
	}

	return results, nil
}

func (ro *RegionOptimizer) getStoragePrices(ctx context.Context, providers []string) ([]PriceData, error) {
	query := `
		SELECT DISTINCT 
			provider,
			region,
			region as region_name,
			AVG(price_per_unit) as avg_price
		FROM normalized_pricing 
		WHERE service_category = 'Storage' 
		AND price_per_unit > 0
		AND ($1 = '' OR provider = ANY($2))
		GROUP BY provider, region
		ORDER BY avg_price ASC
	`

	var args []interface{}
	if len(providers) == 0 {
		args = []interface{}{"", nil}
	} else {
		args = []interface{}{"has_providers", providers}
	}

	rows, err := ro.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query storage prices: %w", err)
	}
	defer rows.Close()

	var results []PriceData
	for rows.Next() {
		var pd PriceData
		err := rows.Scan(&pd.Provider, &pd.Region, &pd.RegionName, &pd.Price)
		if err != nil {
			log.Printf("Error scanning storage price row: %v", err)
			continue
		}
		results = append(results, pd)
	}

	return results, nil
}

func (ro *RegionOptimizer) getEgressPrices(ctx context.Context, providers []string) ([]PriceData, error) {
	query := `
		SELECT DISTINCT 
			provider,
			region,
			region as region_name,
			AVG(price_per_unit) as avg_price
		FROM normalized_pricing 
		WHERE service_category = 'Networking' 
		AND (resource_name ILIKE '%transfer%' OR resource_name ILIKE '%egress%' OR resource_name ILIKE '%bandwidth%')
		AND price_per_unit > 0
		AND ($1 = '' OR provider = ANY($2))
		GROUP BY provider, region
		ORDER BY avg_price ASC
	`

	var args []interface{}
	if len(providers) == 0 {
		args = []interface{}{"", nil}
	} else {
		args = []interface{}{"has_providers", providers}
	}

	rows, err := ro.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query egress prices: %w", err)
	}
	defer rows.Close()

	var results []PriceData
	for rows.Next() {
		var pd PriceData
		err := rows.Scan(&pd.Provider, &pd.Region, &pd.RegionName, &pd.Price)
		if err != nil {
			log.Printf("Error scanning egress price row: %v", err)
			continue
		}
		results = append(results, pd)
	}

	return results, nil
}

func (ro *RegionOptimizer) getSpecificComputePrice(ctx context.Context, provider, region string) (float64, error) {
	query := `
		SELECT AVG(price_per_unit) 
		FROM normalized_pricing 
		WHERE provider = $1 AND region = $2 
		AND service_category = 'Compute & Web' 
		AND price_per_unit > 0
	`

	var price float64
	err := ro.db.QueryRowContext(ctx, query, provider, region).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("query specific compute price: %w", err)
	}

	return price, nil
}

func (ro *RegionOptimizer) getSpecificStoragePrice(ctx context.Context, provider, region string) (float64, error) {
	query := `
		SELECT AVG(price_per_unit) 
		FROM normalized_pricing 
		WHERE provider = $1 AND region = $2 
		AND service_category = 'Storage' 
		AND price_per_unit > 0
	`

	var price float64
	err := ro.db.QueryRowContext(ctx, query, provider, region).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("query specific storage price: %w", err)
	}

	return price, nil
}

func (ro *RegionOptimizer) getSpecificEgressPrice(ctx context.Context, provider, region string) (float64, error) {
	query := `
		SELECT AVG(price_per_unit) 
		FROM normalized_pricing 
		WHERE provider = $1 AND region = $2 
		AND service_category = 'Networking' 
		AND (resource_name ILIKE '%transfer%' OR resource_name ILIKE '%egress%')
		AND price_per_unit > 0
	`

	var price float64
	err := ro.db.QueryRowContext(ctx, query, provider, region).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("query specific egress price: %w", err)
	}

	return price, nil
}

// Helper methods

func (ro *RegionOptimizer) findPriceForRegion(prices []PriceData, provider, region string) float64 {
	for _, p := range prices {
		if p.Provider == provider && p.Region == region {
			return p.Price
		}
	}
	// Return average price if specific region not found
	if len(prices) > 0 {
		var total float64
		for _, p := range prices {
			total += p.Price
		}
		return total / float64(len(prices))
	}
	return 0.025 // Default fallback
}

func (ro *RegionOptimizer) generateRecommendations(compute PriceData, storagePrice, egressPrice float64) []string {
	var recommendations []string

	if compute.Price < 0.05 {
		recommendations = append(recommendations, "Excellent compute pricing")
	}
	if storagePrice < 0.02 {
		recommendations = append(recommendations, "Low storage costs")
	}
	if egressPrice < 0.08 {
		recommendations = append(recommendations, "Competitive egress rates")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Standard pricing tier")
	}

	return recommendations
}

func (ro *RegionOptimizer) getRegionDisplayName(provider, region string) string {
	// Simple mapping - could be expanded with a database lookup
	regionNames := map[string]map[string]string{
		"AWS": {
			"us-east-1":      "US East (N. Virginia)",
			"us-west-2":      "US West (Oregon)",
			"eu-west-1":      "EU (Ireland)",
			"ap-southeast-1": "Asia Pacific (Singapore)",
		},
		"Azure": {
			"eastus":        "East US",
			"westus":        "West US",
			"westeurope":    "West Europe",
			"southeastasia": "Southeast Asia",
		},
	}

	if providerMap, exists := regionNames[provider]; exists {
		if name, exists := providerMap[region]; exists {
			return name
		}
	}

	return region // Fallback to region ID
}

func (ro *RegionOptimizer) determineBestFor(computePrice, storagePrice, egressPrice float64) []string {
	var bestFor []string

	if computePrice < 0.05 {
		bestFor = append(bestFor, "Compute-intensive workloads")
	}
	if storagePrice < 0.02 {
		bestFor = append(bestFor, "Storage-heavy applications")
	}
	if egressPrice < 0.08 {
		bestFor = append(bestFor, "High egress requirements")
	}

	if len(bestFor) == 0 {
		bestFor = append(bestFor, "General purpose workloads")
	}

	return bestFor
}

func (ro *RegionOptimizer) getFallbackOptimizations(workload WorkloadProfile) []RegionOptimization {
	// Fallback data when database queries fail
	fallbackData := []RegionOptimization{
		{
			Provider:        "Azure",
			Region:          "eastus",
			RegionName:      "East US",
			MonthlyCost:     workload.ComputeHours*0.096*30 + workload.StorageGB*0.0184 + workload.EgressGB*0.087,
			ComputeCost:     workload.ComputeHours * 0.096 * 30,
			StorageCost:     workload.StorageGB * 0.0184,
			EgressCost:      workload.EgressGB * 0.087,
			Recommendations: []string{"Competitive pricing", "Good for general workloads"},
		},
		{
			Provider:        "AWS",
			Region:          "us-east-1",
			RegionName:      "US East (N. Virginia)",
			MonthlyCost:     workload.ComputeHours*0.10*30 + workload.StorageGB*0.023 + workload.EgressGB*0.09,
			ComputeCost:     workload.ComputeHours * 0.10 * 30,
			StorageCost:     workload.StorageGB * 0.023,
			EgressCost:      workload.EgressGB * 0.09,
			Recommendations: []string{"Standard pricing", "Wide service availability"},
		},
		{
			Provider:        "Azure",
			Region:          "westeurope",
			RegionName:      "West Europe",
			MonthlyCost:     workload.ComputeHours*0.106*30 + workload.StorageGB*0.0196 + workload.EgressGB*0.087,
			ComputeCost:     workload.ComputeHours * 0.106 * 30,
			StorageCost:     workload.StorageGB * 0.0196,
			EgressCost:      workload.EgressGB * 0.087,
			Recommendations: []string{"EU data residency", "GDPR compliant"},
		},
	}

	// Sort by cost
	sort.Slice(fallbackData, func(i, j int) bool {
		return fallbackData[i].MonthlyCost < fallbackData[j].MonthlyCost
	})

	return fallbackData
}