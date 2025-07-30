package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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

	log.Printf("Starting server on http://localhost:%s/", port)
	log.Printf("GraphQL playground available at http://localhost:%s/", port)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Simple GraphQL handler
func graphQLHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Simple query parsing (just for demo)
		response := graphQLResponse{
			Data: make(map[string]interface{}),
		}

		data := response.Data.(map[string]interface{})

		// Handle different queries
		if contains(req.Query, "hello") {
			data["hello"] = "Hello from Cloud Price Compare GraphQL API!"
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

// Simple playground handler
func playgroundHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
  <title>CPC GraphQL Playground</title>
  <style>
    body {
      font-family: Arial, sans-serif;
      margin: 0;
      padding: 20px;
      background-color: #f5f5f5;
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
      background: white;
      padding: 20px;
      border-radius: 8px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    }
    h1 {
      color: #333;
    }
    .query-area {
      margin: 20px 0;
    }
    textarea {
      width: 100%;
      height: 200px;
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
    }
    button:hover {
      background: #45a049;
    }
    .response {
      margin-top: 20px;
      padding: 10px;
      background: #f9f9f9;
      border: 1px solid #ddd;
      border-radius: 4px;
      font-family: monospace;
      white-space: pre-wrap;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>Cloud Price Compare - GraphQL Playground</h1>
    <div class="query-area">
      <h3>Query:</h3>
      <textarea id="query">{
  hello
  messages {
    id
    content
    createdAt
  }
  providers {
    id
    name
  }
  categories {
    id
    name
    description
  }
}</textarea>
      <br><br>
      <button onclick="executeQuery()">Execute Query</button>
    </div>
    <div class="response" id="response"></div>
  </div>

  <script>
    async function executeQuery() {
      const query = document.getElementById('query').value;
      const response = await fetch('/query', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ query }),
      });
      const data = await response.json();
      document.getElementById('response').textContent = JSON.stringify(data, null, 2);
    }
  </script>
</body>
</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}