# syntax=docker/dockerfile:1.7

FROM golang:1.25.7-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /out/lihatin-go ./main.go

FROM alpine:3.22
WORKDIR /app

RUN addgroup -S app && adduser -S -G app app && \
    apk add --no-cache ca-certificates tzdata curl

COPY --from=builder /out/lihatin-go /usr/local/bin/lihatin-go

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=5 \
  CMD curl -fsS http://127.0.0.1:8080/v1/health || exit 1

ENTRYPOINT ["/usr/local/bin/lihatin-go"]
