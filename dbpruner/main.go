package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hypershift-community/ci-testgrid/dbpruner/db"
	"github.com/hypershift-community/ci-testgrid/dbpruner/types"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CleanupConfig struct {
	RetentionDays int
	DryRun        bool
	BatchSize     int
	TestName      string
}

func parseStartedAt(startedAt string) (time.Time, error) {
	// Try multiple timestamp formats that might be used
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, startedAt); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", startedAt)
}

func findOldJobs(ctx context.Context, collection *mongo.Collection, cutoffDate time.Time, testName string, batchSize int) ([]types.Job, error) {
	filter := bson.M{}

	// Add test name filter if specified
	if testName != "" {
		filter["test_name"] = testName
	}

	// Find all jobs to check their started_at dates (since we need to parse them)
	cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(int64(batchSize)))
	if err != nil {
		return nil, fmt.Errorf("error finding jobs: %v", err)
	}
	defer cursor.Close(ctx)

	var oldJobs []types.Job
	var allJobs []types.Job

	if err := cursor.All(ctx, &allJobs); err != nil {
		return nil, fmt.Errorf("error decoding jobs: %v", err)
	}

	for _, job := range allJobs {
		if job.StartedAt == "" {
			log.Printf("Job %s has empty started_at field, skipping", job.ID)
			continue
		}

		startedAt, err := parseStartedAt(job.StartedAt)
		if err != nil {
			log.Printf("Job %s has invalid started_at format '%s': %v, skipping", job.ID, job.StartedAt, err)
			continue
		}

		if startedAt.Before(cutoffDate) {
			oldJobs = append(oldJobs, job)
		}
	}

	return oldJobs, nil
}

func deleteJobs(ctx context.Context, collection *mongo.Collection, jobIDs []string, dryRun bool) (int64, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}

	if dryRun {
		log.Printf("[DRY RUN] Would delete %d jobs with IDs: %v", len(jobIDs), jobIDs)
		return int64(len(jobIDs)), nil
	}

	filter := bson.M{"_id": bson.M{"$in": jobIDs}}
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error deleting jobs: %v", err)
	}

	return result.DeletedCount, nil
}

func runCleanup(config CleanupConfig) error {
	ctx := context.Background()

	// Connect to MongoDB using the existing connection logic
	client, err := db.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	collection := client.Database("ci").Collection("jobs")

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -config.RetentionDays)

	log.Printf("Starting cleanup process...")
	log.Printf("Retention period: %d days", config.RetentionDays)
	log.Printf("Cutoff date: %s", cutoffDate.Format(time.RFC3339))
	log.Printf("Test name filter: %s", config.TestName)
	log.Printf("Dry run mode: %v", config.DryRun)
	log.Printf("Batch size: %d", config.BatchSize)

	totalDeleted := int64(0)
	batchCount := 0

	for {
		batchCount++
		log.Printf("Processing batch %d...", batchCount)

		// Find old jobs in batches
		oldJobs, err := findOldJobs(ctx, collection, cutoffDate, config.TestName, config.BatchSize)
		if err != nil {
			return fmt.Errorf("error finding old jobs: %v", err)
		}

		if len(oldJobs) == 0 {
			log.Printf("No more old jobs found. Cleanup complete.")
			break
		}

		log.Printf("Found %d jobs older than %s in batch %d", len(oldJobs), cutoffDate.Format("2006-01-02"), batchCount)

		// Extract job IDs for deletion
		jobIDs := make([]string, len(oldJobs))
		for i, job := range oldJobs {
			jobIDs[i] = job.ID
			log.Printf("  - Job %s (%s) started at %s", job.ID, job.TestName, job.StartedAt)
		}

		// Delete the jobs
		deletedCount, err := deleteJobs(ctx, collection, jobIDs, config.DryRun)
		if err != nil {
			return fmt.Errorf("error deleting jobs: %v", err)
		}

		totalDeleted += deletedCount

		if config.DryRun {
			log.Printf("[DRY RUN] Batch %d: Would delete %d jobs", batchCount, deletedCount)
		} else {
			log.Printf("Batch %d: Successfully deleted %d jobs", batchCount, deletedCount)
		}

		// If we got fewer jobs than the batch size, we're done
		if len(oldJobs) < config.BatchSize {
			break
		}
	}

	if config.DryRun {
		log.Printf("Cleanup complete! [DRY RUN] Would have deleted %d total jobs", totalDeleted)
	} else {
		log.Printf("Cleanup complete! Successfully deleted %d total jobs", totalDeleted)
	}

	return nil
}

func createRootCommand() *cobra.Command {
	var config CleanupConfig

	cmd := &cobra.Command{
		Use:   "dbpruner",
		Short: "Clean up old test results from the CI TestGrid database",
		Long: `A tool that removes old test results from the MongoDB database used by the CI TestGrid scraper.
This helps manage database size and performance by removing outdated test results.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCleanup(config)
		},
	}

	cmd.Flags().IntVar(&config.RetentionDays, "retention-days", 30, "Number of days to retain test results (default: 30)")
	cmd.Flags().BoolVar(&config.DryRun, "dry-run", false, "Run in dry-run mode without actually deleting anything")
	cmd.Flags().IntVar(&config.BatchSize, "batch-size", 1000, "Number of jobs to process in each batch (default: 1000)")
	cmd.Flags().StringVar(&config.TestName, "test-name", "", "Only clean up jobs for a specific test name (e.g., 'e2e-aws', 'e2e-aks')")

	return cmd
}

func init() {
	// Check for environment variable overrides
	if dryRun := os.Getenv("DRY_RUN"); dryRun != "" {
		log.Println("DRY_RUN environment variable detected - enabling dry run mode")
	}

	if retentionDays := os.Getenv("RETENTION_DAYS"); retentionDays != "" {
		if _, err := strconv.Atoi(retentionDays); err == nil {
			log.Printf("RETENTION_DAYS environment variable detected - using %s days", retentionDays)
		}
	}
}

func main() {
	rootCmd := createRootCommand()

	// Override dry-run flag if environment variable is set
	if os.Getenv("DRY_RUN") != "" {
		rootCmd.Flag("dry-run").Value.Set("true")
	}

	// Override retention-days flag if environment variable is set
	if retentionDays := os.Getenv("RETENTION_DAYS"); retentionDays != "" {
		if _, err := strconv.Atoi(retentionDays); err == nil {
			rootCmd.Flag("retention-days").Value.Set(retentionDays)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
