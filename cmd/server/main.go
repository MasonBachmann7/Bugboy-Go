package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	bugstack "github.com/MasonBachmann7/bugstack-go"
	"github.com/masonbachmann7/bugboy-go/internal/server"
)

func main() {
	logger := log.New(os.Stdout, "bugboy-go ", log.LstdFlags|log.Lmicroseconds|log.LUTC)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize BugStack error monitoring
	bugstack.Init(bugstack.Config{
		APIKey:   os.Getenv("BUGSTACK_API_KEY"),
		Endpoint: os.Getenv("BUGSTACK_ENDPOINT"),
	})
	defer bugstack.Flush()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           server.NewHandler(logger),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-shutdownSignal.Done()
		logger.Printf("shutdown signal received; stopping server")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Printf("graceful shutdown failed: %v", err)
		}
	}()

	logger.Printf("listening on http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("server failed: %v", err)
	}
}
