# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Go HTTP API 템플릿. Chi v5 + PostgreSQL(pgx/v5, sqlc, goose). 진입점은 `cmd/api/main.go`.

## Commands

```bash
# 실행
make db-up            # PostgreSQL 컨테이너만
make dev              # 로컬 hot reload (air, .env.local 로드)
make local-run        # API + DB 모두 docker compose

# DB 마이그레이션 (goose, migrations/)
make migrate-create name=add_orders   # 새 파일
make migrate-up | migrate-down | migrate-status

# 코드 생성
make sqlc             # migrations/ + */query.sql → internal/database/postgresql/gen
make create-swagger   # docs/api

# 검사
make test             # go test -race
make lint             # ./bin/golangci-lint (없으면 make tools-upgrade)
make build            # 운영 이미지 (Dockerfile.prod, distroless)

# 템플릿에서 새 프로젝트 시작
make rename name=order-api   # go.mod·import·쿠키·compose·이미지 이름 변경 후 build + swagger 재생성
```

## Architecture

```
cmd/api/             DI wiring, 서버 시작/종료
cmd/migrate/         마이그레이션 CLI (up/down/status/create)
config/              env 로드 + 검증 (Load, LoadDatabase)
migrations/          goose SQL. 바이너리에 내장(embed). sqlc 스키마 원본
domain/<name>/       엔티티, Reader/Writer/Repository/UseCase 인터페이스, 도메인 에러, service
  postgresql/        repository 구현 + query.sql
internal/
  auth/              Role, 토큰 TTL, 요청 주체(Identity) context
  api/               HTTP 서버 (graceful shutdown)
  database/          드라이버 무관 TxManager, 공통 에러(ErrNotFound, ErrUniqueViolation...), 락 인터페이스
    postgresql/      pgxpool, 트랜잭션, MapError, 커서 암호화, advisory lock, Migrator
      gen/           sqlc 생성 코드 (수정 금지)
  http/              공통 응답(Response, WriteError, WriteValidationError)
    chi/             라우터, 핸들러, bindJSON
      middleware/    clientip, logger, recover, ratelimit, cors/csrf, security, apikey, auth, swagger_auth
  jobs/              cron 스케줄러 (advisory lock 으로 서버당 한 대만 실행)
  jwt/  validator/  logger/(zerolog, zap)
  blockchain/        선택 모듈 (아래 참고)
  database/redis/    아직 미연결
```

### 의존 방향

`internal/http` → `domain` → `internal/{database, auth, logger}` 인터페이스.
도메인은 HTTP 패키지(`internal/http/...`)나 구현 패키지(`internal/database/postgresql`)를 import 하지 않는다.
`domain/<name>/postgresql` 만 `internal/database/postgresql` 을 쓴다.

### Repository / DB

- `s.db.Queries(ctx)` 가 sqlc 쿼리를 돌려준다. ctx 에 트랜잭션이 있으면 자동으로 그 트랜잭션을 쓴다.
- DB 에러는 repository 에서 `postgresql.MapError(err)` → `database.Err*` → 도메인 에러로 바꿔 돌려준다. 예: `domain/user/postgresql/storage.go` 의 `mapError`.
- 세션 설정(timezone UTC, statement_timeout 10s, lock_timeout 5s, idle_in_transaction 30s)은 연결할 때 건다. 마이그레이션 연결에는 걸지 않는다.
- 커서 페이지네이션: `db.EncryptCursor(id)` / `DecryptCursor` (AES-GCM).

### 트랜잭션

service 에서 직접 Begin/Commit 하지 않는다. `database.TxManager` 를 주입받아 쓴다.

```go
func (s *Service) SomeOperation(ctx context.Context) error {
    return s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 이 ctx 를 넘기면 같은 트랜잭션
        if err := s.repo.DoA(ctx); err != nil { return err }
        return s.repo.DoB(ctx)
    }) // 옵션: database.ReadOnly(), database.WithIsolation(database.Serializable)
}
```

트랜잭션 안에서 쿼리 하나가 실패하면 PostgreSQL 은 이후 쿼리와 Commit 도 실패시킨다. tx 안의 에러는 무시하지 말고 돌려준다.

### 마이그레이션

- 스키마 변경은 항상 새 파일로 (`make migrate-create`). 이미 적용된 파일은 고치지 않는다.
- 파일에는 `-- +goose Up` / `-- +goose Down` 을 둔다. 함수 본문(`$$`)은 `-- +goose StatementBegin/End` 로 감싼다.
- `MIGRATE_ON_START=true`(기본)면 앱 시작 시 Up 을 적용한다. advisory lock 으로 여러 대 중 한 대만 적용한다. Down 은 자동으로 실행하지 않는다.
- 새 테이블 규칙: `id bigint generated always as identity`, `timestamptz`, 문자열은 `text`, `updated_at` 이 있으면 `v1.set_updated_at()` trigger.

### HTTP

라우트 그룹 (`internal/http/chi/api_handler.go`, 핸들러는 `handler.Routes.*` 에 등록):

| Routes 필드 | 경로 | 미들웨어 | body 한도 |
|---|---|---|---|
| `Public` | `/api/v1` | CSRF, API key | 1MB |
| `App` | `/api/v1/app` | CSRF, API key, 사용자 JWT | 1MB |
| `AppUpload` | `/api/v1/app` | 위와 같음 | 20MB |
| `Admin` | `/admin/v1` | CSRF | 1MB (관리자 인증을 붙이기 전에는 라우트 등록 금지) |

system: `/health`(liveness), `/ready`(DB ping), `/swagger/*`(dev 만, basic auth).

전역 미들웨어 순서: `RequestID` → `ClientIP` → `Logger` → `Recovery` → `RateLimit`(IP당 30rps, burst 60) → `CORS` → `SecurityHeaders`.

- **ClientIP**: `TRUSTED_PROXY_HOPS` 로 X-Forwarded-For 오른쪽에서 N번째 값을 쓴다. 다른 헤더는 위조 가능해 읽지 않는다. IP 는 `middleware.ClientIP(r)` 로 읽는다.
- **Logger**: 요청 바디는 64KB 이하 JSON 만, `loggableBodyFields` 에 있는 필드 값만 남기고 나머지는 가린다(허용 목록). 4xx/5xx 는 `v1.error_log` 에 비동기 저장한다(401/403/404/405/413/429/503 제외, 큐가 차면 버림).
- **CSRF**: `http.CrossOriginProtection`. 신뢰 출처는 `CORS_ALLOWED_ORIGINS` (dev 에서 비우면 localhost:3000/5173).

핸들러 작성 규칙:

```go
var req CreateOrderRequest
if !bindJSON(w, r, h.validator, &req) { return }   // 400 / 413 / 필드별 errors 응답까지 처리

result, err := h.useCase.Create(r.Context(), ...)
if err != nil {
    switch {
    case errors.Is(err, order.ErrOrderNotFound):
        response.WriteError(w, http.StatusNotFound, response.NotFound, order.ErrOrderNotFound)
    default: // 5xx 는 원인을 로그로 남기고 응답에는 고정 문구만
        response.WriteErrorWithRequest(w, r, h.logger, http.StatusInternalServerError, response.InternalError, err)
    }
    return
}
response.WriteResponse(w, http.StatusOK, response.Success, nil, result)
```

- 4xx 에는 도메인 에러(사용자에게 보여도 되는 문구)만 넘긴다. DB 에러 원문을 4xx 로 내보내지 않는다.
- 응답 엔티티에 비밀값 필드가 있으면 `json:"-"` (예: `User.PasswordHash`).
- 인증된 사용자: `auth.FromContext(r.Context())` → `Identity{UserID, Role}`. 역할 제한은 `middleware.RequireRole(auth.RoleUser)`.
- Swagger 주석을 단다. 응답 형태: `response.Response{data=<pkg>.<Type>}`.

### Auth

JWT HS256 (`internal/jwt`: HS256 만 허용, exp 필수). claims: `user_id`, `role`. 앱 토큰은 `HttpOnly` 쿠키 `go_template_access_token` (path `/api/v1/app`, `SameSite=Lax`, dev 외 `Secure`), 1시간.
dev 에서는 응답 바디에도 토큰을 주고 `Authorization: Bearer` 도 받는다.

API: `POST /api/v1/user/create`, `POST /api/v1/user/login`, `GET /api/v1/app/user/me`, `POST /api/v1/app/user/logout`.

### 주기 작업

`JOBS_ENABLED=true` 일 때 `cmd/api/main.go` 에서 `scheduler.Add(jobs.Job{Name, Spec, Timeout, Run})`. Spec 은 UTC cron. 서버가 여러 대여도 `Name` 별 advisory lock 을 잡은 한 대만 실행한다.

### 선택 모듈

- `internal/blockchain`: go-ethereum 기반. `eth_addr` 검증 태그는 쓰는 프로젝트만 `validator.NewValidationService(blockchain.ValidationRules()...)` 로 켠다. 안 쓰면 API 바이너리에 go-ethereum 이 들어가지 않는다.
- `internal/database/redis`, `internal/logger/zap`: 아직 연결하지 않음. Redis 는 `docker compose --profile cache up -d cache_db`.

## Environment Variables

`.env.template` 을 `.env.local` 로 복사한다. 설정이 잘못되면 `config.Load()` 가 모든 오류를 모아 시작을 막는다.

| Variable | 설명 |
|---|---|
| `STAGE` | `dev` / `staging` / `prod` (기본 dev) |
| `API_HOST`, `API_PORT` | HTTP 바인드 (기본 포트 8080) |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_HOST`, `DB_PORT`, `DB_SSLMODE` | PostgreSQL. sslmode 기본: dev `disable`, 그 외 `require` |
| `MIGRATE_ON_START` | 시작 시 마이그레이션 Up (기본 true) |
| `API_SECRET_KEY` | JWT 서명 키, 32자 이상 |
| `CURSOR_SECRET` | 커서 암호화 키, 정확히 32자 |
| `API_X_API_KEY` | 값이 있으면 `/api/v1` 에 `X-API-Key` 헤더 요구 |
| `CORS_ALLOWED_ORIGINS` | 쉼표 구분. CORS + CSRF 신뢰 출처. dev 외 필수 |
| `TRUSTED_PROXY_HOPS` | 앞단 프록시 수 (0 직접, 1 ALB/nginx, 2 CloudFront→ALB) |
| `JOBS_ENABLED` | 주기 작업 실행 (기본 false) |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` | Redis (미연결) |
| `SWAGGER_ID`, `SWAGGER_PASSWORD` | `/swagger/*` basic auth. 비우면 접근 불가 |
| `IP_INFO_TOKEN`, `SMTP_USERNAME`, `SMTP_PASSWORD` | 외부 서비스 (선택) |

## 새 도메인 추가 순서

`/add-domain <name>` 스킬이 아래를 자동으로 한다.

1. `make migrate-create name=create_<name>s` → 테이블 작성 → `make migrate-up`
2. `domain/<name>/postgresql/query.sql` 작성, `sqlc.yaml` 의 `queries` 에 경로 추가 → `make sqlc`
3. `domain/<name>/`: `<name>.go`(모델·인터페이스), `error.go`(도메인 에러), `service.go`
4. `domain/<name>/postgresql/storage.go`: sqlc 쿼리 호출 + `mapError`
5. `internal/http/chi/<name>.go`: 핸들러 + Swagger 주석, `handler.go` 의 `Services` 에 필드 추가, `api_handler.go` 에서 등록
6. `cmd/api/main.go`: storage → service → `Services` 순서로 wiring
7. `go build ./... && make test`, `make create-swagger`

## 금지 사항

- Go 파일 안에 raw SQL 작성 금지 → `query.sql` 에 작성 후 `make sqlc`
- 적용된 마이그레이션 파일 수정 금지 → 새 파일 추가
- `internal/database/postgresql/gen/`, `docs/` 직접 수정 금지 (생성 코드)
- DI 프레임워크 사용 금지 → `cmd/api/main.go` 에서 수동 wiring
- 도메인 레이어에서 외부 라이브러리·HTTP 패키지·DB 구현 패키지 import 금지 (인터페이스로 추상화). 예외: 표준 라이브러리, `golang.org/x/crypto`
- 요청 바디 로그 허용 목록(`loggableBodyFields`)에 개인정보·비밀값 필드 추가 금지
