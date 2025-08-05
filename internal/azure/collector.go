package azure

import (
	"context"
	"fmt"
	"log"
)

// Collector orchestrates Azure pricing data collection
type Collector struct {
	regionHandler   RegionHandler
	dataStore       DataStore
	outputHandler   OutputHandler
	progressTracker ProgressTracker
	client          *Client
}

// NewCollector creates a new Azure collector with injected dependencies
func NewCollector(regionHandler RegionHandler, dataStore DataStore, 
                 outputHandler OutputHandler, tracker ProgressTracker) *Collector {
	client := NewClient()
	regionHandler.SetClient(client)
	
	return &Collector{
		regionHandler:   regionHandler,
		dataStore:       dataStore,
		outputHandler:   outputHandler,
		progressTracker: tracker,
		client:          client,
	}
}

// Run executes the data collection process
func (c *Collector) Run(ctx context.Context) error {
	regions := c.regionHandler.GetRegions()
	
	log.Printf("Starting Azure data collection for %d regions", len(regions))
	c.progressTracker.Start(len(regions))
	
	for i, region := range regions {
		log.Printf("Collecting region %d/%d: %s", i+1, len(regions), region)
		
		// Start collection tracking
		collectionID, err := c.dataStore.StartCollection(region)
		if err != nil {
			c.progressTracker.Complete(region, false, 0)
			return fmt.Errorf("failed to start collection for region %s: %w", region, err)
		}
		
		c.progressTracker.Update(region, 0, "collecting")
		
		// Collect data for region
		data, err := c.regionHandler.Collect(ctx, region)
		if err != nil {
			c.progressTracker.Complete(region, false, 0)
			if failErr := c.dataStore.FailCollection(collectionID, err.Error()); failErr != nil {
				log.Printf("Failed to mark collection as failed: %v", failErr)
			}
			return fmt.Errorf("failed to collect data for region %s: %w", region, err)
		}
		
		// Store data
		if err := c.dataStore.Store(ctx, collectionID, region, data); err != nil {
			c.progressTracker.Complete(region, false, len(data))
			if failErr := c.dataStore.FailCollection(collectionID, err.Error()); failErr != nil {
				log.Printf("Failed to mark collection as failed: %v", failErr)
			}
			return fmt.Errorf("failed to store data for region %s: %w", region, err)
		}
		
		// Complete collection tracking
		if err := c.dataStore.CompleteCollection(collectionID, len(data)); err != nil {
			log.Printf("Warning: Failed to mark collection complete: %v", err)
		}
		
		// Output results
		items := make([]PricingItem, len(data))
		for i, raw := range data {
			items[i] = convertRawToPricingItem(raw)
		}
		
		if err := c.outputHandler.Write(items); err != nil {
			log.Printf("Warning: Failed to write output: %v", err)
		}
		
		c.progressTracker.Complete(region, true, len(data))
		log.Printf("✅ Completed region %s: %d items", region, len(data))
	}
	
	// Final statistics
	completed, failed, total, elapsed := c.progressTracker.GetStatus()
	log.Printf("🎉 Collection completed! %d/%d regions successful (%d failed) in %v", 
		completed, total, failed, elapsed)
	
	return nil
}

// RunConcurrent executes concurrent data collection for multiple regions
func (c *Collector) RunConcurrent(ctx context.Context, concurrency int) error {
	regions := c.regionHandler.GetRegions()
	
	log.Printf("Starting concurrent Azure data collection: %d regions, %d workers", 
		len(regions), concurrency)
	c.progressTracker.Start(len(regions))
	
	// Create worker pool
	regionChan := make(chan string, len(regions))
	errorChan := make(chan error, len(regions))
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		go c.worker(ctx, i, regionChan, errorChan)
	}
	
	// Send regions to workers
	for _, region := range regions {
		regionChan <- region
	}
	close(regionChan)
	
	// Collect results
	var errors []error
	for i := 0; i < len(regions); i++ {
		if err := <-errorChan; err != nil {
			errors = append(errors, err)
			log.Printf("Worker error: %v", err)
		}
	}
	
	// Final statistics
	completed, failed, total, elapsed := c.progressTracker.GetStatus()
	log.Printf("🎉 Concurrent collection completed! %d/%d regions successful (%d failed) in %v", 
		completed, total, failed, elapsed)
	
	if len(errors) > 0 {
		return fmt.Errorf("collection completed with %d errors", len(errors))
	}
	
	return nil
}

func (c *Collector) worker(ctx context.Context, workerID int, regionChan <-chan string, errorChan chan<- error) {
	for region := range regionChan {
		c.progressTracker.SetWorking(workerID, region)
		
		// Start collection tracking
		collectionID, err := c.dataStore.StartCollection(region)
		if err != nil {
			c.progressTracker.Complete(region, false, 0)
			c.progressTracker.ClearWorking(workerID)
			errorChan <- fmt.Errorf("worker %d failed to start collection for region %s: %w", workerID, region, err)
			continue
		}
		
		c.progressTracker.Update(region, 0, "collecting")
		
		// Collect data for region
		data, err := c.regionHandler.Collect(ctx, region)
		if err != nil {
			c.progressTracker.Complete(region, false, 0)
			c.progressTracker.ClearWorking(workerID)
			if failErr := c.dataStore.FailCollection(collectionID, err.Error()); failErr != nil {
				log.Printf("Failed to mark collection as failed: %v", failErr)
			}
			errorChan <- fmt.Errorf("worker %d failed to collect data for region %s: %w", workerID, region, err)
			continue
		}
		
		// Store data
		if err := c.dataStore.Store(ctx, collectionID, region, data); err != nil {
			c.progressTracker.Complete(region, false, len(data))
			c.progressTracker.ClearWorking(workerID)
			if failErr := c.dataStore.FailCollection(collectionID, err.Error()); failErr != nil {
				log.Printf("Failed to mark collection as failed: %v", failErr)
			}
			errorChan <- fmt.Errorf("worker %d failed to store data for region %s: %w", workerID, region, err)
			continue
		}
		
		// Complete collection tracking
		if err := c.dataStore.CompleteCollection(collectionID, len(data)); err != nil {
			log.Printf("Warning: Failed to mark collection complete: %v", err)
		}
		
		c.progressTracker.Complete(region, true, len(data))
		c.progressTracker.ClearWorking(workerID)
		log.Printf("✅ Worker %d completed region %s: %d items", workerID, region, len(data))
		
		errorChan <- nil // Success
	}
}