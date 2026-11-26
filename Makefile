# 저장소 루트에서 실행한다. 로컬 실행 타깃은 ENV_FILE(기본 .env.local)을 읽는다.
ENV_FILE ?= .env.local
IMAGE    ?= go-template
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# .env 파일을 환경변수로 올린 뒤 명령을 실행한다
WITH_ENV = set -a && . ./$(ENV_FILE) && set +a &&
# compose 변수(${DB_USER} 등)도 같은 파일에서 읽는다
COMPOSE  = docker compose --env-file $(ENV_FILE)

.DEFAULT_GOAL := help
.PHONY: help dev run local-run db-up down logs \
	migrate-up migrate-down migrate-status migrate-create \
	sqlc create-swagger generate domain \
	test test-cover lint vuln check build tools-upgrade

help: ## 사용 가능한 명령 목록
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## ---------- 실행 ----------
dev: ## 로컬에서 hot reload 실행 (DB 는 make db-up)
	$(WITH_ENV) go tool air -c cmd/api/.air.toml

run: ## 로컬에서 한 번 실행
	$(WITH_ENV) go run ./cmd/api

local-run: ## docker compose 로 API + DB 실행 후 로그 보기
	$(COMPOSE) up -d --build && $(COMPOSE) logs -f api

db-up: ## DB 컨테이너만 실행
	$(COMPOSE) up -d db

down: ## compose 컨테이너 중지 (데이터 볼륨은 유지. 지우려면 docker compose down -v)
	$(COMPOSE) down

logs: ## compose 로그
	$(COMPOSE) logs -f

## ---------- DB 마이그레이션 ----------
migrate-up: ## 아직 적용 안 된 마이그레이션 모두 적용
	$(WITH_ENV) go run ./cmd/migrate up

migrate-down: ## 마지막 마이그레이션 1개 되돌리기
	$(WITH_ENV) go run ./cmd/migrate down

migrate-status: ## 마이그레이션 적용 현황
	$(WITH_ENV) go run ./cmd/migrate status

migrate-create: ## 새 마이그레이션 파일. ex) make migrate-create name=add_orders
	@test -n "$(name)" || (echo "usage: make migrate-create name=<snake_case>" && exit 1)
	go run ./cmd/migrate create $(name)

## ---------- 코드 생성 ----------
sqlc: ## migrations/ 와 */query.sql 로 DB 코드 생성
	go tool sqlc generate

create-swagger: ## Swagger 문서 생성 (docs/api)
	go tool swag init -g ./internal/http/chi/api_handler.go -o ./docs/api --parseDependency --parseInternal --instanceName api

generate: sqlc create-swagger ## sqlc + swagger

domain: ## 테이블로 도메인 뼈대 생성. ex) make domain table=v1.notices name=notice
	go run ./generator -table=$(table) -name=$(name)
	gofmt -w ./domain/$(name)

## ---------- 검사 ----------
test: ## 테스트 (race)
	go test -race ./...

test-cover: ## 커버리지 리포트 (coverage.html)
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## golangci-lint (bin/, 없으면 make tools-upgrade)
	./bin/golangci-lint run ./...

vuln: ## 알려진 취약점 검사
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check: lint test ## CI 와 같은 검사
	go tool sqlc diff

## ---------- 빌드 ----------
build: ## 운영 Docker 이미지 (linux/amd64)
	docker build -f Dockerfile.prod --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

tools-upgrade: ## air, swag, sqlc, golangci-lint 설치/업그레이드
	go get -tool github.com/air-verse/air@latest
	go get -tool github.com/swaggo/swag/cmd/swag@latest
	go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go mod tidy
