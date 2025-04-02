package main

import (
	"context"
	"log"
	"time"

	"github.com/hypershift-community/ci-testgrid/scraper/db"
	"github.com/hypershift-community/ci-testgrid/scraper/processor"
	"github.com/hypershift-community/ci-testgrid/scraper/scraper"
	"github.com/spf13/cobra"
)

type Scraper struct {
	startURL string
	suffix   string
	testName string
}

func NewScraper(startURL, suffix, testName string) *Scraper {
	return &Scraper{
		startURL: startURL,
		suffix:   suffix,
		testName: testName,
	}
}

func (s *Scraper) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to MongoDB.
	client, err := db.Connect()
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()
	collection := client.Database("ci").Collection("jobs")

	jobCount := 0
	pageURL := s.startURL

	for jobCount < 100 {
		log.Printf("Scraping jobs page for %s: %s\n", s.testName, pageURL)
		jobs, nextPage, err := scraper.ScrapeJobs(pageURL, s.suffix)
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
				return nil
			}

			// Process the job: fetch log, extract and parse JUnit XML.
			tests, err := processor.ProcessJob(&job)
			if err != nil {
				log.Printf("Error processing job %s: %v", job.ID, err)
				continue
			}
			job.Tests = tests
			job.TestName = s.testName

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

	log.Printf("Scraping complete for %s. %d jobs processed.\n", s.testName, jobCount)
	return nil
}

func createRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci-scraper",
		Short: "CI TestGrid scraper for OpenShift CI jobs",
		Long:  `A tool that scrapes test results from OpenShift CI jobs and stores them in MongoDB.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create e2e-aws scraper
			awsScraper := NewScraper(
				"https://prow.ci.openshift.org/job-history/gs/test-platform-results/pr-logs/directory/pull-ci-openshift-hypershift-main-e2e-aws",
				"/artifacts/e2e-aws/hypershift-aws-run-e2e-external/build-log.txt",
				"e2e-aws",
			)

			// Create e2e-aks scraper
			aksScraper := NewScraper(
				"https://prow.ci.openshift.org/job-history/gs/test-platform-results/pr-logs/directory/pull-ci-openshift-hypershift-main-e2e-aks",
				"/artifacts/e2e-aks/hypershift-azure-run-e2e/build-log.txt",
				"e2e-aks",
			)

			// Run e2e-aws scraper
			if err := awsScraper.Run(); err != nil {
				log.Printf("Error running e2e-aws scraper: %v", err)
			}

			// Run e2e-aks scraper
			if err := aksScraper.Run(); err != nil {
				log.Printf("Error running e2e-aks scraper: %v", err)
			}

			return nil
		},
	}

	return cmd
}

func main() {
	rootCmd := createRootCommand()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
