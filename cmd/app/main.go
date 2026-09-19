package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/config"
	"github.com/undndnwnkk/go-vk-messenger/internal/controller/restapi"
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
	messageService := service.NewMessageService(messageRepo, *chatService)

	handler := restapi.NewHandler(*userService, jwtService, pgxPool, *chatService, *messageService)
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

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http server shutdown failed: %v", err)
		return
	}
}
