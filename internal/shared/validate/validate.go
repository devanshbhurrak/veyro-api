// Package validate contains small, reusable input validation helpers.
package validate

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	slugPattern  = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,58}[a-z0-9])?$`)
	gstinPattern = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)
	panPattern   = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`)
	statePattern = regexp.MustCompile(`^[0-9]{2}$`)
	phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
)

// Email normalizes and validates an email address.
func Email(value string) (string, bool) {
	normalized := NormalizeEmail(value)
	if len(normalized) > 254 || !emailPattern.MatchString(normalized) {
		return normalized, false
	}
	return normalized, true
}

// NormalizeEmail trims and lowercases an email address consistently.
func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func UUID(value string) bool { return uuidPattern.MatchString(strings.TrimSpace(value)) }

func Slug(value string) bool { return slugPattern.MatchString(value) }

func GSTIN(value string) bool {
	return gstinPattern.MatchString(strings.ToUpper(strings.TrimSpace(value)))
}

func PAN(value string) bool { return panPattern.MatchString(strings.ToUpper(strings.TrimSpace(value))) }

func StateCode(value string) bool { return statePattern.MatchString(strings.TrimSpace(value)) }

func PhoneE164(value string) bool { return phonePattern.MatchString(strings.TrimSpace(value)) }

// Required trims value and checks it is non-empty and within max runes.
func Required(value string, max int) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trimmed, false
	}
	if utf8.RuneCountInString(trimmed) > max {
		return trimmed, false
	}
	return trimmed, true
}

// Slugify converts a display name into a candidate workspace slug.
func Slugify(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if len(result) > 60 {
		result = strings.Trim(result[:60], "-")
	}
	if result == "" {
		return "workspace"
	}
	return result
}
