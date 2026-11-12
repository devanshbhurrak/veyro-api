package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_TOKEN_ENCRYPTION_KEY", "test-key-for-configuration")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.App.Name != "Veyro" {
		t.Fatalf("expected default app name Veyro, got %q", cfg.App.Name)
	}
	if cfg.App.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.App.Port)
	}
	if cfg.Database.Host != "localhost" || cfg.Database.Port != 5432 {
		t.Fatalf("unexpected database defaults: %s:%d", cfg.Database.Host, cfg.Database.Port)
	}
	if cfg.Database.Database != "veyro" || cfg.Database.Username != "postgres" {
		t.Fatalf("unexpected database name/user defaults")
	}
	if cfg.Auth.AccessTokenTTL >= cfg.Auth.RefreshTokenTTL {
		t.Fatalf("access token lifetime must be shorter than refresh lifetime")
	}
	if len(cfg.HTTP.AllowedOrigins) == 0 || cfg.HTTP.AllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("unexpected allowed origins: %v", cfg.HTTP.AllowedOrigins)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default configuration should validate: %v", err)
	}
}

func TestLoadProductionRequiresEncryptionKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_TOKEN_ENCRYPTION_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("production configuration without AUTH_TOKEN_ENCRYPTION_KEY must fail")
	}
}

func TestProductionRequiresExplicitOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_TOKEN_ENCRYPTION_KEY", "test-key-for-configuration")
	t.Setenv("HTTP_ALLOWED_ORIGINS", "")
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	if _, err := Load(); err == nil {
		t.Fatal("production configuration without explicit origins must fail")
	}
}

func TestProductionRequiresSecureCookies(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_TOKEN_ENCRYPTION_KEY", "test-key-for-configuration")
	t.Setenv("HTTP_ALLOWED_ORIGINS", "https://app.veyro.test")
	t.Setenv("AUTH_COOKIE_SECURE", "false")
	if _, err := Load(); err == nil {
		t.Fatal("production configuration without secure cookies must fail")
	}
}

func TestValidateRejectsWeakArgon2Parameters(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Auth.Argon2Memory = 1024
	if err := cfg.Validate(); err == nil {
		t.Fatal("weak argon2 parameters must be rejected")
	}
}

func TestEnvHelpers(t *testing.T) {
	t.Setenv("TEST_KEY", "test-value")
	if getEnv("TEST_KEY", "default") != "test-value" {
		t.Fatal("getEnv did not read the environment")
	}
	if getEnv("TEST_MISSING_KEY", "default") != "default" {
		t.Fatal("getEnv did not return the default")
	}

	t.Setenv("TEST_INT_KEY", "100")
	if getEnvInt("TEST_INT_KEY", 42) != 100 {
		t.Fatal("getEnvInt did not parse the value")
	}
	if getEnvInt("TEST_MISSING_INT", 42) != 42 {
		t.Fatal("getEnvInt did not return the default")
	}

	t.Setenv("TEST_BOOL_KEY", "false")
	if getEnvBool("TEST_BOOL_KEY", true) != false {
		t.Fatal("getEnvBool did not parse the value")
	}

	parts := splitEnvVar("http://a.test, http://b.test")
	if len(parts) != 2 || parts[0] != "http://a.test" || parts[1] != "http://b.test" {
		t.Fatalf("unexpected split result: %v", parts)
	}
	if len(splitEnvVar("")) != 0 {
		t.Fatal("empty value must produce no entries")
	}
}
