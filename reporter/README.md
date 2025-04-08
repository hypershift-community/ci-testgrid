# Test Results Reporter

This program queries MongoDB for e2e-aws and e2e-aks test results and creates/updates comments on all open GitHub PRs that have test results available.

## Prerequisites

- Go 1.21 or later
- MongoDB instance
- GitHub Personal Access Token with repo scope

## Environment Variables

The following environment variables are required:

- `GITHUB_TOKEN`: GitHub Personal Access Token with repo scope
- `MONGO_URI` (optional): MongoDB connection URI (defaults to "mongodb://localhost:27017")

## Building

```bash
go mod download
go build
```

## Running

```bash
export GITHUB_TOKEN="your-github-token"
export MONGO_URI="mongodb://localhost:27017"  # optional
./reporter
```

## Output

The program will:
1. Fetch all open PRs from the repository
2. For each PR, query MongoDB for the latest e2e-aws and e2e-aks test results
3. Create or update a comment on each PR that has test results, including:
   - Test status (PASS/FAIL)
   - Start time
   - Link to the job

The program will log its progress and any errors encountered while processing PRs. 