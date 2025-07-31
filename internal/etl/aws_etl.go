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

// normalizeAWSData normalizes all AWS raw pricing data
func (p *Pipeline) normalizeAWSData(job *Job) error {
	job.Progress.CurrentStage = "Counting AWS raw records"
	job.Progress.LastUpdated = now()
	
	// Get total count for progress tracking
	totalCount, err := p.getAWSRawDataCount(job.ctx, job.Configuration)
	if err != nil {
		return fmt.Errorf("failed to count AWS raw data: %w", err)
	}
	
	job.Progress.TotalRecords = totalCount
	job.Progress.CurrentStage = "Processing AWS raw data"
	
	p.logger.Info("Starting AWS normalization",
		normalizer.Field{"totalRecords", totalCount},
		normalizer.Field{"batchSize", job.Configuration.BatchSize},
		normalizer.Field{"workers", job.Configuration.ConcurrentWorkers},
	)
	
	// Process data in batches with concurrent workers
	return p.processAWSDataInBatches(job)
}

// getAWSRawDataCount counts AWS raw pricing records matching the job configuration
func (p *Pipeline) getAWSRawDataCount(ctx context.Context, config JobConfiguration) (int, error) {
	// Check if aws_pricing_raw table exists
	tableExistsQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'aws_pricing_raw'
		)`
	
	var tableExists bool
	err := p.db.GetConn().QueryRowContext(ctx, tableExistsQuery).Scan(&tableExists)
	if err != nil {
		return 0, fmt.Errorf("failed to check AWS table existence: %w", err)
	}
	
	if !tableExists {
		p.logger.Warn("AWS pricing raw table does not exist, skipping AWS normalization")
		return 0, nil
	}
	
	query := "SELECT COUNT(*) FROM aws_pricing_raw WHERE 1=1"
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
		query += fmt.Sprintf(" AND service_code IN (%s)", strings.Join(placeholders, ","))
	}
	
	var count int
	err = p.db.GetConn().QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count AWS records: %w", err)
	}
	
	return count, nil
}

// processAWSDataInBatches processes AWS data in batches with concurrent workers
func (p *Pipeline) processAWSDataInBatches(job *Job) error {
	if job.Progress.TotalRecords == 0 {
		p.logger.Info("No AWS data to process")
		return nil
	}
	
	batchSize := job.Configuration.BatchSize
	workerCount := job.Configuration.ConcurrentWorkers
	
	// Create worker pool
	batchChan := make(chan *AWSBatch, workerCount*2)
	resultChan := make(chan *BatchResult, workerCount*2)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go p.awsWorker(job, batchChan, resultChan, &wg)
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
		
		batch, err := p.getAWSBatch(job.ctx, job.Configuration, offset, batchSize)
		if err != nil {
			close(batchChan)
			wg.Wait()
			close(resultChan)
			collectorWg.Wait()
			return fmt.Errorf("failed to get AWS batch at offset %d: %w", offset, err)
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

// AWSBatch represents a batch of AWS raw pricing records
type AWSBatch struct {
	Offset  int
	Records []*AWSRawRecord
}

// AWSRawRecord represents a single AWS raw pricing record
type AWSRawRecord struct {
	ID           int
	ServiceCode  string
	ServiceName  string  
	Region       string
	Data         json.RawMessage
	CollectionID string
}

// getAWSBatch retrieves a batch of AWS raw pricing data
func (p *Pipeline) getAWSBatch(ctx context.Context, config JobConfiguration, offset, limit int) (*AWSBatch, error) {
	query := `
		SELECT id, service_code, service_name, region, data, collection_id 
		FROM aws_pricing_raw 
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
		query += fmt.Sprintf(" AND service_code IN (%s)", strings.Join(placeholders, ","))
	}
	
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)
	
	rows, err := p.db.GetConn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query AWS batch: %w", err)
	}
	defer rows.Close()
	
	var records []*AWSRawRecord
	for rows.Next() {
		record := &AWSRawRecord{}
		err := rows.Scan(
			&record.ID,
			&record.ServiceCode,
			&record.ServiceName,
			&record.Region,
			&record.Data,
			&record.CollectionID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan AWS record: %w", err)
		}
		records = append(records, record)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating AWS records: %w", err)
	}
	
	return &AWSBatch{
		Offset:  offset,
		Records: records,
	}, nil
}

// awsWorker processes batches of AWS data
func (p *Pipeline) awsWorker(job *Job, batchChan <-chan *AWSBatch, resultChan chan<- *BatchResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for batch := range batchChan {
		select {
		case <-job.ctx.Done():
			return
		default:
		}
		
		result := p.processAWSBatch(job, batch)
		resultChan <- result
	}
}

// processAWSBatch processes a single batch of AWS data
func (p *Pipeline) processAWSBatch(job *Job, batch *AWSBatch) *BatchResult {
	result := &BatchResult{
		BatchOffset: batch.Offset,
		Errors:      []string{},
	}
	
	var normalizedRecords []database.NormalizedPricing
	
	for _, record := range batch.Records {
		result.ProcessedRecords++
		
		// Create normalization input
		input := database.NormalizationInput{
			Provider:     database.ProviderAWS,
			ServiceCode:  record.ServiceCode,
			Region:       record.Region,
			RawData:      record.Data,
			RawDataID:    record.ID,
			CollectionID: record.CollectionID,
		}
		
		// Normalize the record
		normResult, err := p.awsNormalizer.NormalizePricing(job.ctx, input)
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

// normalizeRegion normalizes data for specific regions
func (p *Pipeline) normalizeRegion(job *Job) error {
	if len(job.Configuration.Regions) == 0 {
		return fmt.Errorf("no regions specified in configuration")
	}
	
	// Process each provider that has the specified regions
	providers := job.Configuration.Providers
	if len(providers) == 0 {
		providers = []string{database.ProviderAWS, database.ProviderAzure}
	}
	
	for _, provider := range providers {
		select {
		case <-job.ctx.Done():
			return fmt.Errorf("job cancelled")
		default:
		}
		
		job.Provider = provider
		job.Progress.CurrentStage = fmt.Sprintf("Processing %s regions: %v", provider, job.Configuration.Regions)
		
		if err := p.normalizeProviderData(job, provider); err != nil {
			return fmt.Errorf("failed to normalize %s regions: %w", provider, err)
		}
	}
	
	return nil
}

// normalizeService normalizes data for specific services
func (p *Pipeline) normalizeService(job *Job) error {
	if len(job.Configuration.Services) == 0 {
		return fmt.Errorf("no services specified in configuration")
	}
	
	// Process each provider that has the specified services
	providers := job.Configuration.Providers
	if len(providers) == 0 {
		providers = []string{database.ProviderAWS, database.ProviderAzure}
	}
	
	for _, provider := range providers {
		select {
		case <-job.ctx.Done():
			return fmt.Errorf("job cancelled")
		default:
		}
		
		job.Provider = provider
		job.Progress.CurrentStage = fmt.Sprintf("Processing %s services: %v", provider, job.Configuration.Services)
		
		if err := p.normalizeProviderData(job, provider); err != nil {
			return fmt.Errorf("failed to normalize %s services: %w", provider, err)
		}
	}
	
	return nil
}

// cleanupNormalized removes old/invalid normalized data
func (p *Pipeline) cleanupNormalized(job *Job) error {
	job.Progress.CurrentStage = "Cleaning up normalized data"
	job.Progress.LastUpdated = now()
	
	// Remove normalized records without corresponding raw data
	query := `
		DELETE FROM normalized_pricing 
		WHERE (aws_raw_id IS NOT NULL AND aws_raw_id NOT IN (SELECT id FROM aws_pricing_raw))
		   OR (azure_raw_id IS NOT NULL AND azure_raw_id NOT IN (SELECT id FROM azure_pricing_raw))`
	
	result, err := p.db.GetConn().ExecContext(job.ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup normalized data: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	job.Progress.ProcessedRecords = int(rowsAffected)
	
	p.logger.Info("Cleaned up normalized data",
		normalizer.Field{"removedRecords", rowsAffected},
	)
	
	return nil
}