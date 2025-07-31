package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
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

// Population response structure
type populationResponse struct {
	Message      string `json:"message"`
	CollectionID string `json:"collectionId,omitempty"`
	Error        string `json:"error,omitempty"`
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
			cmd := exec.Command("go", "run", "cmd/azure-raw-collector/main.go", req.Region)
			if err := cmd.Run(); err != nil {
				log.Printf("Collection failed for %s: %v", collectionID, err)
				db.FailAzureRawCollection(collectionID, err.Error())
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
			cmd := exec.Command("go", "run", "cmd/azure-all-regions/main.go", strconv.Itoa(req.Concurrency))
			if err := cmd.Run(); err != nil {
				log.Printf("All-regions collection failed: %v", err)
			}
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
      <h3>🚀 Population Endpoints</h3>
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
      <button class="populate-btn" onclick="populateAllData(3)" style="background: #FF5722;">🌍 Populate ALL Regions (3 concurrent)</button>
      <button class="populate-btn" onclick="populateAllData(5)" style="background: #FF5722;">🌍 Populate ALL Regions (5 concurrent)</button>
      <br><br>
      <p><strong>Progress Monitoring:</strong></p>
      <button onclick="checkProgress()">📊 Check Collection Progress</button>
      <button onclick="startProgressMonitoring()">🔄 Start Auto-Refresh (10s)</button>
      <button onclick="stopProgressMonitoring()">⏹️ Stop Auto-Refresh</button>
      <br><br>
      <p><em>⚠️ All-regions collection will fetch data from 70+ Azure regions. This may take 30-60 minutes to complete.</em></p>
    </div>
    
    <div class="query-area">
      <h3>GraphQL Query:</h3>
      <textarea id="query">{
  hello
  azureRegions {
    name
  }
  azureServices {
    name
  }
  azureCollections {
    collectionId
    region
    status
    startedAt
    totalItems
  }
  azurePricing {
    serviceName
    productName
    skuName
    retailPrice
    unitOfMeasure
    armRegionName
  }
}</textarea>
      <br><br>
      <button onclick="executeQuery()">Execute Query</button>
      <button onclick="loadSample('basic')">Load Basic Query</button>
      <button onclick="loadSample('pricing')">Load Pricing Query</button>
      <button onclick="loadSample('collections')">Load Collections Query</button>
      <button onclick="loadSample('progress')">Load Progress Query</button>
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
          
          let progressText = '📊 COLLECTION PROGRESS\\n\\n';
          progressText += '🔄 Running: ' + running.length + '\\n';
          progressText += '✅ Completed: ' + completed.length + '\\n';
          progressText += '❌ Failed: ' + failed.length + '\\n\\n';
          
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
      document.getElementById('response').textContent += '\\n\\n🔄 Auto-refresh started (every 10 seconds)...';
    }

    function stopProgressMonitoring() {
      if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
        document.getElementById('response').textContent += '\\n\\n⏹️ Auto-refresh stopped.';
      }
    }

    function loadSample(type) {
      const samples = {
        basic: '{\\n  hello\\n  azureRegions {\\n    name\\n  }\\n  azureServices {\\n    name\\n  }\\n}',
        pricing: '{\\n  azurePricing {\\n    serviceName\\n    productName\\n    retailPrice\\n    unitOfMeasure\\n    armRegionName\\n  }\\n}',
        collections: '{\\n  azureCollections {\\n    collectionId\\n    region\\n    status\\n    startedAt\\n    totalItems\\n    completedAt\\n    duration\\n    progress\\n    errorMessage\\n  }\\n}',
        progress: '{\\n  azureCollections {\\n    region\\n    status\\n    totalItems\\n    progress\\n  }\\n}'
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