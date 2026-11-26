# 저장소 루트에서 실행한다. 로컬 실행 타깃은 ENV_FILE(기본 .env.local)을 읽는다.
ENV_FILE ?= .env.local
IMAGE    ?= go-template
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# .env 파일을 환경변수로 올린 뒤 명령을 실행한다
WITH_ENV = set -a && . ./$(ENV_FILE) && set +a &&
# compose 변수(${DB_USER} 등)도 같은 파일에서 읽는다
COMPOSE  = docker compose --env-file $(ENV_FILE)

.DEFAULT_GOAL := help
.PHONY: help dev local-run db-up down logs \
	migrate-up migrate-down migrate-status migrate-create \
	sqlc create-swagger test lint build tools-upgrade rename

help: ## 사용 가능한 명령 목록
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## ---------- 실행 ----------
dev: ## 로컬에서 hot reload 실행 (DB 는 make db-up)
	$(WITH_ENV) go tool air -c cmd/api/.air.toml

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

## ---------- 검사·빌드 ----------
test: ## 테스트 (race)
	go test -race ./...

lint: ## golangci-lint (bin/, 없으면 make tools-upgrade)
	./bin/golangci-lint run ./...

build: ## 운영 Docker 이미지 (linux/amd64)
	docker build -f Dockerfile.prod --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

tools-upgrade: ## air, swag, sqlc, golangci-lint 설치/업그레이드
	go get -tool github.com/air-verse/air@latest
	go get -tool github.com/swaggo/swag/cmd/swag@latest
	go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go mod tidy

## ---------- 템플릿 ----------
# 템플릿을 새 프로젝트로 가져온 뒤 이름을 바꾼다. 실제 작업은 scripts/rename.sh 가 한다.
#   git clone <템플릿> order-api && cd order-api && make rename name=order-api
#
# - name: 소문자, 숫자, '-' 만 (docker 이미지·compose 이름 규칙). 전체 경로(github.com/...)는 쓰지 않는다.
# - 현재 이름은 go.mod 의 module 에서 읽으므로 여러 번 실행해도 된다.
# - 바꾸는 곳: go.mod module, *.go/*.md import 경로, 인증 쿠키(<name>_access_token, '-' → '_'),
#   docker-compose.yaml 프로젝트 이름, 이 파일의 IMAGE. 끝나면 go build + swagger(docs/api) 재생성.
# - 실행한 폴더의 이름이 바로 바뀌므로 템플릿 원본에서는 돌리지 않는다.
# - compose 프로젝트 이름이 바뀌면 docker 볼륨도 새로 생긴다 (기존 DB 데이터는 옛 볼륨에 남음).
# - 처음 실행할 때 swag 모듈을 내려받느라 인터넷이 필요할 수 있다.
rename: ## 프로젝트 이름 변경 (go.mod, import, 쿠키, compose, 이미지). ex) make rename name=order-api
	@test -n "$(name)" || (echo "usage: make rename name=<project-name>" && exit 1)
	bash scripts/rename.sh $(name)
