# Build stage - using public UBI image (no auth required)
FROM registry.access.redhat.com/ubi8/go-toolset:1.21 AS builder

# Switch to root to install packages and set up build environment
USER root

# Install git (needed for go modules)
RUN dnf update -y && dnf install -y git && dnf clean all

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code and local modules
COPY main.go ./
COPY db/ ./db/
COPY types/ ./types/

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dbpruner main.go

# Final stage - minimal runtime image (public access)
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# Add Red Hat container labels
LABEL name="ci-testgrid-dbpruner" \
      vendor="Hypershift Community" \
      version="1.0" \
      release="1" \
      summary="Database pruner for CI TestGrid MongoDB cleanup" \
      description="A tool that removes old test results from the MongoDB database used by the CI TestGrid scraper" \
      io.k8s.description="Database pruner for CI TestGrid MongoDB cleanup" \
      io.k8s.display-name="CI TestGrid DB Pruner" \
      io.openshift.expose-services="8080:http" \
      io.openshift.tags="database,mongodb,cleanup,ci,testgrid"

# Install ca-certificates, timezone data, and user management tools
RUN microdnf update -y && \
    microdnf install -y ca-certificates tzdata shadow-utils && \
    microdnf clean all

# Create app directory and non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -m -s /bin/bash appuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/dbpruner .

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set default environment variables
ENV RETENTION_DAYS=30
ENV BATCH_SIZE=1000

# Expose health check port (if needed)
EXPOSE 8080

# Health check command (using Red Hat compatible approach)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ps aux | grep -v grep | grep dbpruner || exit 1

# Default command
ENTRYPOINT ["./dbpruner"]

# Default arguments (can be overridden)
CMD ["--retention-days=30"] 