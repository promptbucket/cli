.PHONY: build test vet fmt lint

build:
go build ./...

test:
go test -race ./...

vet:
go vet ./...

fmt:
gofmt -w $(shell find . -name '*.go')

lint:
golangci-lint run
