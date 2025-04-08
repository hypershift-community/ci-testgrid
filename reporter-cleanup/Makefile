BINARY_NAME=bin/reporter-cleanup

build:
	go build -o ${BINARY_NAME} main.go

run: build
	./${BINARY_NAME}

clean:
	go clean
	rm -f ${BINARY_NAME} 