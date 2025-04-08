package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

const (
	commentMarker = "<!-- Test Results Reporter -->"
)

func main() {
	// Get GitHub token from environment
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Check for dry run mode
	dryRun := os.Getenv("DRY_RUN") != ""
	if dryRun {
		log.Println("Running in dry run mode - no comments will be deleted")
	}

	// Create GitHub client
	ctx := context.Background()
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

	// Process each PR
	for _, pr := range prs {
		prNumber := *pr.Number
		log.Printf("Processing PR #%d", prNumber)

		// Get all comments for the PR
		comments, _, err := githubClient.Issues.ListComments(ctx, "openshift", "hypershift", prNumber, nil)
		if err != nil {
			log.Printf("Error listing comments for PR %d: %v", prNumber, err)
			continue
		}

		// Find and delete comments from the reporter
		for _, comment := range comments {
			if comment.User.Login != nil && *comment.User.Login == currentUser && comment.Body != nil && strings.Contains(*comment.Body, commentMarker) {
				if dryRun {
					log.Printf("[DRY RUN] Would delete comment %d from PR %d", *comment.ID, prNumber)
				} else {
					_, err := githubClient.Issues.DeleteComment(ctx, "openshift", "hypershift", *comment.ID)
					if err != nil {
						log.Printf("Error deleting comment %d from PR %d: %v", *comment.ID, prNumber, err)
						continue
					}
					log.Printf("Successfully deleted comment %d from PR %d", *comment.ID, prNumber)
				}
			}
		}
	}
}
