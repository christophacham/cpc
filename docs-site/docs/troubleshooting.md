#  Troubleshooting Guide

This guide helps you resolve common issues when working with Cloud Price Compare (CPC).

##  Common Problems & Solutions

###  Docker & Deployment Issues

#### Container Won't Start

**Problem**: `docker-compose up -d` fails or containers crash immediately.

**Solution**:
```bash
# Check container status
docker-compose ps

# View detailed logs
docker-compose logs app
docker-compose logs postgres

# Check available resources
docker system df
docker stats

# Clean up if needed
docker system prune -f
docker-compose down -v
docker-compose up -d
```

#### Database Connection Failed

**Problem**: API server can't connect to PostgreSQL.

**Solution**:
```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check PostgreSQL logs
docker-compose logs postgres

# Test database connectivity
docker-compose exec postgres pg_isready

# Reset database if corrupted
docker-compose down -v
docker-compose up -d
```

#### Port Already in Use

**Problem**: `Error: bind: address already in use`.

**Solution**:
```bash
# Find process using the port
lsof -i :8080
lsof -i :5432
lsof -i :3000

# Kill the process (replace PID)
kill -9 &lt;PID&gt;

# Or use different ports in docker-compose.yml
ports:
  - "8081:8080"  # API on different port
```

###  API & GraphQL Issues

#### GraphQL Playground Not Loading

**Problem**: Blank screen or errors when accessing http://localhost:8080.

**Solution**:
```bash
# Check API server logs
docker-compose logs app

# Test API endpoint directly
curl http://localhost:8080/query -d '{"query": "{ hello }"}'

# Restart API server
docker-compose restart app

# Check browser console for errors
# Try incognito/private browsing mode
```

#### Query Timeout or Slow Responses

**Problem**: GraphQL queries take too long or timeout.

**Solution**:
```bash
# Check database performance
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"

# Check system resources
docker stats

# Add query limits
query {
  azurePricing(limit: 100) {  # Limit results
    serviceName
    retailPrice
  }
}

# Create database indexes for common queries
psql -h localhost -U postgres -d cpc \
  -c "CREATE INDEX IF NOT EXISTS idx_azure_service ON azure_pricing_raw(data->>'serviceName');"
```

#### Authentication Errors

**Problem**: AWS-related operations fail with authentication errors.

**Solution**:
```bash
# Verify AWS credentials in .env file
cat .env | grep AWS

# Test AWS credentials
aws sts get-caller-identity

# Restart services to pick up new credentials
docker-compose restart app

# Check credential format:
# AWS_ACCESS_KEY_ID=AKIA...
# AWS_SECRET_ACCESS_KEY=...
# AWS_REGION=us-east-1
```

###  Data Collection Issues

#### Azure Collection Stuck

**Problem**: Azure data collection never completes or gets stuck.

**Solution**:
```bash
# Check collection status
curl -s http://localhost:8080/query \
  -d '{"query": "{ azureCollections { region status totalItems progress errorMessage } }"}' | jq

# Check API server logs for errors
docker-compose logs app | grep -i azure

# Restart collection for specific region
curl -X POST http://localhost:8080/populate \
  -H "Content-Type: application/json" \
  -d '{"region": "eastus"}'

# Check network connectivity
curl -s "https://prices.azure.com/api/retail/prices?api-version=2023-01-01-preview" | head
```

#### AWS Collection Fails

**Problem**: AWS data collection fails immediately or returns no data.

**Solution**:
```bash
# Verify credentials are working
aws pricing describe-services --region us-east-1

# Check if service exists in AWS
aws pricing describe-services --region us-east-1 \
  --query 'Services[?ServiceCode==`AmazonEC2`]'

# Test with minimal collection
curl -X POST http://localhost:8080/aws-populate \
  -H "Content-Type: application/json" \
  -d '{"serviceCodes": ["AmazonEC2"], "regions": ["us-east-1"]}'

# Check API server logs
docker-compose logs app | grep -i aws
```

#### Rate Limiting Issues

**Problem**: Collection fails due to API rate limits.

**Solution**:
```bash
# For AWS: Use fewer concurrent workers
curl -X POST http://localhost:8080/aws-populate \
  -d '{"serviceCodes": ["AmazonEC2"], "regions": ["us-east-1"], "concurrency": 1}'

# For Azure: Reduce concurrency
curl -X POST http://localhost:8080/populate-all \
  -d '{"concurrency": 2}'  # Instead of 5

# Monitor rate limiting in logs
docker-compose logs app | grep -i "rate\|limit\|throttle"
```

###  ETL Pipeline Issues

#### ETL Job Stuck in PENDING

**Problem**: ETL job never starts processing.

**Solution**:
```bash
# Check ETL job status
curl -s http://localhost:8080/query \
  -d '{"query": "{ etlJobs { id status error } }"}' | jq

# Check database connections
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT count(*) FROM pg_stat_activity;"

# Check if raw data exists
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT count(*) FROM azure_pricing_raw;"

# Restart with smaller batch size
mutation {
  startNormalization(config: {
    type: NORMALIZE_ALL
    batchSize: 100      # Smaller batches
    concurrentWorkers: 1 # Fewer workers
    dryRun: true        # Test mode first
  }) {
    id
    status
  }
}
```

#### ETL Job Fails with Errors

**Problem**: ETL job moves to FAILED status.

**Solution**:
```bash
# Get detailed error information
curl -s http://localhost:8080/query \
  -d '{"query": "{ etlJob(id: \"your-job-id\") { status error progress { errorRecords } } }"}' | jq

# Check API server logs for detailed errors
docker-compose logs app | grep -i etl

# Check database locks
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT * FROM pg_locks WHERE NOT granted;"

# Test with dry run first
mutation {
  startNormalization(config: {
    type: NORMALIZE_PROVIDER
    providers: ["azure"]
    dryRun: true
  }) {
    id
    status
  }
}
```

#### Slow ETL Processing

**Problem**: ETL job processes records very slowly.

**Solution**:
```bash
# Monitor ETL performance
curl -s http://localhost:8080/query \
  -d '{"query": "{ etlJob(id: \"your-job-id\") { progress { processedRecords rate } } }"}' | jq

# Increase workers and batch size
mutation {
  startNormalization(config: {
    type: NORMALIZE_ALL
    batchSize: 2000      # Larger batches
    concurrentWorkers: 8  # More workers
  }) {
    id
  }
}

# Check system resources
docker stats
htop

# Optimize database
docker-compose exec postgres psql -U postgres -d cpc \
  -c "VACUUM ANALYZE;"
```

###  Database Issues

#### Database Corruption

**Problem**: Database queries fail or return inconsistent results.

**Solution**:
```bash
# Check database integrity
docker-compose exec postgres pg_dump -U postgres cpc > backup.sql

# Reset database completely
docker-compose down -v
docker volume prune -f
docker-compose up -d

# Or repair existing database
docker-compose exec postgres psql -U postgres -d cpc \
  -c "REINDEX DATABASE cpc;"

docker-compose exec postgres psql -U postgres -d cpc \
  -c "VACUUM FULL;"
```

#### Out of Disk Space

**Problem**: Database operations fail due to insufficient disk space.

**Solution**:
```bash
# Check disk usage
df -h
docker system df

# Clean up Docker resources
docker system prune -a -f
docker volume prune -f

# Clean up old data
docker-compose exec postgres psql -U postgres -d cpc \
  -c "DELETE FROM azure_pricing_raw WHERE created_at &lt; NOW() - INTERVAL '30 days';"

# Vacuum to reclaim space
docker-compose exec postgres psql -U postgres -d cpc \
  -c "VACUUM FULL;"
```

#### Database Connection Pool Exhausted

**Problem**: "remaining connection slots are reserved" errors.

**Solution**:
```bash
# Check active connections
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT count(*) FROM pg_stat_activity;"

# Kill idle connections
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle';"

# Restart PostgreSQL
docker-compose restart postgres

# Reduce concurrent workers in ETL jobs
mutation {
  startNormalization(config: {
    concurrentWorkers: 2  # Reduce from 4 or 8
  }) {
    id
  }
}
```

###  System Performance Issues

#### High Memory Usage

**Problem**: System runs out of memory or becomes unresponsive.

**Solution**:
```bash
# Monitor memory usage
docker stats
free -h

# Reduce ETL batch sizes
mutation {
  startNormalization(config: {
    batchSize: 500       # Smaller batches
    concurrentWorkers: 2  # Fewer workers
  }) {
    id
  }
}

# Restart services to clear memory
docker-compose restart

# Add swap space if needed (Linux)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

#### High CPU Usage

**Problem**: System becomes slow due to high CPU usage.

**Solution**:
```bash
# Monitor CPU usage
htop
docker stats

# Reduce concurrent processing
curl -X POST http://localhost:8080/populate-all \
  -d '{"concurrency": 1}'  # Single threaded

# Reduce ETL workers
mutation {
  startNormalization(config: {
    concurrentWorkers: 1
  }) {
    id
  }
}

# Schedule processing during off-hours
# Use cron jobs for large data collections
```

##  Debugging Tools & Commands

### Useful Diagnostic Commands

```bash
# System overview
docker-compose ps
docker stats
df -h
free -h

# Application logs
docker-compose logs --tail=50 app
docker-compose logs --tail=50 postgres

# Database diagnostics
docker-compose exec postgres psql -U postgres -d cpc \
  -c "SELECT schemaname,tablename,n_tup_ins,n_tup_upd,n_tup_del FROM pg_stat_user_tables;"

# Network connectivity
curl -I http://localhost:8080
curl -I https://prices.azure.com/api/retail/prices

# API testing
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{ hello }"}'
```

### Performance Monitoring

```bash
# Watch ETL progress
watch -n 2 'curl -s http://localhost:8080/query -d "{\"query\":\"{ etlJobs { id status progress { rate processedRecords } } }\"}" | jq'

# Monitor collection progress
watch -n 5 'curl -s http://localhost:8080/query -d "{\"query\":\"{ azureCollections { region status totalItems } }\"}" | jq'

# Database activity
watch -n 10 'docker-compose exec postgres psql -U postgres -d cpc -c "SELECT count(*) as active_queries FROM pg_stat_activity WHERE state = '\''active'\'';"'
```

##  Getting Help

### Before Asking for Help

1. **Check the logs**: Always start with `docker-compose logs app`
2. **Try basic connectivity**: Test with `curl http://localhost:8080/query -d '{"query": "{ hello }"}'`
3. **Restart services**: Often fixes transient issues
4. **Check system resources**: Ensure adequate CPU, memory, and disk space

### Where to Get Support

- ** Bug Reports**: [Create an issue](https://github.com/your-org/cpc/issues) with:
  - Complete error messages
  - Steps to reproduce
  - System information (`docker version`, `docker-compose version`)
  - Relevant log snippets

- ** Questions**: [Start a discussion](https://github.com/your-org/cpc/discussions)

- ** Documentation**: Check the [full documentation](../README.md)

### Information to Include in Bug Reports

```bash
# System information
docker version
docker-compose version
uname -a

# Container status
docker-compose ps

# Recent logs
docker-compose logs --tail=100 app
docker-compose logs --tail=100 postgres

# Database status
docker-compose exec postgres pg_isready
```

##  Advanced Troubleshooting

### Enable Debug Logging

Add to your `.env` file:
```bash
LOG_LEVEL=debug
DEBUG=true
```

Then restart:
```bash
docker-compose restart app
```

### Database Query Analysis

```sql
-- Find slow queries
SELECT query, calls, total_time, mean_time 
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;

-- Check table sizes
SELECT schemaname,tablename,pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check indexes
SELECT schemaname,tablename,indexname,idx_tup_read,idx_tup_fetch 
FROM pg_stat_user_indexes 
ORDER BY idx_tup_read DESC;
```

### Clean Database Reset

If all else fails, start fresh:

```bash
# Backup important data first
docker-compose exec postgres pg_dump -U postgres cpc > backup.sql

# Complete reset
docker-compose down -v
docker system prune -f
docker volume prune -f
docker-compose up -d

# Wait for services to start
sleep 30

# Test basic functionality
curl http://localhost:8080/query -d '{"query": "{ hello }"}'
```

---

**Still having issues?** Don't hesitate to [create an issue](https://github.com/your-org/cpc/issues) with detailed information about your problem. The community is here to help! 