package organization

import (
	"testing"
	"time"
)

func validCompanyInput() CreateCompanyInput {
	return CreateCompanyInput{
		TradeName: "Sharma Traders", BusinessType: "proprietorship",
		AddressLine1: "12 Market Road", City: "Pune", StateCode: "27",
		PostalCode: "411001", CountryCode: "IN",
		GSTProfile: "registered", RegistrationType: "regular",
		GSTIN: "27AAPFU0939F1ZV", RegistrationState: "27",
	}
}

func TestValidateCreateCompanyInputAcceptsValidCompany(t *testing.T) {
	in, fields := ValidateCreateCompanyInput(validCompanyInput())
	if len(fields) != 0 {
		t.Fatalf("unexpected validation errors: %v", fields)
	}
	if in.BaseCurrency != "INR" || in.Timezone != "Asia/Kolkata" {
		t.Fatalf("expected India defaults, got %s/%s", in.BaseCurrency, in.Timezone)
	}
	if in.NumberPadding != 4 || in.StartingNumber != 1 || in.InvoicePrefix != "INV" {
		t.Fatalf("expected invoice numbering defaults, got padding=%d start=%d prefix=%q",
			in.NumberPadding, in.StartingNumber, in.InvoicePrefix)
	}
	if in.GSTIN != "27AAPFU0939F1ZV" {
		t.Fatalf("GSTIN was not preserved: %q", in.GSTIN)
	}
}

func TestValidateCreateCompanyInputRejectsGSTINStateMismatch(t *testing.T) {
	in := validCompanyInput()
	in.RegistrationState = "29"
	if _, fields := ValidateCreateCompanyInput(in); fields["gstin"] == "" {
		t.Fatal("expected GSTIN/state mismatch to be rejected")
	}
}

func TestValidateCreateCompanyInputRejectsIncompleteAddress(t *testing.T) {
	in := validCompanyInput()
	in.AddressLine1 = ""
	in.City = ""
	in.PostalCode = ""
	_, fields := ValidateCreateCompanyInput(in)
	for _, key := range []string{"address_line1", "city", "postal_code"} {
		if fields[key] == "" {
			t.Errorf("expected error for %s", key)
		}
	}
}

func TestValidateCreateCompanyInputRejectsUnsupportedCurrency(t *testing.T) {
	in := validCompanyInput()
	in.BaseCurrency = "USD"
	if _, fields := ValidateCreateCompanyInput(in); fields["base_currency"] == "" {
		t.Fatal("expected non-INR currency to be rejected in this release")
	}
}

func TestValidateRegistrationInputRejectsShortGSTINWithoutPanicking(t *testing.T) {
	_, fields := validateRegistrationInput(TaxRegistrationInput{
		GSTIN: "A", StateCode: "27", RegistrationType: "regular",
	}, true)
	if fields["gstin"] == "" {
		t.Fatal("a one-character GSTIN must be rejected with a field error")
	}
}

func TestFinancialYearDefaultsToIndianAprilYear(t *testing.T) {
	in := validCompanyInput()
	start, end, fields := financialYearFor(time.Date(2026, time.November, 11, 0, 0, 0, 0, time.UTC), in)
	if len(fields) != 0 {
		t.Fatalf("unexpected errors: %v", fields)
	}
	if start.Format("2006-01-02") != "2026-04-01" || end.Format("2006-01-02") != "2027-04-01" {
		t.Fatalf("unexpected financial year %s - %s", start, end)
	}

	start, end, _ = financialYearFor(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), in)
	if start.Format("2006-01-02") != "2025-04-01" || end.Format("2006-01-02") != "2026-04-01" {
		t.Fatalf("unexpected prior financial year %s - %s", start, end)
	}
}

func TestFinancialYearRejectsInvalidRange(t *testing.T) {
	in := validCompanyInput()
	in.FinancialYearStart = "2026-04-01"
	in.FinancialYearEnd = "2026-01-01"
	if _, _, fields := financialYearFor(time.Now(), in); len(fields) == 0 {
		t.Fatal("expected invalid range to be rejected")
	}
	in.FinancialYearStart = "2026-04-01"
	in.FinancialYearEnd = ""
	if _, _, fields := financialYearFor(time.Now(), in); len(fields) == 0 {
		t.Fatal("expected incomplete range to be rejected")
	}
}
