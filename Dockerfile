FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .
RUN go install github.com/air-verse/air@latest
RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /app/cmd/api
CMD ["air", "-c", "/app/cmd/api/.air.toml"]
