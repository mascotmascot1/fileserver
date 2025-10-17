package main

import (
	"io"
	"log"
	"os"

	"github.com/mascotmascot1/fileserver/internal/config"
	"github.com/mascotmascot1/fileserver/internal/server"
)

func main() {
	const configPath = "fileserver.yaml"

	// Open the log file for appending. The flags ensure the file is created if it
	// does not exist, and that new log entries are added to the end.
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v\n", err)
	}
	defer logFile.Close()

	// Create a MultiWriter to direct log output to both standard output (the console)
	// and the log file simultaneously.
	mw := io.MultiWriter(os.Stdout, logFile)

	// Initialise the application's logger to use the multi-writer. This instance will
	// be injected as a dependency into other parts of the application.
	logger := log.New(mw, "[FILE SERVER] ", log.LstdFlags)

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
		logger.Fatalf("error starting server: %s\n", err)
	}
}
