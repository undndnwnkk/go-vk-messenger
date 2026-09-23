# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

ENV CGO_ENABLED=0 GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/go-vk-messenger ./cmd/app
RUN go build -trimpath -ldflags="-s -w" -o /out/go-vk-messenger-migrate ./cmd/migrate

# Migration runtime stage
FROM alpine:3.20 AS migrate

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /out/go-vk-messenger-migrate /app/go-vk-messenger-migrate
COPY migrations /migrations

USER app
ENTRYPOINT ["/app/go-vk-messenger-migrate"]

# Application runtime stage
FROM alpine:3.20 AS app

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /out/go-vk-messenger /app/go-vk-messenger

USER app
EXPOSE 8080

ENTRYPOINT ["/app/go-vk-messenger"]
