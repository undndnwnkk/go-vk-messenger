# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/go-vk-messenger ./cmd/app

# Runtime stage
FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /out/go-vk-messenger /app/go-vk-messenger

USER app
EXPOSE 8080

ENTRYPOINT ["/app/go-vk-messenger"]