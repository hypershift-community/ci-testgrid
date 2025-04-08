package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/oauth2"
)

const (
	commentMarker = "<!-- Test Results Reporter -->"
	jobIDsMarker  = "<!-- Job IDs: "
)

type TestResult struct {
	ID        string
	TestName  string
	Result    string
	JobLink   string
	StartedAt string
	PR        int
	Tests     []Test
}

type Test struct {
	Name   string `bson:"name"`
	Result string `bson:"result"`
}

type PRResults struct {
	Results map[string]TestResult
	PR      int
}

func main() {
	// Get environment variables
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Check for dry run mode
	dryRun := os.Getenv("DRY_RUN") != ""
	if dryRun {
		log.Println("Running in dry run mode - no comments will be created or updated")
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Get the jobs collection
	collection := client.Database("ci").Collection("jobs")

	// Create GitHub client
	ctx = context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	// Get current user
	user, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		log.Fatal(err)
	}
	currentUser := *user.Login

	// Get all open PRs
	prs, _, err := githubClient.PullRequests.List(ctx, "openshift", "hypershift", &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a map to store results by PR number
	prResults := make(map[int]*PRResults)

	// Find the latest e2e-aws and e2e-aks test results for each PR
	testTypes := []string{"e2e-aws", "e2e-aks"}
	for _, pr := range prs {
		prNumber := *pr.Number
		prResults[prNumber] = &PRResults{
			Results: make(map[string]TestResult),
			PR:      prNumber,
		}

		for _, testType := range testTypes {
			var job struct {
				ID        string `bson:"_id"`
				TestName  string `bson:"test_name"`
				Result    string `bson:"result"`
				JobLink   string `bson:"job_link"`
				StartedAt string `bson:"started_at"`
				PR        int    `bson:"pr"`
				Tests     []Test `bson:"tests"`
			}

			opts := options.FindOne().SetSort(bson.D{{Key: "started_at", Value: -1}})
			err := collection.FindOne(ctx, bson.M{
				"test_name": testType,
				"pr":        prNumber,
			}, opts).Decode(&job)

			if err != nil {
				if err == mongo.ErrNoDocuments {
					log.Printf("No results found for PR %d, test %s", prNumber, testType)
					continue
				}
				log.Printf("Error finding results for PR %d, test %s: %v", prNumber, testType, err)
				continue
			}

			prResults[prNumber].Results[testType] = TestResult{
				ID:        job.ID,
				TestName:  job.TestName,
				Result:    job.Result,
				JobLink:   job.JobLink,
				StartedAt: job.StartedAt,
				PR:        job.PR,
				Tests:     job.Tests,
			}
		}
	}

	// Post comments for each PR that has results
	for prNumber, results := range prResults {
		if len(results.Results) == 0 {
			continue
		}

		// Create or update comment
		commentBody := formatComment(results.Results)
		comment := &github.IssueComment{
			Body: &commentBody,
		}

		// Try to find existing comment from current user with our marker
		comments, _, err := githubClient.Issues.ListComments(ctx, "openshift", "hypershift", prNumber, nil)
		if err != nil {
			log.Printf("Error listing comments for PR %d: %v", prNumber, err)
			continue
		}

		var existingCommentID int64
		var existingJobIDs string
		for _, c := range comments {
			if c.User.Login != nil && *c.User.Login == currentUser && c.Body != nil && strings.Contains(*c.Body, commentMarker) {
				existingCommentID = *c.ID
				// Extract job IDs from the comment
				if idx := strings.Index(*c.Body, jobIDsMarker); idx != -1 {
					if endIdx := strings.Index((*c.Body)[idx:], " -->"); endIdx != -1 {
						existingJobIDs = (*c.Body)[idx+len(jobIDsMarker) : idx+endIdx]
					}
				}
				break
			}
		}

		// Get current job IDs
		currentJobIDs := getJobIDs(results.Results)

		// Only update if job IDs have changed
		if existingCommentID != 0 && existingJobIDs == currentJobIDs {
			log.Printf("No new jobs to report for PR %d, skipping update", prNumber)
			continue
		}

		if existingCommentID != 0 {
			if dryRun {
				log.Printf("[DRY RUN] Would update existing comment %d on PR %d with new results", existingCommentID, prNumber)
				log.Printf("[DRY RUN] New comment body would be:\n%s", commentBody)
			} else {
				// Update existing comment
				_, _, err = githubClient.Issues.EditComment(ctx, "openshift", "hypershift", existingCommentID, comment)
				if err != nil {
					log.Printf("Error updating comment for PR %d: %v", prNumber, err)
					continue
				}
				log.Printf("Successfully updated comment %d for PR %d", existingCommentID, prNumber)
			}
		} else {
			if dryRun {
				log.Printf("[DRY RUN] Would create new comment on PR %d with results", prNumber)
				log.Printf("[DRY RUN] Comment body would be:\n%s", commentBody)
			} else {
				// Create new comment
				_, _, err = githubClient.Issues.CreateComment(ctx, "openshift", "hypershift", prNumber, comment)
				if err != nil {
					log.Printf("Error creating comment for PR %d: %v", prNumber, err)
					continue
				}
				log.Printf("Successfully created new comment for PR %d", prNumber)
			}
		}
	}
}

func getJobIDs(results map[string]TestResult) string {
	var jobIDs []string
	for _, result := range results {
		jobIDs = append(jobIDs, result.ID)
	}
	sort.Strings(jobIDs)
	return strings.Join(jobIDs, ",")
}

func formatComment(results map[string]TestResult) string {
	// Add job IDs at the start of the comment
	jobIDs := getJobIDs(results)
	comment := fmt.Sprintf("%s\n%s%s -->\n\n## Test Results\n\n", commentMarker, jobIDsMarker, jobIDs)

	for testType, result := range results {
		status := "✅ PASS"
		if result.Result != "SUCCESS" {
			status = "❌ FAIL"
		}

		singleJobURL := fmt.Sprintf("https://testgrid-ci-testgrid.apps.rosa.hypershift-ci-2.1xls.p3.openshiftapps.com/?job=%s&testName=%s", result.ID, testType)
		jobHistoryURL := fmt.Sprintf("https://testgrid-ci-testgrid.apps.rosa.hypershift-ci-2.1xls.p3.openshiftapps.com/?pr=%d&testName=%s", result.PR, testType)

		comment += fmt.Sprintf("### %s\n", testType)
		comment += fmt.Sprintf("- Status: %s\n", status)
		comment += fmt.Sprintf("- Started: %s\n", result.StartedAt)
		comment += fmt.Sprintf("- [View Job](%s)\n", singleJobURL)
		comment += fmt.Sprintf("- [View Job History](%s)\n", jobHistoryURL)

		// Add test failure details if there are any failed tests
		var failedTests []string
		for _, test := range result.Tests {
			if test.Result == "fail" {
				failedTests = append(failedTests, test.Name)
			}
		}

		if len(failedTests) > 0 {
			comment += "\n<details>\n<summary>Failed Tests</summary>\n\n"
			comment += fmt.Sprintf("Total failed tests: %d\n\n", len(failedTests))

			// Show first 5 failed tests
			numToShow := 5
			if len(failedTests) < numToShow {
				numToShow = len(failedTests)
			}

			for i := 0; i < numToShow; i++ {
				comment += fmt.Sprintf("- %s\n", failedTests[i])
			}

			if len(failedTests) > 5 {
				comment += fmt.Sprintf("\n... and %d more failed tests\n", len(failedTests)-5)
			}

			comment += "\n</details>\n"
		}

		comment += "\n"
	}

	return comment
}
