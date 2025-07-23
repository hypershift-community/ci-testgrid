# CI TestGrid Database Pruner Tool

An independent utility program to clean up old test results from the MongoDB database used by the CI TestGrid scraper. This tool is completely self-contained and helps manage database size and performance by removing outdated test results based on configurable retention policies.

## Features

- **Safe by Default**: Runs in dry-run mode by default to prevent accidental data loss
- **Configurable Retention**: Set custom retention periods (default: 30 days)
- **Batch Processing**: Processes jobs in configurable batches for memory efficiency
- **Test-Specific Cleanup**: Option to clean up only specific test types (e.g., `e2e-aws`, `e2e-aks`)
- **Flexible Timestamp Parsing**: Handles multiple timestamp formats automatically
- **Detailed Logging**: Comprehensive logging of cleanup operations
- **Environment Variable Support**: Can be configured via environment variables

## Quick Start

### 1. Install Dependencies
```bash
make deps
```

### 2. Test Run (Safe)
```bash
# Run in dry-run mode to see what would be deleted
go run main.go --dry-run
```

### 3. Actual Cleanup (Destructive)
```bash
# WARNING: This will permanently delete old test results!
make run
```

## Usage Options

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--retention-days` | 30 | Number of days to retain test results |
| `--dry-run` | false | Run without actually deleting anything |
| `--batch-size` | 1000 | Number of jobs to process in each batch |
| `--test-name` | "" | Only clean up jobs for specific test name |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DRY_RUN` | Set to any value to enable dry-run mode |
| `RETENTION_DAYS` | Override default retention period |
| `MONGODB_HOST` | MongoDB host (default: localhost) |
| `MONGODB_USER` | MongoDB username (optional) |
| `MONGODB_PASSWORD` | MongoDB password (optional) |

## Examples

### Basic Usage

```bash
# Safe test run - see what would be deleted
./bin/dbpruner --dry-run

# Delete jobs older than 30 days (default)
./bin/dbpruner

# Delete jobs older than 14 days
./bin/dbpruner --retention-days 14

# Clean up only e2e-aws test jobs
./bin/dbpruner --test-name e2e-aws --dry-run
```

### Using Makefile

```bash
# Dry run with default settings
go run main.go --dry-run

# Dry run with custom retention period
go run main.go --dry-run --retention-days=14

# Actual cleanup (be careful!)
make run

# Clean up specific test type
go run main.go --dry-run --test-name=e2e-aws
```

### Using Environment Variables

```bash
# Force dry-run mode via environment
DRY_RUN=1 ./bin/dbpruner

# Set retention period via environment
RETENTION_DAYS=14 ./bin/dbpruner --dry-run

# Connect to remote MongoDB
MONGODB_HOST=mongo.example.com ./bin/dbpruner --dry-run
```

## Database Connection

The dbpruner tool includes its own MongoDB connection logic (independent of the scraper). It connects to:

- **Database**: `ci`
- **Collection**: `jobs`
- **Default Host**: `localhost:27017`

### Authentication

If your MongoDB requires authentication, set these environment variables:

```bash
export MONGODB_USER=username
export MONGODB_PASSWORD=password
export MONGODB_HOST=your-mongo-host
```

## How It Works

1. **Connection**: Connects to MongoDB using the same connection logic as the scraper
2. **Date Calculation**: Calculates cutoff date based on retention period
3. **Batch Processing**: Processes jobs in configurable batches to manage memory usage
4. **Timestamp Parsing**: Attempts to parse job timestamps using multiple common formats
5. **Filtering**: Optionally filters by test name if specified
6. **Deletion**: Removes jobs older than the cutoff date (or logs what would be removed in dry-run mode)

## Safety Features

- **Dry-run by Default**: Many commands default to dry-run mode
- **Input Validation**: Validates timestamp formats before processing
- **Batch Processing**: Prevents memory issues with large datasets
- **Detailed Logging**: Shows exactly what will be or was deleted
- **Environment Override**: `DRY_RUN` environment variable provides extra safety

## Build and Install

### Native Binary

```bash
# Build the binary
make build

# Install to GOPATH/bin
make install

# Clean build artifacts
make clean
```

### Container Image (Red Hat UBI)

The dbpruner uses Red Hat Universal Base Images (UBI) for enhanced enterprise security and support:

```bash
# Build container image using Red Hat UBI8 (no authentication required)
make image

# Run with podman/docker
podman run --rm -it quay.io/hypershift/ci-testgrid-dbpruner:latest --help

# Run with custom environment variables
podman run --rm -e DRY_RUN=1 -e RETENTION_DAYS=7 \
  quay.io/hypershift/ci-testgrid-dbpruner:latest --test-name=e2e-aws
```

## Troubleshooting

### Common Issues

1. **Connection Failed**: Ensure MongoDB is running and accessible
2. **Authentication Error**: Check `MONGODB_USER` and `MONGODB_PASSWORD` environment variables
3. **No Jobs Found**: Verify database name and collection name are correct
4. **Timestamp Parse Errors**: Check the format of `started_at` fields in your jobs

### Debug Mode

For verbose logging, you can modify the code to set log level to DEBUG, or run with:

```bash
# Enable verbose output (if implemented)
./bin/dbpruner --dry-run --verbose
```

## Contributing

When modifying this tool:

1. Always test with `--dry-run` first
2. Add appropriate logging for new features
3. Update this README with new options
4. Consider backward compatibility with existing data

## Warning

⚠️ **DANGER**: This tool permanently deletes data from your database. Always:

1. Test with `--dry-run` first
2. Backup your database before running cleanup
3. Verify the retention period is correct
4. Check that you're connected to the right database

## Container Registry

The container image is built using Red Hat Universal Base Images (UBI8) and can be pushed to:

- **Red Hat Quay.io**: `quay.io/hypershift/ci-testgrid-dbpruner:latest`
- **OpenShift Internal Registry**: `image-registry.openshift-image-registry.svc:5000/ci-testgrid/dbpruner:latest`
- **Docker Hub**: `hypershift/ci-testgrid-dbpruner:latest`

## License

This tool is part of the CI TestGrid project and follows the same license terms. The base Red Hat UBI images are provided under the Red Hat Universal Base Image End User License Agreement. 