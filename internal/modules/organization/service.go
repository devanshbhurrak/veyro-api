package organization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"backend/internal/platform/audit"
	"backend/internal/platform/database"
	"backend/internal/platform/idempotency"
	"backend/internal/shared/apperr"
	"backend/internal/shared/validate"
)

const defaultIdempotencyTTL = 24 * time.Hour

// Service implements company onboarding and configuration rules.
type Service struct {
	db    *database.Database
	repo  *Repository
	init  *Initializer
	audit *audit.Recorder
}

func NewService(db *database.Database, repo *Repository, init *Initializer, recorder *audit.Recorder) *Service {
	return &Service{db: db, repo: repo, init: init, audit: recorder}
}

var validBusinessTypes = map[string]struct{}{
	"proprietorship": {}, "partnership": {}, "llp": {}, "private_limited": {},
	"public_limited": {}, "huf": {}, "trust": {}, "society": {}, "other": {},
}

var validRegistrationTypes = map[string]struct{}{
	"regular": {}, "composition": {},
}

var stateNames = map[string]string{
	"01": "Jammu and Kashmir", "02": "Himachal Pradesh", "03": "Punjab", "04": "Chandigarh",
	"05": "Uttarakhand", "06": "Haryana", "07": "Delhi", "08": "Rajasthan", "09": "Uttar Pradesh",
	"10": "Bihar", "11": "Sikkim", "12": "Arunachal Pradesh", "13": "Nagaland", "14": "Manipur",
	"15": "Mizoram", "16": "Tripura", "17": "Meghalaya", "18": "Assam", "19": "West Bengal",
	"20": "Jharkhand", "21": "Odisha", "22": "Chhattisgarh", "23": "Madhya Pradesh", "24": "Gujarat",
	"26": "Dadra and Nagar Haveli and Daman and Diu", "27": "Maharashtra", "29": "Karnataka",
	"30": "Goa", "31": "Lakshadweep", "32": "Kerala", "33": "Tamil Nadu", "34": "Puducherry",
	"35": "Andaman and Nicobar Islands", "36": "Telangana", "37": "Andhra Pradesh", "38": "Ladakh",
}

// ValidateCreateCompanyInput normalizes and validates user input, returning a
// field-keyed error map so the API can report every problem at once.
func ValidateCreateCompanyInput(in CreateCompanyInput) (CreateCompanyInput, map[string]string) {
	fields := map[string]string{}

	in.TradeName = strings.TrimSpace(in.TradeName)
	if in.TradeName == "" || len([]rune(in.TradeName)) > 160 {
		fields["trade_name"] = "Trade name is required and must be at most 160 characters."
	}
	in.LegalName = strings.TrimSpace(in.LegalName)
	if len([]rune(in.LegalName)) > 200 {
		fields["legal_name"] = "Legal name must be at most 200 characters."
	}
	in.BusinessType = strings.ToLower(strings.TrimSpace(in.BusinessType))
	if in.BusinessType == "" {
		in.BusinessType = "other"
	}
	if _, ok := validBusinessTypes[in.BusinessType]; !ok {
		fields["business_type"] = "Unsupported business type."
	}
	in.PAN = strings.ToUpper(strings.TrimSpace(in.PAN))
	if in.PAN != "" && !validate.PAN(in.PAN) {
		fields["pan"] = "PAN must be in the format AAAAA9999A."
	}
	if in.Email != "" {
		normalized, ok := validate.Email(in.Email)
		if !ok {
			fields["email"] = "Enter a valid email address."
		}
		in.Email = normalized
	}
	if in.Phone != "" {
		in.Phone = strings.TrimSpace(in.Phone)
		if !validate.PhoneE164(in.Phone) {
			fields["phone"] = "Phone must be in E.164 format, for example +919876543210."
		}
	}
	in.Website = strings.TrimSpace(in.Website)
	if len(in.Website) > 200 {
		fields["website"] = "Website must be at most 200 characters."
	}

	in.AddressLine1 = strings.TrimSpace(in.AddressLine1)
	in.AddressLine2 = strings.TrimSpace(in.AddressLine2)
	in.City = strings.TrimSpace(in.City)
	in.District = strings.TrimSpace(in.District)
	if in.AddressLine1 == "" || len([]rune(in.AddressLine1)) > 200 {
		fields["address_line1"] = "Address line 1 is required and must be at most 200 characters."
	}
	if len([]rune(in.AddressLine2)) > 200 {
		fields["address_line2"] = "Address line 2 must be at most 200 characters."
	}
	if in.City == "" || len([]rune(in.City)) > 100 {
		fields["city"] = "City is required and must be at most 100 characters."
	}
	if len([]rune(in.District)) > 100 {
		fields["district"] = "District must be at most 100 characters."
	}
	in.StateCode = strings.TrimSpace(in.StateCode)
	if _, ok := stateNames[in.StateCode]; !ok {
		fields["state_code"] = "A valid two-digit GST state code is required."
	}
	in.PostalCode = strings.TrimSpace(in.PostalCode)
	if in.PostalCode == "" || len(in.PostalCode) > 10 {
		fields["postal_code"] = "Postal code is required and must be at most 10 characters."
	}
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	if in.CountryCode == "" {
		in.CountryCode = "IN"
	}
	if len(in.CountryCode) != 2 || in.CountryCode != "IN" {
		fields["country_code"] = "Only country code IN is supported."
	}

	in.GSTProfile = strings.ToLower(strings.TrimSpace(in.GSTProfile))
	if in.GSTProfile == "not_registered" || in.GSTProfile == "" {
		in.GSTProfile = "not_registered"
	} else if in.GSTProfile != "registered" {
		fields["gst_profile"] = "GST profile must be registered or not_registered."
	}
	if in.GSTProfile == "registered" {
		in.RegistrationType = strings.ToLower(strings.TrimSpace(in.RegistrationType))
		if in.RegistrationType == "" {
			in.RegistrationType = "regular"
		}
		if _, ok := validRegistrationTypes[in.RegistrationType]; !ok {
			fields["registration_type"] = "Registration type must be regular or composition."
		}
		in.GSTIN = strings.ToUpper(strings.TrimSpace(in.GSTIN))
		if !validate.GSTIN(in.GSTIN) {
			fields["gstin"] = "GSTIN must be a valid 15-character GST identification number."
		}
		in.RegistrationState = strings.TrimSpace(in.RegistrationState)
		if in.RegistrationState == "" {
			in.RegistrationState = in.StateCode
		}
		if _, ok := stateNames[in.RegistrationState]; !ok {
			fields["registration_state"] = "A valid two-digit GST state code is required."
		} else if len(in.GSTIN) >= 2 && in.GSTIN[:2] != in.RegistrationState {
			fields["gstin"] = "GSTIN state prefix does not match the registration state."
		}
		in.RegistrationLegalName = strings.TrimSpace(in.RegistrationLegalName)
		if len([]rune(in.RegistrationLegalName)) > 220 {
			fields["registration_legal_name"] = "Registration legal name must be at most 220 characters."
		}
		in.RegistrationTradeName = strings.TrimSpace(in.RegistrationTradeName)
		if len([]rune(in.RegistrationTradeName)) > 180 {
			fields["registration_trade_name"] = "Registration trade name must be at most 180 characters."
		}
	} else {
		in.GSTIN = ""
		in.RegistrationType = ""
	}

	in.BaseCurrency = strings.ToUpper(strings.TrimSpace(in.BaseCurrency))
	if in.BaseCurrency == "" {
		in.BaseCurrency = "INR"
	}
	if len(in.BaseCurrency) != 3 {
		fields["base_currency"] = "Base currency must be a three-letter ISO code."
	}
	if in.BaseCurrency != "INR" {
		fields["base_currency"] = "INR is the only supported base currency in this release."
	}
	in.Timezone = strings.TrimSpace(in.Timezone)
	if in.Timezone == "" {
		in.Timezone = "Asia/Kolkata"
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		fields["timezone"] = "Unrecognized time zone."
	}
	in.DateFormat = strings.TrimSpace(in.DateFormat)
	if in.DateFormat == "" {
		in.DateFormat = "dd/MM/yyyy"
	}
	if !validDateFormats[in.DateFormat] {
		fields["date_format"] = "Unsupported date format."
	}
	if in.InvoicePrefix == "" {
		in.InvoicePrefix = "INV"
	}
	in.InvoicePrefix = strings.TrimSpace(in.InvoicePrefix)
	if len(in.InvoicePrefix) > 20 {
		fields["invoice_prefix"] = "Invoice prefix must be at most 20 characters."
	}
	if in.NumberPadding == 0 {
		in.NumberPadding = 4
	}
	if in.NumberPadding < 1 || in.NumberPadding > 10 {
		fields["number_padding"] = "Number padding must be between 1 and 10."
	}
	if in.StartingNumber == 0 {
		in.StartingNumber = 1
	}
	if in.StartingNumber < 1 {
		fields["starting_number"] = "Starting number must be positive."
	}

	if len(fields) > 0 {
		return in, fields
	}
	return in, nil
}

var validDateFormats = map[string]bool{
	"dd/MM/yyyy": true, "dd-MM-yyyy": true, "yyyy-MM-dd": true, "MM/dd/yyyy": true,
}

func financialYearFor(now time.Time, in CreateCompanyInput) (time.Time, time.Time, map[string]string) {
	start, end := strings.TrimSpace(in.FinancialYearStart), strings.TrimSpace(in.FinancialYearEnd)
	if start == "" && end == "" {
		loc, _ := time.LoadLocation(in.Timezone)
		today := now.In(loc)
		year := today.Year()
		if today.Month() < time.April {
			year--
		}
		s := time.Date(year, time.April, 1, 0, 0, 0, 0, loc)
		return s, s.AddDate(1, 0, 0), nil
	}
	if start == "" || end == "" {
		return time.Time{}, time.Time{}, map[string]string{
			"financial_year": "Provide both financial year start and end, or neither.",
		}
	}
	s, err := time.Parse("2006-01-02", start)
	if err != nil {
		return time.Time{}, time.Time{}, map[string]string{"financial_year_start": "Use the format YYYY-MM-DD."}
	}
	e, err := time.Parse("2006-01-02", end)
	if err != nil {
		return time.Time{}, time.Time{}, map[string]string{"financial_year_end": "Use the format YYYY-MM-DD."}
	}
	if !e.After(s) || e.Sub(s) > 366*24*time.Hour {
		return time.Time{}, time.Time{}, map[string]string{
			"financial_year_end": "Financial year end must be after the start and at most one year later.",
		}
	}
	return s, e, nil
}

// CreateCompanyResult is the outcome of onboarding a primary company.
type CreateCompanyResult struct {
	Company        *Company
	Registration   *TaxRegistration
	AlreadyExisted bool
	IdempotencyKey string
}

// CreateCompany atomically creates the primary company and seeds all mandatory
// configuration. It is idempotent per tenant when a key is supplied.
func (s *Service) CreateCompany(ctx context.Context, tenantID, actorID, idemKey string, in CreateCompanyInput) (*CreateCompanyResult, error) {
	in, fields := ValidateCreateCompanyInput(in)
	if len(fields) > 0 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Check the company details and retry.").WithDetails(fields)
	}
	fyStart, fyEnd, fyFields := financialYearFor(time.Now(), in)
	if len(fyFields) > 0 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Check the financial year and retry.").WithDetails(fyFields)
	}

	// Fingerprint the complete normalized request so reusing a key with a
	// different payload is rejected instead of silently replaying.
	fingerprint, err := json.Marshal(in)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	result := &CreateCompanyResult{IdempotencyKey: idemKey}
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		record, existingID, acquired, err := idempotency.Reserve(ctx, tx, "company.create", idemKey,
			idempotency.Hash(tenantID, actorID, string(fingerprint), fyStart.Format("2006-01-02")),
			tenantID, "", actorID, defaultIdempotencyTTL)
		if err != nil {
			return err
		}
		if !acquired {
			company, err := s.repo.GetCompany(ctx, tx, existingID)
			if err != nil {
				return apperr.Internal(err)
			}
			if company == nil || company.TenantID != tenantID {
				return apperr.Conflict(apperr.CodeConflict, "The company already exists.")
			}
			result.Company = company
			result.AlreadyExisted = true
			return nil
		}

		if existing, err := s.repo.GetPrimaryCompany(ctx, tx, tenantID); err != nil {
			return apperr.Internal(err)
		} else if existing != nil {
			return apperr.Conflict("company_already_exists", "This workspace already has a primary company.")
		}

		company := &Company{
			TenantID:     tenantID,
			TradeName:    in.TradeName,
			LegalName:    optional(in.LegalName),
			PAN:          optional(in.PAN),
			BusinessType: optional(in.BusinessType),
			GSTProfile:   in.GSTProfile,
			Email:        optional(in.Email),
			Phone:        optional(in.Phone),
			Website:      optional(in.Website),
			AddressLine1: optional(in.AddressLine1),
			AddressLine2: optional(in.AddressLine2),
			City:         optional(in.City),
			District:     optional(in.District),
			StateCode:    optional(in.StateCode),
			PostalCode:   optional(in.PostalCode),
			CountryCode:  in.CountryCode,
			BaseCurrency: in.BaseCurrency,
			Timezone:     in.Timezone,
			IsPrimary:    true,
			Status:       "active",
		}
		if err := s.repo.CreateCompany(ctx, tx, company); err != nil {
			if isUniqueViolation(err, "companies_tenant_active_primary_idx") {
				return apperr.Conflict("company_already_exists", "This workspace already has a primary company.")
			}
			return apperr.Internal(err)
		}

		var registration *TaxRegistration
		if in.GSTProfile == "registered" {
			registration = &TaxRegistration{
				GSTIN:            in.GSTIN,
				StateCode:        in.RegistrationState,
				RegistrationType: in.RegistrationType,
				LegalName:        optional(firstNonEmpty(in.RegistrationLegalName, in.LegalName, in.TradeName)),
				TradeName:        optional(firstNonEmpty(in.RegistrationTradeName, in.TradeName)),
				AddressLine1:     company.AddressLine1,
				AddressLine2:     company.AddressLine2,
				City:             company.City,
				District:         company.District,
				PostalCode:       company.PostalCode,
				CountryCode:      in.CountryCode,
				ValidFrom:        fyStart,
				IsDefault:        true,
				Status:           "active",
			}
			if err := s.repo.CreateTaxRegistration(ctx, tx, tenantID, company.ID, registration); err != nil {
				return apperr.Internal(err)
			}
		}

		if err := s.init.Initialize(ctx, tx, company, registration, fyStart, fyEnd,
			in.InvoicePrefix, in.NumberPadding, in.StartingNumber, in.DateFormat, actorID); err != nil {
			return apperr.Internal(err)
		}

		if err := s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, ActorType: "user",
			Action: "company.created", EntityType: "company", EntityID: &company.ID,
			Metadata: map[string]any{"trade_name": company.TradeName, "gst_profile": company.GSTProfile},
		}); err != nil {
			return apperr.Internal(err)
		}

		if record != nil {
			if err := idempotency.Complete(ctx, tx, record.ID, company.ID, 201); err != nil {
				return apperr.Internal(err)
			}
		}
		result.Company = company
		result.Registration = registration
		return nil
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	return result, nil
}

// GetCompany returns a company only when it belongs to the tenant.
func (s *Service) GetCompany(ctx context.Context, tenantID, companyID string) (*Company, error) {
	if !validate.UUID(companyID) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Company not found.")
	}
	company, err := s.repo.GetCompany(ctx, s.db.Pool, companyID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if company == nil || company.TenantID != tenantID {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Company not found.")
	}
	return company, nil
}

// PrimaryCompany returns the tenant's primary company, or nil when onboarding
// has not completed.
func (s *Service) PrimaryCompany(ctx context.Context, tenantID string) (*Company, error) {
	company, err := s.repo.GetPrimaryCompany(ctx, s.db.Pool, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return company, nil
}

// UpdateCompanyInput carries a partial company update.
type UpdateCompanyInput struct {
	TradeName    *string
	LegalName    *string
	BusinessType *string
	PAN          *string
	Email        *string
	Phone        *string
	Website      *string
	AddressLine1 *string
	AddressLine2 *string
	City         *string
	District     *string
	StateCode    *string
	PostalCode   *string
	CountryCode  *string
	RowVersion   int64
}

// UpdateCompany applies a version-checked partial update.
func (s *Service) UpdateCompany(ctx context.Context, tenantID, companyID, actorID string, in UpdateCompanyInput) (*Company, error) {
	company, err := s.GetCompany(ctx, tenantID, companyID)
	if err != nil {
		return nil, err
	}
	if in.RowVersion <= 0 {
		return nil, apperr.BadRequest(apperr.CodeValidation,
			"row_version is required so concurrent edits are not silently overwritten.").
			WithDetails(map[string]string{"row_version": "Provide the version you last read."})
	}
	if in.TradeName != nil {
		v := strings.TrimSpace(*in.TradeName)
		if v == "" || len([]rune(v)) > 160 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Trade name is invalid.").
				WithDetails(map[string]string{"trade_name": "Trade name is required and must be at most 160 characters."})
		}
		company.TradeName = v
	}
	if in.LegalName != nil {
		v := strings.TrimSpace(*in.LegalName)
		if len([]rune(v)) > 200 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Legal name must be at most 200 characters.")
		}
		company.LegalName = optional(v)
	}
	if in.BusinessType != nil {
		v := strings.ToLower(strings.TrimSpace(*in.BusinessType))
		if _, ok := validBusinessTypes[v]; !ok {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Unsupported business type.")
		}
		company.BusinessType = &v
	}
	if in.PAN != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.PAN))
		if v != "" && !validate.PAN(v) {
			return nil, apperr.BadRequest(apperr.CodeValidation, "PAN must be in the format AAAAA9999A.")
		}
		company.PAN = optional(v)
	}
	if in.Email != nil {
		v, ok := validate.Email(*in.Email)
		if !ok {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Enter a valid email address.")
		}
		company.Email = optional(v)
	}
	if in.Phone != nil {
		v := strings.TrimSpace(*in.Phone)
		if v != "" && !validate.PhoneE164(v) {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Phone must be in E.164 format.")
		}
		company.Phone = optional(v)
	}
	if in.Website != nil {
		v := strings.TrimSpace(*in.Website)
		if len(v) > 200 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Website must be at most 200 characters.")
		}
		company.Website = optional(v)
	}
	if in.AddressLine1 != nil {
		v := strings.TrimSpace(*in.AddressLine1)
		if v == "" || len([]rune(v)) > 200 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Address line 1 is required and must be at most 200 characters.")
		}
		company.AddressLine1 = optional(v)
	}
	if in.AddressLine2 != nil {
		company.AddressLine2 = optional(strings.TrimSpace(*in.AddressLine2))
	}
	if in.City != nil {
		v := strings.TrimSpace(*in.City)
		if v == "" || len([]rune(v)) > 100 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "City is required and must be at most 100 characters.")
		}
		company.City = optional(v)
	}
	if in.District != nil {
		company.District = optional(strings.TrimSpace(*in.District))
	}
	if in.StateCode != nil {
		v := strings.TrimSpace(*in.StateCode)
		if _, ok := stateNames[v]; !ok {
			return nil, apperr.BadRequest(apperr.CodeValidation, "A valid two-digit GST state code is required.")
		}
		company.StateCode = &v
	}
	if in.PostalCode != nil {
		v := strings.TrimSpace(*in.PostalCode)
		if v == "" || len(v) > 10 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Postal code is required and must be at most 10 characters.")
		}
		company.PostalCode = optional(v)
	}
	if in.CountryCode != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.CountryCode))
		if v != "IN" {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Only country code IN is supported.")
		}
		company.CountryCode = v
	}

	var updated bool
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		updated, err = s.repo.UpdateCompany(ctx, tx, company, in.RowVersion)
		if err != nil || !updated {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, Action: "company.updated",
			EntityType: "company", EntityID: &company.ID,
			Changes: map[string]any{"row_version": in.RowVersion + 1},
		})
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if !updated {
		return nil, apperr.Conflict("version_conflict", "The company was changed by another request. Reload and retry.")
	}
	return s.GetCompany(ctx, tenantID, companyID)
}

// Settings returns company configuration.
func (s *Service) Settings(ctx context.Context, tenantID, companyID string) (*Settings, error) {
	if _, err := s.GetCompany(ctx, tenantID, companyID); err != nil {
		return nil, err
	}
	settings, err := s.repo.GetSettings(ctx, s.db.Pool, tenantID, companyID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if settings == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Company settings not found.")
	}
	return settings, nil
}

// UpdateSettingsInput carries a partial settings update.
type UpdateSettingsInput struct {
	DefaultPriceIncludesTax *bool
	NegativeStockPolicy     *string
	InventoryCostMethod     *string
	DefaultCreditDays       *int
	CreditLimitAction       *string
	AmountDisplayScale      *int16
	QuantityDisplayScale    *int16
	DateFormat              *string
	Locale                  *string
	RowVersion              int64
}

// UpdateSettings applies a validated, version-checked settings update.
func (s *Service) UpdateSettings(ctx context.Context, tenantID, companyID, actorID string, in UpdateSettingsInput) (*Settings, error) {
	settings, err := s.Settings(ctx, tenantID, companyID)
	if err != nil {
		return nil, err
	}
	if in.RowVersion <= 0 {
		return nil, apperr.BadRequest(apperr.CodeValidation,
			"row_version is required so concurrent edits are not silently overwritten.")
	}
	if in.DefaultPriceIncludesTax != nil {
		settings.DefaultPriceIncludesTax = *in.DefaultPriceIncludesTax
	}
	if in.NegativeStockPolicy != nil {
		v := strings.ToLower(strings.TrimSpace(*in.NegativeStockPolicy))
		if !oneOf(v, "block", "warn", "allow") {
			return nil, apperr.BadRequest(apperr.CodeValidation, "negative_stock_policy must be block, warn, or allow.")
		}
		settings.NegativeStockPolicy = v
	}
	if in.InventoryCostMethod != nil {
		v := strings.ToLower(strings.TrimSpace(*in.InventoryCostMethod))
		if !oneOf(v, "moving_average", "fifo") {
			return nil, apperr.BadRequest(apperr.CodeValidation, "inventory_cost_method must be moving_average or fifo.")
		}
		settings.InventoryCostMethod = v
	}
	if in.DefaultCreditDays != nil {
		if *in.DefaultCreditDays < 0 || *in.DefaultCreditDays > 365 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "default_credit_days must be between 0 and 365.")
		}
		settings.DefaultCreditDays = *in.DefaultCreditDays
	}
	if in.CreditLimitAction != nil {
		v := strings.ToLower(strings.TrimSpace(*in.CreditLimitAction))
		if !oneOf(v, "warn", "block") {
			return nil, apperr.BadRequest(apperr.CodeValidation, "credit_limit_action must be warn or block.")
		}
		settings.CreditLimitAction = v
	}
	if in.AmountDisplayScale != nil {
		if *in.AmountDisplayScale < 0 || *in.AmountDisplayScale > 4 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "amount_display_scale must be between 0 and 4.")
		}
		settings.AmountDisplayScale = *in.AmountDisplayScale
	}
	if in.QuantityDisplayScale != nil {
		if *in.QuantityDisplayScale < 0 || *in.QuantityDisplayScale > 6 {
			return nil, apperr.BadRequest(apperr.CodeValidation, "quantity_display_scale must be between 0 and 6.")
		}
		settings.QuantityDisplayScale = *in.QuantityDisplayScale
	}
	if in.DateFormat != nil {
		v := strings.TrimSpace(*in.DateFormat)
		if !validDateFormats[v] {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Unsupported date format.")
		}
		settings.DateFormat = v
	}
	if in.Locale != nil {
		v := strings.TrimSpace(*in.Locale)
		if !oneOf(v, "en-IN") {
			return nil, apperr.BadRequest(apperr.CodeValidation, "locale must be en-IN.")
		}
		settings.Locale = v
	}

	var updated bool
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		updated, err = s.repo.UpdateSettings(ctx, tx, tenantID, companyID, settings, in.RowVersion)
		if err != nil || !updated {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, Action: "company.settings_updated",
			EntityType: "company_settings", EntityID: &companyID,
			Changes: map[string]any{"row_version": in.RowVersion + 1},
		})
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if !updated {
		return nil, apperr.Conflict("version_conflict", "Settings were changed by another request. Reload and retry.")
	}
	return s.Settings(ctx, tenantID, companyID)
}

// TaxRegistrations lists a company's GST registrations.
func (s *Service) TaxRegistrations(ctx context.Context, tenantID, companyID string) ([]TaxRegistration, error) {
	if _, err := s.GetCompany(ctx, tenantID, companyID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListTaxRegistrations(ctx, s.db.Pool, tenantID, companyID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if list == nil {
		list = []TaxRegistration{}
	}
	return list, nil
}

// TaxRegistrationInput is a create/update payload for a GST registration.
type TaxRegistrationInput struct {
	GSTIN            string
	StateCode        string
	RegistrationType string
	LegalName        string
	TradeName        string
	AddressLine1     string
	AddressLine2     string
	City             string
	District         string
	PostalCode       string
	CountryCode      string
	ValidFrom        string
	ValidTo          string
	IsDefault        bool
}

func validateRegistrationInput(in TaxRegistrationInput, requireGSTIN bool) (TaxRegistrationInput, map[string]string) {
	fields := map[string]string{}
	in.GSTIN = strings.ToUpper(strings.TrimSpace(in.GSTIN))
	if requireGSTIN {
		if !validate.GSTIN(in.GSTIN) {
			fields["gstin"] = "GSTIN must be a valid 15-character GST identification number."
		}
	}
	in.StateCode = strings.TrimSpace(in.StateCode)
	if _, ok := stateNames[in.StateCode]; !ok {
		fields["state_code"] = "A valid two-digit GST state code is required."
	} else if len(in.GSTIN) >= 2 && in.GSTIN[:2] != in.StateCode {
		fields["gstin"] = "GSTIN state prefix does not match the registration state."
	}
	in.RegistrationType = strings.ToLower(strings.TrimSpace(in.RegistrationType))
	if _, ok := validRegistrationTypes[in.RegistrationType]; !ok {
		fields["registration_type"] = "Registration type must be regular or composition."
	}
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	if in.CountryCode == "" {
		in.CountryCode = "IN"
	}
	if in.CountryCode != "IN" {
		fields["country_code"] = "Only country code IN is supported."
	}
	if len(fields) > 0 {
		return in, fields
	}
	return in, nil
}

// CreateTaxRegistration adds a GST registration to an existing company.
func (s *Service) CreateTaxRegistration(ctx context.Context, tenantID, companyID, actorID string, in TaxRegistrationInput) (*TaxRegistration, error) {
	in, fields := validateRegistrationInput(in, true)
	if len(fields) > 0 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Check the GST registration and retry.").WithDetails(fields)
	}
	if _, err := s.GetCompany(ctx, tenantID, companyID); err != nil {
		return nil, err
	}
	validFrom := time.Now()
	if strings.TrimSpace(in.ValidFrom) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(in.ValidFrom))
		if err != nil {
			return nil, apperr.BadRequest(apperr.CodeValidation, "valid_from must use YYYY-MM-DD.")
		}
		validFrom = parsed
	}
	registration := &TaxRegistration{
		GSTIN: in.GSTIN, StateCode: in.StateCode, RegistrationType: in.RegistrationType,
		LegalName: optional(in.LegalName), TradeName: optional(in.TradeName),
		AddressLine1: optional(in.AddressLine1), AddressLine2: optional(in.AddressLine2),
		City: optional(in.City), District: optional(in.District), PostalCode: optional(in.PostalCode),
		CountryCode: in.CountryCode, ValidFrom: validFrom, IsDefault: in.IsDefault, Status: "active",
	}
	if in.ValidTo != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(in.ValidTo))
		if err != nil || !parsed.After(validFrom) {
			return nil, apperr.BadRequest(apperr.CodeValidation, "valid_to must be after valid_from.")
		}
		registration.ValidTo = &parsed
	}
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if registration.IsDefault {
			if err := s.repo.ClearDefaultTaxRegistrations(ctx, tx, tenantID, companyID, ""); err != nil {
				return apperr.Internal(err)
			}
		}
		if err := s.repo.CreateTaxRegistration(ctx, tx, tenantID, companyID, registration); err != nil {
			if isUniqueViolation(err, "") {
				return apperr.Conflict("tax_registration_exists", "This GSTIN is already registered for the company.")
			}
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, Action: "tax_registration.created",
			EntityType: "company_tax_registration", EntityID: &registration.ID,
			Metadata: map[string]any{"state_code": registration.StateCode},
		})
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	return registration, nil
}

// UpdateTaxRegistration changes mutable registration fields.
func (s *Service) UpdateTaxRegistration(ctx context.Context, tenantID, companyID, id, actorID string, in TaxRegistrationInput) (*TaxRegistration, error) {
	if _, err := s.GetCompany(ctx, tenantID, companyID); err != nil {
		return nil, err
	}
	current, err := s.repo.GetTaxRegistration(ctx, s.db.Pool, tenantID, companyID, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if current == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Tax registration not found.")
	}
	in, fields := validateRegistrationInput(in, false)
	if len(fields) > 0 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Check the GST registration and retry.").WithDetails(fields)
	}
	current.RegistrationType = in.RegistrationType
	current.LegalName = optional(in.LegalName)
	current.TradeName = optional(in.TradeName)
	current.AddressLine1 = optional(in.AddressLine1)
	current.AddressLine2 = optional(in.AddressLine2)
	current.City = optional(in.City)
	current.District = optional(in.District)
	current.PostalCode = optional(in.PostalCode)
	// PATCH never silently unsets the company's only default registration;
	// making another registration the default is the way to change it.
	if in.IsDefault {
		current.IsDefault = true
	}
	if in.ValidTo != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(in.ValidTo))
		if err != nil || !parsed.After(current.ValidFrom) {
			return nil, apperr.BadRequest(apperr.CodeValidation, "valid_to must be after valid_from.")
		}
		current.ValidTo = &parsed
	}

	var updated bool
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if current.IsDefault {
			if err := s.repo.ClearDefaultTaxRegistrations(ctx, tx, tenantID, companyID, current.ID); err != nil {
				return apperr.Internal(err)
			}
		}
		var err error
		updated, err = s.repo.UpdateTaxRegistration(ctx, tx, tenantID, companyID, current)
		if err != nil || !updated {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, Action: "tax_registration.updated",
			EntityType: "company_tax_registration", EntityID: &current.ID,
		})
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if !updated {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Tax registration not found.")
	}
	return s.repo.GetTaxRegistration(ctx, s.db.Pool, tenantID, companyID, id)
}

// OnboardingStatus computes the authoritative readiness checklist.
func (s *Service) OnboardingStatus(ctx context.Context, tenantID, companyID string) (*OnboardingStatus, error) {
	company, err := s.PrimaryCompany(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return &OnboardingStatus{Ready: false, Checklist: []ChecklistItem{}}, nil
	}
	if companyID != "" && company.ID != companyID {
		if _, err := s.GetCompany(ctx, tenantID, companyID); err != nil {
			return nil, err
		}
		company, err = s.GetCompany(ctx, tenantID, companyID)
		if err != nil {
			return nil, err
		}
	}
	facts, err := s.repo.LoadOnboardingFacts(ctx, s.db.Pool, tenantID, company.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	hasAccounts := true
	for _, code := range []string{"cash", "accounts_receivable", "accounts_payable", "input_gst", "output_gst", "inventory", "sales_revenue", "purchases", "cogs"} {
		if !contains(facts.AccountSystemCodes, code) {
			hasAccounts = false
			break
		}
	}
	items := []ChecklistItem{
		{Key: "company_profile", Label: "Company profile", Mandatory: true,
			Complete: company.TradeName != "" && facts.AddressComplete,
			Message:  "Add the business name, address, city, state, and postal code."},
		{Key: "gst_configuration", Label: "GST configuration", Mandatory: true,
			Complete: facts.TaxRegistrationCount > 0 || company.GSTProfile == "not_registered",
			Message:  "Add a GST registration or confirm the business is not registered."},
		{Key: "financial_year", Label: "Financial year and accounting periods", Mandatory: true,
			Complete: facts.HasFinancialYear && facts.PeriodCount == 12,
			Message:  "Create the current financial year with 12 accounting periods."},
		{Key: "chart_of_accounts", Label: "Chart of accounts", Mandatory: true,
			Complete: hasAccounts,
			Message:  "Seed the required system accounts."},
		{Key: "default_branch", Label: "Default branch", Mandatory: true,
			Complete: facts.HasDefaultBranch,
			Message:  "Create a default branch."},
		{Key: "default_warehouse", Label: "Default warehouse", Mandatory: true,
			Complete: facts.HasDefaultWarehouse && facts.WarehouseBranchID == facts.DefaultBranchID && facts.DefaultBranchID != "",
			Message:  "Create a default warehouse in the default branch."},
		{Key: "cash_account", Label: "Default cash or bank account", Mandatory: true,
			Complete: facts.CashAccountExists,
			Message:  "Create a default cash or bank account."},
		{Key: "number_series", Label: "Document number series", Mandatory: true,
			Complete: facts.NumberSeriesCount >= len(requiredSeries),
			Message:  "Create the required document number series."},
		{Key: "invoice_template", Label: "Default invoice template", Mandatory: true,
			Complete: facts.TemplateExists && facts.TemplateVersionCount > 0,
			Message:  "Create a default invoice template."},
		{Key: "tax_codes", Label: "GST tax codes", Mandatory: true,
			Complete: facts.TaxCodeCount > 0 || company.GSTProfile == "not_registered",
			Message:  "Seed GST tax codes for a registered business."},
		{Key: "company_settings", Label: "Company settings", Mandatory: true,
			Complete: facts.HasSettings,
			Message:  "Save company settings."},
		{Key: "company_admin", Label: "Company administrator", Mandatory: true,
			Complete: true,
			Message:  "The company owner is assigned automatically."},
	}

	completed, mandatory, ready := 0, 0, true
	for _, item := range items {
		if item.Mandatory {
			mandatory++
			if !item.Complete {
				ready = false
			}
		}
		if item.Complete {
			completed++
		}
	}
	return &OnboardingStatus{Company: company, Ready: ready, Checklist: items,
		Completed: completed, Mandatory: mandatory}, nil
}

// StateName returns the display name for a GST state code.
func StateName(code string) string { return stateNames[code] }

func optional(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

// ParseInt64 parses a positive integer query/body value.
func ParseInt64(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("empty value")
	}
	return strconv.ParseInt(value, 10, 64)
}
