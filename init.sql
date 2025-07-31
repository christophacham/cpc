-- Simplified database schema for CPC (Cloud Price Compare)
-- Using raw JSON storage approach

-- Create a simple messages table for testing
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert a welcome message
INSERT INTO messages (content) VALUES ('Welcome to Cloud Price Compare!');

-- Create the main pricing tables (basic structure)
CREATE TABLE IF NOT EXISTS providers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS service_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial providers
INSERT INTO providers (name) VALUES ('AWS'), ('Azure');

-- Insert initial categories based on CLAUDE.md
INSERT INTO service_categories (name, description) VALUES 
    ('General', 'Core infrastructure and foundational services'),
    ('Networking', 'Network connectivity, load balancing, CDN'),
    ('Compute & Web', 'Virtual machines, containers, serverless compute'),
    ('Containers', 'Container orchestration and management'),
    ('Databases', 'Relational, NoSQL, and specialized databases'),
    ('Storage', 'Object storage, file systems, backup solutions'),
    ('AI & ML', 'Machine learning, cognitive services, AI tools'),
    ('Analytics & IoT', 'Data analytics, streaming, IoT platforms'),
    ('Virtual Desktop', 'Desktop virtualization and workspace solutions'),
    ('Dev Tools', 'Development, CI/CD, and testing tools'),
    ('Integration', 'API management, messaging, event services'),
    ('Migration', 'Data migration and transfer services'),
    ('Management', 'Monitoring, governance, security tools');

-- Azure Raw Pricing Data - Simplified approach
CREATE TABLE IF NOT EXISTS azure_pricing_raw (
    id SERIAL PRIMARY KEY,
    region VARCHAR(100) NOT NULL,
    service_name VARCHAR(100),
    service_family VARCHAR(100),
    data JSONB NOT NULL, -- Store entire Azure API response
    collected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    collection_id VARCHAR(50), -- For tracking collection batches
    total_items INTEGER DEFAULT 0 -- Number of items in this batch
);

-- Collection tracking for raw data
CREATE TABLE IF NOT EXISTS azure_collections (
    id SERIAL PRIMARY KEY,
    collection_id VARCHAR(50) NOT NULL UNIQUE,
    region VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running', -- running, completed, failed
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    total_items INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB -- Store additional collection metadata
);

-- Create indexes for performance on raw data
CREATE INDEX IF NOT EXISTS idx_azure_pricing_raw_region ON azure_pricing_raw(region);
CREATE INDEX IF NOT EXISTS idx_azure_pricing_raw_service ON azure_pricing_raw(service_name);
CREATE INDEX IF NOT EXISTS idx_azure_pricing_raw_family ON azure_pricing_raw(service_family);
CREATE INDEX IF NOT EXISTS idx_azure_pricing_raw_collected ON azure_pricing_raw(collected_at);
CREATE INDEX IF NOT EXISTS idx_azure_pricing_raw_collection_id ON azure_pricing_raw(collection_id);

-- JSONB indexes for querying inside the data
CREATE INDEX IF NOT EXISTS idx_azure_pricing_raw_data_gin ON azure_pricing_raw USING GIN (data);

-- Collection indexes
CREATE INDEX IF NOT EXISTS idx_azure_collections_region ON azure_collections(region);
CREATE INDEX IF NOT EXISTS idx_azure_collections_status ON azure_collections(status);
CREATE INDEX IF NOT EXISTS idx_azure_collections_started ON azure_collections(started_at);

CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);
CREATE INDEX IF NOT EXISTS idx_categories_name ON service_categories(name);

-- AWS Raw Pricing Data Tables
CREATE TABLE IF NOT EXISTS aws_pricing_raw (
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

-- AWS Collection tracking
CREATE TABLE IF NOT EXISTS aws_collections (
    id SERIAL PRIMARY KEY,
    collection_id VARCHAR(50) NOT NULL UNIQUE,
    service_codes TEXT[], -- Array of service codes
    regions TEXT[], -- Array of regions
    status VARCHAR(20) NOT NULL DEFAULT 'running', -- running, completed, failed
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    total_items INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB
);

-- AWS indexes for performance
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_collection_id ON aws_pricing_raw(collection_id);
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_service_code ON aws_pricing_raw(service_code);
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_location ON aws_pricing_raw(location);
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_instance_type ON aws_pricing_raw(instance_type);
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_term_type ON aws_pricing_raw(term_type);
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_created ON aws_pricing_raw(created_at);

-- JSONB indexes for AWS data
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_attributes_gin ON aws_pricing_raw USING GIN (attributes);
CREATE INDEX IF NOT EXISTS idx_aws_pricing_raw_product_gin ON aws_pricing_raw USING GIN (raw_product);

-- AWS collection indexes
CREATE INDEX IF NOT EXISTS idx_aws_collections_status ON aws_collections(status);
CREATE INDEX IF NOT EXISTS idx_aws_collections_started ON aws_collections(started_at);