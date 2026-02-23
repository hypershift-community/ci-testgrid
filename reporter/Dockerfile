# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder

USER root
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7

# Install ca-certificates for HTTPS requests
RUN microdnf install -y ca-certificates && microdnf clean all

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/bin/ci-reporter /app/reporter

# Set the entrypoint
ENTRYPOINT ["/app/reporter"]
