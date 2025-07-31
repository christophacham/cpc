FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests to Azure API and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Set default environment variables
ENV DATABASE_URL="postgres://cpc_user:cpc_password@postgres:5432/cpc_db?sslmode=disable"
ENV PORT="8080"

# Run the application
CMD ["./server"]