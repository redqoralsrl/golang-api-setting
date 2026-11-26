---
name: add-migration
description: Add a goose database migration to this Go API project (new table, column, index, constraint) with matching Down, then regenerate sqlc code and check the build. Usage: /add-migration <snake_case_name> [what to change]. Example: /add-migration add_users_nickname "users 에 nickname text 추가"
tools: Read, Write, Edit, Glob, Grep, Bash
---

# Add Migration Skill

**Arguments**: `$ARGUMENTS` — first token is the migration name in snake_case (`add_users_nickname`), the rest describes the change. If blank, print `Usage: /add-migration <snake_case_name> [what to change]` and stop.

## Rules

- Never edit a migration that already exists in `migrations/`. Always add a new file.
- Every file has `-- +goose Up` and `-- +goose Down`. Down must undo Up exactly (drop what Up created, in reverse order).
- Wrap function/DO bodies (`$$ ... $$`) in `-- +goose StatementBegin` / `-- +goose StatementEnd`.
- Conventions: `id bigint generated always as identity primary key`, `timestamptz` (never `timestamp`), `text` for strings, `bigint` for ids and foreign keys with `references ... (id)` and an index on the FK column, `updated_at` columns get the `v1.set_updated_at()` trigger. Add `comment on table/column` for new objects.
- Production-safe changes:
  - Adding a `not null` column to an existing table needs a `default`, or add it nullable → backfill → set not null in a later migration.
  - Indexes on large existing tables: `create index concurrently` and put `-- +goose NO TRANSACTION` at the top of the file (concurrently cannot run in a transaction). Mention this to the user.
  - Renaming or dropping a column the running app still reads breaks old instances during a rolling deploy. Warn the user and suggest add-new → deploy → drop-old in separate migrations.

## Steps

1. Read the existing files in `migrations/` (in name order) to know the current schema.
2. Create the file:
   ```bash
   go run ./cmd/migrate create <name>
   ```
3. Write Up and Down in the new file.
4. Regenerate and check:
   ```bash
   go tool sqlc generate
   go build ./... && go test ./...
   ```
   If a changed column is used by queries in `domain/*/postgresql/query.sql`, update those queries and the storage mapping (`to<Pascal>`) and domain model fields so the build passes.
5. If the local DB is up (`docker compose ps db` shows it running), apply and verify the round trip:
   ```bash
   make migrate-up && make migrate-down && make migrate-up
   ```
6. Report the file path, the SQL, and any deploy caveat from the rules above.
