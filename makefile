# Common variables
VERSION ?= 0.0.1
BUILD_INFO := Manual build on $(shell hostname) at $(shell date)
SRC_DIR := ./cmd

# Most likely want to override these when calling `make image`
IMAGE_REG ?= ghcr.io
IMAGE_REPO ?= benc-uk/go-rest-api
IMAGE_TAG ?= latest
IMAGE_NAME := $(IMAGE_REG)/$(IMAGE_REPO)

# Things you don't want to change
REPO_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
# Tools
GOLINT_PATH := $(REPO_DIR)/bin/golangci-lint
AIR_PATH := $(REPO_DIR)/bin/air

.EXPORT_ALL_VARIABLES:
.PHONY: help image push build run lint lint-fix clean
.DEFAULT_GOAL := help

help: ## 💬 This help message :)
	@figlet $@ || true
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## 🔮 Install dev tools into project bin directory
	@figlet $@ || true
	@$(GOLINT_PATH) > /dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin/
	@$(AIR_PATH) -v > /dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh

lint: install-tools ## 🔍 Lint & format check only, sets exit code on error for CI
	@figlet $@ || true
	$(GOLINT_PATH) run --modules-download-mode=mod

lint-fix: install-tools ## 📝 Lint & format, attempts to fix errors & modify code
	@figlet $@ || true
	$(GOLINT_PATH) run --modules-download-mode=mod --fix

image: ## 📦 Build container image
	@figlet $@ || true
	docker build . -f build/Dockerfile \
	--build-arg VERSION=$(VERSION) \
	--build-arg BUILD_INFO='$(BUILD_INFO)' \
	--tag $(IMAGE_NAME):$(IMAGE_TAG)

push: ## 📤 Push container images to registry
	@figlet $@ || true
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

build: ## 🔨 Run a local build without a container
	@figlet $@ || true
	go build -o bin/server \
	  -ldflags "-X main.version=$(VERSION) -X 'main.buildInfo=$(BUILD_INFO)'" \
	  github.com/benc-uk/go-rest-api/cmd

run: install-tools ## 🏃 Run server with hot reload
	@figlet $@ || true
	$(AIR_PATH)

test: ## 🧪 Run tests
	@figlet $@ || true
	go test -v ./...
	
clean: ## 🧹 Clean up the repo
	@figlet $@ || true
	rm -rf bin
	rm -rf tmp
