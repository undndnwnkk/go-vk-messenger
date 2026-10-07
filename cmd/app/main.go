package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/undndnwnkk/go-vk-messenger/internal/config"
	"github.com/undndnwnkk/go-vk-messenger/internal/controller/restapi"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/repository"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("application starting")

	config := config.NewConfig()
	if err := config.Load(); err != nil {
		log.Fatal("error loading config: " + err.Error())
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	dbURL := config.BuildDSN()
	pgxPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("error connecting to database: " + err.Error())
	}
	defer func() {
		pgxPool.Close()
		log.Println("postgres connection closed")
	}()

	if err := pgxPool.Ping(ctx); err != nil {
		log.Fatal("error ping to database: " + err.Error())
	}

	log.Println("pgxpool created")

	userRepo := repository.NewUserRepository(pgxPool)
	jwtService := service.NewJWTService(config.JWTConfig.Secret, config.JWTConfig.TTL)
	userService := service.NewUserService(userRepo, jwtService)
	chatRepo := repository.NewChatRepository(pgxPool)
	chatService := service.NewChatService(chatRepo, *userService)
	messageRepo := repository.NewMessageRepository(pgxPool)
	hub := realtime.NewHub()
	var limiter service.MessageLimiter = service.NewMessageRateLimiter()
	if config.RedisURL != "" {
		options, err := redis.ParseURL(config.RedisURL)
		if err != nil {
			log.Fatal("invalid REDIS_URL")
		}
		options.ContextTimeoutEnabled = true
		options.MaxRetries = -1
		options.DialTimeout = 2 * time.Second
		options.ReadTimeout = 2 * time.Second
		options.WriteTimeout = 2 * time.Second
		client := redis.NewClient(options)
		defer client.Close()
		startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := client.Ping(startupCtx).Err(); err != nil {
			log.Fatal("redis unavailable at startup")
		}
		relay, err := realtime.NewRedisRelay(startupCtx, client, hub, config.RedisPrefix+":events")
		if err != nil {
			log.Fatal(err)
		}
		defer relay.Close()
		limiter = service.NewRedisRateLimiter(client, config.RedisPrefix)
		log.Println("redis realtime relay and shared rate limiter enabled")
	}
	messageService := service.NewMessageServiceWithLimiter(messageRepo, *chatService, limiter)
	handler := restapi.NewHandler(*userService, jwtService, pgxPool, *chatService, *messageService, hub)
	server := http.Server{
		Addr:    config.HTTPAddr(),
		Handler: handler,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("http server started on %s", server.Addr)

		err := server.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")

	case err := <-serverErr:
		if err != nil {
			log.Printf("http server failed: %v", err)
			return
		}

		log.Println("http server stopped")
		return
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	shutdownErr := make(chan error, 1)
	go func() {
		shutdownErr <- server.Shutdown(shutdownCtx)
	}()

	hub.Shutdown()
	log.Println("websocket clients closed")

	if err := <-shutdownErr; err != nil {
		log.Printf("http server shutdown failed: %v", err)
		return
	}
}
