// Package notification defines durable email payloads shared by business
// modules (which enqueue) and the worker (which renders and delivers).
package notification

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Kinds of transactional email.
const (
	KindEmailVerification = "email_verification"
	KindPasswordReset     = "password_reset"
	KindInvitation        = "invitation"
)

// Payload is stored in the outbox. TokenEnc is an AES-GCM encrypted raw token;
// the raw token is never persisted.
type Payload struct {
	Kind          string `json:"kind"`
	To            string `json:"to"`
	RecipientName string `json:"recipient_name,omitempty"`
	TokenEnc      string `json:"token_enc"`
	LinkPath      string `json:"link_path"`
	TenantName    string `json:"tenant_name,omitempty"`
	InviterName   string `json:"inviter_name,omitempty"`
	Message       string `json:"message,omitempty"`
}

// Encode serializes a payload for the outbox.
func Encode(p Payload) (json.RawMessage, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("encode notification payload: %w", err)
	}
	return b, nil
}

// Decode parses a stored payload.
func Decode(raw []byte) (Payload, error) {
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("decode notification payload: %w", err)
	}
	return p, nil
}

// Render returns the subject and plaintext body. It never logs the token.
func Render(p Payload, baseURL, token string) (subject, body string) {
	link := strings.TrimRight(baseURL, "/") + p.LinkPath + "?token=" + url.QueryEscape(token)
	switch p.Kind {
	case KindEmailVerification:
		return "Verify your Veyro email address",
			fmt.Sprintf("Hello %s,\n\nPlease verify your Veyro email address by opening this link:\n\n%s\n\nIf you did not create a Veyro account, you can ignore this message.\n", p.RecipientName, link)
	case KindPasswordReset:
		return "Reset your Veyro password",
			fmt.Sprintf("Hello %s,\n\nReset your Veyro password using this link:\n\n%s\n\nThis link expires shortly and can be used once. If you did not request a reset, ignore this message.\n", p.RecipientName, link)
	case KindInvitation:
		return fmt.Sprintf("You are invited to join %s on Veyro", p.TenantName),
			fmt.Sprintf("Hello,\n\n%s has invited you to join the workspace %q on Veyro.\n\nAccept the invitation here:\n\n%s\n\n%s\n", p.InviterName, p.TenantName, link, p.Message)
	default:
		return "Veyro notification", link
	}
}
