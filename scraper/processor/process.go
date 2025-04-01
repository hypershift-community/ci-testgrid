package processor

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hypershift-community/ci-testgrid/scraper/types"
)

func ProcessJob(job *types.Job) ([]types.Test, error) {
	log.Printf("Fetching test log from %s\n", job.LogURL)
	resp, err := http.Get(job.LogURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return parseLog(resp.Body)
}

// TestEntry holds a test’s name, its result (pass, fail or skip),
// and any associated log lines.
type TestEntry struct {
	Name     string        `json:"name"`
	Result   string        `json:"result"`
	Duration time.Duration `json:"duration"`
	Logs     []string      `json:"logs"`
}

type Tests struct {
	Items []TestEntry `json:"items"`
}

// parseLog processes the test log and returns a slice of TestEntry.
func parseLog(in io.Reader) ([]types.Test, error) {
	scanner := bufio.NewScanner(in)
	// testsMap keeps track of tests by name.
	testsMap := make(map[string]*types.Test)
	var currentTest *types.Test

	// Regex patterns to detect test start and result lines.
	testStartRe := regexp.MustCompile(`^\s*(===)\s+(RUN|CONT|NAME)\s+(\S+)`)
	resultRe := regexp.MustCompile(`^\s*(---)\s+(PASS|FAIL|SKIP):\s+(\S+) \(([^\)]+)\)`)
	resultRe2 := regexp.MustCompile(`^\s*(===)\s+(PASS|FAIL|SKIP): \. (\S+) \(([^\)]+)\)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip lines starting with '+' or '{'
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "{") {
			continue
		}
		// Skip lines that have no indent and start with "FAIL"
		if !strings.HasPrefix(line, " ") && strings.HasPrefix(line, "FAIL") {
			continue
		}

		// Check if the line indicates a test start.
		if matches := testStartRe.FindStringSubmatch(line); matches != nil {
			currentTest = nil
			continue
		}

		if matches := resultRe.FindStringSubmatch(line); matches != nil {
			testName := matches[3]
			durationStr := matches[4]
			duration, _ := time.ParseDuration(durationStr)
			resultType := strings.ToLower(matches[2])
			if t, exists := testsMap[testName]; exists {
				t.Result = resultType
				if duration != 0 {
					t.Duration = duration
				}
			} else {
				currentTest = &types.Test{
					Name:     testName,
					Result:   resultType,
					Duration: duration,
				}
				testsMap[testName] = currentTest
			}
			currentTest = nil
			continue
		}

		// Check if the line is a result line.
		if matches := resultRe2.FindStringSubmatch(line); matches != nil {
			testName := matches[3]
			durationStr := matches[4]
			duration, err := time.ParseDuration(durationStr)
			if err != nil {
				duration = 0
			}
			resultType := strings.ToLower(matches[2]) // "pass", "fail", or "skip"
			// Get (or create) the test entry for this test.
			if t, exists := testsMap[testName]; exists {
				t.Result = resultType
				if duration != 0 {
					t.Duration = duration
				}
				currentTest = t
			} else {
				currentTest = &types.Test{
					Name:     testName,
					Result:   resultType,
					Duration: duration,
				}
				testsMap[testName] = currentTest
			}
			continue
		}

		// If the current test is active and the line is indented...
		if currentTest != nil && strings.HasPrefix(line, " ") {
			trimmed := strings.TrimSpace(line)
			// Only add the line if it does NOT start with "---" or "===".
			if !strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "===") {
				currentTest.Logs = append(currentTest.Logs, line)
			}
		}
	}

	// Convert the map to a slice.
	var tests []types.Test
	for _, test := range testsMap {
		tests = append(tests, *test)
	}
	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Name < tests[j].Name
	})
	return tests, nil
}
