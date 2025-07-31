package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/raulc0399/cpc/internal/database"
	"github.com/raulc0399/cpc/internal/normalizer"
)

// normalizeAzureData normalizes all Azure raw pricing data
func (p *Pipeline) normalizeAzureData(job *Job) error {
	job.Progress.CurrentStage = "Counting Azure raw records"
	job.Progress.LastUpdated = now()
	
	// Get total count for progress tracking
	totalCount, err := p.getAzureRawDataCount(job.ctx, job.Configuration)
	if err != nil {
		return fmt.Errorf("failed to count Azure raw data: %w", err)
	}
	
	job.Progress.TotalRecords = totalCount
	job.Progress.CurrentStage = "Processing Azure raw data"
	
	p.logger.Info("Starting Azure normalization",
		normalizer.Field{"totalRecords", totalCount},
		normalizer.Field{"batchSize", job.Configuration.BatchSize},
		normalizer.Field{"workers", job.Configuration.ConcurrentWorkers},
	)
	
	// Process data in batches with concurrent workers
	return p.processAzureDataInBatches(job)
}

// getAzureRawDataCount counts Azure raw pricing records matching the job configuration
func (p *Pipeline) getAzureRawDataCount(ctx context.Context, config JobConfiguration) (int, error) {
	query := "SELECT COUNT(*) FROM azure_pricing_raw WHERE 1=1"
	args := []interface{}{}
	
	// Add filters based on configuration
	if len(config.Regions) > 0 {
		placeholders := make([]string, len(config.Regions))
		for i, region := range config.Regions {
			placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
			args = append(args, region)
		}
		query += fmt.Sprintf(" AND region IN (%s)", strings.Join(placeholders, ","))
	}
	
	if len(config.Services) > 0 {
		placeholders := make([]string, len(config.Services))
		for i, service := range config.Services {
			placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
			args = append(args, service)
		}
		query += fmt.Sprintf(" AND service_name IN (%s)", strings.Join(placeholders, ","))
	}
	
	var count int
	err := p.db.GetConn().QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count Azure records: %w", err)
	}
	
	return count, nil
}

// processAzureDataInBatches processes Azure data in batches with concurrent workers
func (p *Pipeline) processAzureDataInBatches(job *Job) error {
	batchSize := job.Configuration.BatchSize
	workerCount := job.Configuration.ConcurrentWorkers
	
	// Create worker pool
	batchChan := make(chan *AzureBatch, workerCount*2)
	resultChan := make(chan *BatchResult, workerCount*2)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go p.azureWorker(job, batchChan, resultChan, &wg)
	}
	
	// Start result collector
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go p.collectResults(job, resultChan, &collectorWg)
	
	// Generate batches
	offset := 0
	for {
		select {
		case <-job.ctx.Done():
			close(batchChan)
			wg.Wait()
			close(resultChan)
			collectorWg.Wait()
			return fmt.Errorf("job cancelled")
		default:
		}
		
		batch, err := p.getAzureBatch(job.ctx, job.Configuration, offset, batchSize)
		if err != nil {
			close(batchChan)
			wg.Wait()
			close(resultChan)
			collectorWg.Wait()
			return fmt.Errorf("failed to get batch at offset %d: %w", offset, err)
		}
		
		if len(batch.Records) == 0 {
			break // No more data
		}
		
		batchChan <- batch
		offset += batchSize
	}
	
	// Close channels and wait for workers
	close(batchChan)
	wg.Wait()
	close(resultChan)
	collectorWg.Wait()
	
	return nil
}

// AzureBatch represents a batch of Azure raw pricing records
type AzureBatch struct {
	Offset  int
	Records []*AzureRawRecord
}

// AzureRawRecord represents a single Azure raw pricing record
type AzureRawRecord struct {
	ID           int
	Region       string
	ServiceName  *string
	ServiceFamily *string
	Data         json.RawMessage
	CollectionID string
}

// BatchResult represents the result of processing a batch
type BatchResult struct {
	BatchOffset       int
	ProcessedRecords  int
	NormalizedRecords int
	SkippedRecords    int
	ErrorRecords      int
	Errors            []string
}

// getAzureBatch retrieves a batch of Azure raw pricing data
func (p *Pipeline) getAzureBatch(ctx context.Context, config JobConfiguration, offset, limit int) (*AzureBatch, error) {
	query := `
		SELECT id, region, service_name, service_family, data, collection_id 
		FROM azure_pricing_raw 
		WHERE 1=1`
	args := []interface{}{}
	
	// Add filters
	if len(config.Regions) > 0 {
		placeholders := make([]string, len(config.Regions))
		for i, region := range config.Regions {
			placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
			args = append(args, region)
		}
		query += fmt.Sprintf(" AND region IN (%s)", strings.Join(placeholders, ","))
	}
	
	if len(config.Services) > 0 {
		placeholders := make([]string, len(config.Services))
		for i, service := range config.Services {
			placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
			args = append(args, service)
		}
		query += fmt.Sprintf(" AND service_name IN (%s)", strings.Join(placeholders, ","))
	}
	
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)
	
	rows, err := p.db.GetConn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query Azure batch: %w", err)
	}
	defer rows.Close()
	
	var records []*AzureRawRecord
	for rows.Next() {
		record := &AzureRawRecord{}
		err := rows.Scan(
			&record.ID,
			&record.Region,
			&record.ServiceName,
			&record.ServiceFamily,
			&record.Data,
			&record.CollectionID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan Azure record: %w", err)
		}
		records = append(records, record)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Azure records: %w", err)
	}
	
	return &AzureBatch{
		Offset:  offset,
		Records: records,
	}, nil
}

// azureWorker processes batches of Azure data
func (p *Pipeline) azureWorker(job *Job, batchChan <-chan *AzureBatch, resultChan chan<- *BatchResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for batch := range batchChan {
		select {
		case <-job.ctx.Done():
			return
		default:
		}
		
		result := p.processAzureBatch(job, batch)
		resultChan <- result
	}
}

// processAzureBatch processes a single batch of Azure data
func (p *Pipeline) processAzureBatch(job *Job, batch *AzureBatch) *BatchResult {
	result := &BatchResult{
		BatchOffset: batch.Offset,
		Errors:      []string{},
	}
	
	var normalizedRecords []database.NormalizedPricing
	
	for _, record := range batch.Records {
		result.ProcessedRecords++
		
		// Determine service name for mapping
		var serviceName string
		if record.ServiceName != nil {
			serviceName = *record.ServiceName
		}
		
		// Create normalization input
		input := database.NormalizationInput{
			Provider:     database.ProviderAzure,
			ServiceCode:  serviceName,
			Region:       record.Region,
			RawData:      record.Data,
			RawDataID:    record.ID,
			CollectionID: record.CollectionID,
		}
		
		// Normalize the record
		normResult, err := p.azureNormalizer.NormalizePricing(job.ctx, input)
		if err != nil {
			result.ErrorRecords++
			result.Errors = append(result.Errors, fmt.Sprintf("Record ID %d: %v", record.ID, err))
			continue
		}
		
		if !normResult.Success {
			result.SkippedRecords += normResult.SkippedCount
			if normResult.ErrorCount > 0 {
				result.ErrorRecords += normResult.ErrorCount
				result.Errors = append(result.Errors, normResult.Errors...)
			}
			continue
		}
		
		// Add normalized records to batch
		normalizedRecords = append(normalizedRecords, normResult.NormalizedRecords...)
		result.NormalizedRecords += len(normResult.NormalizedRecords)
	}
	
	// Insert normalized records if not a dry run
	if !job.Configuration.DryRun && len(normalizedRecords) > 0 {
		err := p.pricingRepo.BulkInsert(job.ctx, normalizedRecords)
		if err != nil {
			result.ErrorRecords += len(normalizedRecords)
			result.NormalizedRecords -= len(normalizedRecords)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert batch: %v", err))
		}
	}
	
	return result
}

// collectResults collects results from worker goroutines and updates job progress
func (p *Pipeline) collectResults(job *Job, resultChan <-chan *BatchResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for result := range resultChan {
		p.updateJobProgress(job, 
			result.ProcessedRecords,
			result.NormalizedRecords,
			result.SkippedRecords,
			result.ErrorRecords,
		)
		
		// Log errors if any
		for _, errMsg := range result.Errors {
			p.logger.Error("Batch processing error", normalizer.Field{"error", errMsg})
		}
		
		// Log progress periodically
		if job.Progress.ProcessedRecords%10000 == 0 {
			p.logger.Info("Azure normalization progress",
				normalizer.Field{"processed", job.Progress.ProcessedRecords},
				normalizer.Field{"total", job.Progress.TotalRecords},
				normalizer.Field{"normalized", job.Progress.NormalizedRecords},
				normalizer.Field{"rate", job.Progress.Rate},
			)
		}
	}
}