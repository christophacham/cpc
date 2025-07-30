# CPC - Cloud Price Compare

A production-grade API service that aggregates, normalizes, and serves pricing data from AWS and Azure through a unified GraphQL API.

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Make (optional)

### Running with Docker Compose

1. Start the services:
```bash
docker-compose up -d
```

2. Access the GraphQL playground:
```
http://localhost:8080
```

3. Try a simple query:
```graphql
query {
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
}
```

4. Create a message:
```graphql
mutation {
  createMessage(content: "Testing the GraphQL API!") {
    id
    content
    createdAt
  }
}
```

### Development Setup

1. Install dependencies:
```bash
go mod download
```

2. Run database migrations (handled by init.sql on first start)

3. Run the server:
```bash
go run cmd/server/main.go
```

### Available Make Commands

```bash
make help        # Show available commands
make run         # Run the application locally
make build       # Build the application
make docker-up   # Start Docker containers
make docker-down # Stop Docker containers
make docker-logs # View Docker logs
```

## Project Structure

```
cpc/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── database/
│   │   └── database.go      # Database operations
│   └── graph/
│       ├── schema.graphql   # GraphQL schema definition
│       ├── resolver.go      # GraphQL resolvers
│       ├── model.go         # GraphQL models
│       └── generated.go     # Generated GraphQL code
├── docker-compose.yml       # Docker Compose configuration
├── Dockerfile              # Docker image definition
├── init.sql               # Initial database schema
├── go.mod                 # Go module definition
├── Makefile              # Build commands
└── README.md             # This file
```

## Next Steps

- [ ] Implement AWS pricing data collection
- [ ] Implement Azure pricing data collection
- [ ] Add comprehensive GraphQL schema for pricing data
- [ ] Set up data normalization pipeline
- [ ] Add caching layer
- [ ] Deploy to AWS infrastructure
