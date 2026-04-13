// Package config loads application configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config stores runtime settings for the backend server.
type Config struct {
	// Server
	Host            string
	Port            string
	DownloadsDir    string
	CookieFile      string
	CORSOrigins     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Auth
	JWTSecret          string
	JWTAccessDuration  time.Duration
	JWTRefreshDuration time.Duration
	FrontendURL        string

	// Linuxdo OAuth
	LinuxdoClientID     string
	LinuxdoClientSecret string
	LinuxdoRedirectURI  string

	// Static file serving (set to web/dist path in production)
	StaticDir string

	// SMTP (optional)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
}

// FromEnv loads configuration from environment variables with sensible defaults.
func FromEnv() Config {
	return Config{
		Host:               getEnv("HOST", "0.0.0.0"),
		Port:               getEnv("PORT", "5000"),
		DownloadsDir:       getEnv("DOWNLOADS_DIR", "downloads"),
		CookieFile:         getEnv("COOKIE_FILE", "cookie.txt"),
		CORSOrigins:        getEnv("CORS_ORIGINS", "*"),
		ReadTimeout:        time.Duration(getEnvInt("READ_TIMEOUT_SEC", 20)) * time.Second,
		WriteTimeout:       time.Duration(getEnvInt("WRITE_TIMEOUT_SEC", 60)) * time.Second,
		ShutdownTimeout:    time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SEC", 8)) * time.Second,
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnvInt("DB_PORT", 5432),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", "melovault"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		JWTAccessDuration:  time.Duration(getEnvInt("JWT_ACCESS_DURATION_MIN", 15)) * time.Minute,
		JWTRefreshDuration: time.Duration(getEnvInt("JWT_REFRESH_DURATION_DAY", 7)) * 24 * time.Hour,
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		LinuxdoClientID:     getEnv("LINUXDO_CLIENT_ID", ""),
		LinuxdoClientSecret: getEnv("LINUXDO_CLIENT_SECRET", ""),
		LinuxdoRedirectURI:  getEnv("LINUXDO_REDIRECT_URI", ""),
		StaticDir:          getEnv("STATIC_DIR", ""),
		SMTPHost:           getEnv("SMTP_HOST", ""),
		SMTPPort:           getEnvInt("SMTP_PORT", 587),
		SMTPUser:           getEnv("SMTP_USER", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
	}
}

// Addr returns host:port for HTTP server binding.
func (c Config) Addr() string {
	return c.Host + ":" + c.Port
}

// DBConfig returns database configuration.
func (c Config) DBConfig() struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
} {
	return struct {
		Host     string
		Port     int
		User     string
		Password string
		Database string
		SSLMode  string
	}{
		Host:     c.DBHost,
		Port:     c.DBPort,
		User:     c.DBUser,
		Password: c.DBPassword,
		Database: c.DBName,
		SSLMode:  c.DBSSLMode,
	}
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
