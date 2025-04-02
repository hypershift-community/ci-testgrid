package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hypershift-community/ci-testgrid/scraper/types"
)

type Pull struct {
	Number     int    `json:"number"`
	Author     string `json:"author"`
	SHA        string `json:"sha"`
	Title      string `json:"title"`
	HeadRef    string `json:"head_ref"`
	Link       string `json:"link"`
	CommitLink string `json:"commit_link"`
	AuthorLink string `json:"author_link"`
}

type Refs struct {
	Org      string `json:"org"`
	Repo     string `json:"repo"`
	RepoLink string `json:"repo_link"`
	BaseRef  string `json:"base_ref"`
	BaseSHA  string `json:"base_sha"`
	BaseLink string `json:"base_link"`
	Pulls    []Pull `json:"pulls"`
}

type Build struct {
	SpyglassLink string `json:"SpyglassLink"`
	ID           string `json:"ID"`
	Started      string `json:"Started"`
	Duration     int64  `json:"Duration"`
	Result       string `json:"Result"`
	Refs         Refs   `json:"Refs"`
}

func ScrapeJobs(url, suffix string) ([]types.Job, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, "", err
	}

	var jsonData string
	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		content := s.Text()
		re := regexp.MustCompile(`var\s+allBuilds\s*=\s*(\[.*\]);`)
		match := re.FindStringSubmatch(content)
		if len(match) == 2 {
			jsonData = match[1]
			return false
		}
		return true
	})

	if jsonData == "" {
		return nil, "", nil
	}

	var builds []Build
	if err := json.Unmarshal([]byte(jsonData), &builds); err != nil {
		return nil, "", err
	}

	var jobs []types.Job
	for _, build := range builds {
		if build.Result != "SUCCESS" && build.Result != "FAILURE" {
			continue
		}
		logURL, err := spyglassToLogURL(build.SpyglassLink, suffix)
		if err != nil {
			logURL = ""
		}
		var pr int
		for _, pull := range build.Refs.Pulls {
			pr = pull.Number
			break
		}
		jobs = append(jobs, types.Job{
			ID:        build.ID,
			Name:      build.Refs.Repo,
			Result:    build.Result,
			StartedAt: build.Started,
			LogURL:    logURL,
			PR:        pr,
			JobLink:   build.SpyglassLink,
		})
	}

	return jobs, getOlderRunsURL(doc), nil
}

func spyglassToLogURL(spyglassLink, suffix string) (string, error) {
	const (
		prefix       = "/view/gs/"
		gcswebPrefix = "https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/"
	)

	if !strings.HasPrefix(spyglassLink, prefix) {
		return "", fmt.Errorf("invalid SpyglassLink: missing %q prefix", prefix)
	}

	// Remove the "/view/gs/" prefix
	gcsPath := strings.TrimPrefix(spyglassLink, prefix)

	// Construct the final log URL
	logURL := gcswebPrefix + gcsPath + suffix
	return logURL, nil
}

func getOlderRunsURL(doc *goquery.Document) string {
	var olderRunsURL string
	// Find the link with text "<- Older Runs"
	doc.Find("a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.Text() == "<- Older Runs" {
			href, exists := s.Attr("href")
			if exists {
				decodedRef, err := url.QueryUnescape(href)
				if err == nil {
					olderRunsURL = decodedRef
				}
			}
			return false // break loop
		}
		return true // keep looking
	})

	return "https://prow.ci.openshift.org" + olderRunsURL
}
