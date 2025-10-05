package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Noooste/azuretls-api/internal/common"
	"github.com/Noooste/azuretls-api/internal/server"
)

func main() {
	var (
		host                  = flag.String("host", "localhost", "Server host address")
		port                  = flag.Int("port", 8080, "Server port")
		maxSessions           = flag.Int("max_sessions", 1000, "Maximum concurrent sessions")
		maxConcurrentRequests = flag.Int("max_concurrent_requests", 100, "Maximum concurrent requests per session")
		readTimeout           = flag.Int("read_timeout", 30, "Server read timeout (seconds)")
		writeTimeout          = flag.Int("write_timeout", 30, "Server write timeout (seconds)")
		logLevel              = flag.String("log_level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	config := common.ServerConfig{
		Host:                  *host,
		Port:                  *port,
		MaxSessions:           *maxSessions,
		MaxConcurrentRequests: *maxConcurrentRequests,
		ReadTimeout:           time.Duration(*readTimeout) * time.Second,
		WriteTimeout:          time.Duration(*writeTimeout) * time.Second,
		LogLevel:              *logLevel,
	}

	srv := server.NewServer(config)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		srv.Stop()
	}()

	log.Printf("Starting AzureTLS server on %s:%d", *host, *port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Server stopped gracefully")
}
