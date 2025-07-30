-- Initial database schema for CPC (Cloud Price Compare)

-- Create a simple messages table for testing
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert a welcome message
INSERT INTO messages (content) VALUES ('Welcome to Cloud Price Compare!');

-- Create the main pricing tables (basic structure for now)
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