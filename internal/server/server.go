package server

import (
	"log"
	"net/http"

	"github.com/mascotmascot1/fileserver/internal/config"
	"github.com/mascotmascot1/fileserver/internal/handlers"
)

// Server represents the application's HTTP server, encapsulating its
// configuration and logger.
type Server struct {
	HTTP   *http.Server
	Logger *log.Logger
}

// NewServer creates and returns a new Server instance.
//
// It sets up the HTTP router, registers request handlers with their dependencies,
// and configures server settings such as address and timeouts.
func NewServer(cfg *config.Config, logger *log.Logger) *Server {
	// Initialise the handlers with their required dependencies (config and logger).
	h := handlers.NewHandlers(cfg, logger)

	// Initialise the handlers with their required dependencies (config and logger).
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", h.UploadHandler)
	mux.HandleFunc("/download/", h.DownloadHandle)
	mux.HandleFunc("/download/list.txt", h.DownloadList)

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		ErrorLog:     logger,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &Server{
		HTTP:   srv,
		Logger: logger,
	}
}
