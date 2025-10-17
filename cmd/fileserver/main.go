package main

import (
	"log"
	"os"

	"github.com/mascotmascot1/fileserver/internal/config"
	"github.com/mascotmascot1/fileserver/internal/server"
)

func main() {
	const configPath = "fileserver.yaml"

	// Load application configuration from the specified path.
	logger := log.New(os.Stdout, "[FILE SERVER] ", log.LstdFlags)

	// Load application configuration from the specified path.
	cfg, err := config.NewConfig(configPath, logger)
	if err != nil {
		logger.Fatalf("error loading config %s\n", err)
	}

	// Create and configure the new HTTP server.
	s := server.NewServer(cfg, logger)
	logger.Printf("starting server on %s\n", s.HTTP.Addr)

	// Start the server and block until it returns an error.
	if err := s.HTTP.ListenAndServe(); err != nil {
		logger.Fatalf("error starting server: %s", err)
	}
}
