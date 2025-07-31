# Azure Pricing API

Detailed information about Azure pricing data structure and queries.

## Data Structure

Azure pricing data is normalized across multiple tables:

- **Services** - High-level service categories (e.g., "Virtual Machines")
- **Products** - Specific products within services  
- **SKUs** - Detailed configurations and variants
- **Regions** - Geographic locations
- **Pricing** - Actual price points with metadata

## Sample Data

Here's what typical Azure pricing data looks like:

```json
{
  "serviceName": "Virtual Machines",
  "serviceFamily": "Compute",
  "productName": "Virtual Machines BS Series",
  "skuName": "B1s",
  "region": "East US",
  "meterName": "B1s Compute Hour",
  "retailPrice": 0.0104,
  "unitOfMeasure": "1 Hour",
  "effectiveDate": "2024-01-01"
}
```

## Key Fields

| Field | Description | Example |
|-------|-------------|---------|
| `serviceName` | Azure service name | "Virtual Machines" |
| `serviceFamily` | Service category | "Compute" |
| `productName` | Specific product line | "Virtual Machines BS Series" |
| `skuName` | SKU identifier | "B1s" |
| `meterName` | Billing meter name | "B1s Compute Hour" |
| `retailPrice` | Price per unit | 0.0104 |
| `unitOfMeasure` | Billing unit | "1 Hour" |
| `region` | Geographic region | "East US" |

## Data Collection

Data is collected from Azure's public Retail Pricing API:
- **No authentication required**
- **Real-time pricing information**
- **71+ regions available**
- **2000+ pricing items per region**

## Filtering and Queries

Currently, the API returns sample data. Future versions will support:

- Filter by service name
- Filter by region
- Price range filtering
- Date range queries
- Comparison across regions

## Units of Measure

Azure pricing comes in various units:
- `1 Hour` - Hourly compute resources
- `1 GB/Month` - Storage pricing
- `10K` - Transaction-based pricing
- `1` - Per-item pricing

## Price Types

- **Consumption** - Pay-as-you-go pricing
- **Reservation** - Reserved instance pricing
- **DevTestConsumption** - Dev/test pricing