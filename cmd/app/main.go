package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/undndnwnkk/go-vk-messenger/internal/config"
	"github.com/undndnwnkk/go-vk-messenger/internal/controller/restapi"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("application starting")

	config := config.NewConfig()
	config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	dbURL := config.BuildDSN()
	pgxPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("error connecting to database: " + err.Error())
	}

	if err := pgxPool.Ping(ctx); err != nil {
		log.Fatal("error ping to database: " + err.Error())
	}

	log.Println("pgxpool created")

	handler := restapi.NewHandler(pgxPool)
	server := http.Server{
		Addr:    config.HTTPConfig.Addr,
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("error starting server: " + err.Error())
		}
	}()
	log.Println("server started at port: " + server.Addr)

	<-ctx.Done()
	log.Println("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("http server stopped")
	pgxPool.Close()
	log.Println("postgres connection closed")
}
