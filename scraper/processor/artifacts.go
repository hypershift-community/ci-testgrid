package processor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hypershift-community/ci-testgrid/scraper/types"
)

// fetchArtifacts populates HostedCluster and NodePools fields on each test
// by fetching YAML artifacts from GCS. Errors are logged and skipped.
func fetchArtifacts(logURL string, tests []types.Test) {
	artifactBaseURL := strings.TrimSuffix(logURL, "build-log.txt") + "artifacts/"

	for i := range tests {
		hc, nps, err := fetchTestArtifacts(artifactBaseURL, tests[i].Name)
		if err != nil {
			log.Printf("Error fetching artifacts for test %s: %v", tests[i].Name, err)
			continue
		}
		tests[i].HostedCluster = hc
		tests[i].NodePools = nps
	}
}

// fetchTestArtifacts fetches the HostedCluster and NodePool YAMLs for a single test.
func fetchTestArtifacts(artifactBaseURL, testName string) (string, []string, error) {
	namespacesURL := artifactBaseURL + testName + "/namespaces/"
	namespaces, err := listGCSDirectory(namespacesURL)
	if err != nil {
		return "", nil, fmt.Errorf("listing namespaces: %w", err)
	}

	var hostedCluster string
	var nodePools []string

	for _, ns := range namespaces {
		// Each namespace entry should end with "/"
		if !strings.HasSuffix(ns, "/") {
			continue
		}

		hcDir := namespacesURL + ns + "hypershift.openshift.io/hostedclusters/"
		hcFiles, err := listGCSDirectory(hcDir)
		if err == nil {
			for _, f := range hcFiles {
				if !strings.HasSuffix(f, ".yaml") {
					continue
				}
				if hostedCluster == "" {
					content, err := fetchFileContent(hcDir + f)
					if err != nil {
						log.Printf("Error fetching hostedcluster file %s: %v", f, err)
						continue
					}
					hostedCluster = content
				}
			}
		}

		npDir := namespacesURL + ns + "hypershift.openshift.io/nodepools/"
		npFiles, err := listGCSDirectory(npDir)
		if err == nil {
			for _, f := range npFiles {
				if !strings.HasSuffix(f, ".yaml") {
					continue
				}
				content, err := fetchFileContent(npDir + f)
				if err != nil {
					log.Printf("Error fetching nodepool file %s: %v", f, err)
					continue
				}
				nodePools = append(nodePools, content)
			}
		}
	}

	return hostedCluster, nodePools, nil
}

// listGCSDirectory fetches a gcsweb HTML directory listing and returns entry names.
func listGCSDirectory(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching directory listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("directory listing returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing directory listing HTML: %w", err)
	}

	var entries []string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		// Only consider links pointing to GCS paths (filters out UI elements like gsutil/gcloud links)
		if !strings.HasPrefix(href, "/gcs/") {
			return
		}
		text := strings.TrimSpace(s.Text())
		// Skip parent directory link
		if text == "" || text == ".." || text == "../" {
			return
		}
		entries = append(entries, text)
	})

	return entries, nil
}

// fetchFileContent fetches a raw file from gcsweb and returns its content as a string.
func fetchFileContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading file body: %w", err)
	}

	return string(body), nil
}
