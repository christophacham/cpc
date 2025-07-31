-- Normalized Cloud Pricing Database Schema
-- This schema provides a unified view of AWS and Azure pricing data
-- for easy querying and cross-provider comparisons

-- Drop existing tables if they exist (for clean migrations)
DROP TABLE IF EXISTS normalized_pricing CASCADE;
DROP TABLE IF EXISTS service_mappings CASCADE;
DROP TABLE IF EXISTS normalized_regions CASCADE;

-- Region mapping table to standardize region codes across providers
CREATE TABLE normalized_regions (
    id SERIAL PRIMARY KEY,
    normalized_code VARCHAR(50) NOT NULL UNIQUE, -- e.g., 'us-east', 'eu-west'
    aws_region VARCHAR(50), -- e.g., 'us-east-1'
    azure_region VARCHAR(50), -- e.g., 'eastus'
    display_name VARCHAR(100) NOT NULL, -- e.g., 'US East (Virginia)'
    country VARCHAR(50),
    continent VARCHAR(50)
);

-- Service mapping table to map provider-specific services to normalized categories
CREATE TABLE service_mappings (
    id SERIAL PRIMARY KEY,
    provider VARCHAR(10) NOT NULL CHECK (provider IN ('aws', 'azure')),
    provider_service_name VARCHAR(200) NOT NULL, -- e.g., 'AmazonEC2', 'Virtual Machines'
    provider_service_code VARCHAR(100), -- e.g., 'AmazonEC2' for AWS
    normalized_service_type VARCHAR(100) NOT NULL, -- e.g., 'Virtual Machines'
    service_category VARCHAR(50) NOT NULL, -- From our 13 categories
    service_family VARCHAR(100) NOT NULL, -- e.g., 'Compute', 'Storage'
    UNIQUE(provider, provider_service_name)
);

-- Main normalized pricing table
CREATE TABLE normalized_pricing (
    id SERIAL PRIMARY KEY,
    
    -- Provider information
    provider VARCHAR(10) NOT NULL CHECK (provider IN ('aws', 'azure')),
    provider_service_code VARCHAR(100) NOT NULL, -- Original service code
    provider_sku VARCHAR(200), -- Provider-specific SKU/product ID
    
    -- Unified service categorization (foreign key to service_mappings)
    service_mapping_id INTEGER REFERENCES service_mappings(id),
    service_category VARCHAR(50) NOT NULL, -- Denormalized for query performance
    service_family VARCHAR(100) NOT NULL, -- Denormalized for query performance
    service_type VARCHAR(100) NOT NULL, -- Denormalized for query performance
    
    -- Location (foreign key to normalized_regions)
    region_id INTEGER REFERENCES normalized_regions(id),
    normalized_region VARCHAR(50) NOT NULL, -- Denormalized for query performance
    provider_region VARCHAR(50) NOT NULL, -- Original provider region
    
    -- Resource specifications
    resource_name VARCHAR(200) NOT NULL, -- e.g., 't3.medium', 'Standard_D4s_v3'
    resource_description TEXT, -- Human-readable description
    
    -- Structured resource specifications as JSONB for flexibility
    resource_specs JSONB NOT NULL DEFAULT '{}',
    /* Expected structure:
    {
        "vcpu": 2,
        "memory_gb": 4,
        "storage_gb": 0,
        "gpu_count": 0,
        "gpu_memory_gb": 0,
        "network_performance": "moderate",
        "storage_type": "ssd",
        "processor_type": "Intel Xeon",
        "processor_features": ["avx", "avx2"]
    }
    */
    
    -- Pricing information
    price_per_unit DECIMAL(20,10) NOT NULL,
    unit VARCHAR(50) NOT NULL, -- Normalized: 'hour', 'gb_month', 'request', 'transaction'
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    pricing_model VARCHAR(50) NOT NULL, -- 'on_demand', 'reserved_1yr', 'reserved_3yr', 'spot', 'savings_plan'
    
    -- Additional pricing details as JSONB
    pricing_details JSONB DEFAULT '{}',
    /* Expected structure for reserved/savings:
    {
        "term_length": "1yr",
        "payment_option": "all_upfront",
        "upfront_cost": 1234.56,
        "hourly_rate": 0.123
    }
    */
    
    -- Metadata
    effective_date DATE,
    expiration_date DATE, -- For limited-time offers
    minimum_commitment INTEGER DEFAULT 1, -- Minimum units (e.g., minimum hours)
    
    -- References to raw data
    aws_raw_id INTEGER, -- Reference to aws_pricing_raw.id
    azure_raw_id INTEGER, -- Reference to azure_pricing_raw.id
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_provider_service ON normalized_pricing(provider, service_type);
CREATE INDEX idx_category_region ON normalized_pricing(service_category, normalized_region);
CREATE INDEX idx_resource_name ON normalized_pricing(resource_name);
CREATE INDEX idx_pricing_model ON normalized_pricing(pricing_model);
CREATE INDEX idx_price_per_unit ON normalized_pricing(price_per_unit);

-- JSONB indexes for querying resource specifications
CREATE INDEX idx_resource_specs_gin ON normalized_pricing USING GIN (resource_specs);
CREATE INDEX idx_resource_vcpu ON normalized_pricing ((resource_specs->>'vcpu'));
CREATE INDEX idx_resource_memory ON normalized_pricing ((resource_specs->>'memory_gb'));

-- Composite indexes for common query patterns
CREATE INDEX idx_vm_pricing ON normalized_pricing(service_type, normalized_region, pricing_model) 
    WHERE service_type = 'Virtual Machines';
CREATE INDEX idx_serverless_pricing ON normalized_pricing(service_type, normalized_region, pricing_model) 
    WHERE service_family = 'Serverless';

-- Function to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to update the updated_at column
CREATE TRIGGER update_normalized_pricing_updated_at 
    BEFORE UPDATE ON normalized_pricing 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Insert common region mappings
INSERT INTO normalized_regions (normalized_code, aws_region, azure_region, display_name, country, continent) VALUES
-- US Regions
('us-east', 'us-east-1', 'eastus', 'US East (Virginia)', 'USA', 'North America'),
('us-east-2', 'us-east-2', 'eastus2', 'US East (Ohio)', 'USA', 'North America'),
('us-west', 'us-west-2', 'westus', 'US West (Oregon)', 'USA', 'North America'),
('us-west-1', 'us-west-1', 'westus2', 'US West (California)', 'USA', 'North America'),
('us-central', NULL, 'centralus', 'US Central (Iowa)', 'USA', 'North America'),
('us-south-central', NULL, 'southcentralus', 'US South Central (Texas)', 'USA', 'North America'),

-- Europe Regions
('eu-west', 'eu-west-1', 'westeurope', 'Europe West (Ireland/Netherlands)', 'Ireland/Netherlands', 'Europe'),
('eu-north', 'eu-north-1', 'northeurope', 'Europe North (Stockholm/Ireland)', 'Sweden/Ireland', 'Europe'),
('eu-central', 'eu-central-1', 'germanywestcentral', 'Europe Central (Frankfurt)', 'Germany', 'Europe'),
('uk-south', 'eu-west-2', 'uksouth', 'UK South (London)', 'UK', 'Europe'),

-- Asia Pacific Regions
('asia-southeast', 'ap-southeast-1', 'southeastasia', 'Asia Southeast (Singapore)', 'Singapore', 'Asia'),
('asia-east', 'ap-northeast-1', 'eastasia', 'Asia East (Tokyo/Hong Kong)', 'Japan/Hong Kong', 'Asia'),
('asia-northeast', 'ap-northeast-2', 'koreacentral', 'Asia Northeast (Seoul)', 'South Korea', 'Asia'),
('australia-east', 'ap-southeast-2', 'australiaeast', 'Australia East (Sydney)', 'Australia', 'Oceania'),

-- Other Regions
('canada-central', 'ca-central-1', 'canadacentral', 'Canada Central', 'Canada', 'North America'),
('brazil-south', 'sa-east-1', 'brazilsouth', 'Brazil South (São Paulo)', 'Brazil', 'South America'),
('india-south', 'ap-south-1', 'centralindia', 'India South (Mumbai)', 'India', 'Asia');

-- Comments for documentation
COMMENT ON TABLE normalized_pricing IS 'Unified pricing data from AWS and Azure in a normalized format for easy querying and comparison';
COMMENT ON COLUMN normalized_pricing.resource_specs IS 'JSONB field containing normalized resource specifications like vCPU, memory, storage';
COMMENT ON COLUMN normalized_pricing.pricing_model IS 'Standardized pricing model: on_demand, reserved_1yr, reserved_3yr, spot, savings_plan';
COMMENT ON COLUMN normalized_pricing.unit IS 'Normalized billing unit: hour, gb_month, request, transaction, etc.';