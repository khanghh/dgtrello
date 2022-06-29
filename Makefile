VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse HEAD)
DATE=$(shell date)

LDFLAGS=-ldflags "-w -s -X 'main.Version=$(VERSION)' -X 'main.CommitHash=$(COMMIT)' -X 'main.BuiltTime=$(DATE)'"

trellobot:
	@echo "Building target: $@" 
	go build $(LDFLAGS) -o bin/$@ cmd/*.go

clean:
	@rm bin/*

.PHONY: clean

all: trellobot
