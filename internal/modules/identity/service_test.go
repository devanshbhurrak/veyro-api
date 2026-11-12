package identity

import (
	"errors"
	"testing"
	"time"

	"backend/internal/platform/audit"
	"backend/internal/platform/auth"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/outbox"
	"backend/internal/platform/throttle"
	"backend/internal/shared/apperr"
	"backend/internal/shared/validate"
)

func TestRegisterValidation(t *testing.T) {
	svc := &Service{
		authCfg: config.AuthConfig{
			LoginWindow:        time.Minute,
			LoginMaxAttempts:   10,
		},
		registerLimiter: throttle.New(time.Minute, 10),
	}

	err := svc.Register(nil, "", "user@example.com", "Password123", RequestMeta{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}

	err = svc.Register(nil, "Test User", "not-an-email", "Password123", RequestMeta{})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}

	err = svc.Register(nil, "Test User", "user@example.com", "short", RequestMeta{})
	if err == nil {
		t.Fatal("expected error for weak password")
	}
}

func TestLoginValidation(t *testing.T) {
	svc := &Service{
		authCfg: config.AuthConfig{
			LoginWindow:        time.Minute,
			LoginMaxAttempts:   10,
		},
		loginLimiter:  throttle.New(time.Minute, 10),
		dummyHash:     "dummy",
		repo:          &Repository{},
		db:            &database.Database{},
	}

	_, err := svc.Login(nil, "", "Password123", RequestMeta{})
	if err == nil {
		t.Fatal("expected error for empty email")
	}

	_, err = svc.Login(nil, "not-an-email", "Password123", RequestMeta{})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestThrottleIntegration(t *testing.T) {
	limiter := throttle.New(time.Minute, 2)
	if !limiter.Allow("ip:1") || !limiter.Allow("ip:1") {
		t.Fatal("first two attempts should be allowed")
	}
	if limiter.Allow("ip:1") {
		t.Fatal("third attempt should be blocked")
	}
	limiter.Reset("ip:1")
	if !limiter.Allow("ip:1") {
		t.Fatal("after reset, attempt should be allowed")
	}
}

func TestValidatePasswordDelegation(t *testing.T) {
	valid := []string{"Password123", "abcdefgh12", "Veyro2026!"}
	for _, pw := range valid {
		if err := validatePassword(pw); err != nil {
			t.Errorf("expected %q to be valid: %v", pw, err)
		}
	}

	invalid := []string{"short", "onlyletters", "1234567890"}
	for _, pw := range invalid {
		if err := validatePassword(pw); err == nil {
			t.Errorf("expected %q to be rejected", pw)
		}
	}
}

func validatePassword(password string) error {
	return auth.ValidatePassword(password)
}

func TestEmailValidation(t *testing.T) {
	valid := []string{"User@Example.COM", "a.b+tag@sub.domain.in"}
	for _, input := range valid {
		_, ok := validate.Email(input)
		if !ok {
			t.Errorf("expected %q to be valid email", input)
		}
	}

	invalid := []string{"", "not-an-email", "a@b", "@example.com"}
	for _, input := range invalid {
		if _, ok := validate.Email(input); ok {
			t.Errorf("expected %q to be invalid email", input)
		}
	}
}

func TestApperrClassification(t *testing.T) {
	err := apperr.BadRequest(apperr.CodeValidation, "test")
	if err.Status != 400 || err.Code != apperr.CodeValidation {
		t.Fatal("bad request error classification incorrect")
	}

	err = apperr.Unauthorized(apperr.CodeUnauthenticated, "test")
	if err.Status != 401 {
		t.Fatal("unauthorized error status incorrect")
	}

	err = apperr.Internal(errors.New("boom"))
	if err.Status != 500 || errors.Unwrap(err) == nil {
		t.Fatal("internal error classification incorrect")
	}
}

func TestAuditRecorderSignature(t *testing.T) {
	rec := audit.NewRecorder()
	if rec == nil {
		t.Fatal("expected non-nil audit recorder")
	}
	_ = rec
}

func TestOutboxRepository(t *testing.T) {
	repo := outbox.NewRepository()
	if repo == nil {
		t.Fatal("expected non-nil outbox repository")
	}
	_ = repo
}

func TestDatabaseConfigFields(t *testing.T) {
	cfg := database.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Database:        "veyro",
		Username:        "postgres",
		Password:        "change-me",
		SSLMode:         "disable",
		PoolSize:        10,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
		ConnectTimeout:  5 * time.Second,
	}
	if cfg.Host != "localhost" || cfg.Port != 5432 {
		t.Fatal("database config fields not set correctly")
	}
}

func TestAuthConfigStructure(t *testing.T) {
	cfg := config.AuthConfig{
		AccessTokenTTL:      15 * time.Minute,
		RefreshTokenTTL:     24 * time.Hour,
		EmailTokenTTL:       24 * time.Hour,
		PasswordResetTTL:    30 * time.Minute,
		InvitationTTL:       7 * 24 * time.Hour,
		TokenEncryptionKey:  "test-key",
	}
	if cfg.AccessTokenTTL >= cfg.RefreshTokenTTL {
		t.Fatal("access token TTL must be less than refresh token TTL")
	}
}

func TestPasswordHasherRoundTrip(t *testing.T) {
	hasher := auth.NewPasswordHasher(auth.PasswordParams{
		Memory:      8192,
		Iterations:  1,
		Parallelism: 1,
	})
	hash, err := hasher.Hash("correct horse 42")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !hasher.Verify("correct horse 42", hash) {
		t.Fatal("expected password to verify")
	}
	if hasher.Verify("wrong password", hash) {
		t.Fatal("wrong password must not verify")
	}
}

func TestValidateEmailNormalization(t *testing.T) {
	got, ok := validate.Email("  User@Example.COM ")
	if !ok || got != "user@example.com" {
		t.Errorf("expected normalized email, got %q, %v", got, ok)
	}
}