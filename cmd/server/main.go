// Package main provides the entry point for the MeloVault server.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/h0sh1-no/MeloVault/internal/config"
	"github.com/h0sh1-no/MeloVault/internal/database"
	"github.com/h0sh1-no/MeloVault/internal/server"
)

func main() {
	_ = loadDotEnv(".env")

	logger := log.New(os.Stdout, "[melovault] ", log.LstdFlags|log.Lmicroseconds)
	cfg := config.FromEnv()

	if err := os.MkdirAll(cfg.DownloadsDir, 0o755); err != nil {
		logger.Fatalf("create downloads dir failed: %v", err)
	}

	app := server.New(cfg, logger)

	// Initialize database if configured
	if cfg.DBHost != "" && cfg.DBPassword != "" {
		dbCfg := database.Config{
			Host:     cfg.DBHost,
			Port:     cfg.DBPort,
			User:     cfg.DBUser,
			Password: cfg.DBPassword,
			Database: cfg.DBName,
			SSLMode:  cfg.DBSSLMode,
		}

		pool, err := database.NewPool(dbCfg)
		if err != nil {
			logger.Printf("database connection failed: %v (running without auth features)", err)
		} else {
			defer pool.Close()

			// Run migrations
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := pool.Migrate(ctx); err != nil {
				logger.Printf("database migration failed: %v", err)
			}
			cancel()

			// Attach pool to server
			app.WithDB(pool)
			logger.Println("database initialized with auth features enabled")
		}
	} else {
		logger.Println("database not configured, running without auth features")
	}

	httpServer := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      app.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  90 * time.Second,
	}

	printBanner(cfg)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("start server failed: %v", err)
		}
	}()

	waitForShutdown(httpServer, cfg.ShutdownTimeout, logger)
}

func waitForShutdown(srv *http.Server, timeout time.Duration, logger *log.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Printf("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("graceful shutdown failed: %v", err)
		if closeErr := srv.Close(); closeErr != nil {
			logger.Printf("force close failed: %v", closeErr)
		}
	}
	logger.Printf("server stopped")
}

func printBanner(cfg config.Config) {
	host := cfg.Host
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("MeloVault Music Server")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Listen:    http://%s:%s\n", host, cfg.Port)
	fmt.Printf("Downloads: %s\n", cfg.DownloadsDir)
	fmt.Println("API endpoints:")
	fmt.Println("  GET  /health              - Health check")
	fmt.Println("  GET/POST /song            - Song details / URL")
	fmt.Println("  GET/POST /search          - Search music")
	fmt.Println("  GET/POST /playlist        - Playlist details")
	fmt.Println("  GET/POST /album           - Album details")
	fmt.Println("  GET/POST /download        - Download music")
	fmt.Println("  POST /api/auth/register   - Register")
	fmt.Println("  POST /api/auth/login      - Login")
	fmt.Println("  GET  /api/auth/linuxdo    - LinuxDo OAuth")
	fmt.Println(strings.Repeat("=", 60))
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return scanner.Err()
}
