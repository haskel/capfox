.PHONY: build run test lint clean fmt vet tidy

APP_NAME := capfox
BUILD_DIR := bin
CMD_DIR := cmd/capfox

GO := go
GOFLAGS := -v

build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)

run: build
	./$(BUILD_DIR)/$(APP_NAME)

test:
	$(GO) test $(GOFLAGS) ./...

test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...
	gofmt -s -w .

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

deps:
	$(GO) mod download

all: fmt vet lint test build
