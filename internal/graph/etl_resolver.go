package graph

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/raulc0399/cpc/internal/etl"
)

// ETL-related resolver methods

// EtlJob retrieves a specific ETL job by ID
func (r *queryResolver) EtlJob(ctx context.Context, id string) (*ETLJob, error) {
	if r.pipeline == nil {
		return nil, fmt.Errorf("ETL pipeline not initialized")
	}
	
	job, exists := r.pipeline.GetJob(id)
	if !exists {
		return nil, nil // GraphQL handles null return for not found
	}
	
	return convertJobToGraphQL(job), nil
}

// EtlJobs retrieves all ETL jobs
func (r *queryResolver) EtlJobs(ctx context.Context) ([]*ETLJob, error) {
	if r.pipeline == nil {
		return nil, fmt.Errorf("ETL pipeline not initialized")
	}
	
	jobs := r.pipeline.GetAllJobs()
	result := make([]*ETLJob, len(jobs))
	
	for i, job := range jobs {
		result[i] = convertJobToGraphQL(job)
	}
	
	return result, nil
}

// StartNormalization starts a new normalization job
func (r *mutationResolver) StartNormalization(ctx context.Context, config NormalizationConfigInput) (*ETLJob, error) {
	if r.pipeline == nil {
		return nil, fmt.Errorf("ETL pipeline not initialized")
	}
	
	// Convert GraphQL input to ETL configuration
	etlConfig := etl.JobConfiguration{
		BatchSize:         1000, // default
		ConcurrentWorkers: 4,    // default
	}
	
	// Set optional fields
	if config.BatchSize != nil {
		etlConfig.BatchSize = *config.BatchSize
	}
	if config.ConcurrentWorkers != nil {
		etlConfig.ConcurrentWorkers = *config.ConcurrentWorkers
	}
	if config.ClearExisting != nil {
		etlConfig.ClearExisting = *config.ClearExisting
	}
	if config.DryRun != nil {
		etlConfig.DryRun = *config.DryRun
	}
	if len(config.Providers) > 0 {
		etlConfig.Providers = config.Providers
	}
	if len(config.Regions) > 0 {
		etlConfig.Regions = config.Regions
	}
	if len(config.Services) > 0 {
		etlConfig.Services = config.Services
	}
	
	// Convert job type
	jobType := convertGraphQLJobType(config.Type)
	
	// Start the job
	job, err := r.pipeline.StartJob(jobType, etlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start normalization job: %w", err)
	}
	
	return convertJobToGraphQL(job), nil
}

// CancelETLJob cancels a running ETL job
func (r *mutationResolver) CancelETLJob(ctx context.Context, id string) (bool, error) {
	if r.pipeline == nil {
		return false, fmt.Errorf("ETL pipeline not initialized")
	}
	
	err := r.pipeline.CancelJob(id)
	if err != nil {
		return false, fmt.Errorf("failed to cancel job: %w", err)
	}
	
	return true, nil
}

// Helper functions

// convertJobToGraphQL converts an ETL job to GraphQL format
func convertJobToGraphQL(job *etl.Job) *ETLJob {
	result := &ETLJob{
		ID:       job.ID,
		Type:     convertJobTypeToGraphQL(job.Type),
		Status:   convertJobStatusToGraphQL(job.Status),
		Provider: &job.Provider,
		StartedAt: job.StartedAt.Format(time.RFC3339),
		Configuration: &ETLJobConfiguration{
			BatchSize:         job.Configuration.BatchSize,
			ConcurrentWorkers: job.Configuration.ConcurrentWorkers,
			ClearExisting:     job.Configuration.ClearExisting,
			DryRun:            job.Configuration.DryRun,
		},
	}
	
	if len(job.Configuration.Providers) > 0 {
		result.Configuration.Providers = job.Configuration.Providers
	}
	if len(job.Configuration.Regions) > 0 {
		result.Configuration.Regions = job.Configuration.Regions
	}
	if len(job.Configuration.Services) > 0 {
		result.Configuration.Services = job.Configuration.Services
	}
	
	if job.CompletedAt != nil {
		completedAt := job.CompletedAt.Format(time.RFC3339)
		result.CompletedAt = &completedAt
	}
	
	if job.Error != "" {
		result.Error = &job.Error
	}
	
	if job.Progress != nil {
		result.Progress = &ETLJobProgress{
			TotalRecords:      job.Progress.TotalRecords,
			ProcessedRecords:  job.Progress.ProcessedRecords,
			NormalizedRecords: job.Progress.NormalizedRecords,
			SkippedRecords:    job.Progress.SkippedRecords,
			ErrorRecords:      job.Progress.ErrorRecords,
			CurrentStage:      job.Progress.CurrentStage,
			LastUpdated:       job.Progress.LastUpdated.Format(time.RFC3339),
			Rate:              job.Progress.Rate,
		}
	}
	
	return result
}

// convertJobTypeToGraphQL converts ETL job type to GraphQL enum
func convertJobTypeToGraphQL(jobType etl.JobType) ETLJobType {
	switch jobType {
	case etl.JobTypeNormalizeAll:
		return ETLJobTypeNormalizeAll
	case etl.JobTypeNormalizeProvider:
		return ETLJobTypeNormalizeProvider
	case etl.JobTypeNormalizeRegion:
		return ETLJobTypeNormalizeRegion
	case etl.JobTypeNormalizeService:
		return ETLJobTypeNormalizeService
	case etl.JobTypeCleanupNormalized:
		return ETLJobTypeCleanupNormalized
	default:
		return ETLJobTypeNormalizeAll
	}
}

// convertGraphQLJobType converts GraphQL job type to ETL job type
func convertGraphQLJobType(jobType ETLJobType) etl.JobType {
	switch jobType {
	case ETLJobTypeNormalizeAll:
		return etl.JobTypeNormalizeAll
	case ETLJobTypeNormalizeProvider:
		return etl.JobTypeNormalizeProvider
	case ETLJobTypeNormalizeRegion:
		return etl.JobTypeNormalizeRegion
	case ETLJobTypeNormalizeService:
		return etl.JobTypeNormalizeService
	case ETLJobTypeCleanupNormalized:
		return etl.JobTypeCleanupNormalized
	default:
		return etl.JobTypeNormalizeAll
	}
}

// convertJobStatusToGraphQL converts ETL job status to GraphQL enum
func convertJobStatusToGraphQL(status etl.JobStatus) ETLJobStatus {
	switch status {
	case etl.StatusPending:
		return ETLJobStatusPending
	case etl.StatusRunning:
		return ETLJobStatusRunning
	case etl.StatusCompleted:
		return ETLJobStatusCompleted
	case etl.StatusFailed:
		return ETLJobStatusFailed
	case etl.StatusCancelled:
		return ETLJobStatusCancelled
	default:
		return ETLJobStatusPending
	}
}

// GraphQL types for ETL functionality are now auto-generated in models_gen.go