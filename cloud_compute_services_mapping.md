# AWS vs Azure Compute Services - Complete Mapping

## Virtual Machines & Basic Compute

### Virtual Machines
- **AWS**: Amazon EC2 (Elastic Compute Cloud)
- **Azure**: Azure Virtual Machines

### Auto Scaling Virtual Machines
- **AWS**: EC2 Auto Scaling Groups
- **Azure**: Virtual Machine Scale Sets (VMSS)

### Spot/Low-Priority Instances
- **AWS**: EC2 Spot Instances
- **Azure**: Spot Virtual Machines

### Dedicated Hardware
- **AWS**: EC2 Dedicated Hosts, EC2 Dedicated Instances
- **Azure**: Azure Dedicated Host

## Container Services

### Managed Kubernetes
- **AWS**: Elastic Kubernetes Service (EKS)
- **Azure**: Azure Kubernetes Service (AKS)

### Container Orchestration Platforms
- **AWS**: Elastic Container Service (ECS)
- **Azure**: Service Fabric

### Serverless Containers
- **AWS**: AWS Fargate (works with ECS/EKS)
- **Azure**: Azure Container Instances (ACI)

### Container Management Tools
- **AWS**: AWS Copilot (CLI for containerized apps)
- **Azure**: No direct equivalent

### Container Registries
- **AWS**: Elastic Container Registry (ECR)
- **Azure**: Azure Container Registry (ACR)

### Specialized Container Platforms
- **AWS**: No direct equivalent
- **Azure**: Azure Red Hat OpenShift, Azure Container Apps

## Serverless Computing

### Function as a Service (FaaS)
- **AWS**: AWS Lambda
- **Azure**: Azure Functions

### Serverless Workflows & Orchestration
- **AWS**: AWS Step Functions, Simple Workflow Service (SWF)
- **Azure**: Logic Apps, Durable Functions

### Serverless Application Repository
- **AWS**: AWS Serverless Application Repository
- **Azure**: No direct equivalent

## Platform as a Service (PaaS)

### Web Application Platforms
- **AWS**: AWS Elastic Beanstalk, AWS App Runner
- **Azure**: Azure App Service (Web Apps, API Apps, Mobile Apps)

### Static Web Hosting
- **AWS**: S3 Static Website Hosting, AWS Amplify
- **Azure**: Azure Static Web Apps

### Simplified Cloud Platforms
- **AWS**: Amazon Lightsail (VPS service)
- **Azure**: No direct equivalent

### Spring Boot Applications
- **AWS**: No direct equivalent
- **Azure**: Azure Spring Apps

## Batch & High-Performance Computing

### Batch Processing
- **AWS**: AWS Batch
- **Azure**: Azure Batch

### High-Performance Computing
- **AWS**: AWS ParallelCluster, HPC on EC2
- **Azure**: Azure CycleCloud, HPC clusters on VMs

## Edge Computing & Hybrid Cloud

### On-Premises Cloud Infrastructure
- **AWS**: AWS Outposts
- **Azure**: Azure Stack Hub, Azure Stack HCI

### Edge Computing Appliances
- **AWS**: No direct equivalent
- **Azure**: Azure Stack Edge

### Ultra-Low Latency Computing
- **AWS**: AWS Local Zones (metropolitan areas)
- **Azure**: No direct equivalent

### 5G/Mobile Edge Computing
- **AWS**: AWS Wavelength (5G networks)
- **Azure**: No direct equivalent

### VMware Integration
- **AWS**: VMware Cloud on AWS
- **Azure**: Azure VMware Solution

## Development & Testing

### Development Environments
- **AWS**: AWS Cloud9 (IDE), EC2 for development
- **Azure**: Azure Dev Box (cloud workstations)

### Chaos Engineering
- **AWS**: AWS Fault Injection Simulator
- **Azure**: Azure Chaos Studio

## GPU & Specialized Computing

### GPU Computing Instances
- **AWS**: EC2 P-series (AI/ML), G-series (graphics), VT1 (video)
- **Azure**: N-series (NC, ND, NV variants)

### Quantum Computing
- **AWS**: Amazon Braket
- **Azure**: Azure Quantum

## IoT & Edge Services

### IoT Edge Computing
- **AWS**: AWS IoT Greengrass
- **Azure**: Azure IoT Edge

## Key Architectural Differences

### AWS Approach
- **More granular services**: Separate ECS vs EKS vs Fargate
- **Specialized edge options**: Local Zones, Wavelength for different use cases
- **Step Functions**: Complex workflow orchestration
- **Lightsail**: Simplified VPS offering

### Azure Approach
- **More integrated platforms**: App Service covers multiple scenarios
- **Comprehensive hybrid stack**: Stack Edge/Hub/HCI family
- **Durable Functions**: Extends Azure Functions for stateful scenarios
- **Logic Apps**: Low-code workflow automation
- **Container Apps**: Modern microservices platform

## Service Categories Summary

| Category | AWS Count | Azure Count |
|----------|-----------|-------------|
| Virtual Machines | 4 | 3 |
| Container Services | 6 | 6 |
| Serverless | 3 | 3 |
| PaaS Platforms | 4 | 3 |
| Batch/HPC | 2 | 2 |
| Edge/Hybrid | 4 | 4 |
| Development | 2 | 2 |
| GPU/Specialized | 2 | 2 |
| IoT Edge | 1 | 1 |

**Total Services Mapped**: 28 AWS services, 26 Azure services

## Notes

- **Direct Equivalents**: Most core services have direct counterparts (EC2↔VMs, EKS↔AKS, Lambda↔Functions)
- **AWS Unique**: Lightsail (simple VPS), Local Zones (metro edge), Wavelength (5G edge), ECS (native orchestration)
- **Azure Unique**: Container Apps (modern microservices), Stack family (comprehensive hybrid), Spring Apps (managed Spring Boot), Dev Box (cloud workstations)
- **Both Evolving**: This mapping reflects 2025 service availability and continues to evolve as both platforms add new offerings