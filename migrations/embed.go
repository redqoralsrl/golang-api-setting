// Package migrations 는 goose 마이그레이션 SQL 을 바이너리에 내장한다.
// 새 파일: make migrate-create name=add_orders  (파일명 앞 숫자가 적용 순서)
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
