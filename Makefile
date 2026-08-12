GO ?= go
GOFMT ?= gofmt
GOCACHE ?= $(CURDIR)/.gocache
export GOCACHE

BINARY := ytextract
ifeq ($(OS),Windows_NT)
BINARY := $(BINARY).exe
SHELL := cmd.exe
.SHELLFLAGS := /C
endif

.DEFAULT_GOAL := help

.PHONY: help build run test integration race vet fmt fmt-check tidy tidy-check check clean

help: ## Show the available targets.
	@echo "VoxScripta development targets:"
	@echo "  make build       Build the ytextract CLI"
	@echo "  make run         Run the CLI (pass options with ARGS='...')"
	@echo "  make test        Run all unit tests"
	@echo "  make integration Run opt-in live yt-dlp integration tests"
	@echo "  make race        Run all unit tests with the race detector"
	@echo "  make vet         Run go vet"
	@echo "  make fmt         Format all Go source files"
	@echo "  make fmt-check   Check that Go source files are formatted"
	@echo "  make tidy        Update Go module metadata"
	@echo "  make tidy-check  Check that Go module metadata is tidy"
	@echo "  make check       Run formatting, module, test, vet, and build checks"
	@echo "  make clean       Remove the built CLI"

build: ## Build the ytextract CLI.
	$(GO) build -o $(BINARY) ./cmd/ytextract

run: ## Run the ytextract CLI; for example, make run ARGS="--version".
	$(GO) run ./cmd/ytextract $(ARGS)

test: ## Run all unit tests.
	$(GO) test ./...

integration: ## Run live yt-dlp integration tests; requires yt-dlp and network access.
ifeq ($(OS),Windows_NT)
	set VOXSCRIPTA_YTDLP_INTEGRATION=1&& $(GO) test -run TestYTDLPIntegration -v .
else
	VOXSCRIPTA_YTDLP_INTEGRATION=1 $(GO) test -run TestYTDLPIntegration -v .
endif

race: ## Run all unit tests with the race detector.
	$(GO) test -race ./...

vet: ## Analyze all packages with go vet.
	$(GO) vet ./...

fmt: ## Format all Go source files.
	$(GOFMT) -w .

fmt-check: ## Fail when any Go source file needs formatting.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$files = & '$(GOFMT)' -l .; if ($$files) { Write-Host 'Unformatted Go files:'; $$files | ForEach-Object { Write-Host $$_ }; exit 1 }"
else
	@files=`$(GOFMT) -l .`; test -z "$$files" || { echo "Unformatted Go files:"; echo "$$files"; exit 1; }
endif

tidy: ## Update Go module metadata.
	$(GO) mod tidy

tidy-check: ## Fail when go mod tidy would change tracked module metadata.
	$(GO) mod tidy -diff

check: fmt-check tidy-check test vet build ## Run the standard development checks.

clean: ## Remove build output.
ifeq ($(OS),Windows_NT)
	@if exist $(BINARY) del /Q $(BINARY)
else
	$(RM) $(BINARY)
endif
