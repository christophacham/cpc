-- Azure Pricing Database Schema
-- Designed for efficient storage and querying of pricing data

-- Core service reference tables
CREATE TABLE IF NOT EXISTS azure_services (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL UNIQUE,
    service_family VARCHAR(50) NOT NULL,
    category_id INTEGER REFERENCES service_categories(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS azure_regions (
    id SERIAL PRIMARY KEY,
    arm_region_name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS azure_products (
    id SERIAL PRIMARY KEY,
    service_id INTEGER NOT NULL REFERENCES azure_services(id),
    product_name VARCHAR(200) NOT NULL,
    product_id VARCHAR(100) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(service_id, product_name)
);

CREATE TABLE IF NOT EXISTS azure_skus (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES azure_products(id),
    sku_name VARCHAR(100) NOT NULL,
    sku_id VARCHAR(100) UNIQUE,
    arm_sku_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(product_id, sku_name)
);

-- Main pricing table - optimized for queries
CREATE TABLE IF NOT EXISTS azure_pricing (
    id BIGSERIAL PRIMARY KEY,
    
    -- References to normalized tables
    service_id INTEGER NOT NULL REFERENCES azure_services(id),
    product_id INTEGER NOT NULL REFERENCES azure_products(id),
    sku_id INTEGER NOT NULL REFERENCES azure_skus(id),
    region_id INTEGER NOT NULL REFERENCES azure_regions(id),
    
    -- Pricing details
    meter_id VARCHAR(100) NOT NULL,
    meter_name VARCHAR(200) NOT NULL,
    retail_price DECIMAL(15,6) NOT NULL,
    unit_price DECIMAL(15,6) NOT NULL,
    tier_minimum_units DECIMAL(15,6) DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    unit_of_measure VARCHAR(50) NOT NULL,
    
    -- Pricing type and terms
    price_type VARCHAR(20) NOT NULL DEFAULT 'Consumption',
    reservation_term VARCHAR(20),
    
    -- Temporal data
    effective_start_date DATE NOT NULL,
    is_primary_meter_region BOOLEAN DEFAULT false,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    collection_version INTEGER DEFAULT 1,
    
    -- Indexes for common queries
    INDEX idx_pricing_service_region (service_id, region_id),
    INDEX idx_pricing_product_region (product_id, region_id),
    INDEX idx_pricing_meter (meter_id),
    INDEX idx_pricing_effective_date (effective_start_date),
    INDEX idx_pricing_price (retail_price),
    INDEX idx_pricing_collection (collection_version, created_at),
    
    -- Unique constraint to prevent duplicates
    UNIQUE(service_id, product_id, sku_id, region_id, meter_id, effective_start_date)
);

-- Collection tracking
CREATE TABLE IF NOT EXISTS azure_collection_runs (
    id SERIAL PRIMARY KEY,
    version INTEGER NOT NULL UNIQUE,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'running', -- running, completed, failed
    total_items INTEGER DEFAULT 0,
    regions_collected TEXT[], -- array of region names
    error_message TEXT,
    collection_metadata JSONB,
    
    INDEX idx_collection_version (version),
    INDEX idx_collection_status (status),
    INDEX idx_collection_date (started_at)
);

-- Update existing service_categories if needed
INSERT INTO service_categories (name, description) VALUES 
    ('Compute', 'Virtual machines, containers, serverless compute'),
    ('Networking', 'Network connectivity, load balancing, CDN')
ON CONFLICT (name) DO NOTHING;

-- Create indexes on existing tables
CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);
CREATE INDEX IF NOT EXISTS idx_categories_name ON service_categories(name);