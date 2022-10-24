MAKEFILE_PATH := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PATH := $(MAKEFILE_PATH):$(PATH)

export GOBIN := $(MAKEFILE_PATH)/bin

PATH := $(GOBIN):$(PATH)

.PHONY: all
all: clean format build lint test

.PHONY: clean
clean:
	@echo clean
	@go clean

.PHONY: build
build:
	@echo build
	@go build -o $(GOBIN)/final-surge-bot

.PHONY: test
test:
	@echo test
	@go test -count=1 -race -v ./...

.PHONY: lint
lint: ensure-golangci-lint
	@echo lint
	@$(GOBIN)/golangci-lint run

.PHONY: format
format:
	@echo format
	@go fmt $(PKGS)

.PHONY: generate
generate: ensure-mockgen
	@echo generate
	@go generate ./...

.PHONY: ensure-golangci-lint
ensure-golangci-lint:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1

.PHONY: ensure-mockgen
ensure-mockgen:
	@go install github.com/golang/mock/mockgen@v1.3.1
