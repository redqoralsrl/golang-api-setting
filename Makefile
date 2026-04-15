# Makefile
# include .env.local
.PHONY: help create-swagger test test-cover mocks build local-run dev-run logs lint sqlc tools-upgrade

help: ## This help dialog.
	@IFS=$$'\n' ; \
	help_lines=(`fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//'`); \
	for help_line in $${help_lines[@]}; do \
		IFS=$$'#' ; \
		help_split=($$help_line) ; \
		help_command=`echo $${help_split[0]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		help_info=`echo $${help_split[2]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		printf "%-30s %s\n" $$help_command $$help_info ; \
	done

create-swagger: ## Create swagger documentation
	go tool swag init -g ./internal/http/chi/api_handler.go -o ./docs/api --parseDependency --parseInternal --instanceName api --tags Users

test: ## Run tests
	go test -v ./...

test-cover: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

mocks: ## Generate mocks
	@read -p "Enter full package path(ex: template/domain/book): " package; \
	if ! grep -q "\"$$package\"" .mockery.yaml; then \
		echo "  \"$$package\":" >> .mockery.yaml; \
		echo "    config:" >> .mockery.yaml; \
		echo "      all: true" >> .mockery.yaml; \
		echo "Added $$package to .mockery.yaml"; \
	else \
		echo "Package $$package already exists in .mockery.yaml"; \
	fi; \
	go tool mockery

build: ## Build Prod Docker Image
	docker build -f Dockerfile.prod --platform linux/amd64 -t appname .     # Build the image

local-run: ## Run docker compose with local env file
	docker compose --env-file .env.local up -d && docker compose logs -f

dev-run: ## Run docker container with development env file
	docker run --env-file .env.development -p 8080:8080 -v .:/app -it appname

logs: ## Show logs
	docker compose logs -f

lint: ## Run linter
	golangci-lint run ./...

sqlc: ## Generate sqlc
	go tool sqlc generate

tools-upgrade: ## Go tool setting
	go get -tool github.com/air-verse/air@latest
	go get -tool github.com/swaggo/swag/cmd/swag@latest
	go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go mod tidy
