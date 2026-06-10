.PHONY: all build vet test fmt
all: build vet test

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

fmt:
	gofmt -l -w .
