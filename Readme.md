# Go Template

`chi`, `postgres`, `redis`, `sqlc`, `swag` 기반의 Go API 템플릿입니다.  
현재 기본 예제로 유저 생성/로그아웃 API와 에러 로그 적재 흐름이 포함되어 있습니다.

## Requirements

- Go `1.26.x`
- Docker / Docker Compose

## Quick Start

가장 쉬운 방법은 Docker Compose로 전체 스택을 올리는 방식입니다.

### 1. 환경 파일 작성

프로젝트 루트에 `.env.local` 파일을 만들고 아래 값을 채웁니다.

```env
STAGE=dev

DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=go_template
DB_HOST=db
DB_PORT=5432

REDIS_HOST=cache_db
REDIS_PORT=6379

API_HOST=localhost
API_PORT=8080
API_X_API_KEY=local-api-key
API_SECRET_KEY=change-this-secret
CURSOR_SECRET=change-this-cursor-secret

IP_INFO_TOKEN=
SMTP_USERNAME=
SMTP_PASSWORD=

SWAGGER_ID=admin
SWAGGER_PASSWORD=admin
```

### 2. 전체 스택 실행

```bash
make local-run
```

이 명령은 아래를 실행합니다.

- `api` 컨테이너 실행
- `postgres` 실행
- `redis` 실행
- 실시간 로그 출력

기본 포트는 `8080`입니다.

## Local Development

앱은 로컬에서 직접 실행하고, DB/Redis만 Docker로 띄우고 싶다면 이렇게 하면 됩니다.

### 1. DB / Redis만 실행

```bash
docker compose --env-file .env.local up -d db cache_db
```

로컬에서 앱을 띄울 때는 `.env.local`의 DB/Redis host를 아래처럼 바꿔야 합니다.

```env
DB_HOST=localhost
REDIS_HOST=localhost
```

### 2. 앱 실행

```bash
go run ./cmd/api
```

핫리로드가 필요하면 `air`를 사용합니다.

```bash
go tool air -c ./cmd/api/.air.toml
```

## Swagger

개발 환경에서만 Swagger 라우트가 열립니다.

- URL: `http://localhost:8080/swagger/index.html`
- Basic Auth ID: `SWAGGER_ID`
- Basic Auth Password: `SWAGGER_PASSWORD`

문서 재생성:

```bash
make create-swagger
```

## Useful Commands

도구 설치/업데이트:

```bash
make tools-upgrade
```

테스트:

```bash
make test
```

커버리지:

```bash
make test-cover
```

`sqlc` 생성:

```bash
make sqlc
```

Swagger 생성:

```bash
make create-swagger
```

로그 보기:

```bash
make logs
```

## Project Structure

```text
cmd/api                  API entrypoint
config                   env 기반 설정 로딩
domain                   도메인 레이어
internal/http/chi        router, handler, middleware
internal/database        db wrapper, transaction, generated queries
ops/db/init.sql          초기 스키마 및 테이블 생성
docs/api                 swagger generated files
```

## Current API Examples

유저 생성:

```bash
curl -X POST http://localhost:8080/api/v1/user/create \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

로그아웃:

```bash
curl -X POST http://localhost:8080/api/v1/user/logout
```

헬스체크:

```bash
curl http://localhost:8080/health
```

## Notes

- DB 스키마는 컨테이너 최초 실행 시 [ops/db/init.sql](/Users/minki/Desktop/go-template/ops/db/init.sql:1)로 초기화됩니다.
- `API_HOST=localhost`이면 `JobsEnabled()`가 `false`가 되어 잡 등록을 건너뜁니다.
- 운영 이미지는 [Dockerfile.prod](/Users/minki/Desktop/go-template/Dockerfile.prod:1), 로컬 개발 컨테이너는 [Dockerfile](/Users/minki/Desktop/go-template/Dockerfile:1)을 사용합니다.
