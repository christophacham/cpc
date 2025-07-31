# Development Setup

Get your local development environment up and running.

## Prerequisites

### Required Software
- **Go 1.24+** - [Download](https://golang.org/dl/)
- **Docker Desktop** - [Download](https://www.docker.com/products/docker-desktop/)
- **Git** - [Download](https://git-scm.com/downloads)

### Optional Tools
- **PostgreSQL Client** - For direct database access
- **cURL** - For API testing
- **Postman** - Alternative API testing

## Project Setup

### 1. Clone Repository
```bash
git clone https://github.com/raulc0399/cpc.git
cd cpc
```

### 2. Initialize Go Module
```bash
go mod tidy
```

### 3. Start Database
```bash
docker-compose up -d postgres
```

### 4. Verify Database Connection
```bash
docker-compose logs postgres
```

## Running the Application

### Start API Server
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

### Collect Sample Data
```bash
# Explore Azure services
go run cmd/azure-explorer/main.go

# Collect and store pricing data
go run cmd/azure-db-collector/main.go
```

## Development Workflow

### Project Structure
```
cpc/
├── cmd/                    # Application entry points
│   ├── server/            # GraphQL API server
│   ├── azure-explorer/    # Azure API exploration
│   ├── azure-collector/   # Data collection tools
│   └── azure-db-collector/# Database population
├── internal/              # Internal packages
│   └── database/         # Database operations
├── docs/                 # Project documentation
├── docs-site/           # Docusaurus documentation
├── docker-compose.yml   # Docker services
├── init.sql            # Database schema
└── go.mod              # Go dependencies
```

### Database Access
Connect directly to PostgreSQL:
```bash
# Using docker exec
docker exec -it cpc-postgres-1 psql -U cpc_user -d cpc_db

# Using external client
psql -h localhost -p 5432 -U cpc_user -d cpc_db
```

### Environment Variables
```bash
# Database connection (optional)
export DATABASE_URL="postgres://cpc_user:cpc_password@localhost:5432/cpc_db?sslmode=disable"

# API server port (optional)
export PORT="8080"
```

## IDE Configuration

### VS Code
Recommended extensions:
- Go extension by Google
- Docker extension
- GraphQL extension

### GoLand
Built-in Go support with excellent debugging capabilities.

## Troubleshooting

### Common Issues

**Port 8080 already in use:**
```bash
# Find process using port 8080
lsof -i :8080
# Kill the process
kill -9 <PID>
```

**Docker not running:**
- Start Docker Desktop
- Verify with `docker ps`

**Go command not found:**
- Ensure Go is in your PATH
- On Windows: Add `C:\Program Files\Go\bin` to PATH

**Database connection failed:**
- Check if PostgreSQL container is running
- Verify credentials in docker-compose.yml
- Check firewall settings

### Reset Database
```bash
# Stop and remove containers
docker-compose down

# Remove volumes (WARNING: deletes all data)
docker-compose down -v

# Restart fresh
docker-compose up -d postgres
```