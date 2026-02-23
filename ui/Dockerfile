# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder

USER root
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ci-testgrid-ui main.go

# Final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7

# Install runtime dependencies
RUN microdnf install -y ca-certificates tzdata shadow-utils && microdnf clean all

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -m -s /bin/bash appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ci-testgrid-ui .

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./ci-testgrid-ui"]
