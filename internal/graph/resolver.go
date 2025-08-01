package graph

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/raulc0399/cpc/internal/database"
	"github.com/raulc0399/cpc/internal/etl"
)

// Resolver is the root resolver
type Resolver struct {
	DB       *database.DB
	pipeline *etl.Pipeline
}

// SetPipeline sets the ETL pipeline for the resolver
func (r *Resolver) SetPipeline(pipeline *etl.Pipeline) {
	r.pipeline = pipeline
}

// Query returns the query resolver
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

// Mutation returns the mutation resolver
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

type queryResolver struct{ *Resolver }

// Hello is a simple hello world query
func (r *queryResolver) Hello(ctx context.Context) (string, error) {
	return "Hello from Cloud Price Compare GraphQL API!", nil
}

// Messages retrieves all messages
func (r *queryResolver) Messages(ctx context.Context) ([]*Message, error) {
	dbMessages, err := r.DB.GetMessages()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*Message, len(dbMessages))
	for i, msg := range dbMessages {
		messages[i] = &Message{
			ID:        strconv.Itoa(msg.ID),
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return messages, nil
}

// Providers retrieves all providers
func (r *queryResolver) Providers(ctx context.Context) ([]*Provider, error) {
	dbProviders, err := r.DB.GetProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	providers := make([]*Provider, len(dbProviders))
	for i, p := range dbProviders {
		providers[i] = &Provider{
			ID:        strconv.Itoa(p.ID),
			Name:      p.Name,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return providers, nil
}

// Categories retrieves all categories
func (r *queryResolver) Categories(ctx context.Context) ([]*Category, error) {
	dbCategories, err := r.DB.GetCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	categories := make([]*Category, len(dbCategories))
	for i, c := range dbCategories {
		categories[i] = &Category{
			ID:          strconv.Itoa(c.ID),
			Name:        c.Name,
			Description: c.Description,
			CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return categories, nil
}

// AWS returns the AWS provider resolver
func (r *queryResolver) Aws(ctx context.Context) (*AWSProvider, error) {
	return &AWSProvider{resolver: r.Resolver}, nil
}

// Azure returns the Azure provider resolver
func (r *queryResolver) Azure(ctx context.Context) (*AzureProvider, error) {
	return &AzureProvider{resolver: r.Resolver}, nil
}

type mutationResolver struct{ *Resolver }

// CreateMessage creates a new message
func (r *mutationResolver) CreateMessage(ctx context.Context, content string) (*Message, error) {
	msg, err := r.DB.CreateMessage(content)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &Message{
		ID:        strconv.Itoa(msg.ID),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// AWS Provider Implementation
type AWSProvider struct {
	resolver *Resolver
}

func (p *AWSProvider) Compute(ctx context.Context, region string) (*AWSCompute, error) {
	return &AWSCompute{resolver: p.resolver, region: region}, nil
}

func (p *AWSProvider) Storage(ctx context.Context, region string) (*AWSStorage, error) {
	return &AWSStorage{resolver: p.resolver, region: region}, nil
}

func (p *AWSProvider) DataTransfer(ctx context.Context, region string) (*AWSDataTransfer, error) {
	return &AWSDataTransfer{resolver: p.resolver, region: region}, nil
}

// AWS Compute Implementation
type AWSCompute struct {
	resolver *Resolver
	region   string
}

func (c *AWSCompute) InstancePrice(ctx context.Context, instanceType string) (float64, error) {
	// Query normalized pricing data for AWS EC2 instances
	filter := database.PricingFilter{
		Provider:         &[]string{database.ProviderAWS}[0],
		ServiceCategory:  &[]string{database.CategoryComputeWeb}[0],
		ServiceType:      &[]string{"ec2"}[0],
		NormalizedRegion: &c.region,
		PricingModel:     &[]string{database.PricingModelOnDemand}[0],
	}

	pricings, err := c.resolver.DB.QueryNormalizedPricing(filter)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query AWS instance pricing: %w", err)
	}

	// Find matching instance type
	for _, pricing := range pricings {
		if strings.Contains(strings.ToLower(pricing.ResourceName), strings.ToLower(instanceType)) {
			return pricing.PricePerUnit, nil
		}
	}

	// If not found in normalized data, return static fallback
	return getAWSInstancePricing(instanceType), nil
}

func (c *AWSCompute) Instances(ctx context.Context) ([]*AWSInstance, error) {
	// Query all AWS EC2 instances from normalized pricing
	filter := database.PricingFilter{
		Provider:         &[]string{database.ProviderAWS}[0],
		ServiceCategory:  &[]string{database.CategoryComputeWeb}[0],
		ServiceType:      &[]string{"ec2"}[0],
		NormalizedRegion: &c.region,
		PricingModel:     &[]string{database.PricingModelOnDemand}[0],
		Limit:           &[]int{50}[0], // Limit to 50 for performance
	}

	pricings, err := c.resolver.DB.QueryNormalizedPricing(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query AWS instances: %w", err)
	}

	instances := make([]*AWSInstance, 0, len(pricings))
	for _, pricing := range pricings {
		instances = append(instances, &AWSInstance{
			Type:         pricing.ResourceName,
			Vcpu:         pricing.ResourceSpecs.GetVCPU(),
			MemoryGb:     pricing.ResourceSpecs.GetMemoryGB(),
			PricePerHour: pricing.PricePerUnit,
			Architecture: pricing.ResourceSpecs.Architecture,
			Burstable:    pricing.ResourceSpecs.Burstable,
		})
	}

	return instances, nil
}

// AWS Storage Implementation
type AWSStorage struct {
	resolver *Resolver
	region   string
}

func (s *AWSStorage) PricePerGb(ctx context.Context, tier string) (float64, error) {
	filter := database.PricingFilter{
		Provider:         &[]string{database.ProviderAWS}[0],
		ServiceCategory:  &[]string{database.CategoryStorage}[0],
		ServiceType:      &[]string{"s3"}[0],
		NormalizedRegion: &s.region,
		PricingModel:     &[]string{database.PricingModelOnDemand}[0],
	}

	pricings, err := s.resolver.DB.QueryNormalizedPricing(filter)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query AWS storage pricing: %w", err)
	}

	// Find matching storage tier
	for _, pricing := range pricings {
		if strings.Contains(strings.ToLower(pricing.ResourceName), strings.ToLower(tier)) {
			return pricing.PricePerUnit, nil
		}
	}

	return getAWSStoragePricing(tier), nil
}

func (s *AWSStorage) Tiers(ctx context.Context) ([]*AWSStorageTier, error) {
	return []*AWSStorageTier{
		{Name: "standard", PricePerGb: 0.023, Description: "S3 Standard storage"},
		{Name: "infrequent_access", PricePerGb: 0.0125, Description: "S3 Infrequent Access"},
		{Name: "glacier", PricePerGb: 0.004, Description: "S3 Glacier"},
		{Name: "deep_archive", PricePerGb: 0.00099, Description: "S3 Glacier Deep Archive"},
	}, nil
}

// AWS Data Transfer Implementation
type AWSDataTransfer struct {
	resolver *Resolver
	region   string
}

func (dt *AWSDataTransfer) PricePerGb(ctx context.Context, direction string) (float64, error) {
	if direction == "in" {
		return 0.0, nil // AWS inbound is free
	}
	return 0.09, nil // Default outbound pricing
}

func (dt *AWSDataTransfer) Inbound(ctx context.Context) (float64, error) {
	return 0.0, nil // AWS inbound is always free
}

func (dt *AWSDataTransfer) Outbound(ctx context.Context) (float64, error) {
	return 0.09, nil // Standard outbound pricing
}

// Azure Provider Implementation
type AzureProvider struct {
	resolver *Resolver
}

func (p *AzureProvider) Compute(ctx context.Context, region string) (*AzureCompute, error) {
	return &AzureCompute{resolver: p.resolver, region: region}, nil
}

func (p *AzureProvider) Storage(ctx context.Context, region string) (*AzureStorage, error) {
	return &AzureStorage{resolver: p.resolver, region: region}, nil
}

func (p *AzureProvider) DataTransfer(ctx context.Context, region string) (*AzureDataTransfer, error) {
	return &AzureDataTransfer{resolver: p.resolver, region: region}, nil
}

// Azure Compute Implementation
type AzureCompute struct {
	resolver *Resolver
	region   string
}

func (c *AzureCompute) VmPrice(ctx context.Context, size string) (float64, error) {
	filter := database.PricingFilter{
		Provider:         &[]string{database.ProviderAzure}[0],
		ServiceCategory:  &[]string{database.CategoryComputeWeb}[0],
		ServiceType:      &[]string{"virtual_machines"}[0],
		NormalizedRegion: &c.region,
		PricingModel:     &[]string{database.PricingModelOnDemand}[0],
	}

	pricings, err := c.resolver.DB.QueryNormalizedPricing(filter)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query Azure VM pricing: %w", err)
	}

	// Find matching VM size
	for _, pricing := range pricings {
		if strings.Contains(strings.ToLower(pricing.ResourceName), strings.ToLower(size)) {
			return pricing.PricePerUnit, nil
		}
	}

	return getAzureVMPricing(size), nil
}

func (c *AzureCompute) Vms(ctx context.Context) ([]*AzureVM, error) {
	filter := database.PricingFilter{
		Provider:         &[]string{database.ProviderAzure}[0],
		ServiceCategory:  &[]string{database.CategoryComputeWeb}[0],
		ServiceType:      &[]string{"virtual_machines"}[0],
		NormalizedRegion: &c.region,
		PricingModel:     &[]string{database.PricingModelOnDemand}[0],
		Limit:           &[]int{50}[0],
	}

	pricings, err := c.resolver.DB.QueryNormalizedPricing(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query Azure VMs: %w", err)
	}

	vms := make([]*AzureVM, 0, len(pricings))
	for _, pricing := range pricings {
		vms = append(vms, &AzureVM{
			Size:         pricing.ResourceName,
			Vcpu:         pricing.ResourceSpecs.GetVCPU(),
			MemoryGb:     pricing.ResourceSpecs.GetMemoryGB(),
			PricePerHour: pricing.PricePerUnit,
			Architecture: pricing.ResourceSpecs.Architecture,
			Burstable:    pricing.ResourceSpecs.Burstable,
		})
	}

	return vms, nil
}

// Azure Storage Implementation
type AzureStorage struct {
	resolver *Resolver
	region   string
}

func (s *AzureStorage) PricePerGb(ctx context.Context, tier string) (float64, error) {
	filter := database.PricingFilter{
		Provider:         &[]string{database.ProviderAzure}[0],
		ServiceCategory:  &[]string{database.CategoryStorage}[0],
		ServiceType:      &[]string{"blob_storage"}[0],
		NormalizedRegion: &s.region,
		PricingModel:     &[]string{database.PricingModelOnDemand}[0],
	}

	pricings, err := s.resolver.DB.QueryNormalizedPricing(filter)
	if err != nil {
		return 0.0, fmt.Errorf("failed to query Azure storage pricing: %w", err)
	}

	for _, pricing := range pricings {
		if strings.Contains(strings.ToLower(pricing.ResourceName), strings.ToLower(tier)) {
			return pricing.PricePerUnit, nil
		}
	}

	return getAzureStoragePricing(tier), nil
}

func (s *AzureStorage) Tiers(ctx context.Context) ([]*AzureStorageTier, error) {
	return []*AzureStorageTier{
		{Name: "hot", PricePerGb: 0.0184, Description: "Hot access tier"},
		{Name: "cool", PricePerGb: 0.01, Description: "Cool access tier"},
		{Name: "archive", PricePerGb: 0.00099, Description: "Archive access tier"},
	}, nil
}

// Azure Data Transfer Implementation
type AzureDataTransfer struct {
	resolver *Resolver
	region   string
}

func (dt *AzureDataTransfer) PricePerGb(ctx context.Context, direction string) (float64, error) {
	if direction == "in" {
		return 0.0, nil // Azure inbound is free
	}
	return 0.0877, nil // Default outbound pricing
}

func (dt *AzureDataTransfer) Inbound(ctx context.Context) (float64, error) {
	return 0.0, nil
}

func (dt *AzureDataTransfer) Outbound(ctx context.Context) (float64, error) {
	return 0.0877, nil
}

// Helper functions for fallback pricing
func getAWSInstancePricing(instanceType string) float64 {
	pricing := map[string]float64{
		"t3.micro":  0.0104,
		"t3.small":  0.0208,
		"t3.medium": 0.0416,
		"m5.large":  0.096,
		"m5.xlarge": 0.192,
		"c5.large":  0.085,
		"c5.xlarge": 0.17,
		"r5.large":  0.126,
		"r5.xlarge": 0.252,
	}
	
	if price, exists := pricing[instanceType]; exists {
		return price
	}
	return 0.10 // Default fallback
}

func getAWSStoragePricing(tier string) float64 {
	pricing := map[string]float64{
		"standard":          0.023,
		"infrequent_access": 0.0125,
		"glacier":           0.004,
		"deep_archive":      0.00099,
	}
	
	if price, exists := pricing[tier]; exists {
		return price
	}
	return 0.023 // Default to standard
}

func getAzureVMPricing(size string) float64 {
	pricing := map[string]float64{
		"Standard_B1s":         0.0104,
		"Standard_B2s":         0.0416,
		"Standard_D2s_v3":      0.096,
		"Standard_D4s_v3":      0.192,
		"Standard_F2s_v2":      0.085,
		"Standard_F4s_v2":      0.17,
		"Standard_NC8as_T4_v3": 1.204,
		"Standard_NC16as_T4_v3": 2.408,
	}
	
	if price, exists := pricing[size]; exists {
		return price
	}
	return 0.10 // Default fallback
}

func getAzureStoragePricing(tier string) float64 {
	pricing := map[string]float64{
		"hot":     0.0184,
		"cool":    0.01,
		"archive": 0.00099,
	}
	
	if price, exists := pricing[tier]; exists {
		return price
	}
	return 0.0184 // Default to hot
}