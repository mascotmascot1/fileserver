package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds settings specific to the HTTP server.
type ServerConfig struct {
	Addr         string        `yaml:"address"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
}

// UploaderConfig holds settings related to the file uploading functionality.
// Size limits are specified in megabytes (MB) in the configuration file.
type UploaderConfig struct {
	StorageDir       string `yaml:"storageDir"`
	MaxUploadSizeMB  int64  `yaml:"maxUploadSizeMB"`
	MaxFormMemSizeMB int64  `yaml:"maxFormMemSizeMB"`
}

// Config is the root structure that encapsulates all application settings.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Uploader UploaderConfig `yaml:"uploader"`
}

// GetMaxUploadSize returns the maximum permitted upload size in bytes.
// It converts the megabyte value from the configuration into bytes.
func (uc *UploaderConfig) GetMaxUploadSize() int64 {
	return uc.MaxUploadSizeMB << 20
}

// GetMaxFormMemSize returns the maximum memory to use for multipart form parsing in bytes.
// It converts the megabyte value from the configuration into bytes.
func (uc *UploaderConfig) GetMaxFormMemSize() int64 {
	return uc.MaxFormMemSizeMB << 20
}

// NewConfig loads the application configuration from the specified YAML file path.
// If the file does not exist, it logs a warning and returns a default configuration.
// It returns an error for any other file access or parsing issues.
func NewConfig(path string, logger *log.Logger) (*Config, error) {
	// Initialise with default values, which will be used if the config file is not found.
	var cfg = Config{
		Server: ServerConfig{
			Addr:         ":8090",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		},
		Uploader: UploaderConfig{
			StorageDir:       "storage",
			MaxUploadSizeMB:  3072,
			MaxFormMemSizeMB: 32,
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("warn: config file '%s' not found, using default settings.\n", path)
			return &cfg, nil
		}
		// Any other error (e.g., permissions) is considered fatal.
		return nil, err
	}

	// If the file exists, unmarshal its content over the default configuration.
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
