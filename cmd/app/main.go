package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/undndnwnkk/go-vk-messenger/internal/controller/restapi"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("application starting")

	if err := godotenv.Load(); err != nil {
		log.Fatal("error downloading env: " + err.Error())
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	dbURL := buildDSN()
	pgxPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("error connecting to database: " + err.Error())
	}

	if err := pgxPool.Ping(ctx); err != nil {
		log.Fatal("error ping to database: " + err.Error())
	}

	log.Println("pgxpool created")

	handler := handler.NewHandler(pgxPool)
	server := http.Server{
		Addr:    os.Getenv("HTTP_PORT"),
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("error starting server: " + err.Error())
		}
	}()
	log.Println("server started")

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

func buildDSN() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
}
