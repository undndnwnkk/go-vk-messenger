package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/undndnwnkk/go-vk-messenger/internal/config"
	"github.com/undndnwnkk/go-vk-messenger/internal/controller/restapi"
	"github.com/undndnwnkk/go-vk-messenger/internal/health"
	"github.com/undndnwnkk/go-vk-messenger/internal/observability"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.NewConfig()
	if err := cfg.Load(); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	startup, cancelStartup := context.WithTimeout(ctx, 10*time.Second)
	defer cancelStartup()
	db, err := pgxpool.New(startup, cfg.BuildDSN())
	if err != nil {
		return errors.New("invalid database configuration")
	}
	defer db.Close()
	if err := db.Ping(startup); err != nil {
		return errors.New("postgres unavailable at startup")
	}
	ready := health.Checker{Checks: []func(context.Context) error{db.Ping}}
	hub := realtime.NewHub()
	defer hub.Shutdown()
	var limiter service.MessageLimiter = service.NewMessageRateLimiter()
	if cfg.RedisURL != "" {
		options, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return errors.New("invalid REDIS_URL")
		}
		options.ContextTimeoutEnabled = true
		options.MaxRetries = -1
		options.DialTimeout, options.ReadTimeout, options.WriteTimeout = 2*time.Second, 2*time.Second, 2*time.Second
		client := redis.NewClient(options)
		defer client.Close()
		if err := client.Ping(startup).Err(); err != nil {
			return errors.New("redis unavailable at startup")
		}
		relay, err := realtime.NewRedisRelay(startup, client, hub, cfg.RedisPrefix+":events")
		if err != nil {
			return err
		}
		defer relay.Close()
		limiter = service.NewRedisRateLimiter(client, cfg.RedisPrefix)
		ready.Checks = append(ready.Checks, func(ctx context.Context) error { return client.Ping(ctx).Err() })
		log.Println("redis realtime relay and shared rate limiter enabled")
	}
	jwt := service.NewJWTService(cfg.JWTConfig.Secret, cfg.JWTConfig.TTL)
	users := service.NewUserService(repository.NewUserRepository(db), jwt)
	chats := service.NewChatService(repository.NewChatRepository(db), *users)
	messages := service.NewMessageServiceWithLimiter(repository.NewMessageRepository(db), *chats, limiter)
	metrics := observability.New(hub)
	handler := restapi.NewHandler(*users, jwt, ready, *chats, *messages, hub, metrics.Middleware)
	api := &http.Server{Addr: cfg.HTTPAddr(), Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", metrics.Handler())
	admin := &http.Server{Addr: cfg.MetricsAddr, Handler: metricsMux, ReadHeaderTimeout: 5 * time.Second}
	apiListener, err := net.Listen("tcp", api.Addr)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer apiListener.Close()
	metricsListener, err := net.Listen("tcp", admin.Addr)
	if err != nil {
		return fmt.Errorf("listen metrics: %w", err)
	}
	defer metricsListener.Close()
	serverErr := make(chan error, 2)
	go func() { serverErr <- api.Serve(apiListener) }()
	go func() { serverErr <- admin.Serve(metricsListener) }()
	log.Printf("HTTP listening on %s; metrics on %s", api.Addr, admin.Addr)
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-serverErr:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Shutdown does not close hijacked WebSockets; close them explicitly first.
	hub.Shutdown()
	if err := api.Shutdown(shutdown); err != nil {
		_ = api.Close()
	}
	if err := admin.Shutdown(shutdown); err != nil {
		_ = admin.Close()
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return nil
}
