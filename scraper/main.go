package main

import (
	"context"
	"log"
	"time"

	"github.com/hypershift-community/ci-testgrid/scraper/db"
	"github.com/hypershift-community/ci-testgrid/scraper/processor"
	"github.com/hypershift-community/ci-testgrid/scraper/scraper"
)

func main() {
	startURL := "https://prow.ci.openshift.org/job-history/gs/test-platform-results/pr-logs/directory/pull-ci-openshift-hypershift-main-e2e-aws"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to MongoDB.
	client, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatalf("Error disconnecting MongoDB: %v", err)
		}
	}()
	collection := client.Database("ci").Collection("jobs")

	jobCount := 0
	pageURL := startURL

	for jobCount < 100 {
		log.Printf("Scraping jobs page: %s\n", pageURL)
		jobs, nextPage, err := scraper.ScrapeJobs(pageURL)
		if err != nil {
			log.Printf("Error scraping page: %v", err)
			break
		}
		if len(jobs) == 0 {
			log.Println("No jobs found on page.")
			break
		}

		for _, job := range jobs {
			// Stop scraping if the job is already in MongoDB.
			exists, err := db.JobExists(ctx, collection, job.ID)
			if err != nil {
				log.Printf("Error checking job %s: %v\n", job.ID, err)
				continue
			}
			if exists {
				log.Printf("Job %s already stored. Stopping scraping.\n", job.ID)
				return
			}

			// Process the job: fetch log, extract and parse JUnit XML.
			tests, err := processor.ProcessJob(&job)
			if err != nil {
				log.Printf("Error processing job %s: %v", job.ID, err)
				continue
			}
			job.Tests = tests

			// Store the job in MongoDB.
			err = db.InsertJob(ctx, collection, &job)
			if err != nil {
				log.Printf("Error storing job %s: %v", job.ID, err)
				continue
			}
			log.Printf("Stored job %s successfully.\n", job.ID)
			jobCount++
			if jobCount >= 100 {
				break
			}
		}

		if nextPage == "" {
			log.Println("No more pages to scrape.")
			break
		}
		pageURL = nextPage
	}

	log.Printf("Scraping complete. %d jobs processed.\n", jobCount)
}
