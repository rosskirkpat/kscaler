
.DEFAULT_GOAL := help

SHELL 										:= $(shell which bash) -c

VERSION 									?= $(shell git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD)
COMMIT_HASH									= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE 									= $(shell date +%FT%T%z)


GO_BIN										?= $(GOROOT)/bin

build: generate
	go build -o kscaler github.com/rosskirkpat/kscaler/cmd

generate: ## generate stubs
	go build -o $(GO_BIN)/cmd-gen internal/cmd-gen/main.go
	go build -o $(GO_BIN)/stub-gen internal/stub-gen/main.go
	go generate github.com/rosskirkpat/kscaler/pkg/kubernetes
	go generate github.com/rosskirkpat/kscaler/cmd

test: build
	KUBECONFIG=$(HOME)/.kube/config ./kscaler namespace -n 5

##############
## Helpers

.PHONY: help
help: ### Show this help
	@egrep -h '\s###\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?### "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
