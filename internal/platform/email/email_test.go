package email

import "testing"

func TestSanitizeHeaderRemovesCRLF(t *testing.T) {
	got := sanitizeHeader("Hello\r\nBcc: attacker@example.com")
	if got != "Hello  Bcc: attacker@example.com" {
		t.Fatalf("unexpected sanitized header: %q", got)
	}
	if sanitizeHeader("  Subject  ") != "Subject" {
		t.Fatal("header values should be trimmed")
	}
}
