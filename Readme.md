# Go Template

Go HTTP API 템플릿. Chi v5, PostgreSQL(pgx/v5 + sqlc + goose), JWT 쿠키 인증, 구조화 로그.

## Requirements

- Go 1.26+
- Docker (로컬 DB)
- make, bash (Windows 는 Git Bash 또는 WSL)

## Quick Start

```bash
# 0. 프로젝트 이름 변경 (go.mod, import, 인증 쿠키, compose, 이미지 이름)
make rename name=order-api

# 1. 환경 파일
cp .env.template .env.local
# API_SECRET_KEY, CURSOR_SECRET 를 채운다 (비어 있으면 서버가 시작되지 않는다)
#   API_SECRET_KEY=$(openssl rand -hex 32)
#   CURSOR_SECRET=$(openssl rand -hex 16)

# 2-a. 전부 docker 로
make local-run

# 2-b. DB 만 docker, API 는 로컬 hot reload
make db-up
make dev
```

시작할 때 `migrations/` 가 자동으로 적용된다 (`MIGRATE_ON_START=true`).

```bash
curl localhost:8080/health   # {"code":200,"message":"success","data":{"status":"ok"}}
curl localhost:8080/ready    # DB 연결까지 확인
```

## API

| Method | Path | 인증 | 설명 |
|---|---|---|---|
| POST | `/api/v1/user/create` | - | 가입 + 토큰 쿠키 발급 |
| POST | `/api/v1/user/login` | - | 로그인 + 토큰 쿠키 발급 |
| GET | `/api/v1/app/user/me` | JWT | 내 정보 |
| POST | `/api/v1/app/user/logout` | JWT | 쿠키 삭제 |

응답 형태:

```json
{ "code": 400, "message": "invalid input", "error": "validation failed",
  "errors": [{ "field": "email", "message": "email must be a valid email address" }],
  "data": null }
```

dev 에서는 토큰이 응답 바디(`data.access_token`)에도 오고 `Authorization: Bearer <token>` 도 받는다. 그 외에는 `HttpOnly` 쿠키로만 쓴다.

Swagger(dev 만): `make create-swagger` 후 `http://localhost:8080/swagger/index.html` (`SWAGGER_ID` / `SWAGGER_PASSWORD`).

## DB 마이그레이션 (goose)

스키마 변경은 파일 하나씩 쌓는다. DB 의 `goose_db_version` 테이블에 적용 기록이 남아, 안 된 것만 실행된다.

```bash
make migrate-create name=add_users_nickname   # migrations/<시각>_add_users_nickname.sql 생성
```

```sql
-- +goose Up
alter table v1.users add column nickname text;

-- +goose Down
alter table v1.users drop column nickname;
```

```bash
make migrate-up       # 적용
make migrate-status   # 현황
make migrate-down     # 마지막 1개 되돌리기
make sqlc             # 스키마가 바뀌었으면 쿼리 코드 다시 생성
```

- 운영: 기본은 앱 시작 시 자동 적용. 여러 대가 동시에 떠도 한 대만 적용한다. 큰 테이블을 건드리는 오래 걸리는 변경은 `MIGRATE_ON_START=false` 로 두고 배포 전에 `go run ./cmd/migrate up` 을 따로 실행한다.
- 이미 적용된 파일은 고치지 않는다. 고칠 게 있으면 새 파일을 만든다.

## Commands

```bash
make help             # 전체 목록
make dev | local-run | db-up | down | logs
make migrate-up | migrate-down | migrate-status | migrate-create name=...
make sqlc | create-swagger
make test | lint
make build            # 운영 이미지 (distroless, nonroot)
make tools-upgrade    # air, swag, sqlc, golangci-lint
make rename name=...  # 프로젝트 이름 변경 (scripts/rename.sh)
```

push 하면 CI(`.github/workflows/ci.yml`)가 검사한다: build, vet, test -race, sqlc diff, govulncheck, golangci-lint, 운영 이미지 빌드.

## Project Structure

```
cmd/api/            서버 진입점 (wiring, 종료 순서)
cmd/migrate/        마이그레이션 CLI
config/             env 로드·검증
migrations/         goose SQL (바이너리에 내장)
domain/<name>/      도메인 모델·인터페이스·서비스, postgresql/ 에 repository + query.sql
internal/
  http/chi/         라우터·핸들러·미들웨어
  database/         TxManager·공통 에러, postgresql/ 에 pgx 구현과 sqlc 생성 코드
  auth/ jwt/ validator/ logger/ jobs/ api/
  blockchain/       선택 모듈 (go-ethereum)
  database/redis/   미연결
```

자세한 규칙은 [CLAUDE.md](CLAUDE.md).

## 운영 배포 체크리스트

- `STAGE=prod`, `CORS_ALLOWED_ORIGINS` 에 프론트 출처
- `TRUSTED_PROXY_HOPS` 를 실제 프록시 단계 수로 (잘못 두면 IP 위조로 rate limit 우회)
- `DB_SSLMODE` (기본 `require`, 가능하면 `verify-full`)
- 헬스체크: liveness `/health`, readiness `/ready`
- 이미지는 `make build VERSION=<tag>`, 로그는 stdout JSON
