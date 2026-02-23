package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/masonbachmann7/bugboy-go/internal/bugstack"
	"github.com/masonbachmann7/bugboy-go/internal/server"
	"github.com/MasonBachmann7/bugstack-go"
)

func main() {
	// BugStack error monitoring
	bugstack.Init(os.Getenv("BUGSTACK_API_KEY"))

	logger := log.New(os.Stdout, "bugboy-go ", log.LstdFlags|log.Lmicroseconds|log.LUTC)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	bugstackClient := bugstack.InitFromEnv(logger)
	defer bugstackClient.Flush()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           server.NewHandlerWithReporter(logger, bugstackClient),
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
