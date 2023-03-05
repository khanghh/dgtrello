.DEFAULT_GOAL := trellobot
BUILD_DIR=$(CURDIR)/build/bin

COMMIT=$(shell git rev-parse HEAD)
DATE=$(shell date)
TAG=$(shell git describe --tags)

LDFLAGS=-ldflags "-w -s -X 'main.gitCommit=$(COMMIT)' -X 'main.gitDate=$(DATE)' main.gitTag=$(TAG)'"

help:
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

trellobot: ## Build trellobot
	@echo "Building target: $@" 
	go build $(LDFLAGS) -o $(BUILD_DIR)/$@ $(CURDIR)/cmd/$@
	@echo "Done building."

clean: ## Clean build directory
	@rm -rf $(BUILD_DIR)/*

all: trellobot
