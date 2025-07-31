package etl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raulc0399/cpc/internal/database"
	"github.com/raulc0399/cpc/internal/normalizer"
)

// Pipeline manages the ETL process for normalizing raw pricing data
type Pipeline struct {
	db                 *database.DB
	awsNormalizer      *normalizer.AWSNormalizerV2
	azureNormalizer    *normalizer.AzureNormalizerV2
	serviceMappingRepo normalizer.ServiceMappingRepository
	regionMappingRepo  normalizer.RegionMappingRepository
	unitNormalizer     normalizer.UnitNormalizer
	pricingRepo        normalizer.NormalizedPricingRepository
	logger             normalizer.Logger
	mu                 sync.RWMutex
	runningJobs        map[string]*Job
}

// Job represents an ETL job
type Job struct {
	ID               string                    `json:"id"`
	Type             JobType                   `json:"type"`
	Provider         string                    `json:"provider"`
	Status           JobStatus                 `json:"status"`
	Progress         *JobProgress              `json:"progress"`
	StartedAt        time.Time                 `json:"startedAt"`
	CompletedAt      *time.Time                `json:"completedAt"`
	Error            string                    `json:"error,omitempty"`
	Configuration    JobConfiguration          `json:"configuration"`
	ctx              context.Context
	cancel           context.CancelFunc
}

// JobType represents the type of ETL job
type JobType string

const (
	JobTypeNormalizeAll          JobType = "normalize_all"
	JobTypeNormalizeProvider     JobType = "normalize_provider"
	JobTypeNormalizeRegion       JobType = "normalize_region"
	JobTypeNormalizeService      JobType = "normalize_service"
	JobTypeCleanupNormalized     JobType = "cleanup_normalized"
)

// JobStatus represents the status of an ETL job
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

// JobProgress tracks the progress of an ETL job
type JobProgress struct {
	TotalRecords      int       `json:"totalRecords"`
	ProcessedRecords  int       `json:"processedRecords"`
	NormalizedRecords int       `json:"normalizedRecords"`
	SkippedRecords    int       `json:"skippedRecords"`
	ErrorRecords      int       `json:"errorRecords"`
	CurrentStage      string    `json:"currentStage"`
	LastUpdated       time.Time `json:"lastUpdated"`
	Rate              float64   `json:"rate"` // records per second
}

// JobConfiguration holds configuration for ETL jobs
type JobConfiguration struct {
	Providers         []string `json:"providers,omitempty"`         // AWS, Azure
	Regions           []string `json:"regions,omitempty"`           // Specific regions
	Services          []string `json:"services,omitempty"`          // Specific services
	BatchSize         int      `json:"batchSize"`                   // Records per batch
	ConcurrentWorkers int      `json:"concurrentWorkers"`           // Parallel workers
	ClearExisting     bool     `json:"clearExisting"`               // Clear existing normalized data
	DryRun            bool     `json:"dryRun"`                      // Don't actually insert
}

// NewPipeline creates a new ETL pipeline
func NewPipeline(db *database.DB) (*Pipeline, error) {
	logger := normalizer.NewSimpleLogger()
	
	// Create repositories
	serviceMappingRepo := normalizer.NewServiceMappingRepository(db, logger)
	regionMappingRepo := normalizer.NewRegionMappingRepository(db, logger)
	unitNormalizer := normalizer.NewStandardUnitNormalizer()
	validator := normalizer.NewInputValidator()
	pricingRepo := normalizer.NewNormalizedPricingRepository(db, logger)
	
	// Create normalizers
	awsNormalizer := normalizer.NewAWSNormalizerV2(
		serviceMappingRepo,
		regionMappingRepo,
		unitNormalizer,
		validator,
		logger,
	)
	
	azureNormalizer := normalizer.NewAzureNormalizerV2(
		serviceMappingRepo,
		regionMappingRepo,
		unitNormalizer,
		validator,
		logger,
	)
	
	return &Pipeline{
		db:                 db,
		awsNormalizer:      awsNormalizer,
		azureNormalizer:    azureNormalizer,
		serviceMappingRepo: serviceMappingRepo,
		regionMappingRepo:  regionMappingRepo,
		unitNormalizer:     unitNormalizer,
		pricingRepo:        pricingRepo,
		logger:             logger,
		runningJobs:        make(map[string]*Job),
	}, nil
}

// StartJob starts a new ETL job
func (p *Pipeline) StartJob(jobType JobType, config JobConfiguration) (*Job, error) {
	jobID := fmt.Sprintf("%s-%d", jobType, time.Now().Unix())
	
	// Set default configuration
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}
	if config.ConcurrentWorkers == 0 {
		config.ConcurrentWorkers = 4
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	job := &Job{
		ID:            jobID,
		Type:          jobType,
		Status:        StatusPending,
		StartedAt:     time.Now(),
		Configuration: config,
		Progress: &JobProgress{
			CurrentStage: "Initializing",
			LastUpdated:  time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}
	
	p.mu.Lock()
	p.runningJobs[jobID] = job
	p.mu.Unlock()
	
	// Start job in goroutine
	go p.executeJob(job)
	
	p.logger.Info("Started ETL job",
		normalizer.Field{"jobId", jobID},
		normalizer.Field{"type", jobType},
		normalizer.Field{"config", config},
	)
	
	return job, nil
}

// executeJob executes an ETL job
func (p *Pipeline) executeJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			job.Status = StatusFailed
			job.Error = fmt.Sprintf("Job panicked: %v", r)
			completedAt := time.Now()
			job.CompletedAt = &completedAt
		}
		
		p.mu.Lock()
		delete(p.runningJobs, job.ID)
		p.mu.Unlock()
	}()
	
	job.Status = StatusRunning
	
	p.logger.Info("Executing ETL job",
		normalizer.Field{"jobId", job.ID},
		normalizer.Field{"type", job.Type},
	)
	
	var err error
	switch job.Type {
	case JobTypeNormalizeAll:
		err = p.normalizeAll(job)
	case JobTypeNormalizeProvider:
		err = p.normalizeProvider(job)
	case JobTypeNormalizeRegion:
		err = p.normalizeRegion(job)
	case JobTypeNormalizeService:
		err = p.normalizeService(job)
	case JobTypeCleanupNormalized:
		err = p.cleanupNormalized(job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}
	
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	
	if err != nil {
		job.Status = StatusFailed
		job.Error = err.Error()
		p.logger.Error("ETL job failed",
			normalizer.Field{"jobId", job.ID},
			normalizer.Field{"error", err},
		)
	} else {
		job.Status = StatusCompleted
		p.logger.Info("ETL job completed",
			normalizer.Field{"jobId", job.ID},
			normalizer.Field{"processed", job.Progress.ProcessedRecords},
			normalizer.Field{"normalized", job.Progress.NormalizedRecords},
			normalizer.Field{"skipped", job.Progress.SkippedRecords},
			normalizer.Field{"errors", job.Progress.ErrorRecords},
		)
	}
}

// normalizeAll normalizes all raw pricing data
func (p *Pipeline) normalizeAll(job *Job) error {
	job.Progress.CurrentStage = "Preparing normalization"
	job.Progress.LastUpdated = time.Now()
	
	// Clear existing normalized data if requested
	if job.Configuration.ClearExisting {
		job.Progress.CurrentStage = "Clearing existing normalized data"
		if err := p.clearNormalizedData(job.ctx); err != nil {
			return fmt.Errorf("failed to clear existing data: %w", err)
		}
	}
	
	// Determine providers to process
	providers := job.Configuration.Providers
	if len(providers) == 0 {
		providers = []string{database.ProviderAWS, database.ProviderAzure}
	}
	
	// Process each provider
	for _, provider := range providers {
		select {
		case <-job.ctx.Done():
			return fmt.Errorf("job cancelled")
		default:
		}
		
		job.Provider = provider
		job.Progress.CurrentStage = fmt.Sprintf("Processing %s data", provider)
		job.Progress.LastUpdated = time.Now()
		
		if err := p.normalizeProviderData(job, provider); err != nil {
			return fmt.Errorf("failed to normalize %s data: %w", provider, err)
		}
	}
	
	return nil
}

// normalizeProvider normalizes data for a specific provider
func (p *Pipeline) normalizeProvider(job *Job) error {
	if len(job.Configuration.Providers) == 0 {
		return fmt.Errorf("no provider specified in configuration")
	}
	
	provider := job.Configuration.Providers[0]
	job.Provider = provider
	
	return p.normalizeProviderData(job, provider)
}

// normalizeProviderData normalizes data for a specific provider
func (p *Pipeline) normalizeProviderData(job *Job, provider string) error {
	switch provider {
	case database.ProviderAzure:
		return p.normalizeAzureData(job)
	case database.ProviderAWS:
		return p.normalizeAWSData(job)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// GetJob retrieves a job by ID
func (p *Pipeline) GetJob(jobID string) (*Job, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	job, exists := p.runningJobs[jobID]
	return job, exists
}

// GetAllJobs retrieves all jobs
func (p *Pipeline) GetAllJobs() []*Job {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	jobs := make([]*Job, 0, len(p.runningJobs))
	for _, job := range p.runningJobs {
		jobs = append(jobs, job)
	}
	
	return jobs
}

// CancelJob cancels a running job
func (p *Pipeline) CancelJob(jobID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	job, exists := p.runningJobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	
	if job.Status == StatusCompleted || job.Status == StatusFailed {
		return fmt.Errorf("cannot cancel job in status: %s", job.Status)
	}
	
	job.cancel()
	job.Status = StatusCancelled
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	
	p.logger.Info("Cancelled ETL job", normalizer.Field{"jobId", jobID})
	
	return nil
}

// clearNormalizedData clears all normalized pricing data
func (p *Pipeline) clearNormalizedData(ctx context.Context) error {
	query := `DELETE FROM normalized_pricing`
	_, err := p.db.GetConn().ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear normalized data: %w", err)
	}
	
	p.logger.Info("Cleared existing normalized data")
	return nil
}

// updateJobProgress updates job progress
func (p *Pipeline) updateJobProgress(job *Job, processed, normalized, skipped, errors int) {
	job.Progress.ProcessedRecords += processed
	job.Progress.NormalizedRecords += normalized
	job.Progress.SkippedRecords += skipped
	job.Progress.ErrorRecords += errors
	job.Progress.LastUpdated = time.Now()
	
	// Calculate rate (records per second)
	duration := time.Since(job.StartedAt).Seconds()
	if duration > 0 {
		job.Progress.Rate = float64(job.Progress.ProcessedRecords) / duration
	}
}