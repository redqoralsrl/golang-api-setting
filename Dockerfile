# 로컬 개발용 (docker compose). 소스는 볼륨으로 마운트하고 air 가 변경 시 다시 빌드한다.
FROM golang:1.26-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ENV TZ=UTC
EXPOSE 8080

CMD ["go", "tool", "air", "-c", "cmd/api/.air.toml"]
