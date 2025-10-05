package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Noooste/fhttp"

	"github.com/Noooste/azuretls-api/common"
	"github.com/Noooste/azuretls-api/rest"
)

type Server struct {
	config         common.ServerConfig
	sessionManager common.SessionManager
	httpServer     *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewServer(config common.ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	// Set log level from config
	common.SetLogLevel(config.LogLevel)

	sessionManager := NewSessionManager()

	server := &Server{
		config:         config,
		sessionManager: sessionManager,
		ctx:            ctx,
		cancel:         cancel,
	}

	handler := rest.SetupRoutes(server)

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      handler,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return server
}

func (s *Server) Start() error {
	log.Printf("Starting server on %s:%d", s.config.Host, s.config.Port)

	go func() {
		<-s.ctx.Done()
		log.Println("Shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		err := s.sessionManager.CleanupSessions()
		if err != nil {
			return
		}
	}()

	if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	log.Println("Stopping server...")
	s.cancel()
}

func (s *Server) GetConfig() common.ServerConfig {
	return s.config
}

func (s *Server) GetSessionManager() common.SessionManager {
	return s.sessionManager
}
