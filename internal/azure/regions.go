package azure

// AllRegions contains the complete list of Azure regions
var AllRegions = []string{
	"eastus", "eastus2", "southcentralus", "westus2", "westus3", "australiaeast",
	"southeastasia", "northeurope", "swedencentral", "uksouth", "westeurope",
	"centralus", "southafricanorth", "centralindia", "eastasia", "japaneast",
	"koreacentral", "canadacentral", "francecentral", "germanywestcentral",
	"norwayeast", "switzerlandnorth", "uaenorth", "brazilsouth", "eastus2euap",
	"qatarcentral", "centralusstage", "eastusstage", "eastus2stage", "northcentralusstage",
	"southcentralusstage", "westusstage", "westus2stage", "asia", "asiapacific",
	"australia", "brazil", "canada", "europe", "france", "germany", "global",
	"india", "japan", "korea", "norway", "southafrica", "switzerland", "uae",
	"uk", "unitedstates", "unitedstateseuap", "eastasiastage", "southeastasiastage",
	"northcentralus", "westus", "jioindiawest", "centraluseuap", "westcentralus",
	"southafricawest", "australiacentral", "australiacentral2", "australiasoutheast",
	"japanwest", "jioindiacentral", "koreasouth", "southindia", "westindia",
	"canadaeast", "francesouth", "germanynorth", "norwaywest", "switzerlandwest",
	"ukwest", "uaecentral", "brazilsoutheast",
}

// TestRegions contains a subset of regions for testing and exploration
var TestRegions = []string{
	"eastus", "westus", "northeurope", "southeastasia",
}

// MajorRegions contains primary regions for focused collection
var MajorRegions = []string{
	"eastus", "eastus2", "westus2", "northeurope", "westeurope", 
	"southeastasia", "australiaeast", "japaneast", "uksouth",
}

// GetRegionsByScope returns regions based on scope configuration
func GetRegionsByScope(scope string, targetRegion string) []string {
	switch scope {
	case "all":
		return AllRegions
	case "major":
		return MajorRegions
	case "limited", "test":
		return TestRegions
	case "single":
		if targetRegion != "" {
			return []string{targetRegion}
		}
		return []string{"eastus"}
	default:
		return []string{"eastus"}
	}
}

// ValidateRegion checks if a region is valid
func ValidateRegion(region string) bool {
	for _, r := range AllRegions {
		if r == region {
			return true
		}
	}
	return false
}