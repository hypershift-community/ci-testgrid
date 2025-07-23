# Database Pruner Tool for CI TestGrid
# Removes old test results from MongoDB database

# Container engine (podman preferred for Red Hat ecosystem)
CONTAINER_ENGINE ?= podman
IMAGE_NAME ?= quay.io/hypershift/ci-testgrid-dbpruner:latest

.PHONY: build clean run help deps image

# Build the dbpruner binary
build:
	go build -o bin/dbpruner main.go

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run actual cleanup (30 days retention) - BE CAREFUL!
run:
	go run main.go

# Container image targets
image:
	$(CONTAINER_ENGINE) build --arch=amd64 --os=linux -t $(IMAGE_NAME) .



# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build the dbpruner binary"
	@echo "  clean           - Remove build artifacts"
	@echo "  deps            - Install Go dependencies"
	@echo "  run             - Run actual cleanup (DANGEROUS - removes data!)"
	@echo "  image           - Build Red Hat UBI container image"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  DRY_RUN=1       - Enable dry-run mode"
	@echo "  RETENTION_DAYS  - Number of days to retain (default: 30)"
	@echo "  MONGODB_HOST    - MongoDB host (default: localhost)"
	@echo "  MONGODB_USER    - MongoDB username (optional)"
	@echo "  MONGODB_PASSWORD - MongoDB password (optional)"
	@echo ""
	@echo "Examples:"
	@echo "  make build                      # Build the binary"
	@echo "  make image                      # Build Red Hat UBI-based container"
	@echo "  DRY_RUN=1 make run              # Force dry-run via env var"
	@echo "  go run main.go --dry-run        # Safe test run"
	@echo "  go run main.go --retention-days=14 --dry-run  # Custom retention" 