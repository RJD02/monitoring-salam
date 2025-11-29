package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Mode        string            `yaml:"mode"` // test or prod
	Server      ServerConfig      `yaml:"server"`
	Paths       PathsConfig       `yaml:"paths"`
	Services    ServicesConfig    `yaml:"services"`
	Informatica InformaticaConfig `yaml:"informatica"`
	Logging     LoggingConfig     `yaml:"logging"`
	Database    DatabaseConfig    `yaml:"database"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// PathsConfig holds path configuration for different modes
type PathsConfig struct {
	NFSRoot     string `yaml:"nfs_root"`
	NFSRootTest string `yaml:"nfs_root_test"`
	NFSRootProd string `yaml:"nfs_root_prod"`
	LogDir      string `yaml:"log_dir"`
}

// ServicesConfig holds external service configurations
type ServicesConfig struct {
	YarnRMURL     string            `yaml:"yarn_rm_url"`
	YarnRMURLTest string            `yaml:"yarn_rm_url_test"`
	InformaticaDB InformaticaConfig `yaml:"informatica_db"`
}

// InformaticaConfig holds Informatica database configuration
type InformaticaConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Database   string `yaml:"database"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	TimeOffset int    `yaml:"time_offset"` // hours offset for timezone conversion
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level"`
	FilePath string `yaml:"file_path"`
	FileLog  bool   `yaml:"file_log"`
	JSONLog  bool   `yaml:"json_log"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	SQLitePath string `yaml:"sqlite_path"`
}

// GetNFSRoot returns the appropriate NFS root path based on mode
func (c *Config) GetNFSRoot() string {
	// If direct nfs_root is set, use it
	if c.Paths.NFSRoot != "" {
		return c.Paths.NFSRoot
	}
	// Fall back to mode-specific paths
	if c.Mode == "test" && c.Paths.NFSRootTest != "" {
		return c.Paths.NFSRootTest
	}
	if c.Mode == "prod" && c.Paths.NFSRootProd != "" {
		return c.Paths.NFSRootProd
	}
	// Default fallback
	if c.Mode == "test" {
		return "./nfs_backup/monitoring"
	}
	return "/home/informaticaadmin/nfs_backup/monitoring"
}

// GetYarnURL returns the appropriate Yarn URL based on mode
func (c *Config) GetYarnURL() string {
	if c.Mode == "test" {
		return c.Services.YarnRMURLTest
	}
	return c.Services.YarnRMURL
}

// IsProdMode returns true if running in production mode
func (c *Config) IsProdMode() bool {
	return c.Mode == "prod"
}

// IsTestMode returns true if running in test mode
func (c *Config) IsTestMode() bool {
	return c.Mode == "test"
}

// LoadFromEnv creates configuration entirely from environment variables
func LoadFromEnv() *Config {
	// Parse port with default
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// Parse Informatica DB port
	infDBPort := 1433
	if portStr := os.Getenv("INFORMATICA_DB_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			infDBPort = p
		}
	}

	// Parse Informatica time offset
	timeOffset := 3
	if offsetStr := os.Getenv("INFORMATICA_TIME_OFFSET"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			timeOffset = o
		}
	}

	// Parse boolean values
	fileLog := GetEnvWithDefault("LOG_FILE_ENABLED", "true") == "true"
	jsonLog := GetEnvWithDefault("LOG_JSON_ENABLED", "false") == "true"

	return &Config{
		Mode: GetEnvWithDefault("ENV", "test"),
		Server: ServerConfig{
			Port: port,
			Host: GetEnvWithDefault("HOST", "0.0.0.0"),
		},
		Paths: PathsConfig{
			NFSRoot:     GetEnvWithDefault("NFS_ROOT", ""),
			NFSRootTest: GetEnvWithDefault("NFS_ROOT_TEST", "./nfs_backup/monitoring"),
			NFSRootProd: GetEnvWithDefault("NFS_ROOT_PROD", "/home/informaticaadmin/nfs_backup/monitoring"),
			LogDir:      GetEnvWithDefault("LOG_DIR", "./logs"),
		},
		Services: ServicesConfig{
			YarnRMURL:     GetEnvWithDefault("YARN_RM_URL", "http://rm-host:8088"),
			YarnRMURLTest: GetEnvWithDefault("YARN_RM_URL_TEST", "./mock/yarn/apps.json"),
			InformaticaDB: InformaticaConfig{
				Host:       GetEnvWithDefault("INFORMATICA_DB_HOST", "localhost"),
				Port:       infDBPort,
				Database:   GetEnvWithDefault("INFORMATICA_DB_NAME", "INFORMATICA"),
				Username:   GetEnvWithDefault("INFORMATICA_DB_USER", "repo_read"),
				Password:   GetEnvWithDefault("INFORMATICA_DB_PASS", "password"),
				TimeOffset: timeOffset,
			},
		},
		Logging: LoggingConfig{
			Level:    GetEnvWithDefault("LOG_LEVEL", "info"),
			FilePath: GetEnvWithDefault("LOG_FILE_PATH", "./logs"),
			FileLog:  fileLog,
			JSONLog:  jsonLog,
		},
		Database: DatabaseConfig{
			SQLitePath: GetEnvWithDefault("SQLITE_PATH", "data/history.db"),
		},
	}
}

// LoadConfig loads configuration from file with environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	// If configPath is provided and it's a .env file, load it first and use env-based config
	if configPath != "" && strings.HasSuffix(strings.ToLower(configPath), ".env") {
		if err := LoadEnvFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
		// Create config from environment variables
		return LoadFromEnv(), nil
	}

	// Load default .env file if it exists
	LoadEnvFile(".env")

	// Set default configuration
	config := &Config{
		Mode: GetEnvWithDefault("ENV", "test"),
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Paths: PathsConfig{
			NFSRoot:     "./nfs_backup/monitoring",
			NFSRootTest: "./nfs_backup/monitoring",
			NFSRootProd: "/home/informaticaadmin/nfs_backup/monitoring",
			LogDir:      "./logs",
		},
		Services: ServicesConfig{
			YarnRMURL:     "http://rm-host:8088",
			YarnRMURLTest: "./mock/yarn/apps.json",
			InformaticaDB: InformaticaConfig{
				Host:       "172.16.1.100",
				Port:       1433,
				Database:   "INFORMATICA_PROD",
				Username:   "repo_read",
				Password:   "password",
				TimeOffset: 3,
			},
		},
		Logging: LoggingConfig{
			Level:    "info",
			FilePath: "./logs",
			FileLog:  true,
			JSONLog:  false,
		},
		Database: DatabaseConfig{
			SQLitePath: "data/history.db",
		},
	}

	// Determine config file to load
	var configFiles []string
	if configPath != "" {
		configFiles = []string{configPath}
	} else {
		// Default config file locations based on mode
		mode := GetEnvWithDefault("ENV", "test")
		if mode == "prod" || mode == "production" {
			configFiles = []string{
				"prod-config.yaml",
				"./config/prod-config.yaml",
				"config/prod-config.yaml",
				"./prod-config.yaml",
			}
		} else {
			configFiles = []string{
				"config.yaml",
				"./config/config.yaml",
				"config/config.yaml",
				"./config.yaml",
			}
		}
	}

	// Try to load config from files
	configLoaded := false
	for _, file := range configFiles {
		if fileExists(file) {
			if err := loadConfigFile(config, file); err == nil {
				configLoaded = true
				break
			}
		}
	}

	if !configLoaded {
		fmt.Printf("Warning: No config file found, using defaults\n")
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	// Log final configuration (without sensitive data)
	fmt.Printf("Final configuration:\n")
	fmt.Printf("  Mode: %s\n", config.Mode)
	fmt.Printf("  Yarn RM URL: %s\n", config.Services.YarnRMURL)
	fmt.Printf("  NFS Root: %s\n", config.GetNFSRoot())

	return config, nil
}

// loadConfigFile loads configuration from a specific file
func loadConfigFile(config *Config, filename string) error {
	fmt.Printf("Loading config from: %s\n", filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	fmt.Printf("Successfully loaded config from: %s\n", filename)
	fmt.Printf("  Loaded Yarn URL: %s\n", config.Services.YarnRMURL)
	return nil
}

// applyEnvOverrides applies environment variable overrides to configuration
func applyEnvOverrides(config *Config) {
	// Mode override
	if env := os.Getenv("ENV"); env != "" {
		config.Mode = env
	}

	// Server overrides
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}

	// Path overrides
	if nfsTest := os.Getenv("NFS_ROOT_TEST"); nfsTest != "" {
		config.Paths.NFSRootTest = nfsTest
	}

	if nfsProd := os.Getenv("NFS_ROOT_PROD"); nfsProd != "" {
		config.Paths.NFSRootProd = nfsProd
	}

	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		config.Paths.LogDir = logDir
	}

	// Service overrides
	if yarnURL := os.Getenv("YARN_RM_URL"); yarnURL != "" {
		config.Services.YarnRMURL = yarnURL
	}

	if yarnTestURL := os.Getenv("YARN_RM_URL_TEST"); yarnTestURL != "" {
		config.Services.YarnRMURLTest = yarnTestURL
	}

	// Informatica DB overrides
	if dbHost := os.Getenv("INF_DB_HOST"); dbHost != "" {
		config.Services.InformaticaDB.Host = dbHost
	}

	if dbPort := os.Getenv("INF_DB_PORT"); dbPort != "" {
		if p, err := strconv.Atoi(dbPort); err == nil {
			config.Services.InformaticaDB.Port = p
		}
	}

	if dbService := os.Getenv("INFORMATICA_DB_NAME"); dbService != "" {
		config.Services.InformaticaDB.Database = dbService
	}

	if dbUser := os.Getenv("INF_DB_USER"); dbUser != "" {
		config.Services.InformaticaDB.Username = dbUser
	}

	if dbPass := os.Getenv("INF_DB_PASSWORD"); dbPass != "" {
		config.Services.InformaticaDB.Password = dbPass
	}

	if offset := os.Getenv("TIME_OFFSET_HOURS"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			config.Services.InformaticaDB.TimeOffset = o
		}
	}

	// Logging overrides
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}

	if fileLog := os.Getenv("LOG_FILE"); fileLog != "" {
		config.Logging.FileLog = fileLog == "true"
	}

	if jsonLog := os.Getenv("LOG_JSON"); jsonLog != "" {
		config.Logging.JSONLog = jsonLog == "true"
	}
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
