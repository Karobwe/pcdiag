# PC Diag - developer tasks. Run "make help" for the list.

APP      := pcdiag
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
FYNE     ?= fyne
# Packages that can be vetted for another OS without cgo.
CROSS_PKGS := ./internal/collector/ ./internal/config/ ./internal/format/ ./internal/health/ ./internal/report/
# Numeric version for the packaged metadata (v1.2.3-4-gabc -> 1.2.3).
APP_VERSION := $(shell echo $(VERSION) | sed -nE 's/^v([0-9]+\.[0-9]+\.[0-9]+).*/\1/p')

.PHONY: help run build build-windows test race cover lint fmt vet tidy screenshots package clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

run: ## Run the application
	go run .

build: ## Build a binary for the current platform
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP)$(shell go env GOEXE) .

build-windows: ## Build a Windows GUI binary (no console window); needs a Windows host or mingw-w64
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS) -H=windowsgui" -o $(APP).exe .

test: ## Run the unit tests
	go test ./...

race: ## Run the unit tests with the race detector
	go test -race ./...

cover: ## Run the tests with a coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format the code
	gofmt -s -w .

vet: ## Run go vet for every supported OS
	go vet ./...
	# The UI needs cgo, which is disabled when cross-compiling: vet the OS-specific packages only.
	GOOS=windows go vet $(CROSS_PKGS)
	GOOS=darwin go vet $(CROSS_PKGS)

tidy: ## Tidy go.mod / go.sum
	go mod tidy

screenshots: ## Regenerate the documentation screenshots
	PCDIAG_SCREENSHOTS=$(CURDIR)/docs/screenshots go test ./internal/ui -run TestRenderAllTabs -count=1
	LANG=fr_FR.UTF-8 PCDIAG_SCREENSHOTS=$(CURDIR)/docs/screenshots/fr go test ./internal/ui -run TestRenderAllTabs -count=1

package: ## Package for the current OS with the fyne CLI (go install fyne.io/tools/cmd/fyne@latest)
	GOFLAGS="-ldflags=-X=main.version=$(VERSION)" $(FYNE) package --release $(if $(APP_VERSION),--app-version $(APP_VERSION))

clean: ## Remove build outputs
	rm -rf $(APP) $(APP).exe *.syso "PC Diag"* dist coverage.out
