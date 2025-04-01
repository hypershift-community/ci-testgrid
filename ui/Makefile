# Binary name
BINARY_NAME=ci-testgrid-ui
CONTAINER_ENGINE ?= podman
IMAGE_NAME=quay.io/hypershift/ci-testgrid-ui:latest

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Build directory
BUILD_DIR=bin

# Main target
all: build

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Build the application
build: $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) main.go

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build directory
clean:
	rm -rf $(BUILD_DIR)

# Install dependencies
deps:
	$(GOGET) ./...

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

image:
	${CONTAINER_ENGINE} build --arch=amd64 --os=linux -t ${IMAGE_NAME} .

.PHONY: all build test clean deps run image