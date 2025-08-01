package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
	"github.com/raulc0399/cpc/internal/database"
	"github.com/raulc0399/cpc/internal/etl"
	"github.com/raulc0399/cpc/internal/graph"
	"github.com/raulc0399/cpc/internal/graph/generated"
)

const defaultPort = "8080"

func main() {
	// Connect to database
	dbConn, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Test database connection
	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create database handler
	db := database.New(dbConn)

	// Create ETL pipeline
	pipeline, err := etl.NewPipeline(db)
	if err != nil {
		log.Fatalf("Failed to create ETL pipeline: %v", err)
	}

	// Create resolver
	resolver := &graph.Resolver{
		DB: db,
	}
	resolver.SetPipeline(pipeline)

	// Create GraphQL server with introspection enabled
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	// Set up routes
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	// Add the population endpoints from the original server
	http.HandleFunc("/populate", populateHandler(db))
	http.HandleFunc("/populate-all", populateAllHandler(db))
	http.HandleFunc("/aws-populate", awsPopulateHandler(db))
	http.HandleFunc("/aws-populate-all", awsPopulateAllHandler(db))
	http.HandleFunc("/aws-populate-comprehensive", awsPopulateComprehensiveHandler(db))
	http.HandleFunc("/aws-populate-everything", awsPopulateEverythingHandler(db))

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("Starting GraphQL server on http://localhost:%s/", port)
	log.Printf("GraphQL playground available at http://localhost:%s/", port)
	log.Printf("GraphQL introspection is enabled at http://localhost:%s/query", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Placeholder handlers - these would need to be implemented with the actual logic
// from the original server

func populateHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement from original server
		http.Error(w, "Not implemented yet", http.StatusNotImplemented)
	}
}

func populateAllHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement from original server
		http.Error(w, "Not implemented yet", http.StatusNotImplemented)
	}
}

func awsPopulateHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement from original server
		http.Error(w, "Not implemented yet", http.StatusNotImplemented)
	}
}

func awsPopulateAllHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement from original server
		http.Error(w, "Not implemented yet", http.StatusNotImplemented)
	}
}

func awsPopulateComprehensiveHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement from original server
		http.Error(w, "Not implemented yet", http.StatusNotImplemented)
	}
}

func awsPopulateEverythingHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement from original server
		http.Error(w, "Not implemented yet", http.StatusNotImplemented)
	}
}