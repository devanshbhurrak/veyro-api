package validate

import "testing"

func TestEmail(t *testing.T) {
	valid := map[string]string{
		"  User@Example.COM ":   "user@example.com",
		"a.b+tag@sub.domain.in": "a.b+tag@sub.domain.in",
	}
	for input, want := range valid {
		got, ok := Email(input)
		if !ok || got != want {
			t.Errorf("Email(%q) = %q, %v; want %q", input, got, ok, want)
		}
	}
	for _, input := range []string{"", "not-an-email", "a@b", "@example.com", "a @example.com"} {
		if _, ok := Email(input); ok {
			t.Errorf("Email(%q) should be invalid", input)
		}
	}
}

func TestGSTINAndPAN(t *testing.T) {
	if !GSTIN("27AAPFU0939F1ZV") {
		t.Fatal("expected valid GSTIN")
	}
	if GSTIN("27AAPFU0939F1Z") || GSTIN("") {
		t.Fatal("malformed GSTIN must be rejected")
	}
	if !PAN("AAPFU0939F") {
		t.Fatal("expected valid PAN")
	}
	if PAN("AAPFU0939") {
		t.Fatal("malformed PAN must be rejected")
	}
}

func TestStateAndPhone(t *testing.T) {
	for _, code := range []string{"01", "27", "38"} {
		if !StateCode(code) {
			t.Errorf("expected state code %s to be valid", code)
		}
	}
	for _, code := range []string{"", "1", "123", "AB"} {
		if StateCode(code) {
			t.Errorf("expected state code %q to be invalid", code)
		}
	}
	if !PhoneE164("+919876543210") {
		t.Fatal("expected valid E.164 phone")
	}
	for _, phone := range []string{"9876543210", "+91 98765 43210", "+0123456789"} {
		if PhoneE164(phone) {
			t.Errorf("expected phone %q to be invalid", phone)
		}
	}
}

func TestSlugifyAndSlug(t *testing.T) {
	if got := Slugify("  Sharma & Sons Traders "); got != "sharma-sons-traders" {
		t.Fatalf("Slugify returned %q", got)
	}
	if got := Slugify("!!!"); got != "workspace" {
		t.Fatalf("expected fallback slug, got %q", got)
	}
	if !Slug("sharma-sons") || Slug("-lead") || Slug("UPPER") {
		t.Fatal("slug validation is incorrect")
	}
}
