package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	SMTP        SMTPConfig
	BankingSync BankingSyncConfig
	User        UserConfig
}

// UserConfig holds user authentication configuration.
type UserConfig struct {
	Username string
	Password string
}

// SMTPConfig holds SMTP email configuration.
type SMTPConfig struct {
	Host     string
	Port     int
	From     string
	Username string
	Password string
	UseTLS   bool
	BaseURL  string // Base URL for password reset links
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port         string
	CORSOrigins  []string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
	Issuer        string
}

// BankingSyncConfig holds configuration for the banking sync runner.
type BankingSyncConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8081"),
			CORSOrigins:  getEnvSlice("CORS_ORIGINS", []string{"*"}),
			ReadTimeout:  getEnvDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable&search_path=fees"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "dev-secret-change-in-production"),
			AccessExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
			Issuer:        getEnv("JWT_ISSUER", "kita-fees"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnvInt("SMTP_PORT", 587),
			From:     getEnv("SMTP_FROM", ""),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			UseTLS:   getEnvBool("SMTP_USE_TLS", true),
			BaseURL:  getEnv("APP_BASE_URL", "http://localhost:5175"),
		},
		BankingSync: BankingSyncConfig{
			BaseURL: getEnv("BANKING_SYNC_URL", ""),
			Token:   getEnv("BANKING_SYNC_TOKEN", ""),
			Timeout: getEnvDuration("BANKING_SYNC_TIMEOUT", 30*time.Second),
		},
		User: UserConfig{
			Username: getEnv("USER_NAME", ""),
			Password: getEnv("USER_PASSWORD", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		var result []string
		current := ""
		for _, c := range value {
			if c == ',' {
				if current != "" {
					result = append(result, current)
				}
				current = ""
			} else {
				current += string(c)
			}
		}
		if current != "" {
			result = append(result, current)
		}
		return result
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
