#!/usr/bin/env bash
# 프로젝트 이름을 바꾼다. 현재 이름은 go.mod 의 module 에서 읽는다.
#   usage: scripts/rename.sh <new-name>   (make rename name=<new-name>)
#
# 바꾸는 곳
#   go.mod module, *.go / *.md 의 import 경로, 인증 쿠키 이름(<name>_access_token),
#   docker-compose.yaml 프로젝트 이름, Makefile IMAGE
# 마지막에 go build 와 swagger 재생성(docs/api)을 한다.
#
# 치환은 perl 로 한다. Git Bash 의 sed -i 는 CRLF 를 LF 로 바꿔 버린다.
set -euo pipefail

cd "$(dirname "$0")/.."

new="${1:-}"
if [[ ! "$new" =~ ^[a-z]([a-z0-9-]*[a-z0-9])?$ ]]; then
	echo "usage: make rename name=<project-name>" >&2
	echo "이름은 소문자, 숫자, '-' 만 쓴다 (docker 이미지·compose 이름 규칙)" >&2
	exit 1
fi

old="$(awk '$1 == "module" { print $2; exit }' go.mod | tr -d '\r')"
if [[ -z "$old" ]]; then
	echo "go.mod 에서 module 을 찾지 못했다" >&2
	exit 1
fi
if [[ "$old" == "$new" ]]; then
	echo "이미 이름이 $new 이다"
	exit 0
fi

export OLD="$old" NEW="$new" OLD_SNAKE="${old//-/_}" NEW_SNAKE="${new//-/_}"

# 생성물(docs/)과 바이너리(bin/)는 건드리지 않는다. docs 는 아래에서 다시 만든다.
find_files() {
	grep -rlF \
		--include='*.go' --include='*.md' \
		--exclude-dir=.git --exclude-dir=docs --exclude-dir=bin --exclude-dir=tmp \
		-- "$1" . || true
}

echo "rename: $old -> $new"

go mod edit -module "$new"

mapfile -t files < <(find_files "$old/")
if ((${#files[@]})); then
	perl -pi -e 's{\Q$ENV{OLD}\E/}{$ENV{NEW}/}g' "${files[@]}"
fi
echo "  import 경로: ${#files[@]}개 파일"

mapfile -t files < <(find_files "${OLD_SNAKE}_access_token")
if ((${#files[@]})); then
	perl -pi -e 's{\Q$ENV{OLD_SNAKE}\E_access_token}{$ENV{NEW_SNAKE}_access_token}g' "${files[@]}"
fi
echo "  쿠키 이름: ${NEW_SNAKE}_access_token"

perl -pi -e 's{:-\Q$ENV{OLD}\E\}}{:-$ENV{NEW}\}}' docker-compose.yaml
perl -pi -e 's{^(IMAGE\s*\?=\s*)\Q$ENV{OLD}\E(?=\s|$)}{$1$ENV{NEW}}' Makefile
echo "  compose 프로젝트 / 이미지: $new"

echo "go build ./..."
go build ./...

echo "swagger 재생성 (docs/api)"
go tool swag init -g ./internal/http/chi/api_handler.go -o ./docs/api --parseDependency --parseInternal --instanceName api >/dev/null

echo "완료. 남은 '$old' 흔적:"
grep -rnF --exclude-dir=.git --exclude-dir=bin --exclude-dir=tmp --exclude-dir=scripts --exclude=go.sum \
	-- "$old" . || echo "  없음"
