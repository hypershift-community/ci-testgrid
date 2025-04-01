BINARY_NAME=bin/ci-scraper
CONTAINER_ENGINE ?= podman
IMAGE_NAME=quay.io/hypershift/ci-scraper:latest

build:
	go build -o ${BINARY_NAME} main.go

run: build
	./${BINARY_NAME}

test:
	go test ./...

clean:
	go clean
	rm -f ${BINARY_NAME}

image:
	${CONTAINER_ENGINE} build --arch=amd64 --os=linux -t ${IMAGE_NAME} .

