# Docker Guide

Using Docker for development and production deployment.

## Development Setup

### Docker Compose Services

The project uses Docker Compose to manage services:

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: cpc_db
      POSTGRES_USER: cpc_user
      POSTGRES_PASSWORD: cpc_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql

  docs:
    build: ./docs-site
    ports:
      - "3000:3000"
    volumes:
      - ./docs-site:/app
      - /app/node_modules
```

### Basic Commands

```bash
# Start all services
docker-compose up -d

# Start specific service
docker-compose up -d postgres

# View logs
docker-compose logs postgres
docker-compose logs docs

# Stop services
docker-compose down

# Remove volumes (deletes data)
docker-compose down -v

# Rebuild services
docker-compose build
docker-compose up -d --build
```

## Documentation Site

### Building the Docs
```bash
# Build docs service
docker-compose build docs

# Start docs site
docker-compose up -d docs
```

The documentation will be available at [http://localhost:3000](http://localhost:3000)

### Development Mode
```bash
# For live reload during development
cd docs-site
npm install
npm start
```

## Database Management

### Accessing PostgreSQL
```bash
# Using docker exec
docker exec -it cpc-postgres-1 psql -U cpc_user -d cpc_db

# Using docker-compose
docker-compose exec postgres psql -U cpc_user -d cpc_db
```

### Database Backup
```bash
# Create backup
docker exec cpc-postgres-1 pg_dump -U cpc_user cpc_db > backup.sql

# Restore backup
docker exec -i cpc-postgres-1 psql -U cpc_user -d cpc_db < backup.sql
```

### Reset Database
```bash
# Stop and remove database
docker-compose down
docker volume rm cpc_postgres_data

# Start fresh
docker-compose up -d postgres
```

## Production Deployment

### Environment Variables
Create a `.env` file for production:

```env
# Database
POSTGRES_DB=cpc_db
POSTGRES_USER=cpc_user
POSTGRES_PASSWORD=secure_password_here

# API
DATABASE_URL=postgres://cpc_user:secure_password_here@postgres:5432/cpc_db?sslmode=disable
PORT=8080

# Documentation
NODE_ENV=production
```

### Production Docker Compose
```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped

  api:
    build: .
    environment:
      DATABASE_URL: ${DATABASE_URL}
      PORT: ${PORT}
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    restart: unless-stopped

  docs:
    build: ./docs-site
    ports:
      - "3000:3000"
    restart: unless-stopped

volumes:
  postgres_data:
```

### API Server Dockerfile
Create `Dockerfile` in project root:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

## Container Management

### Health Checks
```bash
# Check container status
docker-compose ps

# View resource usage
docker stats

# Check container health
docker inspect cpc-postgres-1 | grep Health -A 10
```

### Debugging
```bash
# Execute commands in running container
docker exec -it cpc-postgres-1 bash

# View container filesystem
docker exec -it cpc-postgres-1 ls -la /var/lib/postgresql/data

# Follow logs in real-time
docker-compose logs -f postgres
```

## Networking

### Container Communication
Containers communicate using service names:
- API connects to `postgres:5432`
- Documentation links to `api:8080`

### Port Mapping
- PostgreSQL: `localhost:5432` → `postgres:5432`
- API Server: `localhost:8080` → `api:8080`
- Documentation: `localhost:3000` → `docs:3000`

## Volumes and Data Persistence

### Database Data
```bash
# Inspect volume
docker volume inspect cpc_postgres_data

# Backup volume
docker run --rm -v cpc_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_backup.tar.gz -C /data .

# Restore volume
docker run --rm -v cpc_postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/postgres_backup.tar.gz -C /data
```

### Documentation Development
```bash
# Mount docs for live editing
docker run -it --rm -v $(pwd)/docs-site:/app -p 3000:3000 node:18 sh -c "cd /app && npm install && npm start"
```

## Troubleshooting

### Common Issues

**Port conflicts:**
```bash
# Find processes using ports
lsof -i :5432
lsof -i :3000
lsof -i :8080
```

**Container won't start:**
```bash
# Check logs
docker-compose logs service_name

# Inspect container
docker inspect container_name
```

**Database connection issues:**
```bash
# Test database connection
docker exec cpc-postgres-1 pg_isready -U cpc_user -d cpc_db

# Check database logs
docker-compose logs postgres
```

**Build failures:**
```bash
# Clean build
docker-compose down
docker system prune -f
docker-compose build --no-cache
```