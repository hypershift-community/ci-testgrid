package testgrid

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Job represents the CI job metadata and test results
type Job struct {
	ID        string `json:"id" bson:"_id"`
	Name      string `json:"name" bson:"name"`
	Result    string `json:"result" bson:"result"`
	StartedAt string `json:"started_at" bson:"started_at"`
	LogURL    string `json:"log_url" bson:"log_url"`
	PR        int    `json:"pr" bson:"pr"`
	Tests     []Test `json:"tests" bson:"tests"`
	JobLink   string `json:"job_link" bson:"job_link"`
}

// Test represents individual test details
type Test struct {
	Name          string        `json:"name" bson:"name"`
	Result        string        `json:"result" bson:"result"`
	Duration      time.Duration `json:"duration" bson:"duration"`
	Logs          []string      `json:"logs" bson:"logs"`
	HostedCluster interface{}   `json:"hosted_cluster" bson:"hosted_cluster"`
	NodePools     interface{}   `json:"nodepools" bson:"nodepools"`
}

// TestGridViewModel represents the data for the test grid view
type TestGridViewModel struct {
	Jobs       []Job
	TestGroups []string
	FilterPR   int  // The PR number being filtered on, if any
	Filtered   bool // Whether we're currently filtering
}

// TestResultInfo contains additional test result information
type TestResultInfo struct {
	Result string
	Logs   []string
}

// TestGroupInfo contains information about a test group for sorting
type TestGroupInfo struct {
	Name             string
	LastFailureAt    time.Time
	HasRecentFailure bool
}

// JobDetailsViewModel represents the data for the job details view
type JobDetailsViewModel struct {
	Job         Job
	Summary     TestSummary
	FailedTests []Test
	ExpandTest  string // The name of the test to expand, if any
}

// TestSummary contains statistics about test results
type TestSummary struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
}

// Handler handles the testgrid HTTP requests
type Handler struct {
	templates *template.Template
}

// NewHandler creates a new testgrid handler
func NewHandler(templateFS embed.FS) (*Handler, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"getTestResultInfo": getTestResultInfo,
		"formatTime":        formatTime,
		"getJobStatusColor": getJobStatusColor,
	}).ParseFS(templateFS, "templates/testgrid.html", "templates/jobdetails.html")

	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	return &Handler{
		templates: tmpl,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if this is a job details request
	if jobID := r.URL.Query().Get("job"); jobID != "" {
		h.handleJobDetails(w, r, jobID)
		return
	}

	// Handle the main test grid view
	h.handleTestGrid(w, r)
}

// handleTestGrid handles the main test grid view
func (h *Handler) handleTestGrid(w http.ResponseWriter, r *http.Request) {
	// Parse PR filter from query parameters
	var filterPR int
	var filtered bool
	if prStr := r.URL.Query().Get("pr"); prStr != "" {
		if pr, err := fmt.Sscanf(prStr, "%d", &filterPR); err == nil && pr == 1 {
			filtered = true
		}
	}

	// Fetch jobs from MongoDB
	jobs, err := fetchJobsFromMongoDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching jobs: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter jobs by PR if specified
	if filtered {
		var filteredJobs []Job
		for _, job := range jobs {
			if job.PR == filterPR {
				filteredJobs = append(filteredJobs, job)
			}
		}
		jobs = filteredJobs
	}

	// Sort jobs by StartedAt timestamp
	sort.Slice(jobs, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, jobs[i].StartedAt)
		timeJ, _ := time.Parse(time.RFC3339, jobs[j].StartedAt)
		return timeI.After(timeJ)
	})

	// Prepare view model
	viewModel := TestGridViewModel{
		Jobs:       jobs,
		TestGroups: extractTestGroups(jobs),
		FilterPR:   filterPR,
		Filtered:   filtered,
	}

	// Execute template
	err = h.templates.ExecuteTemplate(w, "testgrid.html", viewModel)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleJobDetails handles the job details view
func (h *Handler) handleJobDetails(w http.ResponseWriter, r *http.Request, jobID string) {
	// Fetch job from MongoDB
	job, err := fetchJobFromMongoDB(jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching job: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate test summary
	summary := calculateTestSummary(job.Tests)

	// Get failed tests
	var failedTests []Test
	for _, test := range job.Tests {
		if strings.ToLower(test.Result) == "fail" {
			failedTests = append(failedTests, test)
		}
	}

	// Get the test to expand from query parameters
	expandTest := r.URL.Query().Get("test")

	// Prepare view model
	viewModel := JobDetailsViewModel{
		Job:         *job,
		Summary:     summary,
		FailedTests: failedTests,
		ExpandTest:  expandTest,
	}

	// Execute template
	err = h.templates.ExecuteTemplate(w, "jobdetails.html", viewModel)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, fmt.Sprintf("Error rendering template: %v", err), http.StatusInternalServerError)
		return
	}
}

// getMongoDBURI returns the MongoDB URI from environment or falls back to localhost
func getMongoDBURI() string {
	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		return uri
	}
	return "mongodb://localhost:27017"
}

// fetchJobFromMongoDB retrieves a single job from MongoDB
func fetchJobFromMongoDB(jobID string) (*Job, error) {
	// MongoDB connection configuration
	clientOptions := options.Client().ApplyURI(getMongoDBURI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	// Select database and collection
	collection := client.Database("ci").Collection("jobs")

	// Fetch job by ID
	var job Job
	err = collection.FindOne(context.TODO(), bson.M{"_id": jobID}).Decode(&job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

// calculateTestSummary calculates statistics about test results
func calculateTestSummary(tests []Test) TestSummary {
	summary := TestSummary{
		Total: len(tests),
	}

	for _, test := range tests {
		result := strings.ToLower(test.Result)
		switch result {
		case "pass":
			summary.Passed++
		case "fail":
			summary.Failed++
		case "skip":
			summary.Skipped++
		default:
			summary.Failed++ // Treat unknown results as failures
		}
	}

	return summary
}

// fetchJobsFromMongoDB retrieves jobs from MongoDB
func fetchJobsFromMongoDB() ([]Job, error) {
	// MongoDB connection configuration
	clientOptions := options.Client().ApplyURI(getMongoDBURI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	// Select database and collection
	collection := client.Database("ci").Collection("jobs")

	// Fetch jobs (last 7 days)
	cursor, err := collection.Find(context.TODO(), bson.M{
		"started_at": bson.M{
			"$gte": time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var jobs []Job
	if err = cursor.All(context.TODO(), &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// extractTestGroups gets unique test group names and sorts them by failure status
func extractTestGroups(jobs []Job) []string {
	testGroupMap := make(map[string]*TestGroupInfo)

	// First pass: collect all test groups and their failure information
	for _, job := range jobs {
		jobTime, _ := time.Parse(time.RFC3339, job.StartedAt)
		for _, test := range job.Tests {
			if info, exists := testGroupMap[test.Name]; exists {
				// Update failure information if this is a more recent failure
				if strings.ToLower(test.Result) == "fail" {
					if jobTime.After(info.LastFailureAt) {
						info.LastFailureAt = jobTime
						info.HasRecentFailure = true
					}
				}
			} else {
				// Create new test group info
				info := &TestGroupInfo{
					Name:             test.Name,
					LastFailureAt:    time.Time{}, // Zero time for no failures
					HasRecentFailure: false,
				}
				if strings.ToLower(test.Result) == "fail" {
					info.LastFailureAt = jobTime
					info.HasRecentFailure = true
				}
				testGroupMap[test.Name] = info
			}
		}
	}

	// Convert map to slice for sorting
	var testGroups []TestGroupInfo
	for _, info := range testGroupMap {
		testGroups = append(testGroups, *info)
	}

	// Sort test groups
	sort.Slice(testGroups, func(i, j int) bool {
		// First, sort by failure status
		if testGroups[i].HasRecentFailure != testGroups[j].HasRecentFailure {
			return testGroups[i].HasRecentFailure
		}

		// If both have failures, sort by most recent failure
		if testGroups[i].HasRecentFailure && testGroups[j].HasRecentFailure {
			return testGroups[i].LastFailureAt.After(testGroups[j].LastFailureAt)
		}

		// If neither has failures or both have the same failure time, sort alphabetically
		return testGroups[i].Name < testGroups[j].Name
	})

	// Extract just the names in the sorted order
	var result []string
	for _, info := range testGroups {
		result = append(result, info.Name)
	}

	return result
}

// formatTime formats the timestamp for display
func formatTime(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return t.Format("01-02 15:04")
}

// getTestResultInfo finds the result and logs for a specific test in a job
func getTestResultInfo(job Job, testName string) TestResultInfo {
	for _, test := range job.Tests {
		if test.Name == testName {
			// Normalize result to lowercase
			result := strings.ToLower(test.Result)
			if result == "" {
				return TestResultInfo{Result: "unknown", Logs: []string{}}
			}
			return TestResultInfo{
				Result: result,
				Logs:   test.Logs,
			}
		}
	}
	return TestResultInfo{Result: "unknown", Logs: []string{}}
}

// getJobStatusColor returns the CSS class for the job status
func getJobStatusColor(job Job) string {
	result := strings.ToLower(job.Result)
	switch result {
	case "success":
		return "job-success"
	case "failure":
		return "job-failure"
	default:
		return "job-unknown"
	}
}
