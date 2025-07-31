# AWS Pricing API

The AWS pricing integration provides access to AWS Price List Query API data through REST endpoints and GraphQL queries.

## Authentication

AWS pricing collection requires valid AWS credentials:

```bash
# Create .env file with your credentials
cp .env.example .env

# Edit .env file:
# AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY
# AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY
# AWS_DEFAULT_REGION=us-east-1
```

> **Note**: The AWS Pricing API only works in the `us-east-1` region, but can collect pricing data for all AWS regions.

## Supported Services

Currently supported AWS services:

- **AmazonEC2**: Elastic Compute Cloud instances
- **AmazonS3**: Simple Storage Service

Additional services can be easily added by extending the service collection logic.

## REST Endpoints

### Single Service Collection

**Endpoint**: `POST /aws-populate`

Collect AWS pricing data for specific services and regions.

```bash
curl -X POST http://localhost:8080/aws-populate \
  -H "Content-Type: application/json" \
  -d '{
    "serviceCodes": ["AmazonEC2"],
    "regions": ["us-east-1", "us-west-2"],
    "instanceTypes": ["t3.micro", "t3.small", "t3.medium"]
  }'
```

**Request Parameters**:
- `serviceCodes` (array): AWS service codes to collect (e.g., ["AmazonEC2", "AmazonS3"])
- `regions` (array): AWS regions to collect from (e.g., ["us-east-1", "eu-west-1"])
- `instanceTypes` (array, optional): For EC2, specific instance types to collect

**Response**:
```json
{
  "message": "AWS data collection started for services: [AmazonEC2], regions: [us-east-1]",
  "collectionId": "aws_1753953464"
}
```

### Multi-Region Collection

**Endpoint**: `POST /aws-populate-all`

Collect AWS pricing data from multiple regions concurrently.

```bash
curl -X POST http://localhost:8080/aws-populate-all \
  -H "Content-Type: application/json" \
  -d '{
    "serviceCodes": ["AmazonEC2", "AmazonS3"],
    "concurrency": 3
  }'
```

**Request Parameters**:
- `serviceCodes` (array, optional): Services to collect. Defaults to ["AmazonEC2", "AmazonS3"]
- `concurrency` (integer, optional): Number of concurrent workers. Defaults to 3

**Regions Covered**:
- us-east-1, us-east-2, us-west-1, us-west-2
- eu-west-1, eu-west-2, eu-central-1
- ap-southeast-1, ap-southeast-2, ap-northeast-1

## Service-Specific Details

### EC2 Pricing

**Filters Applied**:
- Instance types: Configurable (default: t3.micro, t3.small, t3.medium)
- Tenancy: Shared
- Operating System: Linux
- Pre-installed Software: None

**Sample Response Data**:
```json
{
  "serviceCode": "Compute Instance",
  "serviceName": "Amazon Elastic Compute Cloud",
  "location": "US East (N. Virginia)",
  "instanceType": "t3.micro",
  "pricePerUnit": 0.0104,
  "unit": "Hrs",
  "currency": "USD",
  "termType": "OnDemand"
}
```

### S3 Pricing

**Filters Applied**:
- Storage Class: General Purpose (Standard)
- Location: Specific region

**Sample Response Data**:
```json
{
  "serviceCode": "Storage",
  "serviceName": "Amazon Simple Storage Service",
  "location": "US East (N. Virginia)",
  "pricePerUnit": 0.023,
  "unit": "GB-Mo",
  "currency": "USD",
  "termType": "OnDemand"
}
```

## Data Structure

### Raw Pricing Table

The `aws_pricing_raw` table stores complete AWS pricing data:

```sql
CREATE TABLE aws_pricing_raw (
    id SERIAL PRIMARY KEY,
    collection_id VARCHAR(50) NOT NULL,
    service_code VARCHAR(100) NOT NULL,
    service_name VARCHAR(100),
    location VARCHAR(100),
    instance_type VARCHAR(50),
    price_per_unit DECIMAL(10,6),
    unit VARCHAR(50),
    currency VARCHAR(10) DEFAULT 'USD',
    term_type VARCHAR(20), -- OnDemand, Reserved
    attributes JSONB, -- All product attributes
    raw_product JSONB NOT NULL, -- Complete AWS product JSON
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Collection Tracking

The `aws_collections` table tracks collection runs:

```sql
CREATE TABLE aws_collections (
    id SERIAL PRIMARY KEY,
    collection_id VARCHAR(50) NOT NULL UNIQUE,
    service_codes TEXT[], -- Array of service codes
    regions TEXT[], -- Array of regions
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    total_items INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB
);
```

## Region Mapping

AWS regions are mapped to location names for pricing API queries:

| Region Code | Location Name |
|-------------|---------------|
| us-east-1 | US East (N. Virginia) |
| us-east-2 | US East (Ohio) |
| us-west-1 | US West (N. California) |
| us-west-2 | US West (Oregon) |
| eu-west-1 | Europe (Ireland) |
| eu-west-2 | Europe (London) |
| eu-central-1 | Europe (Frankfurt) |
| ap-southeast-1 | Asia Pacific (Singapore) |
| ap-southeast-2 | Asia Pacific (Sydney) |
| ap-northeast-1 | Asia Pacific (Tokyo) |

## Error Handling

Common error scenarios:

1. **Invalid Credentials**: Ensure AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are valid
2. **Rate Limiting**: AWS API has built-in rate limiting with retry logic
3. **Invalid Service Code**: Only supported services (EC2, S3) are accepted
4. **Region Not Found**: Unknown regions are skipped with warnings

## Performance Characteristics

- **EC2 Collection**: ~20-30 items per region (varies by instance type filters)
- **S3 Collection**: ~3-5 items per region (varies by storage classes)
- **Collection Time**: 10-30 seconds per region depending on service
- **Concurrent Workers**: Configurable, recommended 3-5 for optimal performance

## Future Enhancements

Planned additions:
- **RDS Pricing**: Database instance pricing
- **Lambda Pricing**: Serverless compute pricing
- **Reserved Instance Pricing**: Long-term commitment pricing
- **Spot Instance Pricing**: Variable market-based pricing