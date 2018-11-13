.PHONY: default test lint check build
default: check

module := $(shell go list -m)
go_bin := $(shell go env GOPATH)/bin

test:
	go test -race -timeout 1m $(module)/...

lint: $(go_bin)/golangci-lint
	golangci-lint run

$(go_bin)/golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

check: test lint

install:
	go install $(module)

coverage: $(go_bin)/goveralls
	goveralls -service=travis-ci

$(go_bin)/goveralls:
	go install github.com/mattn/goveralls
