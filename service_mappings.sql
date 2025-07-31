-- Service mappings from cloud_compute_services_mapping.md
-- Maps provider-specific services to normalized categories

-- Clear existing mappings
DELETE FROM service_mappings;

-- Virtual Machines & Basic Compute
INSERT INTO service_mappings (provider, provider_service_name, provider_service_code, normalized_service_type, service_category, service_family) VALUES
('aws', 'Amazon EC2', 'AmazonEC2', 'Virtual Machines', 'Compute & Web', 'Virtual Machines'),
('azure', 'Virtual Machines', 'Virtual Machines', 'Virtual Machines', 'Compute & Web', 'Virtual Machines'),

('aws', 'EC2 Auto Scaling Groups', 'AmazonEC2', 'Auto Scaling VMs', 'Compute & Web', 'Virtual Machines'),
('azure', 'Virtual Machine Scale Sets', 'Virtual Machine Scale Sets', 'Auto Scaling VMs', 'Compute & Web', 'Virtual Machines'),

('aws', 'EC2 Spot Instances', 'AmazonEC2', 'Spot Instances', 'Compute & Web', 'Virtual Machines'),
('azure', 'Spot Virtual Machines', 'Virtual Machines', 'Spot Instances', 'Compute & Web', 'Virtual Machines'),

('aws', 'EC2 Dedicated Hosts', 'AmazonEC2', 'Dedicated Hardware', 'Compute & Web', 'Virtual Machines'),
('aws', 'EC2 Dedicated Instances', 'AmazonEC2', 'Dedicated Hardware', 'Compute & Web', 'Virtual Machines'),
('azure', 'Azure Dedicated Host', 'Dedicated Host', 'Dedicated Hardware', 'Compute & Web', 'Virtual Machines'),

-- Container Services
('aws', 'Elastic Kubernetes Service', 'AmazonEKS', 'Managed Kubernetes', 'Containers', 'Container Orchestration'),
('azure', 'Azure Kubernetes Service', 'AKS', 'Managed Kubernetes', 'Containers', 'Container Orchestration'),

('aws', 'Elastic Container Service', 'AmazonECS', 'Container Orchestration', 'Containers', 'Container Orchestration'),
('azure', 'Service Fabric', 'Service Fabric', 'Container Orchestration', 'Containers', 'Container Orchestration'),

('aws', 'AWS Fargate', 'AWSFargate', 'Serverless Containers', 'Containers', 'Serverless Containers'),
('azure', 'Azure Container Instances', 'Container Instances', 'Serverless Containers', 'Containers', 'Serverless Containers'),

('aws', 'Elastic Container Registry', 'AmazonECR', 'Container Registry', 'Containers', 'Container Management'),
('azure', 'Azure Container Registry', 'Container Registry', 'Container Registry', 'Containers', 'Container Management'),

('azure', 'Azure Red Hat OpenShift', 'Red Hat OpenShift', 'Specialized Container Platform', 'Containers', 'Container Orchestration'),
('azure', 'Azure Container Apps', 'Container Apps', 'Specialized Container Platform', 'Containers', 'Container Orchestration'),

-- Serverless Computing
('aws', 'AWS Lambda', 'AWSLambda', 'Serverless Functions', 'Compute & Web', 'Serverless'),
('azure', 'Azure Functions', 'Functions', 'Serverless Functions', 'Compute & Web', 'Serverless'),

('aws', 'AWS Step Functions', 'AWSStepFunctions', 'Serverless Workflows', 'Compute & Web', 'Serverless'),
('aws', 'Simple Workflow Service', 'AmazonSWF', 'Serverless Workflows', 'Compute & Web', 'Serverless'),
('azure', 'Logic Apps', 'Logic Apps', 'Serverless Workflows', 'Compute & Web', 'Serverless'),
('azure', 'Durable Functions', 'Functions', 'Serverless Workflows', 'Compute & Web', 'Serverless'),

('aws', 'AWS Serverless Application Repository', 'ServerlessApplicationRepository', 'Serverless App Repository', 'Compute & Web', 'Serverless'),

-- Platform as a Service (PaaS)
('aws', 'AWS Elastic Beanstalk', 'AWSElasticBeanstalk', 'Web Application Platform', 'Compute & Web', 'PaaS'),
('aws', 'AWS App Runner', 'AWSAppRunner', 'Web Application Platform', 'Compute & Web', 'PaaS'),
('azure', 'Azure App Service', 'App Service', 'Web Application Platform', 'Compute & Web', 'PaaS'),

('aws', 'S3 Static Website Hosting', 'AmazonS3', 'Static Web Hosting', 'Compute & Web', 'PaaS'),
('aws', 'AWS Amplify', 'AWSAmplify', 'Static Web Hosting', 'Compute & Web', 'PaaS'),
('azure', 'Azure Static Web Apps', 'Static Web Apps', 'Static Web Hosting', 'Compute & Web', 'PaaS'),

('aws', 'Amazon Lightsail', 'AmazonLightsail', 'Simplified Cloud Platform', 'Compute & Web', 'PaaS'),
('azure', 'Azure Spring Apps', 'Spring Apps', 'Spring Boot Platform', 'Compute & Web', 'PaaS'),

-- Batch & High-Performance Computing
('aws', 'AWS Batch', 'AWSBatch', 'Batch Processing', 'Compute & Web', 'Batch Computing'),
('azure', 'Azure Batch', 'Batch', 'Batch Processing', 'Compute & Web', 'Batch Computing'),

('aws', 'AWS ParallelCluster', 'ParallelCluster', 'High-Performance Computing', 'Compute & Web', 'HPC'),
('azure', 'Azure CycleCloud', 'CycleCloud', 'High-Performance Computing', 'Compute & Web', 'HPC'),

-- Edge Computing & Hybrid Cloud
('aws', 'AWS Outposts', 'AWSOutposts', 'On-Premises Cloud', 'Compute & Web', 'Hybrid Cloud'),
('azure', 'Azure Stack Hub', 'Stack Hub', 'On-Premises Cloud', 'Compute & Web', 'Hybrid Cloud'),
('azure', 'Azure Stack HCI', 'Stack HCI', 'On-Premises Cloud', 'Compute & Web', 'Hybrid Cloud'),

('azure', 'Azure Stack Edge', 'Stack Edge', 'Edge Computing Appliance', 'Compute & Web', 'Edge Computing'),

('aws', 'AWS Local Zones', 'LocalZones', 'Ultra-Low Latency Computing', 'Compute & Web', 'Edge Computing'),
('aws', 'AWS Wavelength', 'Wavelength', '5G/Mobile Edge Computing', 'Compute & Web', 'Edge Computing'),

('aws', 'VMware Cloud on AWS', 'VMwareCloudOnAWS', 'VMware Integration', 'Compute & Web', 'Hybrid Cloud'),
('azure', 'Azure VMware Solution', 'VMware Solution', 'VMware Integration', 'Compute & Web', 'Hybrid Cloud'),

-- Development & Testing
('aws', 'AWS Cloud9', 'AWSCloud9', 'Development Environment', 'Dev Tools', 'Development'),
('azure', 'Azure Dev Box', 'Dev Box', 'Development Environment', 'Dev Tools', 'Development'),

('aws', 'AWS Fault Injection Simulator', 'FaultInjectionSimulator', 'Chaos Engineering', 'Dev Tools', 'Testing'),
('azure', 'Azure Chaos Studio', 'Chaos Studio', 'Chaos Engineering', 'Dev Tools', 'Testing'),

-- GPU & Specialized Computing
('aws', 'EC2 P-series', 'AmazonEC2', 'GPU Computing - AI/ML', 'Compute & Web', 'GPU Computing'),
('aws', 'EC2 G-series', 'AmazonEC2', 'GPU Computing - Graphics', 'Compute & Web', 'GPU Computing'),
('aws', 'EC2 VT1', 'AmazonEC2', 'GPU Computing - Video', 'Compute & Web', 'GPU Computing'),
('azure', 'N-series VMs', 'Virtual Machines', 'GPU Computing', 'Compute & Web', 'GPU Computing'),

('aws', 'Amazon Braket', 'AmazonBraket', 'Quantum Computing', 'AI & ML', 'Quantum Computing'),
('azure', 'Azure Quantum', 'Quantum', 'Quantum Computing', 'AI & ML', 'Quantum Computing'),

-- IoT & Edge Services
('aws', 'AWS IoT Greengrass', 'AWSIoTGreengrass', 'IoT Edge Computing', 'Analytics & IoT', 'IoT Edge'),
('azure', 'Azure IoT Edge', 'IoT Edge', 'IoT Edge Computing', 'Analytics & IoT', 'IoT Edge');

-- Check the results
SELECT 
    provider,
    COUNT(*) as service_count,
    array_agg(DISTINCT service_category) as categories
FROM service_mappings 
GROUP BY provider 
ORDER BY provider;