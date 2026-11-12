package organization

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/database"
)

// Repository provides company persistence.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

const companyColumns = `id, tenant_id, trade_name, legal_name, pan, business_type, gst_profile,
	email::text, phone_e164, website, address_line1, address_line2, city, district, state_code,
	postal_code, country_code, base_currency, timezone, is_primary, status, created_at, row_version`

func scanCompany(row pgx.Row) (*Company, error) {
	var c Company
	err := row.Scan(&c.ID, &c.TenantID, &c.TradeName, &c.LegalName, &c.PAN, &c.BusinessType, &c.GSTProfile,
		&c.Email, &c.Phone, &c.Website, &c.AddressLine1, &c.AddressLine2, &c.City, &c.District,
		&c.StateCode, &c.PostalCode, &c.CountryCode, &c.BaseCurrency, &c.Timezone, &c.IsPrimary,
		&c.Status, &c.CreatedAt, &c.RowVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) GetCompany(ctx context.Context, q database.Querier, companyID string) (*Company, error) {
	return scanCompany(q.QueryRow(ctx, `SELECT `+companyColumns+` FROM companies WHERE id=$1`, companyID))
}

func (r *Repository) GetPrimaryCompany(ctx context.Context, q database.Querier, tenantID string) (*Company, error) {
	return scanCompany(q.QueryRow(ctx, `SELECT `+companyColumns+` FROM companies
		WHERE tenant_id=$1 AND is_primary=true AND status='active'`, tenantID))
}

// CreateCompany inserts the company row. Primary uniqueness is enforced by the
// database; callers map the violation to a conflict.
func (r *Repository) CreateCompany(ctx context.Context, q database.Querier, c *Company) error {
	return q.QueryRow(ctx, `
		INSERT INTO companies
			(tenant_id, trade_name, legal_name, pan, business_type, gst_profile, email, phone_e164,
			 website, address_line1, address_line2, city, district, state_code, postal_code,
			 country_code, base_currency, timezone, is_primary, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,true,'active')
		RETURNING id, created_at, row_version`,
		c.TenantID, c.TradeName, c.LegalName, c.PAN, c.BusinessType, c.GSTProfile, c.Email, c.Phone,
		c.Website, c.AddressLine1, c.AddressLine2, c.City, c.District, c.StateCode, c.PostalCode,
		c.CountryCode, c.BaseCurrency, c.Timezone).Scan(&c.ID, &c.CreatedAt, &c.RowVersion)
}

// UpdateCompany performs an optimistic-concurrency update.
func (r *Repository) UpdateCompany(ctx context.Context, q database.Querier, c *Company, expectedVersion int64) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE companies SET
			trade_name=$2, legal_name=$3, pan=$4, business_type=$5, gst_profile=$6, email=$7,
			phone_e164=$8, website=$9, address_line1=$10, address_line2=$11, city=$12, district=$13,
			state_code=$14, postal_code=$15, country_code=$16, base_currency=$17, timezone=$18,
			updated_at=now(), row_version=row_version+1
		WHERE id=$1 AND row_version=$19`,
		c.ID, c.TradeName, c.LegalName, c.PAN, c.BusinessType, c.GSTProfile, c.Email, c.Phone,
		c.Website, c.AddressLine1, c.AddressLine2, c.City, c.District, c.StateCode, c.PostalCode,
		c.CountryCode, c.BaseCurrency, c.Timezone, expectedVersion)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) GetSettings(ctx context.Context, q database.Querier, tenantID, companyID string) (*Settings, error) {
	var s Settings
	err := q.QueryRow(ctx, `
		SELECT default_branch_id, default_warehouse_id, default_cash_bank_account_id,
		       default_price_includes_tax, negative_stock_policy, inventory_cost_method,
		       default_credit_days, credit_limit_action, amount_display_scale,
		       quantity_display_scale, date_format, locale, row_version
		FROM company_settings WHERE tenant_id=$1 AND company_id=$2`, tenantID, companyID).Scan(
		&s.DefaultBranchID, &s.DefaultWarehouseID, &s.DefaultCashBankAccountID,
		&s.DefaultPriceIncludesTax, &s.NegativeStockPolicy, &s.InventoryCostMethod,
		&s.DefaultCreditDays, &s.CreditLimitAction, &s.AmountDisplayScale,
		&s.QuantityDisplayScale, &s.DateFormat, &s.Locale, &s.RowVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateSettings performs an optimistic-concurrency update.
func (r *Repository) UpdateSettings(ctx context.Context, q database.Querier, tenantID, companyID string, s *Settings, expectedVersion int64) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE company_settings SET
			default_price_includes_tax=$3, negative_stock_policy=$4, inventory_cost_method=$5,
			default_credit_days=$6, credit_limit_action=$7, amount_display_scale=$8,
			quantity_display_scale=$9, date_format=$10, locale=$11,
			updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND company_id=$2 AND row_version=$12`,
		tenantID, companyID, s.DefaultPriceIncludesTax, s.NegativeStockPolicy, s.InventoryCostMethod,
		s.DefaultCreditDays, s.CreditLimitAction, s.AmountDisplayScale, s.QuantityDisplayScale,
		s.DateFormat, s.Locale, expectedVersion)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

const registrationColumns = `id, gstin, state_code, registration_type, legal_name, trade_name,
	address_line1, address_line2, city, district, postal_code, country_code, valid_from, valid_to,
	is_default, status`

func scanRegistration(row pgx.Row) (*TaxRegistration, error) {
	var t TaxRegistration
	err := row.Scan(&t.ID, &t.GSTIN, &t.StateCode, &t.RegistrationType, &t.LegalName, &t.TradeName,
		&t.AddressLine1, &t.AddressLine2, &t.City, &t.District, &t.PostalCode, &t.CountryCode,
		&t.ValidFrom, &t.ValidTo, &t.IsDefault, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) ListTaxRegistrations(ctx context.Context, q database.Querier, tenantID, companyID string) ([]TaxRegistration, error) {
	rows, err := q.Query(ctx, `SELECT `+registrationColumns+` FROM company_tax_registrations
		WHERE tenant_id=$1 AND company_id=$2 ORDER BY valid_from DESC`, tenantID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var registrations []TaxRegistration
	for rows.Next() {
		t, err := scanRegistration(rows)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, *t)
	}
	return registrations, rows.Err()
}

func (r *Repository) GetTaxRegistration(ctx context.Context, q database.Querier, tenantID, companyID, id string) (*TaxRegistration, error) {
	return scanRegistration(q.QueryRow(ctx, `SELECT `+registrationColumns+`
		FROM company_tax_registrations WHERE tenant_id=$1 AND company_id=$2 AND id=$3`, tenantID, companyID, id))
}

// ClearDefaultTaxRegistrations unsets the default flag on other registrations
// so a company never exposes more than one default GST registration.
func (r *Repository) ClearDefaultTaxRegistrations(ctx context.Context, q database.Querier, tenantID, companyID, exceptID string) error {
	var except any
	if exceptID != "" {
		except = exceptID
	}
	_, err := q.Exec(ctx, `
		UPDATE company_tax_registrations
		SET is_default=false, row_version=row_version+1
		WHERE tenant_id=$1 AND company_id=$2 AND is_default=true
		  AND ($3::uuid IS NULL OR id<>$3::uuid)`, tenantID, companyID, except)
	return err
}

func (r *Repository) CreateTaxRegistration(ctx context.Context, q database.Querier, tenantID, companyID string, t *TaxRegistration) error {
	return q.QueryRow(ctx, `
		INSERT INTO company_tax_registrations
			(tenant_id, company_id, gstin, state_code, registration_type, legal_name, trade_name,
			 address_line1, address_line2, city, district, postal_code, country_code,
			 valid_from, valid_to, is_default, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,'active')
		RETURNING id`,
		tenantID, companyID, t.GSTIN, t.StateCode, t.RegistrationType, t.LegalName, t.TradeName,
		t.AddressLine1, t.AddressLine2, t.City, t.District, t.PostalCode, t.CountryCode,
		t.ValidFrom, t.ValidTo, t.IsDefault).Scan(&t.ID)
}

func (r *Repository) UpdateTaxRegistration(ctx context.Context, q database.Querier, tenantID, companyID string, t *TaxRegistration) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE company_tax_registrations SET
			registration_type=$4, legal_name=$5, trade_name=$6, address_line1=$7, address_line2=$8,
			city=$9, district=$10, postal_code=$11, valid_to=$12, is_default=$13
		WHERE tenant_id=$1 AND company_id=$2 AND id=$3`,
		tenantID, companyID, t.ID, t.RegistrationType, t.LegalName, t.TradeName, t.AddressLine1,
		t.AddressLine2, t.City, t.District, t.PostalCode, t.ValidTo, t.IsDefault)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// OnboardingFacts contains authoritative counts used to determine readiness.
type OnboardingFacts struct {
	HasDefaultBranch     bool
	HasDefaultWarehouse  bool
	WarehouseBranchID    string
	DefaultBranchID      string
	AccountSystemCodes   []string
	CashAccountExists    bool
	HasFinancialYear     bool
	CurrentFinancialYear bool
	PeriodCount          int
	HasSettings          bool
	NumberSeriesCount    int
	TaxCodeCount         int
	TemplateExists       bool
	TemplateVersionCount int
	TaxRegistrationCount int
	AddressComplete      bool
}

func (r *Repository) LoadOnboardingFacts(ctx context.Context, q database.Querier, tenantID, companyID string) (*OnboardingFacts, error) {
	var f OnboardingFacts
	if err := q.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM branches WHERE tenant_id=$1 AND company_id=$2 AND is_default=true AND status='active'),
			EXISTS (SELECT 1 FROM warehouses WHERE tenant_id=$1 AND company_id=$2 AND is_default=true AND status='active'),
			COALESCE((SELECT b.id::text FROM company_settings cs JOIN branches b ON b.id=cs.default_branch_id
				WHERE cs.tenant_id=$1 AND cs.company_id=$2), ''),
			COALESCE((SELECT w.branch_id::text FROM company_settings cs JOIN warehouses w ON w.id=cs.default_warehouse_id
				WHERE cs.tenant_id=$1 AND cs.company_id=$2), ''),
			COALESCE((SELECT array_agg(DISTINCT a.system_code) FROM accounts a
				WHERE a.tenant_id=$1 AND a.company_id=$2 AND a.system_code IS NOT NULL), '{}'),
			EXISTS (SELECT 1 FROM cash_bank_accounts WHERE tenant_id=$1 AND company_id=$2 AND is_default=true AND status='active'),
			EXISTS (SELECT 1 FROM financial_years WHERE tenant_id=$1 AND company_id=$2),
			EXISTS (SELECT 1 FROM financial_years WHERE tenant_id=$1 AND company_id=$2 AND is_current=true),
			(SELECT COUNT(*) FROM accounting_periods WHERE tenant_id=$1 AND company_id=$2),
			EXISTS (SELECT 1 FROM company_settings WHERE tenant_id=$1 AND company_id=$2),
			(SELECT COUNT(*) FROM document_number_series WHERE tenant_id=$1 AND company_id=$2 AND is_active=true),
			(SELECT COUNT(*) FROM tax_codes WHERE tenant_id=$1 AND company_id=$2 AND status='active'),
			EXISTS (SELECT 1 FROM document_templates WHERE tenant_id=$1 AND company_id=$2 AND is_default=true),
			(SELECT COUNT(*) FROM document_template_versions v JOIN document_templates t ON t.id=v.template_id
				WHERE t.tenant_id=$1 AND t.company_id=$2),
			(SELECT COUNT(*) FROM company_tax_registrations WHERE tenant_id=$1 AND company_id=$2 AND status='active'),
			(address_line1 IS NOT NULL AND city IS NOT NULL AND state_code IS NOT NULL AND postal_code IS NOT NULL)
		FROM companies WHERE id=$2`, tenantID, companyID).Scan(
		&f.HasDefaultBranch, &f.HasDefaultWarehouse, &f.WarehouseBranchID, &f.DefaultBranchID,
		&f.AccountSystemCodes, &f.CashAccountExists, &f.HasFinancialYear, &f.CurrentFinancialYear,
		&f.PeriodCount, &f.HasSettings, &f.NumberSeriesCount, &f.TaxCodeCount, &f.TemplateExists,
		&f.TemplateVersionCount, &f.TaxRegistrationCount, &f.AddressComplete); err != nil {
		return nil, err
	}
	return &f, nil
}
