.DEFAULT_GOAL := help
PROJECT_BIN = $(shell pwd)/bin
$(shell [ -f bin ] || mkdir -p $(PROJECT_BIN))
PATH := $(PROJECT_BIN):$(PATH)
GOOS = linux
GOARCH = amd64
CGO_ENABLED = 0
VERS = $(shell git describe --tags --abbrev=0)
# LDFLAGS = -w -s -X main.vers=$(VERS)
LDFLAGS = -w -s
GCFLAGS = "-trimpath"
ASMFLAGS = "-trimpath"
APP := $(notdir $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST))))))

.PHONY: build \
		run \
		all \
		.install-linter \
		lint \
		.install-nil \
		nil-check \
		upx \
		help

all: build upx
	
upx: ## Pack upx
	upx $(PROJECT_BIN)/$(APP)

build:  ## Build
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="$(LDFLAGS)" -gcflags=$(GCFLAGS) -asmflags=$(ASMFLAGS) -o $(PROJECT_BIN)/$(APP) cmd/linux/main.go
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=$(GOARCH) go build -ldflags="$(LDFLAGS)" -gcflags=$(GCFLAGS) -asmflags=$(ASMFLAGS) -o $(PROJECT_BIN)/$(APP).exe cmd/windows/main.go

lint: .install-linter ## Run linter
	golangci-lint run ./...

nil-check: .install-nil ## Run nil check linter
	nilaway ./...

.install-linter: ## Install linter
	[ -f $(PROJECT_BIN)/golangci-lint ] || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(PROJECT_BIN) v1.54.2

.install-nil: ## Install nil check
	[ -f $(PROJECT_BIN)/nilaway ] || go install go.uber.org/nilaway/cmd/nilaway@latest && cp $(GOPATH)/bin/nilaway $(PROJECT_BIN)

help:
	@cat $(MAKEFILE_LIST) | grep -E '^[a-zA-Z_-]+:.*?## .*$$' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
