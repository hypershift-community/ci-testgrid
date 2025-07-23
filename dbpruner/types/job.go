package types

import "time"

// Job represents the CI job metadata and test results.
// This is a simplified version for database cleanup operations.
type Job struct {
	ID        string `json:"id" bson:"_id"`
	Name      string `json:"name" bson:"name"`
	Result    string `json:"result" bson:"result"`
	StartedAt string `json:"started_at" bson:"started_at"`
	LogURL    string `json:"log_url" bson:"log_url"`
	PR        int    `json:"pr" bson:"pr"`
	Tests     []Test `json:"tests" bson:"tests"`
	JobLink   string `json:"job_link" bson:"job_link"`
	TestName  string `json:"test_name" bson:"test_name"`
}

// Test represents individual test results within a job.
// Included for completeness but mainly used for data structure compatibility.
type Test struct {
	Name          string        `json:"name" bson:"name"`
	Result        string        `json:"result" bson:"result"`
	Duration      time.Duration `json:"duration" bson:"duration"`
	Logs          []string      `json:"logs" bson:"logs"`
	HostedCluster any           `json:"hosted_cluster" bson:"hosted_cluster"`
	NodePools     any           `json:"nodepools" bson:"nodepools"`
}
