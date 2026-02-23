//go:build integration

package processor

import (
	"os"
	"strings"
	"testing"
)

// These tests require the TEST_LOG_URL environment variable to be set to a
// gcsweb build-log.txt URL, e.g.:
//
//	TEST_LOG_URL="https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/.../build-log.txt" \
//	  go test -tags=integration ./processor/ -v

func testArtifactBaseURL(t *testing.T) string {
	t.Helper()
	logURL := os.Getenv("TEST_LOG_URL")
	if logURL == "" {
		t.Fatal("TEST_LOG_URL environment variable is required")
	}
	if !strings.HasSuffix(logURL, "build-log.txt") {
		t.Fatalf("TEST_LOG_URL must end with build-log.txt, got: %s", logURL)
	}
	return strings.TrimSuffix(logURL, "build-log.txt") + "artifacts/"
}

func TestFetchTestArtifacts(t *testing.T) {
	artifactBaseURL := testArtifactBaseURL(t)
	testName := "TestCreateCluster"

	hostedCluster, nodePools, err := fetchTestArtifacts(artifactBaseURL, testName)
	if err != nil {
		t.Fatalf("fetchTestArtifacts returned error: %v", err)
	}

	if hostedCluster == "" {
		t.Fatal("expected non-empty hostedCluster YAML")
	}
	if !strings.Contains(hostedCluster, "kind: HostedCluster") {
		t.Errorf("hostedCluster YAML missing 'kind: HostedCluster', got:\n%s", hostedCluster)
	}

	if len(nodePools) == 0 {
		t.Fatal("expected at least one nodePool YAML")
	}
	for i, np := range nodePools {
		if !strings.Contains(np, "kind: NodePool") {
			t.Errorf("nodePool[%d] YAML missing 'kind: NodePool', got:\n%s", i, np)
		}
	}

	t.Logf("HostedCluster YAML length: %d bytes", len(hostedCluster))
	t.Logf("Found %d NodePool(s)", len(nodePools))
}

func TestListGCSDirectory(t *testing.T) {
	artifactBaseURL := testArtifactBaseURL(t)
	url := artifactBaseURL + "TestCreateCluster/namespaces/"

	entries, err := listGCSDirectory(url)
	if err != nil {
		t.Fatalf("listGCSDirectory returned error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one namespace directory entry")
	}

	t.Logf("Found %d entries: %v", len(entries), entries)
}
