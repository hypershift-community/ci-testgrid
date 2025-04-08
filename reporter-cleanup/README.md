# Test Results Reporter Cleanup

This program removes comments added by the Test Results Reporter program from all open GitHub PRs.

## Prerequisites

- Go 1.21 or later
- GitHub Personal Access Token with repo scope

## Environment Variables

The following environment variables are required:

- `GITHUB_TOKEN`: GitHub Personal Access Token with repo scope
- `DRY_RUN` (optional): If set to any value, the program will run in dry-run mode and only show what comments would be deleted without actually deleting them

## Building

```bash
go mod download
go build
```

## Running

```bash
export GITHUB_TOKEN="your-github-token"
./reporter-cleanup
```

To run in dry-run mode (no comments will be deleted):
```bash
export GITHUB_TOKEN="your-github-token"
export DRY_RUN=1
./reporter-cleanup
```

## Output

The program will:
1. Fetch all open PRs from the repository
2. For each PR, find comments that contain the Test Results Reporter marker
3. Delete those comments (unless in dry-run mode)

The program will log its progress and any errors encountered while processing PRs. 