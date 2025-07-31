package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/raulc0399/cpc/internal/database"
	"github.com/raulc0399/cpc/internal/etl"
)

func main() {
	// Connect to database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:password@localhost/cpc?sslmode=disable"
	}
	
	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()
	
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	db := database.New(sqlDB)

	// Create ETL pipeline
	pipeline, err := etl.NewPipeline(db)
	if err != nil {
		log.Fatalf("Failed to create ETL pipeline: %v", err)
	}

	// Test job configuration
	config := etl.JobConfiguration{
		Providers:         []string{"azure"},
		BatchSize:         100,
		ConcurrentWorkers: 2,
		DryRun:            true, // Don't actually insert data
	}

	fmt.Println("🚀 Starting ETL test job...")
	
	// Start the job
	job, err := pipeline.StartJob(etl.JobTypeNormalizeProvider, config)
	if err != nil {
		log.Fatalf("Failed to start job: %v", err)
	}

	fmt.Printf("✅ Job started with ID: %s\n", job.ID)
	fmt.Println("📊 Monitoring progress...")

	// Monitor progress
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentJob, exists := pipeline.GetJob(job.ID)
			if !exists {
				fmt.Println("❌ Job no longer exists")
				return
			}

			progress := currentJob.Progress
			fmt.Printf("🔄 [%s] %s - Processed: %d/%d, Normalized: %d, Rate: %.1f/sec\n",
				currentJob.Status,
				progress.CurrentStage,
				progress.ProcessedRecords,
				progress.TotalRecords,
				progress.NormalizedRecords,
				progress.Rate,
			)

			if currentJob.Status == etl.StatusCompleted {
				fmt.Printf("✅ Job completed successfully!\n")
				fmt.Printf("📈 Final stats: Processed=%d, Normalized=%d, Skipped=%d, Errors=%d\n",
					progress.ProcessedRecords,
					progress.NormalizedRecords,
					progress.SkippedRecords,
					progress.ErrorRecords,
				)
				return
			}

			if currentJob.Status == etl.StatusFailed {
				fmt.Printf("❌ Job failed: %s\n", currentJob.Error)
				return
			}

		case <-time.After(30 * time.Second):
			fmt.Println("⏰ Test timeout reached, cancelling job...")
			if err := pipeline.CancelJob(job.ID); err != nil {
				fmt.Printf("❌ Failed to cancel job: %v\n", err)
			}
			return
		}
	}
}