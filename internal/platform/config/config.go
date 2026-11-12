package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the complete application configuration, loaded from environment
// variables. Secrets are never logged.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	HTTP     HTTPConfig
	Auth     AuthConfig
	Email    EmailConfig
	Outbox   OutboxConfig
	Platform PlatformConfig
}

// AppConfig contains general application settings.
type AppConfig struct {
	Name        string
	Environment string
	Port        int
	LogLevel    string
	BaseURL     string
}

// DatabaseConfig contains PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	SSLMode         string
	PoolSize        int
	MinConns        int
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	HealthTimeout   time.Duration
}

// DSN returns a PostgreSQL connection string with escaped credentials. It
// must never be logged.
func (c DatabaseConfig) DSN() string {
	return (&url.URL{
		Scheme:   "postgresql",
		User:     url.UserPassword(c.Username, c.Password),
		Host:     net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:     "/" + c.Database,
		RawQuery: url.Values{"sslmode": []string{c.SSLMode}}.Encode(),
	}).String()
}

// HTTPConfig contains HTTP server settings.
type HTTPConfig struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

// AuthConfig contains authentication, token and password settings.
type AuthConfig struct {
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	EmailTokenTTL      time.Duration
	PasswordResetTTL   time.Duration
	InvitationTTL      time.Duration
	Argon2Memory       uint32
	Argon2Iterations   uint32
	Argon2Parallelism  uint8
	Argon2SaltLength   uint32
	Argon2KeyLength    uint32
	TokenEncryptionKey string

	RefreshCookieName string
	CookieDomain      string
	CookieSecure      bool
	CookieSameSite    string

	LoginWindow              time.Duration
	LoginMaxAttempts         int
	ResendWindow             time.Duration
	ResendMaxAttempts        int
	PasswordResetWindow      time.Duration
	PasswordResetMaxAttempts int
}

// EmailConfig contains transactional email delivery settings.
type EmailConfig struct {
	Provider     string
	FromAddress  string
	FromName     string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPUseTLS   bool
	LocalDir     string
}

// OutboxConfig controls asynchronous outbox processing.
type OutboxConfig struct {
	WorkerInterval time.Duration
	BatchSize      int
	MaxAttempts    int
	LockDuration   time.Duration
}

// PlatformConfig controls internal platform administration.
type PlatformConfig struct {
	AdminEmails []string
}

// Load reads configuration from the environment and applies safe defaults.
func Load() (*Config, error) {
	environment := getEnv("APP_ENV", "development")
	defaultOrigins := "http://localhost:3000"
	if environment == "production" {
		// Production must opt in explicitly; never default to a development origin.
		defaultOrigins = ""
	}
	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "Veyro"),
			Environment: environment,
			Port:        getEnvInt("APP_PORT", 8080),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			BaseURL:     getEnv("APP_BASE_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			Database:        getEnv("DB_NAME", "veyro"),
			Username:        getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			PoolSize:        getEnvInt("DB_POOL_SIZE", 10),
			MinConns:        getEnvInt("DB_MIN_CONNS", 0),
			MaxConnLifetime: time.Duration(getEnvInt("DB_MAX_CONN_LIFETIME_SECONDS", 3600)) * time.Second,
			ConnectTimeout:  time.Duration(getEnvInt("DB_CONNECT_TIMEOUT_SECONDS", 5)) * time.Second,
			HealthTimeout:   time.Duration(getEnvInt("DB_HEALTH_TIMEOUT_SECONDS", 5)) * time.Second,
		},
		HTTP: HTTPConfig{
			ReadTimeout:     time.Duration(getEnvInt("HTTP_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout:    time.Duration(getEnvInt("HTTP_WRITE_TIMEOUT", 30)) * time.Second,
			IdleTimeout:     time.Duration(getEnvInt("HTTP_IDLE_TIMEOUT", 60)) * time.Second,
			ShutdownTimeout: time.Duration(getEnvInt("HTTP_SHUTDOWN_TIMEOUT", 30)) * time.Second,
			AllowedOrigins:  splitEnvVar(getEnv("HTTP_ALLOWED_ORIGINS", defaultOrigins)),
		},
		Auth: AuthConfig{
			AccessTokenTTL:           time.Duration(getEnvInt("AUTH_ACCESS_TOKEN_TTL_MINUTES", 15)) * time.Minute,
			RefreshTokenTTL:          time.Duration(getEnvInt("AUTH_REFRESH_TOKEN_TTL_DAYS", 30)) * 24 * time.Hour,
			EmailTokenTTL:            time.Duration(getEnvInt("AUTH_EMAIL_TOKEN_TTL_HOURS", 24)) * time.Hour,
			PasswordResetTTL:         time.Duration(getEnvInt("AUTH_PASSWORD_RESET_TTL_MINUTES", 30)) * time.Minute,
			InvitationTTL:            time.Duration(getEnvInt("AUTH_INVITATION_TTL_DAYS", 7)) * 24 * time.Hour,
			Argon2Memory:             uint32(getEnvInt("AUTH_ARGON2_MEMORY_KIB", 65536)),
			Argon2Iterations:         uint32(getEnvInt("AUTH_ARGON2_ITERATIONS", 3)),
			Argon2Parallelism:        uint8(getEnvInt("AUTH_ARGON2_PARALLELISM", 2)),
			Argon2SaltLength:         uint32(getEnvInt("AUTH_ARGON2_SALT_LENGTH", 16)),
			Argon2KeyLength:          uint32(getEnvInt("AUTH_ARGON2_KEY_LENGTH", 32)),
			TokenEncryptionKey:       getEnv("AUTH_TOKEN_ENCRYPTION_KEY", ""),
			RefreshCookieName:        getEnv("AUTH_REFRESH_COOKIE_NAME", "veyro_refresh"),
			CookieDomain:             getEnv("AUTH_COOKIE_DOMAIN", ""),
			CookieSecure:             getEnvBool("AUTH_COOKIE_SECURE", false),
			CookieSameSite:           getEnv("AUTH_COOKIE_SAME_SITE", "lax"),
			LoginWindow:              time.Duration(getEnvInt("AUTH_LOGIN_WINDOW_SECONDS", 300)) * time.Second,
			LoginMaxAttempts:         getEnvInt("AUTH_LOGIN_MAX_ATTEMPTS", 10),
			ResendWindow:             time.Duration(getEnvInt("AUTH_RESEND_WINDOW_SECONDS", 3600)) * time.Second,
			ResendMaxAttempts:        getEnvInt("AUTH_RESEND_MAX_ATTEMPTS", 3),
			PasswordResetWindow:      time.Duration(getEnvInt("AUTH_PASSWORD_RESET_WINDOW_SECONDS", 3600)) * time.Second,
			PasswordResetMaxAttempts: getEnvInt("AUTH_PASSWORD_RESET_MAX_ATTEMPTS", 5),
		},
		Email: EmailConfig{
			Provider:     getEnv("EMAIL_PROVIDER", "log"),
			FromAddress:  getEnv("EMAIL_FROM_ADDRESS", "no-reply@veyro.local"),
			FromName:     getEnv("EMAIL_FROM_NAME", "Veyro"),
			SMTPHost:     getEnv("EMAIL_SMTP_HOST", "localhost"),
			SMTPPort:     getEnvInt("EMAIL_SMTP_PORT", 587),
			SMTPUsername: getEnv("EMAIL_SMTP_USERNAME", ""),
			SMTPPassword: getEnv("EMAIL_SMTP_PASSWORD", ""),
			SMTPUseTLS:   getEnvBool("EMAIL_SMTP_USE_TLS", true),
			LocalDir:     getEnv("EMAIL_LOCAL_DIR", ".tmp/emails"),
		},
		Outbox: OutboxConfig{
			WorkerInterval: time.Duration(getEnvInt("OUTBOX_WORKER_INTERVAL_SECONDS", 5)) * time.Second,
			BatchSize:      getEnvInt("OUTBOX_BATCH_SIZE", 20),
			MaxAttempts:    getEnvInt("OUTBOX_MAX_ATTEMPTS", 8),
			LockDuration:   time.Duration(getEnvInt("OUTBOX_LOCK_SECONDS", 60)) * time.Second,
		},
		Platform: PlatformConfig{
			AdminEmails: splitEnvVar(getEnv("PLATFORM_ADMIN_EMAILS", "")),
		},
	}

	if cfg.Auth.TokenEncryptionKey == "" {
		if cfg.App.Environment == "production" {
			return nil, fmt.Errorf("AUTH_TOKEN_ENCRYPTION_KEY is required in production")
		}
		// Development-only deterministic fallback. Never used in production.
		cfg.Auth.TokenEncryptionKey = "veyro-development-only-token-encryption-key"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks required configuration and safe ranges.
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("APP_NAME is required")
	}
	if c.App.Port <= 0 {
		return fmt.Errorf("APP_PORT must be positive")
	}
	if c.Database.Host == "" || c.Database.Port <= 0 || c.Database.Database == "" || c.Database.Username == "" {
		return fmt.Errorf("database host, port, name and user are required")
	}
	if c.Database.PoolSize <= 0 {
		return fmt.Errorf("DB_POOL_SIZE must be positive")
	}
	if c.Auth.AccessTokenTTL <= 0 || c.Auth.RefreshTokenTTL <= 0 {
		return fmt.Errorf("token lifetimes must be positive")
	}
	if c.Auth.Argon2Memory < 8192 || c.Auth.Argon2Iterations < 1 || c.Auth.Argon2Parallelism < 1 {
		return fmt.Errorf("argon2 parameters are too weak")
	}
	if c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		return fmt.Errorf("refresh token lifetime must exceed access token lifetime")
	}
	if c.Auth.EmailTokenTTL <= 0 || c.Auth.PasswordResetTTL <= 0 || c.Auth.InvitationTTL <= 0 {
		return fmt.Errorf("verification, reset and invitation token lifetimes must be positive")
	}
	switch c.Auth.CookieSameSite {
	case "lax", "strict", "none":
	default:
		return fmt.Errorf("AUTH_COOKIE_SAME_SITE must be lax, strict or none")
	}
	if c.Auth.CookieSameSite == "none" && !c.Auth.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true when AUTH_COOKIE_SAME_SITE is none")
	}
	if c.App.Environment == "production" && len(c.HTTP.AllowedOrigins) == 0 {
		return fmt.Errorf("HTTP_ALLOWED_ORIGINS must be set in production")
	}
	if c.App.Environment == "production" && !c.Auth.CookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true in production")
	}
	if c.Outbox.BatchSize <= 0 || c.Outbox.MaxAttempts <= 0 || c.Outbox.WorkerInterval <= 0 {
		return fmt.Errorf("outbox batch size, max attempts and worker interval must be positive")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func splitEnvVar(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
