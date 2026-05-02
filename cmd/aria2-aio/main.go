package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/user/aria2-aio/internal/api"
	"github.com/user/aria2-aio/internal/config"
	"github.com/user/aria2-aio/internal/instance"
	"github.com/user/aria2-aio/internal/store"
	"github.com/user/aria2-aio/internal/task"
	"github.com/user/aria2-aio/internal/web"
	"github.com/user/aria2-aio/internal/ws"
	"github.com/user/aria2-aio/ui"
)

func main() {
	dataDir := flag.String("data-dir", "./data", "Path to data directory")
	configFile := flag.String("config", "", "Path to config file (default: <data-dir>/config.yaml)")
	host := flag.String("host", "", "Override HTTP server host")
	port := flag.Int("port", 0, "Override HTTP server port")
	aria2c := flag.String("aria2c", "", "Path to aria2c binary")
	logLevel := flag.String("log-level", "", "Override log level")
	dev := flag.Bool("dev", false, "Development mode: serve frontend from filesystem")
	showVersion := flag.Bool("version", false, "Show version info")
	flag.Parse()

	if *showVersion {
		fmt.Println("aria2-aio v0.1.0")
		return
	}

	// Determine config file path
	cfgPath := *configFile
	if cfgPath == "" {
		cfgPath = filepath.Join(*dataDir, "config.yaml")
	}

	// Load config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Apply CLI overrides
	if *dataDir != "./data" {
		cfg.DataDir = *dataDir
	}
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *aria2c != "" {
		cfg.Defaults.Aria2cPath = *aria2c
	}
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}

	// Resolve aria2c path
	aria2cPath := cfg.ResolveAria2cPath()
	log.Printf("Using aria2c at: %s", aria2cPath)

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	log.Printf("aria2-aio starting with data_dir=%s, server=%s:%d, dev=%v",
		cfg.DataDir, cfg.Server.Host, cfg.Server.Port, *dev)

	// Initialize SQLite
	db, err := store.OpenDB(cfg.DBPath())
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create WebSocket hub
	hub := ws.NewHub()

	// Create process supervisor
	supervisor := instance.NewProcessSupervisor(aria2cPath)

	// Create instance manager
	manager := instance.NewInstanceManager(db, supervisor, cfg)

	// Create task tracker
	tracker := task.NewTaskTracker(hub, db.Tasks())

	// Recover previously running instances
	manager.RecoverRunningInstances()
	for id, ai := range manager.ListActive() {
		tracker.Register(id, ai.RPCClient, ai.Listener, cfg.TaskTracker.PollInterval)
	}

	// Create API server
	apiServer := api.NewServer(manager, db, tracker, hub)
	mux := http.NewServeMux()
	apiServer.SetupRoutes(mux)

	// SPA handler
	spaHandler := web.SPAHandler(*dev, "frontend/dist", ui.DistFS)
	mux.Handle("/", spaHandler)

	// Start hub
	go hub.Run()

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: api.LoggingMiddleware(mux),
	}

	go func() {
		log.Printf("HTTP server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop all running instances
	manager.StopAll()
	tracker.UnregisterAll()

	// Close database
	db.Close()

	// Shutdown HTTP server
	server.Shutdown(shutdownCtx)

	log.Println("Shutdown complete")
}