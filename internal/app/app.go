// Package app wires platform infrastructure and business modules into the API
// and worker entry points. Explicit constructors keep dependencies visible.
package app

import (
	"fmt"
	"net/http"

	"backend/internal/modules/authorization"
	"backend/internal/modules/identity"
	"backend/internal/modules/invitations"
	"backend/internal/modules/organization"
	"backend/internal/modules/platformadmin"
	"backend/internal/modules/tenancy"
	"backend/internal/platform/audit"
	"backend/internal/platform/auth"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/email"
	"backend/internal/platform/logger"
	"backend/internal/platform/outbox"
	"backend/internal/server"
)

// App holds constructed dependencies shared by the API and worker.
type App struct {
	Config    *config.Config
	Log       *logger.Logger
	DB        *database.Database
	Encryptor *auth.Encryptor
	Outbox    *outbox.Processor
	Handler   http.Handler
}

// New builds the application graph.
func New(cfg *config.Config) (*App, error) {
	log := logger.New(cfg.App.LogLevel, cfg.App.Environment)

	db, err := database.NewDatabase(database.DatabaseConfig{
		Host: cfg.Database.Host, Port: cfg.Database.Port, Database: cfg.Database.Database,
		Username: cfg.Database.Username, Password: cfg.Database.Password,
		SSLMode: cfg.Database.SSLMode, PoolSize: cfg.Database.PoolSize,
		MinConns: cfg.Database.MinConns, MaxConnLifetime: cfg.Database.MaxConnLifetime,
		ConnectTimeout: cfg.Database.ConnectTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	hasher := auth.NewPasswordHasher(auth.PasswordParams{
		Memory: cfg.Auth.Argon2Memory, Iterations: cfg.Auth.Argon2Iterations,
		Parallelism: cfg.Auth.Argon2Parallelism, SaltLength: cfg.Auth.Argon2SaltLength,
		KeyLength: cfg.Auth.Argon2KeyLength,
	})
	encryptor, err := auth.NewEncryptor(cfg.Auth.TokenEncryptionKey)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize token encryption: %w", err)
	}
	sender, err := newEmailSender(cfg.Email)
	if err != nil {
		db.Close()
		return nil, err
	}

	auditRecorder := audit.NewRecorder()
	outboxRepo := outbox.NewRepository()

	identitySvc, err := identity.NewService(db, identity.NewRepository(), hasher, encryptor,
		cfg.Auth, cfg.App, outboxRepo, auditRecorder)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize identity service: %w", err)
	}
	authzRepo := authorization.NewRepository()
	authzSvc := authorization.NewService(db, authzRepo, auditRecorder)
	tenancySvc := tenancy.NewService(db, tenancy.NewRepository(), authzSvc, auditRecorder)
	orgSvc := organization.NewService(db, organization.NewRepository(),
		organization.NewInitializer(), auditRecorder)
	invitationSvc := invitations.NewService(db, invitations.NewRepository(), authzRepo,
		tenancy.NewRepository(), outboxRepo, encryptor, auditRecorder, cfg.App.BaseURL, cfg.Auth.InvitationTTL)
	platformSvc := platformadmin.NewService(db, auditRecorder)

	authzMiddleware := authorization.NewMiddleware(db.Pool, authzRepo, identitySvc, cfg.Platform.AdminEmails)

	handler := server.NewRouter(server.Dependencies{
		Config: cfg, Logger: log, DB: db, Authz: authzMiddleware,
		Identity:      identity.NewHandler(identitySvc, cfg.Auth),
		Authorization: authorization.NewHandler(authzSvc),
		Tenancy:       tenancy.NewHandler(tenancySvc),
		Organization:  organization.NewHandler(orgSvc, tenancySvc, identitySvc),
		Invitations:   invitations.NewHandler(invitationSvc),
		Platform:      platformadmin.NewHandler(platformSvc),
	})

	processor := outbox.NewProcessor(db, outboxRepo, sender, encryptor, log, cfg.App.BaseURL,
		cfg.Outbox.BatchSize, cfg.Outbox.MaxAttempts, cfg.Outbox.LockDuration)

	return &App{Config: cfg, Log: log, DB: db, Encryptor: encryptor,
		Outbox: processor, Handler: handler}, nil
}

func newEmailSender(cfg config.EmailConfig) (email.Sender, error) {
	switch cfg.Provider {
	case "smtp":
		return &email.SMTPSender{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword, UseTLS: cfg.SMTPUseTLS, From: cfg.FromAddress,
			FromName: cfg.FromName,
		}, nil
	default:
		return email.NewLogSender(cfg.LocalDir)
	}
}

// Close releases shared resources.
func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}
