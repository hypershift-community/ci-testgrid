BINARY_NAME=bin/ci-reporter
CONTAINER_ENGINE ?= podman
IMAGE_TAG ?= $(shell date +%Y-%m-%d)
IMAGE_NAME=quay.io/hypershift/ci-reporter:$(IMAGE_TAG)
AUTHFILE ?= $(HOME)/.hypershift-push

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

push: image
	${CONTAINER_ENGINE} push --authfile ${AUTHFILE} ${IMAGE_NAME}