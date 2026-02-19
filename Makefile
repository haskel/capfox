.PHONY: build run test lint clean fmt vet tidy profile-start profile-cpu profile-heap profile-all

APP_NAME := capfox
BUILD_DIR := bin
CMD_DIR := cmd/capfox
PROFILE_DIR := profiles

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

# Profiling targets
profile-start:
	docker compose -f docker-compose.profiling.yml up --build -d
	@echo "Profiling server started. pprof available at http://localhost:8080/debug/pprof/"

profile-stop:
	docker compose -f docker-compose.profiling.yml down

profile-cpu:
	@mkdir -p $(PROFILE_DIR)
	./scripts/profile.sh cpu 30

profile-heap:
	@mkdir -p $(PROFILE_DIR)
	./scripts/profile.sh heap

profile-all:
	@mkdir -p $(PROFILE_DIR)
	./scripts/profile.sh all

profile-web-cpu:
	./scripts/profile.sh web cpu

profile-web-heap:
	./scripts/profile.sh web heap

# Benchmarking targets
bench-ask:
	./scripts/benchmark.sh ask 1000 10

bench-train:
	./scripts/benchmark.sh train 200

bench-mixed:
	./scripts/benchmark.sh mixed 60

bench-report:
	./scripts/benchmark.sh report
