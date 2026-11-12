package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/auth"
	"backend/internal/platform/database"
	"backend/internal/platform/email"
	"backend/internal/platform/logger"
	"backend/internal/platform/notification"
)

// Processor delivers queued notification events. Delivery is at-least-once;
// consumers must tolerate a duplicate email after a crash between send and mark.
type Processor struct {
	db        *database.Database
	repo      *Repository
	sender    email.Sender
	encryptor *auth.Encryptor
	log       *logger.Logger
	baseURL   string
	batch     int
	maxTries  int
	lockFor   time.Duration
}

func NewProcessor(db *database.Database, repo *Repository, sender email.Sender,
	encryptor *auth.Encryptor, log *logger.Logger, baseURL string,
	batch, maxTries int, lockFor time.Duration) *Processor {
	return &Processor{db: db, repo: repo, sender: sender, encryptor: encryptor, log: log,
		baseURL: baseURL, batch: batch, maxTries: maxTries, lockFor: lockFor}
}

// Run processes batches until the context is cancelled.
func (p *Processor) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	p.processOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processOnce(ctx)
		}
	}
}

// ProcessOnce processes a single batch and returns the number attempted.
func (p *Processor) ProcessOnce(ctx context.Context) (int, error) {
	return p.processOnceResult(ctx)
}

func (p *Processor) processOnce(ctx context.Context) {
	if _, err := p.processOnceResult(ctx); err != nil {
		p.log.Error("outbox batch failed", "error", err.Error())
	}
}

func (p *Processor) processOnceResult(ctx context.Context) (int, error) {
	processed := 0
	err := p.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		events, err := p.repo.Claim(ctx, tx, "worker", p.batch, p.lockFor)
		if err != nil {
			return err
		}
		for _, ev := range events {
			processed++
			if err := p.deliver(ctx, tx, ev); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return processed, fmt.Errorf("process outbox: %w", err)
	}
	return processed, nil
}

func (p *Processor) deliver(ctx context.Context, tx pgx.Tx, ev Event) error {
	payload, err := notification.Decode(ev.Payload)
	if err != nil {
		// Corrupt payloads are permanent failures.
		return p.repo.MarkFailed(ctx, tx, ev.ID, "invalid_payload", "notification payload could not be decoded", p.maxTries, p.maxTries)
	}
	token, err := p.encryptor.Decrypt(payload.TokenEnc)
	if err != nil {
		return p.repo.MarkFailed(ctx, tx, ev.ID, "decrypt_failed", "notification token could not be decrypted", p.maxTries, p.maxTries)
	}
	subject, body := notification.Render(payload, p.baseURL, token)
	sendErr := p.sender.Send(ctx, email.Message{To: payload.To, Subject: subject, Text: body})
	if sendErr != nil {
		code := "delivery_failed"
		// Claim already incremented attempt_count for this delivery.
		attempt := ev.AttemptCount + 1
		if errors.Is(sendErr, email.ErrPermanent) {
			code = "permanent_delivery_failure"
			attempt = p.maxTries
		}
		p.log.Warn("outbox delivery failed", "event_type", ev.EventType, "attempt", ev.AttemptCount)
		return p.repo.MarkFailed(ctx, tx, ev.ID, code, sendErr.Error(), attempt, p.maxTries)
	}
	if err := p.repo.MarkProcessed(ctx, tx, ev.ID); err != nil {
		return err
	}
	p.log.Info("outbox event delivered", "event_type", ev.EventType, "event_id", ev.ID)
	return nil
}
