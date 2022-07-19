BUILD = $(CURDIR)/build
LINT_FILE = $(CURDIR)/lint.toml
PROJECT_NAME = gh_exporter
GOBIN=$(shell pwd)/bin

help: ## Show help dialog
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

.PHONY: install
install: ## Install dependencies
	go get -d github.com/mgechev/revive; \
	go get -d github.com/goreleaser/goreleaser

.PHONY: build
build: ## Build the project
	go build -o $(BUILD)/$(PROJECT_NAME) $(CURDIR)/cmd/main.go

.PHONY: clean
clean: ## Clean project
	go clean; \
	rm -rf $(BUILD);

.PHONY: run
run: ## Run project locally
	$(BUILD)/$(PROJECT_NAME) -dir ../

.PHONY: fmt
fmt: ## Format project
	go fmt $(CURDIR)/...

.PHONY: lint
lint: ## Lint project
	revive -config $(LINT_FILE) -formatter friendly $(CURDIR)/...

.PHONY: check
check: ## Format and lint project
check: fmt lint

.PHONY: tidy
tidy: ## Tidy up go modules
tidy:
	go mod tidy

.PHONY: release
release: ## Publish package to github
release:
	goreleaser release --rm-dist

.PHONY: setup
setup: ## Setup project
setup: clean install tidy