package normalizer

import (
	"context"
	"fmt"

	"github.com/raulc0399/cpc/internal/database"
)

// MockServiceMappingRepository is a mock implementation of ServiceMappingRepository
type MockServiceMappingRepository struct {
	Mappings map[string]*database.ServiceMapping
	Error    error
	CallCount int
}

// NewMockServiceMappingRepository creates a new mock service mapping repository
func NewMockServiceMappingRepository() *MockServiceMappingRepository {
	return &MockServiceMappingRepository{
		Mappings: make(map[string]*database.ServiceMapping),
	}
}

// GetServiceMappingByProvider mock implementation
func (m *MockServiceMappingRepository) GetServiceMappingByProvider(ctx context.Context, provider, serviceName string) (*database.ServiceMapping, error) {
	m.CallCount++
	if m.Error != nil {
		return nil, m.Error
	}
	key := fmt.Sprintf("%s:%s", provider, serviceName)
	return m.Mappings[key], nil
}

// GetAllServiceMappings mock implementation
func (m *MockServiceMappingRepository) GetAllServiceMappings(ctx context.Context) ([]database.ServiceMapping, error) {
	m.CallCount++
	if m.Error != nil {
		return nil, m.Error
	}
	var mappings []database.ServiceMapping
	for _, mapping := range m.Mappings {
		if mapping != nil {
			mappings = append(mappings, *mapping)
		}
	}
	return mappings, nil
}

// AddMapping adds a mapping to the mock repository
func (m *MockServiceMappingRepository) AddMapping(provider, serviceName string, mapping *database.ServiceMapping) {
	key := fmt.Sprintf("%s:%s", provider, serviceName)
	m.Mappings[key] = mapping
}

// MockRegionMappingRepository is a mock implementation of RegionMappingRepository
type MockRegionMappingRepository struct {
	Regions   map[string]*database.NormalizedRegion
	Error     error
	CallCount int
}

// NewMockRegionMappingRepository creates a new mock region mapping repository
func NewMockRegionMappingRepository() *MockRegionMappingRepository {
	return &MockRegionMappingRepository{
		Regions: make(map[string]*database.NormalizedRegion),
	}
}

// GetNormalizedRegionByProvider mock implementation
func (m *MockRegionMappingRepository) GetNormalizedRegionByProvider(ctx context.Context, provider, providerRegion string) (*database.NormalizedRegion, error) {
	m.CallCount++
	if m.Error != nil {
		return nil, m.Error
	}
	key := fmt.Sprintf("%s:%s", provider, providerRegion)
	return m.Regions[key], nil
}

// GetAllNormalizedRegions mock implementation
func (m *MockRegionMappingRepository) GetAllNormalizedRegions(ctx context.Context) ([]database.NormalizedRegion, error) {
	m.CallCount++
	if m.Error != nil {
		return nil, m.Error
	}
	var regions []database.NormalizedRegion
	for _, region := range m.Regions {
		if region != nil {
			regions = append(regions, *region)
		}
	}
	return regions, nil
}

// AddRegion adds a region to the mock repository
func (m *MockRegionMappingRepository) AddRegion(provider, providerRegion string, region *database.NormalizedRegion) {
	key := fmt.Sprintf("%s:%s", provider, providerRegion)
	m.Regions[key] = region
}

// MockUnitNormalizer is a mock implementation of UnitNormalizer
type MockUnitNormalizer struct {
	Mappings  map[string]string
	CallCount int
}

// NewMockUnitNormalizer creates a new mock unit normalizer
func NewMockUnitNormalizer() *MockUnitNormalizer {
	return &MockUnitNormalizer{
		Mappings: make(map[string]string),
	}
}

// NormalizeUnit mock implementation
func (m *MockUnitNormalizer) NormalizeUnit(provider, originalUnit string) string {
	m.CallCount++
	key := fmt.Sprintf("%s:%s", provider, originalUnit)
	if normalized, exists := m.Mappings[key]; exists {
		return normalized
	}
	// Default behavior - return original
	return originalUnit
}

// AddMapping adds a unit mapping
func (m *MockUnitNormalizer) AddMapping(provider, originalUnit, normalizedUnit string) {
	key := fmt.Sprintf("%s:%s", provider, originalUnit)
	m.Mappings[key] = normalizedUnit
}

// MockLogger is a mock implementation of Logger
type MockLogger struct {
	Messages []LogMessage
}

// LogMessage represents a logged message
type LogMessage struct {
	Level   string
	Message string
	Fields  []Field
}

// NewMockLogger creates a new mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: make([]LogMessage, 0),
	}
}

// Debug mock implementation
func (l *MockLogger) Debug(msg string, fields ...Field) {
	l.Messages = append(l.Messages, LogMessage{
		Level:   "DEBUG",
		Message: msg,
		Fields:  fields,
	})
}

// Info mock implementation
func (l *MockLogger) Info(msg string, fields ...Field) {
	l.Messages = append(l.Messages, LogMessage{
		Level:   "INFO",
		Message: msg,
		Fields:  fields,
	})
}

// Warn mock implementation
func (l *MockLogger) Warn(msg string, fields ...Field) {
	l.Messages = append(l.Messages, LogMessage{
		Level:   "WARN",
		Message: msg,
		Fields:  fields,
	})
}

// Error mock implementation
func (l *MockLogger) Error(msg string, fields ...Field) {
	l.Messages = append(l.Messages, LogMessage{
		Level:   "ERROR",
		Message: msg,
		Fields:  fields,
	})
}

// HasMessage checks if a message with given level and text exists
func (l *MockLogger) HasMessage(level, message string) bool {
	for _, msg := range l.Messages {
		if msg.Level == level && msg.Message == message {
			return true
		}
	}
	return false
}

// GetMessages returns all messages of a specific level
func (l *MockLogger) GetMessages(level string) []LogMessage {
	var messages []LogMessage
	for _, msg := range l.Messages {
		if msg.Level == level {
			messages = append(messages, msg)
		}
	}
	return messages
}

// MockNormalizedPricingRepository is a mock implementation of NormalizedPricingRepository
type MockNormalizedPricingRepository struct {
	Records   []database.NormalizedPricing
	Error     error
	CallCount int
}

// NewMockNormalizedPricingRepository creates a new mock normalized pricing repository
func NewMockNormalizedPricingRepository() *MockNormalizedPricingRepository {
	return &MockNormalizedPricingRepository{
		Records: make([]database.NormalizedPricing, 0),
	}
}

// Insert mock implementation
func (m *MockNormalizedPricingRepository) Insert(ctx context.Context, pricing *database.NormalizedPricing) error {
	m.CallCount++
	if m.Error != nil {
		return m.Error
	}
	m.Records = append(m.Records, *pricing)
	return nil
}

// BulkInsert mock implementation
func (m *MockNormalizedPricingRepository) BulkInsert(ctx context.Context, pricings []database.NormalizedPricing) error {
	m.CallCount++
	if m.Error != nil {
		return m.Error
	}
	m.Records = append(m.Records, pricings...)
	return nil
}

// Query mock implementation
func (m *MockNormalizedPricingRepository) Query(ctx context.Context, filter database.PricingFilter) ([]database.NormalizedPricing, error) {
	m.CallCount++
	if m.Error != nil {
		return nil, m.Error
	}
	
	// Simple filter implementation for testing
	var results []database.NormalizedPricing
	for _, record := range m.Records {
		match := true
		
		if filter.Provider != nil && record.Provider != *filter.Provider {
			match = false
		}
		if filter.ServiceType != nil && record.ServiceType != *filter.ServiceType {
			match = false
		}
		if filter.NormalizedRegion != nil && record.NormalizedRegion != *filter.NormalizedRegion {
			match = false
		}
		
		if match {
			results = append(results, record)
		}
	}
	
	return results, nil
}

// Clear clears all records
func (m *MockNormalizedPricingRepository) Clear() {
	m.Records = make([]database.NormalizedPricing, 0)
	m.CallCount = 0
}