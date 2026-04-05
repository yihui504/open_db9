package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-db9/db9/internal/api/handlers"
	"github.com/open-db9/db9/internal/api/router"
	"github.com/open-db9/db9/internal/auth"
	"github.com/open-db9/db9/internal/config"
	"github.com/open-db9/db9/internal/database"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize database manager
	poolConfig := &database.PoolConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Database: cfg.Database.Database,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		SSLMode:  "disable",
		// Pool settings with defaults
		MaxConnections:    25,
		MinConnections:    5,
		MaxConnLifetime:   0,                 // No limit
		MaxConnIdleTime:   0,                 // No limit
		HealthCheckPeriod: 1 * time.Minute,   // Required positive value
		ConnectTimeout:    0,                 // Use pgx default
	}

	manager, err := database.NewManager(poolConfig)
	if err != nil {
		log.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.Health(ctx); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}

	// Initialize dbRegistry and register the database
	// Using ID 1 as the default database connection
	handlers.RegisterDatabase(1, manager)

	// Initialize auth manager
	authManager, err := auth.NewManager()
	if err != nil {
		log.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create router with middleware and auth
	mux := router.NewRouter(authManager)

	// Configure server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.Timeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("DB9 API Server starting on %s", server.Addr)
		log.Printf("Database connected to %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	// Unregister database
	handlers.UnregisterDatabase(1)

	log.Println("Server shutdown complete")
}
