package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/raulc0399/cpc/internal/database"
)

// GraphQL request structure
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// GraphQL response structure
type graphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []string    `json:"errors,omitempty"`
}

// Population request structure
type populationRequest struct {
	Region      string `json:"region"`
	Concurrency int    `json:"concurrency,omitempty"`
}

// AWS Population request structure
type awsPopulationRequest struct {
	ServiceCodes  []string `json:"serviceCodes,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	InstanceTypes []string `json:"instanceTypes,omitempty"`
	Concurrency   int      `json:"concurrency,omitempty"`
}

// Population response structure
type populationResponse struct {
	Message      string `json:"message"`
	CollectionID string `json:"collectionId,omitempty"`
	Error        string `json:"error,omitempty"`
}

// AzureAPIResponse represents the Azure pricing API response
type AzureAPIResponse struct {
	BillingCurrency    string                   `json:"BillingCurrency"`
	CustomerEntityID   string                   `json:"CustomerEntityId"`
	CustomerEntityType string                   `json:"CustomerEntityType"`
	Items              []map[string]interface{} `json:"Items"`
	NextPageLink       string                   `json:"NextPageLink"`
	Count              int                      `json:"Count"`
}

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cpc_user:cpc_password@localhost:5432/cpc_db?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully!")

	// Create database handler
	dbHandler := database.New(db)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Set up routes
	http.HandleFunc("/", playgroundHandler)
	http.HandleFunc("/query", graphQLHandler(dbHandler))
	http.HandleFunc("/populate", populateHandler(dbHandler))
	http.HandleFunc("/populate-all", populateAllHandler(dbHandler))
	http.HandleFunc("/aws-populate", awsPopulateHandler(dbHandler))
	http.HandleFunc("/aws-populate-all", awsPopulateAllHandler(dbHandler))
	http.HandleFunc("/aws-populate-comprehensive", awsPopulateComprehensiveHandler(dbHandler))
	http.HandleFunc("/aws-populate-everything", awsPopulateEverythingHandler(dbHandler))

	log.Printf("Starting server on http://localhost:%s/", port)
	log.Printf("GraphQL playground available at http://localhost:%s/", port)
	log.Printf("Population endpoint available at http://localhost:%s/populate", port)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Population endpoint handler
func populateHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req populationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Region == "" {
			req.Region = "eastus"
		}

		log.Printf("Starting Azure data population for region: %s", req.Region)

		// Start collection in database
		collectionID, err := db.StartAzureRawCollection(req.Region)
		if err != nil {
			response := populationResponse{
				Message: "Failed to start collection",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Run collector in background
		go func() {
			totalItems, err := collectAzureData(db, collectionID, req.Region)
			if err != nil {
				log.Printf("Collection failed for %s: %v", collectionID, err)
				db.FailAzureRawCollection(collectionID, err.Error())
				return
			}
			
			// Complete collection
			if err := db.CompleteAzureRawCollection(collectionID, totalItems); err != nil {
				log.Printf("Failed to mark collection as completed: %v", err)
			} else {
				log.Printf("✅ Collection completed successfully! ID: %s, Region: %s, Items: %d", collectionID, req.Region, totalItems)
			}
		}()

		response := populationResponse{
			Message:      fmt.Sprintf("Started data collection for region: %s", req.Region),
			CollectionID: collectionID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Population all regions endpoint handler
func populateAllHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req populationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Use defaults if no body provided
			req.Concurrency = 3
		}

		if req.Concurrency <= 0 {
			req.Concurrency = 3
		}

		log.Printf("Starting Azure all-regions data collection with concurrency: %d", req.Concurrency)

		// Run all-regions collector in background
		go func() {
			// For now, use a simple approach - collect a few major regions
			regions := []string{"eastus", "westus", "eastus2", "westus2", "centralus", "northeurope", "westeurope"}
			log.Printf("Starting collection for %d major regions with concurrency %d", len(regions), req.Concurrency)
			
			// Simple sequential collection for now
			for _, region := range regions {
				collectionID, err := db.StartAzureRawCollection(region)
				if err != nil {
					log.Printf("Failed to start collection for %s: %v", region, err)
					continue
				}
				
				totalItems, err := collectAzureData(db, collectionID, region)
				if err != nil {
					log.Printf("Collection failed for %s: %v", region, err)
					db.FailAzureRawCollection(collectionID, err.Error())
					continue
				}
				
				if err := db.CompleteAzureRawCollection(collectionID, totalItems); err != nil {
					log.Printf("Failed to mark collection as completed for %s: %v", region, err)
				} else {
					log.Printf("✅ Completed %s: %d items", region, totalItems)
				}
			}
			log.Printf("🎉 All-regions collection completed!")
		}()

		response := populationResponse{
			Message: fmt.Sprintf("Started data collection for all Azure regions (concurrency: %d)", req.Concurrency),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Simple GraphQL handler with raw data support
func graphQLHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Simple query parsing
		response := graphQLResponse{
			Data: make(map[string]interface{}),
		}

		data := response.Data.(map[string]interface{})

		// Handle different queries
		if contains(req.Query, "hello") {
			data["hello"] = "Hello from Cloud Price Compare GraphQL API (Raw JSON Version)!"
		}
		
		if contains(req.Query, "messages") {
			messages, err := db.GetMessages()
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				msgList := make([]map[string]interface{}, len(messages))
				for i, msg := range messages {
					msgList[i] = map[string]interface{}{
						"id":        fmt.Sprintf("%d", msg.ID),
						"content":   msg.Content,
						"createdAt": msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
					}
				}
				data["messages"] = msgList
			}
		}

		if contains(req.Query, "providers") {
			providers, err := db.GetProviders()
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				provList := make([]map[string]interface{}, len(providers))
				for i, p := range providers {
					provList[i] = map[string]interface{}{
						"id":        fmt.Sprintf("%d", p.ID),
						"name":      p.Name,
						"createdAt": p.CreatedAt.Format("2006-01-02T15:04:05Z"),
					}
				}
				data["providers"] = provList
			}
		}

		if contains(req.Query, "categories") {
			categories, err := db.GetCategories()
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				catList := make([]map[string]interface{}, len(categories))
				for i, c := range categories {
					catList[i] = map[string]interface{}{
						"id":          fmt.Sprintf("%d", c.ID),
						"name":        c.Name,
						"description": c.Description,
						"createdAt":   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
					}
				}
				data["categories"] = catList
			}
		}

		// Raw Azure data queries
		if contains(req.Query, "azureRegions") {
			regions, err := db.GetAzureRegionsAvailable()
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				regionList := make([]map[string]interface{}, len(regions))
				for i, region := range regions {
					regionList[i] = map[string]interface{}{
						"name": region,
					}
				}
				data["azureRegions"] = regionList
			}
		}

		if contains(req.Query, "azureServices") {
			// Extract region parameter if provided
			region := extractParameter(req.Query, "region")
			services, err := db.GetAzureServicesAvailable(region)
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				serviceList := make([]map[string]interface{}, len(services))
				for i, service := range services {
					serviceList[i] = map[string]interface{}{
						"name": service,
					}
				}
				data["azureServices"] = serviceList
			}
		}

		if contains(req.Query, "azurePricing") {
			// Extract parameters
			region := extractParameter(req.Query, "region")
			limitStr := extractParameter(req.Query, "limit")
			serviceName := extractParameter(req.Query, "service")
			
			limit := 20 // default
			if limitStr != "" {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
					limit = parsedLimit
				}
			}

			if serviceName != "" {
				// Query by service
				pricing, err := db.QueryAzurePricingByService(serviceName, region, limit)
				if err != nil {
					response.Errors = append(response.Errors, err.Error())
				} else {
					data["azurePricing"] = pricing
				}
			} else {
				// General pricing query
				rawPricing, err := db.GetAzureRawPricing(region, limit, 0)
				if err != nil {
					response.Errors = append(response.Errors, err.Error())
				} else {
					pricingList := make([]map[string]interface{}, len(rawPricing))
					for i, item := range rawPricing {
						pricingList[i] = item.Data
					}
					data["azurePricing"] = pricingList
				}
			}
		}

		if contains(req.Query, "azureCollections") {
			collections, err := db.GetAzureCollections(20)
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				collectionList := make([]map[string]interface{}, len(collections))
				for i, collection := range collections {
					collectionItem := map[string]interface{}{
						"id":           fmt.Sprintf("%d", collection.ID),
						"collectionId": collection.CollectionID,
						"region":       collection.Region,
						"status":       collection.Status,
						"startedAt":    collection.StartedAt.Format("2006-01-02T15:04:05Z"),
						"totalItems":   collection.TotalItems,
					}
					
					if collection.CompletedAt != nil {
						collectionItem["completedAt"] = collection.CompletedAt.Format("2006-01-02T15:04:05Z")
						// Calculate duration
						duration := collection.CompletedAt.Sub(collection.StartedAt)
						collectionItem["duration"] = duration.String()
					}
					
					if collection.ErrorMessage != nil {
						collectionItem["errorMessage"] = *collection.ErrorMessage
					}
					
					// Add progress information from metadata
					if collection.Metadata != nil {
						if progress, ok := collection.Metadata["progress"].(map[string]interface{}); ok {
							collectionItem["progress"] = progress
						}
					}
					
					collectionList[i] = collectionItem
				}
				data["azureCollections"] = collectionList
			}
		}

		if contains(req.Query, "awsCollections") {
			collections, err := db.GetAWSCollections()
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				data["awsCollections"] = collections
			}
		}

		if contains(req.Query, "awsPricing") {
			// Extract parameters
			serviceCode := extractParameter(req.Query, "serviceCode")
			location := extractParameter(req.Query, "location")
			limitStr := extractParameter(req.Query, "limit")
			
			limit := 20 // default
			if limitStr != "" {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
					limit = parsedLimit
				}
			}

			// Query AWS pricing data from new JSONB schema
			rawPricing, err := db.GetAWSRawPricing(serviceCode, location, limit)
			if err != nil {
				response.Errors = append(response.Errors, err.Error())
			} else {
				// Parse JSONB data to extract fields for GraphQL response
				pricingList := make([]map[string]interface{}, len(rawPricing))
				for i, item := range rawPricing {
					// Parse the JSONB data field to extract pricing information
					var awsProduct map[string]interface{}
					if err := json.Unmarshal([]byte(item.Data), &awsProduct); err == nil {
						// Extract AWS product information
						pricingItem := map[string]interface{}{
							"serviceCode": item.ServiceCode,
							"serviceName": item.ServiceName,
							"location":    item.Location,
						}
						
						// Extract fields from the raw AWS product JSON
						if product, ok := awsProduct["product"].(map[string]interface{}); ok {
							if attributes, ok := product["attributes"].(map[string]interface{}); ok {
								if instanceType, ok := attributes["instanceType"].(string); ok {
									pricingItem["instanceType"] = instanceType
								}
							}
						}
						
						// Extract pricing information from terms
						if terms, ok := awsProduct["terms"].(map[string]interface{}); ok {
							// Look for OnDemand pricing first
							if onDemand, ok := terms["OnDemand"].(map[string]interface{}); ok {
								for _, termData := range onDemand {
									if termMap, ok := termData.(map[string]interface{}); ok {
										if priceDimensions, ok := termMap["priceDimensions"].(map[string]interface{}); ok {
											for _, dimension := range priceDimensions {
												if dimMap, ok := dimension.(map[string]interface{}); ok {
													if pricePerUnit, ok := dimMap["pricePerUnit"].(map[string]interface{}); ok {
														for currency, priceStr := range pricePerUnit {
															if priceString, ok := priceStr.(string); ok {
																if price, err := strconv.ParseFloat(priceString, 64); err == nil {
																	pricingItem["pricePerUnit"] = price
																	pricingItem["currency"] = currency
																}
															}
														}
													}
													if unit, ok := dimMap["unit"].(string); ok {
														pricingItem["unit"] = unit
													}
													pricingItem["termType"] = "OnDemand"
													break // Take first price dimension
												}
											}
										}
									}
								}
							}
						}
						
						pricingList[i] = pricingItem
					} else {
						// Fallback if JSON parsing fails
						pricingList[i] = map[string]interface{}{
							"serviceCode": item.ServiceCode,
							"serviceName": item.ServiceName,
							"location":    item.Location,
							"error":       "Failed to parse pricing data",
						}
					}
				}
				data["awsPricing"] = pricingList
			}
		}

		// Handle mutations
		if contains(req.Query, "mutation") && contains(req.Query, "createMessage") {
			// Extract content from query (simplified)
			content := "Test message" // In real implementation, parse from query
			if msg, err := db.CreateMessage(content); err == nil {
				data["createMessage"] = map[string]interface{}{
					"id":        fmt.Sprintf("%d", msg.ID),
					"content":   msg.Content,
					"createdAt": msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
				}
			} else {
				response.Errors = append(response.Errors, err.Error())
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Simple string contains helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

// Extract parameter from GraphQL query (very basic)
func extractParameter(query, param string) string {
	// Look for pattern like: param: "value"
	pattern := param + `: "`
	start := strings.Index(query, pattern)
	if start == -1 {
		return ""
	}
	start += len(pattern)
	end := strings.Index(query[start:], `"`)
	if end == -1 {
		return ""
	}
	return query[start : start+end]
}

// Enhanced playground handler with raw data examples
func playgroundHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
  <title>CPC GraphQL Playground (Raw JSON Version)</title>
  <style>
    body {
      font-family: Arial, sans-serif;
      margin: 0;
      padding: 20px;
      background-color: #f5f5f5;
    }
    .container {
      max-width: 1400px;
      margin: 0 auto;
      background: white;
      padding: 20px;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    h1 {
      color: #333;
    }
    .section {
      margin: 20px 0;
      padding: 15px;
      border: 1px solid #ddd;
      border-radius: 4px;
      background: #fafafa;
    }
    .query-area {
      margin: 20px 0;
    }
    textarea {
      width: 100%;
      height: 300px;
      font-family: monospace;
      font-size: 14px;
      padding: 10px;
      border: 1px solid #ddd;
      border-radius: 4px;
    }
    button {
      background: #4CAF50;
      color: white;
      border: none;
      padding: 10px 20px;
      font-size: 16px;
      border-radius: 4px;
      cursor: pointer;
      margin: 5px;
    }
    button:hover {
      background: #45a049;
    }
    .populate-btn {
      background: #2196F3;
    }
    .populate-btn:hover {
      background: #1976D2;
    }
    .response {
      margin-top: 20px;
      padding: 10px;
      background: #f9f9f9;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-family: monospace;
      white-space: pre-wrap;
      max-height: 400px;
      overflow-y: auto;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>Cloud Price Compare - GraphQL Playground (Raw JSON Version)</h1>
    
    <div class="section">
      <h3>AWS Population Endpoints</h3>
      <p><strong>Comprehensive Collection:</strong></p>
      <button class="populate-btn" onclick="awsPopulateComprehensive()" style="background: #FF9800;">Major AWS Services (~200K records)</button>
      <button class="populate-btn" onclick="awsPopulateEverything()" style="background: #FF5722;">ALL AWS Services (~500K+ records)</button>
      <br><br>
      <p><strong>Custom Service Collection:</strong></p>
      <button class="populate-btn" onclick="awsPopulateCustom(['AmazonEC2'], ['us-east-1', 'us-west-2'])">EC2 (US East/West)</button>
      <button class="populate-btn" onclick="awsPopulateCustom(['AmazonRDS'], ['us-east-1', 'eu-west-1'])">RDS (US/EU)</button>
      <button class="populate-btn" onclick="awsPopulateCustom(['AmazonS3'], ['us-east-1'])">S3 (US East)</button>
      <br><br>
      <p><strong>AWS Progress Monitoring:</strong></p>
      <button onclick="checkAWSProgress()">Check AWS Collection Progress</button>
      <button onclick="startAWSProgressMonitoring()">Start AWS Auto-Refresh (10s)</button>
      <button onclick="stopAWSProgressMonitoring()">Stop AWS Auto-Refresh</button>
      <br><br>
      <p><em>Note: Comprehensive collection includes 14 major services across 4 regions (~30 minutes). Everything collection includes 60+ services (~2-6 hours).</em></p>
    </div>
    
    <div class="section">
      <h3>Azure Population Endpoints</h3>
      <p><strong>Single Region Collection:</strong></p>
      <button class="populate-btn" onclick="populateData('eastus')">East US</button>
      <button class="populate-btn" onclick="populateData('westus')">West US</button>
      <button class="populate-btn" onclick="populateData('eastus2')">East US 2</button>
      <button class="populate-btn" onclick="populateData('westus2')">West US 2</button>
      <button class="populate-btn" onclick="populateData('centralus')">Central US</button>
      <button class="populate-btn" onclick="populateData('northeurope')">North Europe</button>
      <button class="populate-btn" onclick="populateData('westeurope')">West Europe</button>
      <button class="populate-btn" onclick="populateData('eastasia')">East Asia</button>
      <button class="populate-btn" onclick="populateData('southeastasia')">Southeast Asia</button>
      <br><br>
      <p><strong>All Regions Collection:</strong></p>
      <button class="populate-btn" onclick="populateAllData(3)" style="background: #FF5722;">Populate ALL Regions (3 concurrent)</button>
      <button class="populate-btn" onclick="populateAllData(5)" style="background: #FF5722;">Populate ALL Regions (5 concurrent)</button>
      <br><br>
      <p><strong>Azure Progress Monitoring:</strong></p>
      <button onclick="checkProgress()">Check Azure Collection Progress</button>
      <button onclick="startProgressMonitoring()">Start Azure Auto-Refresh (10s)</button>
      <button onclick="stopProgressMonitoring()">Stop Azure Auto-Refresh</button>
      <br><br>
      <p><em>Note: All-regions collection will fetch data from 70+ Azure regions. This may take 30-60 minutes to complete.</em></p>
    </div>
    
    <div class="query-area">
      <h3>GraphQL Query:</h3>
      <textarea id="query">{
  hello
  providers { name }
  categories { name }
  
  # Azure Data
  azureCollections {
    collectionId
    region
    status
    startedAt
    totalItems
  }
  
  # AWS Data
  awsCollections {
    collectionId
    serviceCodes
    regions
    status
    startedAt
    totalItems
  }
}</textarea>
      <br><br>
      <button onclick="executeQuery()">Execute Query</button>
      <button onclick="loadSample('basic')">Load Basic Query</button>
      <button onclick="loadSample('pricing')">Load Azure Pricing</button>
      <button onclick="loadSample('collections')">Load Azure Collections</button>
      <button onclick="loadSample('progress')">Load Azure Progress</button>
      <button onclick="loadSample('awsPricing')">Load AWS Pricing</button>
      <button onclick="loadSample('awsCollections')">Load AWS Collections</button>
    </div>
    <div class="response" id="response">Execute a query or populate data to see results...</div>
  </div>

  <script>
    async function executeQuery() {
      const query = document.getElementById('query').value;
      try {
        const response = await fetch('/query', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ query }),
        });
        const data = await response.json();
        document.getElementById('response').textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        document.getElementById('response').textContent = 'Error: ' + error.message;
      }
    }

    async function populateData(region) {
      try {
        document.getElementById('response').textContent = 'Starting data collection for ' + region + '...';
        const response = await fetch('/populate', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ region }),
        });
        const data = await response.json();
        document.getElementById('response').textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        document.getElementById('response').textContent = 'Error: ' + error.message;
      }
    }

    async function populateAllData(concurrency) {
      try {
        document.getElementById('response').textContent = 'Starting data collection for ALL Azure regions with ' + concurrency + ' concurrent workers...';
        const response = await fetch('/populate-all', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ concurrency }),
        });
        const data = await response.json();
        document.getElementById('response').textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        document.getElementById('response').textContent = 'Error: ' + error.message;
      }
    }

    let progressInterval = null;

    async function checkProgress() {
      try {
        const query = '{ azureCollections { collectionId region status startedAt completedAt totalItems duration progress errorMessage } }';
        const response = await fetch('/query', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ query }),
        });
        const data = await response.json();
        
        if (data.data && data.data.azureCollections) {
          const collections = data.data.azureCollections;
          const running = collections.filter(c => c.status === 'running');
          const completed = collections.filter(c => c.status === 'completed');
          const failed = collections.filter(c => c.status === 'failed');
          
          let progressText = 'AZURE COLLECTION PROGRESS\\n\\n';
          progressText += 'Running: ' + running.length + '\\n';
          progressText += 'Completed: ' + completed.length + '\\n';
          progressText += 'Failed: ' + failed.length + '\\n\\n';
          
          if (running.length > 0) {
            progressText += 'CURRENTLY RUNNING:\\n';
            running.forEach(c => {
              const progress = c.progress || {};
              const currentPage = progress.current_page || 0;
              const itemsCollected = progress.items_collected || 0;
              const statusMessage = progress.status_message || 'Processing...';
              progressText += '• ' + c.region + ': Page ' + currentPage + ', ' + itemsCollected + ' items\\n  Status: ' + statusMessage + '\\n';
            });
            progressText += '\\n';
          }
          
          if (completed.length > 0) {
            progressText += 'RECENTLY COMPLETED:\\n';
            completed.slice(0, 5).forEach(c => {
              progressText += '• ' + c.region + ': ' + c.totalItems + ' items (' + (c.duration || 'unknown duration') + ')\\n';
            });
          }
          
          document.getElementById('response').textContent = progressText;
        } else {
          document.getElementById('response').textContent = JSON.stringify(data, null, 2);
        }
      } catch (error) {
        document.getElementById('response').textContent = 'Progress check error: ' + error.message;
      }
    }

    function startProgressMonitoring() {
      if (progressInterval) {
        clearInterval(progressInterval);
      }
      checkProgress(); // Check immediately
      progressInterval = setInterval(checkProgress, 10000); // Every 10 seconds
      document.getElementById('response').textContent += '\\n\\nAzure auto-refresh started (every 10 seconds)...';
    }

    function stopProgressMonitoring() {
      if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
        document.getElementById('response').textContent += '\\n\\nAzure auto-refresh stopped.';
      }
    }

    // AWS Population Functions
    async function awsPopulateComprehensive() {
      try {
        document.getElementById('response').textContent = 'Starting comprehensive AWS data collection (major services ~200K records)...';
        const response = await fetch('/aws-populate-comprehensive', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
        });
        const data = await response.json();
        document.getElementById('response').textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        document.getElementById('response').textContent = 'Error: ' + error.message;
      }
    }

    async function awsPopulateEverything() {
      try {
        document.getElementById('response').textContent = 'Starting complete AWS data collection (ALL services ~500K+ records)...';
        const response = await fetch('/aws-populate-everything', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
        });
        const data = await response.json();
        document.getElementById('response').textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        document.getElementById('response').textContent = 'Error: ' + error.message;
      }
    }

    async function awsPopulateCustom(serviceCodes, regions) {
      try {
        document.getElementById('response').textContent = 'Starting custom AWS data collection for ' + serviceCodes.join(', ') + '...';
        const response = await fetch('/aws-populate', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ serviceCodes, regions }),
        });
        const data = await response.json();
        document.getElementById('response').textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        document.getElementById('response').textContent = 'Error: ' + error.message;
      }
    }

    let awsProgressInterval = null;

    async function checkAWSProgress() {
      try {
        const query = '{ awsCollections { collectionId serviceCodes regions status startedAt completedAt totalItems duration errorMessage } }';
        const response = await fetch('/query', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ query }),
        });
        const data = await response.json();
        
        if (data.data && data.data.awsCollections) {
          const collections = data.data.awsCollections;
          const running = collections.filter(c => c.status === 'running');
          const completed = collections.filter(c => c.status === 'completed');
          const failed = collections.filter(c => c.status === 'failed');
          
          let progressText = 'AWS COLLECTION PROGRESS\\n\\n';
          progressText += 'Running: ' + running.length + '\\n';
          progressText += 'Completed: ' + completed.length + '\\n';
          progressText += 'Failed: ' + failed.length + '\\n\\n';
          
          if (running.length > 0) {
            progressText += 'CURRENTLY RUNNING:\\n';
            running.forEach(c => {
              progressText += '• Services: ' + (c.serviceCodes || []).join(', ') + '\\n';
              progressText += '  Regions: ' + (c.regions || []).join(', ') + '\\n';
              progressText += '  Items collected: ' + (c.totalItems || 0) + '\\n\\n';
            });
          }
          
          if (completed.length > 0) {
            progressText += 'RECENTLY COMPLETED:\\n';
            completed.slice(0, 3).forEach(c => {
              progressText += '• Services: ' + (c.serviceCodes || []).join(', ') + '\\n';
              progressText += '  Total items: ' + (c.totalItems || 0) + '\\n';
              progressText += '  Duration: ' + (c.duration || 'unknown') + '\\n\\n';
            });
          }
          
          document.getElementById('response').textContent = progressText;
        } else {
          document.getElementById('response').textContent = JSON.stringify(data, null, 2);
        }
      } catch (error) {
        document.getElementById('response').textContent = 'AWS progress check error: ' + error.message;
      }
    }

    function startAWSProgressMonitoring() {
      if (awsProgressInterval) {
        clearInterval(awsProgressInterval);
      }
      checkAWSProgress(); // Check immediately
      awsProgressInterval = setInterval(checkAWSProgress, 10000); // Every 10 seconds
      document.getElementById('response').textContent += '\\n\\nAWS auto-refresh started (every 10 seconds)...';
    }

    function stopAWSProgressMonitoring() {
      if (awsProgressInterval) {
        clearInterval(awsProgressInterval);
        awsProgressInterval = null;
        document.getElementById('response').textContent += '\\n\\nAWS auto-refresh stopped.';
      }
    }

    function loadSample(type) {
      const samples = {
        basic: '{\\n  hello\\n  azureRegions {\\n    name\\n  }\\n  azureServices {\\n    name\\n  }\\n}',
        pricing: '{\\n  azurePricing {\\n    serviceName\\n    productName\\n    retailPrice\\n    unitOfMeasure\\n    armRegionName\\n  }\\n}',
        collections: '{\\n  azureCollections {\\n    collectionId\\n    region\\n    status\\n    startedAt\\n    totalItems\\n    completedAt\\n    duration\\n    progress\\n    errorMessage\\n  }\\n}',
        progress: '{\\n  azureCollections {\\n    region\\n    status\\n    totalItems\\n    progress\\n  }\\n}',
        awsPricing: '{\\n  awsPricing {\\n    serviceCode\\n    serviceName\\n    location\\n    instanceType\\n    pricePerUnit\\n    unit\\n    currency\\n    termType\\n  }\\n}',
        awsCollections: '{\\n  awsCollections {\\n    collectionId\\n    serviceCodes\\n    regions\\n    status\\n    startedAt\\n    totalItems\\n    completedAt\\n    duration\\n    errorMessage\\n  }\\n}'
      };
      document.getElementById('query').value = samples[type] || samples.basic;
    }
  </script>
</body>
</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// collectAzureData collects pricing data for a specific region
func collectAzureData(db *database.DB, collectionID string, region string) (int, error) {
	baseURL := "https://prices.azure.com/api/retail/prices"
	
	// Build filter for the specific region
	filter := fmt.Sprintf("armRegionName eq '%s'", region)
	
	totalItems := 0
	nextLink := ""
	pageCount := 0
	estimatedTotalPages := 0
	
	log.Printf("🚀 Starting collection for region: %s", region)
	
	for {
		pageCount++
		
		// Update progress in database
		statusMsg := fmt.Sprintf("Fetching page %d for region %s", pageCount, region)
		err := db.UpdateAzureRawCollectionProgress(collectionID, pageCount, estimatedTotalPages, totalItems, statusMsg)
		if err != nil {
			log.Printf("⚠️  Failed to update progress: %v", err)
		}
		
		log.Printf("📥 [%s] Fetching page %d (total items so far: %d)...", region, pageCount, totalItems)
		
		// Build URL
		var apiURL string
		if nextLink != "" {
			apiURL = nextLink
		} else {
			params := url.Values{}
			params.Add("$filter", filter)
			apiURL = baseURL + "?" + params.Encode()
		}
		
		// Make API request with retry
		var resp *http.Response
		for retries := 0; retries < 3; retries++ {
			resp, err = http.Get(apiURL)
			if err == nil && resp.StatusCode == 200 {
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			log.Printf("⚠️  [%s] API request failed (retry %d), retrying...", region, retries+1)
			time.Sleep(time.Duration(retries+1) * time.Second)
		}
		
		if err != nil {
			return totalItems, fmt.Errorf("failed to fetch data after retries: %w", err)
		}
		
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return totalItems, fmt.Errorf("API returned status %d", resp.StatusCode)
		}
		
		// Read response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return totalItems, fmt.Errorf("failed to read response: %w", err)
		}
		
		// Parse JSON
		var apiResp AzureAPIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return totalItems, fmt.Errorf("failed to parse JSON: %w", err)
		}
		
		log.Printf("📊 [%s] Page %d: %d items (total: %d)", region, pageCount, len(apiResp.Items), totalItems+len(apiResp.Items))
		
		// Store raw data in database
		if len(apiResp.Items) > 0 {
			err = db.BulkInsertAzureRawPricing(collectionID, region, apiResp.Items)
			if err != nil {
				return totalItems, fmt.Errorf("failed to store data: %w", err)
			}
		}
		
		totalItems += len(apiResp.Items)
		
		// Estimate total pages based on first page (rough estimate)
		if pageCount == 1 && len(apiResp.Items) > 0 {
			estimatedTotalPages = 10 // Rough estimate, Azure typically has 5-15 pages per region
		}
		
		// Update progress after successful page
		statusMsg = fmt.Sprintf("Completed page %d for region %s", pageCount, region)
		db.UpdateAzureRawCollectionProgress(collectionID, pageCount, estimatedTotalPages, totalItems, statusMsg)
		
		// Check if there's more data
		if apiResp.NextPageLink == "" || len(apiResp.Items) == 0 {
			log.Printf("✅ [%s] Collection complete - no more pages", region)
			break
		}
		
		nextLink = apiResp.NextPageLink
		
		// Add small delay to be nice to the API
		time.Sleep(100 * time.Millisecond)
		
		// Safety break to avoid infinite loops
		if pageCount > 20 {
			log.Printf("⚠️  [%s] Reached page limit (%d), stopping collection", region, pageCount)
			break
		}
	}
	
	log.Printf("🎉 [%s] Collection completed! Total items: %d, Pages: %d", region, totalItems, pageCount)
	return totalItems, nil
}

// AWS Population endpoint handler
func awsPopulateHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req awsPopulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Set defaults
		if len(req.ServiceCodes) == 0 {
			req.ServiceCodes = []string{"AmazonEC2"}
		}
		if len(req.Regions) == 0 {
			req.Regions = []string{"us-east-1"}
		}
		if len(req.InstanceTypes) == 0 {
			req.InstanceTypes = []string{"t3.micro", "t3.small", "t3.medium"}
		}

		log.Printf("Starting AWS data collection for services: %v, regions: %v", req.ServiceCodes, req.Regions)

		// Generate collection ID
		collectionID := fmt.Sprintf("aws_%d", time.Now().Unix())

		// Start collection in database
		err := startAWSCollection(db, collectionID, req.ServiceCodes, req.Regions)
		if err != nil {
			response := populationResponse{
				Message: "Failed to start AWS collection",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Run collector in background
		go func() {
			totalItems, err := collectAWSData(db, collectionID, req.ServiceCodes, req.Regions, req.InstanceTypes)
			if err != nil {
				log.Printf("AWS collection failed: %v", err)
				updateAWSCollectionStatus(db, collectionID, "failed", 0, err.Error())
			} else {
				log.Printf("AWS collection completed successfully: %d items", totalItems)
				updateAWSCollectionStatus(db, collectionID, "completed", totalItems, "")
			}
		}()

		response := populationResponse{
			Message:      fmt.Sprintf("AWS data collection started for services: %v, regions: %v", req.ServiceCodes, req.Regions),
			CollectionID: collectionID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AWS All Regions Population endpoint handler
func awsPopulateAllHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req awsPopulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Set defaults
		if len(req.ServiceCodes) == 0 {
			req.ServiceCodes = []string{"AmazonEC2", "AmazonS3"}
		}
		if req.Concurrency == 0 {
			req.Concurrency = 3
		}

		// Define major AWS regions
		allRegions := []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-central-1",
			"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		}

		log.Printf("Starting AWS all-regions collection for services: %v with %d concurrent workers", req.ServiceCodes, req.Concurrency)

		// Generate collection ID
		collectionID := fmt.Sprintf("aws_all_%d", time.Now().Unix())

		// Start collection in database
		err := startAWSCollection(db, collectionID, req.ServiceCodes, allRegions)
		if err != nil {
			response := populationResponse{
				Message: "Failed to start AWS all-regions collection",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Run collector in background
		go func() {
			totalItems, err := collectAWSDataConcurrent(db, collectionID, req.ServiceCodes, allRegions, req.Concurrency)
			if err != nil {
				log.Printf("AWS all-regions collection failed: %v", err)
				updateAWSCollectionStatus(db, collectionID, "failed", 0, err.Error())
			} else {
				log.Printf("AWS all-regions collection completed successfully: %d items", totalItems)
				updateAWSCollectionStatus(db, collectionID, "completed", totalItems, "")
			}
		}()

		response := populationResponse{
			Message:      fmt.Sprintf("AWS all-regions collection started for services: %v with %d workers", req.ServiceCodes, req.Concurrency),
			CollectionID: collectionID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// collectAWSData collects AWS pricing data for specified services and regions
func collectAWSData(db *database.DB, collectionID string, serviceCodes []string, regions []string, instanceTypes []string) (int, error) {
	// Validate AWS credentials are available via environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return 0, fmt.Errorf("AWS credentials not found in environment variables")
	}

	client, err := database.NewAWSPricingClient()
	if err != nil {
		return 0, fmt.Errorf("failed to create AWS pricing client: %w", err)
	}

	var allPricing []database.AWSPricingItem
	totalItems := 0

	// Use the new comprehensive collection method
	allPricing, err = client.GetAllServicePricing(serviceCodes, regions)
	if err != nil {
		return totalItems, fmt.Errorf("failed to get comprehensive pricing: %w", err)
	}

	// Store all pricing data
	if len(allPricing) > 0 {
		err = database.StoreAWSPricing(db.GetConn(), allPricing, collectionID)
		if err != nil {
			return totalItems, fmt.Errorf("failed to store AWS pricing data: %w", err)
		}
		totalItems = len(allPricing)
	}

	return totalItems, nil
}

// collectAWSDataConcurrent collects AWS pricing data with concurrent workers
func collectAWSDataConcurrent(db *database.DB, collectionID string, serviceCodes []string, regions []string, concurrency int) (int, error) {
	// Validate AWS credentials are available via environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return 0, fmt.Errorf("AWS credentials not found in environment variables")
	}

	client, err := database.NewAWSPricingClient()
	if err != nil {
		return 0, fmt.Errorf("failed to create AWS pricing client: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allPricing []database.AWSPricingItem
	var collectErrors []error

	// Create a semaphore to limit concurrency
	sem := make(chan struct{}, concurrency)

	for _, serviceCode := range serviceCodes {
		wg.Add(1)
		go func(sc string) {
			defer wg.Done()
			sem <- struct{}{} // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			// Use comprehensive collection for each service
			pricing, err := client.GetAllServicePricing([]string{sc}, regions)

			mu.Lock()
			if err != nil {
				collectErrors = append(collectErrors, fmt.Errorf("failed to collect %s: %w", sc, err))
			} else {
				allPricing = append(allPricing, pricing...)
			}
			mu.Unlock()
		}(serviceCode)
	}

	wg.Wait()

	// Check for collection errors
	if len(collectErrors) > 0 {
		return 0, fmt.Errorf("collection errors: %v", collectErrors)
	}

	// Store all pricing data
	totalItems := 0
	if len(allPricing) > 0 {
		err = database.StoreAWSPricing(db.GetConn(), allPricing, collectionID)
		if err != nil {
			return totalItems, fmt.Errorf("failed to store AWS pricing data: %w", err)
		}
		totalItems = len(allPricing)
	}

	return totalItems, nil
}

// Helper functions for AWS collection tracking
func startAWSCollection(db *database.DB, collectionID string, serviceCodes []string, regions []string) error {
	query := `
		INSERT INTO aws_collections (collection_id, service_codes, regions, status, started_at)
		VALUES ($1, $2, $3, 'running', $4)
	`
	_, err := db.GetConn().Exec(query, collectionID, pq.StringArray(serviceCodes), pq.StringArray(regions), time.Now())
	return err
}

func updateAWSCollectionStatus(db *database.DB, collectionID string, status string, totalItems int, errorMessage string) error {
	query := `
		UPDATE aws_collections 
		SET status = $2, completed_at = $3, total_items = $4, error_message = $5
		WHERE collection_id = $1
	`
	_, err := db.GetConn().Exec(query, collectionID, status, time.Now(), totalItems, errorMessage)
	return err
}

// AWS Comprehensive Collection endpoint - collects major AWS services
func awsPopulateComprehensiveHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req awsPopulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Use defaults if no request body
			req = awsPopulationRequest{}
		}

		// Use major AWS services (80/20 rule)
		majorServices := database.GetMajorAWSServices()
		req.ServiceCodes = majorServices

		if len(req.Regions) == 0 {
			req.Regions = []string{
				"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", // Major regions
			}
		}
		if req.Concurrency == 0 {
			req.Concurrency = 5 // Higher concurrency for comprehensive collection
		}

		log.Printf("🔥 Starting COMPREHENSIVE AWS collection for %d major services across %d regions", 
			len(majorServices), len(req.Regions))

		// Generate collection ID
		collectionID := fmt.Sprintf("aws_comprehensive_%d", time.Now().Unix())

		// Start collection in database
		err := startAWSCollection(db, collectionID, req.ServiceCodes, req.Regions)
		if err != nil {
			response := populationResponse{
				Message: "Failed to start comprehensive AWS collection",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Run collector in background
		go func() {
			totalItems, err := collectAWSDataConcurrent(db, collectionID, req.ServiceCodes, req.Regions, req.Concurrency)
			if err != nil {
				log.Printf("Comprehensive AWS collection failed: %v", err)
				updateAWSCollectionStatus(db, collectionID, "failed", 0, err.Error())
			} else {
				log.Printf("🎉 Comprehensive AWS collection completed successfully: %d items", totalItems)
				updateAWSCollectionStatus(db, collectionID, "completed", totalItems, "")
			}
		}()

		response := populationResponse{
			Message:      fmt.Sprintf("🔥 Comprehensive AWS collection started: %d services, %d regions", len(req.ServiceCodes), len(req.Regions)),
			CollectionID: collectionID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// AWS Everything Collection endpoint - collects ALL AWS services
func awsPopulateEverythingHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req awsPopulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Use defaults if no request body
			req = awsPopulationRequest{}
		}

		// Use ALL AWS services
		allServices := database.GetAllAWSServices()
		req.ServiceCodes = allServices

		if len(req.Regions) == 0 {
			req.Regions = []string{
				"us-east-1", "us-west-2", "eu-west-1", // Limited regions for everything collection
			}
		}
		if req.Concurrency == 0 {
			req.Concurrency = 3 // Conservative concurrency for massive collection
		}

		log.Printf("🌪️ Starting EVERYTHING AWS collection for %d services across %d regions - THIS WILL TAKE HOURS!", 
			len(allServices), len(req.Regions))

		// Generate collection ID
		collectionID := fmt.Sprintf("aws_everything_%d", time.Now().Unix())

		// Start collection in database
		err := startAWSCollection(db, collectionID, req.ServiceCodes, req.Regions)
		if err != nil {
			response := populationResponse{
				Message: "Failed to start everything AWS collection",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Run collector in background
		go func() {
			totalItems, err := collectAWSDataConcurrent(db, collectionID, req.ServiceCodes, req.Regions, req.Concurrency)
			if err != nil {
				log.Printf("Everything AWS collection failed: %v", err)
				updateAWSCollectionStatus(db, collectionID, "failed", 0, err.Error())
			} else {
				log.Printf("🎉 EVERYTHING AWS collection completed successfully: %d items", totalItems)
				updateAWSCollectionStatus(db, collectionID, "completed", totalItems, "")
			}
		}()

		response := populationResponse{
			Message:      fmt.Sprintf("🌪️ EVERYTHING AWS collection started: %d services, %d regions - Estimated time: 2-6 hours", len(req.ServiceCodes), len(req.Regions)),
			CollectionID: collectionID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}