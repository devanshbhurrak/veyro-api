package organization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Initializer seeds a company's mandatory default configuration inside the
// company-creation transaction. Any error rolls back the whole company.
type Initializer struct{}

func NewInitializer() *Initializer { return &Initializer{} }

type accountSeed struct {
	Code        string
	Name        string
	Type        string
	Subtype     string
	Normal      string
	SystemCode  string
	Control     bool
	ControlRole string
	AllowManual bool
}

// chartOfAccounts is the reviewed default accounting template.
var chartOfAccounts = []accountSeed{
	{"1000", "Cash", "asset", "current_asset", "debit", "cash", false, "", true},
	{"1010", "Bank", "asset", "current_asset", "debit", "bank", false, "", true},
	{"1020", "Accounts Receivable", "asset", "current_asset", "debit", "accounts_receivable", true, "receivable", false},
	{"1030", "Input GST", "asset", "current_asset", "debit", "input_gst", true, "gst_input", false},
	{"1040", "Inventory", "asset", "current_asset", "debit", "inventory", true, "inventory", false},
	{"1050", "Supplier Advances", "asset", "current_asset", "debit", "supplier_advances", true, "supplier_advances", false},
	{"2000", "Accounts Payable", "liability", "current_liability", "credit", "accounts_payable", true, "payable", false},
	{"2010", "Output GST", "liability", "current_liability", "credit", "output_gst", true, "gst_output", false},
	{"2020", "Customer Advances", "liability", "current_liability", "credit", "customer_advances", true, "customer_advances", false},
	{"3000", "Owner's Equity", "equity", "equity", "credit", "owners_equity", false, "", true},
	{"3900", "Retained Earnings", "equity", "equity", "credit", "retained_earnings", false, "", true},
	{"4000", "Sales Revenue", "revenue", "operating_revenue", "credit", "sales_revenue", false, "", true},
	{"4900", "Other Income", "revenue", "other_revenue", "credit", "other_income", false, "", true},
	{"5000", "Purchases", "expense", "cost_of_goods_sold", "debit", "purchases", false, "", true},
	{"5100", "Cost of Goods Sold", "expense", "cost_of_goods_sold", "debit", "cogs", true, "cogs", false},
	{"5200", "Operating Expenses", "expense", "operating_expense", "debit", "operating_expenses", false, "", true},
	{"5300", "Discounts Given", "expense", "operating_expense", "debit", "discounts_given", false, "", true},
	{"5900", "Round Off", "expense", "other_expense", "debit", "round_off", false, "", true},
}

type unitSeed struct {
	Code      string
	Name      string
	Symbol    string
	Precision int16
}

var standardUnits = []unitSeed{
	{"PCS", "Pieces", "pcs", 0},
	{"KG", "Kilogram", "kg", 3},
	{"G", "Gram", "g", 3},
	{"L", "Litre", "l", 3},
	{"ML", "Millilitre", "ml", 3},
	{"M", "Metre", "m", 2},
	{"BOX", "Box", "box", 0},
	{"PKT", "Packet", "pkt", 0},
}

type gstSlab struct {
	Rate int
	Code string
	Name string
}

var gstSlabs = []gstSlab{
	{0, "GST0", "GST 0%"},
	{5, "GST5", "GST 5%"},
	{12, "GST12", "GST 12%"},
	{18, "GST18", "GST 18%"},
	{28, "GST28", "GST 28%"},
}

type seriesSeed struct {
	DocumentType string
	Prefix       string
}

var requiredSeries = []seriesSeed{
	{"sales_invoice", "INV"},
	{"purchase_bill", "PB"},
	{"estimate", "EST"},
	{"payment", "PMT"},
	{"receipt", "RCT"},
	{"expense", "EXP"},
	{"journal", "JV"},
	{"credit_note", "CN"},
	{"debit_note", "DN"},
	{"stock_adjustment", "STK"},
}

// Initialize creates every mandatory default record. The caller owns the
// surrounding transaction.
func (i *Initializer) Initialize(
	ctx context.Context,
	tx pgx.Tx,
	company *Company,
	registration *TaxRegistration,
	fyStart, fyEnd time.Time,
	invoicePrefix string,
	padding int,
	startNumber int64,
	dateFormat string,
	actorID string,
) error {
	tenantID, companyID := company.TenantID, company.ID

	// Default branch.
	var branchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO branches
			(tenant_id, company_id, code, name, address_line1, address_line2, city, district,
			 state_code, postal_code, country_code, phone_e164, default_tax_registration_id, is_default, status)
		VALUES ($1,$2,'HO',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,true,'active')
		RETURNING id`,
		tenantID, companyID, company.TradeName, company.AddressLine1, company.AddressLine2,
		company.City, company.District, company.StateCode, company.PostalCode, company.CountryCode,
		company.Phone, registrationID(registration)).Scan(&branchID); err != nil {
		return fmt.Errorf("create default branch: %w", err)
	}

	// Default warehouse.
	var warehouseID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO warehouses (tenant_id, company_id, branch_id, code, name, warehouse_kind, is_default, status)
		VALUES ($1,$2,$3,'WH-001','Main Warehouse','physical',true,'active')
		RETURNING id`, tenantID, companyID, branchID).Scan(&warehouseID); err != nil {
		return fmt.Errorf("create default warehouse: %w", err)
	}

	// Chart of accounts and system mappings.
	cashAccountID, err := i.seedAccounts(ctx, tx, tenantID, companyID)
	if err != nil {
		return err
	}

	// Default cash account.
	var cashBankID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO cash_bank_accounts (tenant_id, company_id, account_id, account_kind, display_name, is_default, status)
		VALUES ($1,$2,$3,'cash','Cash',true,'active')
		RETURNING id`, tenantID, companyID, cashAccountID).Scan(&cashBankID); err != nil {
		return fmt.Errorf("create default cash account: %w", err)
	}

	// Financial year and accounting periods.
	var fyID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO financial_years (tenant_id, company_id, name, start_date, end_date, status, is_current)
		VALUES ($1,$2,$3,$4,$5,'open',true) RETURNING id`,
		tenantID, companyID, fyStart.Format("2006-04")+" to "+fyEnd.AddDate(0, 0, -1).Format("2006-03"),
		fyStart, fyEnd).Scan(&fyID); err != nil {
		return fmt.Errorf("create financial year: %w", err)
	}
	if err := i.seedPeriods(ctx, tx, tenantID, companyID, fyID, fyStart, fyEnd); err != nil {
		return err
	}

	// Standard units.
	for _, unit := range standardUnits {
		if _, err := tx.Exec(ctx, `
			INSERT INTO units (tenant_id, company_id, unit_code, name, symbol, precision_scale, is_system_seed, status)
			VALUES ($1,$2,$3,$4,$5,$6,true,'active')`,
			tenantID, companyID, unit.Code, unit.Name, unit.Symbol, unit.Precision); err != nil {
			return fmt.Errorf("seed unit %s: %w", unit.Code, err)
		}
	}

	// Document numbering series.
	for _, series := range requiredSeries {
		prefix := series.Prefix
		next := int64(1)
		if series.DocumentType == "sales_invoice" && invoicePrefix != "" {
			prefix = invoicePrefix
			next = startNumber
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_number_series
				(tenant_id, company_id, document_type, financial_year_id, branch_id, tax_registration_id,
				 prefix, suffix, padding_width, next_number, is_default, is_active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'',$8,$9,true,true)`,
			tenantID, companyID, series.DocumentType, fyID, branchID, registrationID(registration),
			prefix, padding, next); err != nil {
			return fmt.Errorf("create number series %s: %w", series.DocumentType, err)
		}
	}

	// Approved tax configuration (independent India GST slabs).
	if registration != nil {
		if err := i.seedTaxConfig(ctx, tx, tenantID, companyID, fyStart); err != nil {
			return err
		}
	}

	// Company settings.
	if _, err := tx.Exec(ctx, `
		INSERT INTO company_settings
			(tenant_id, company_id, default_branch_id, default_warehouse_id, default_cash_bank_account_id,
			 date_format, locale)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		tenantID, companyID, branchID, warehouseID, cashBankID, dateFormat, "en-IN"); err != nil {
		return fmt.Errorf("create company settings: %w", err)
	}

	// Default invoice template and immutable first version.
	var templateID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO document_templates (tenant_id, company_id, document_type, name, is_default, status, created_by)
		VALUES ($1,$2,'sales_invoice','Default Sales Invoice',true,'active',$3) RETURNING id`,
		tenantID, companyID, actorID).Scan(&templateID); err != nil {
		return fmt.Errorf("create default template: %w", err)
	}
	layout, _ := json.Marshal(map[string]any{
		"sections": []string{"header", "party", "items", "totals", "footer"},
		"paper":    "A4",
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO document_template_versions
			(tenant_id, company_id, template_id, version_number, config_schema_version, layout_config, status, created_by, published_at)
		VALUES ($1,$2,$3,1,1,$4::jsonb,'published',$5,now())`,
		tenantID, companyID, templateID, layout, actorID); err != nil {
		return fmt.Errorf("create default template version: %w", err)
	}

	return nil
}

func (i *Initializer) seedAccounts(ctx context.Context, tx pgx.Tx, tenantID, companyID string) (string, error) {
	var cashAccountID string
	for _, a := range chartOfAccounts {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO accounts
				(tenant_id, company_id, account_code, system_code, name, account_type, account_subtype,
				 normal_balance, is_control_account, control_account_role, allow_manual_posting, is_system_account, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),$11,true,'active')
			RETURNING id`,
			tenantID, companyID, a.Code, a.SystemCode, a.Name, a.Type, a.Subtype,
			a.Normal, a.Control, a.ControlRole, a.AllowManual).Scan(&id)
		if err != nil {
			return "", fmt.Errorf("seed account %s: %w", a.Code, err)
		}
		if a.SystemCode == "cash" {
			cashAccountID = id
		}
	}
	if cashAccountID == "" {
		return "", fmt.Errorf("cash account was not seeded")
	}
	return cashAccountID, nil
}

func (i *Initializer) seedPeriods(ctx context.Context, tx pgx.Tx, tenantID, companyID, fyID string, start, end time.Time) error {
	periodStart := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	for number := 1; number <= 12; number++ {
		next := periodStart.AddDate(0, 1, 0)
		periodEnd := next.AddDate(0, 0, -1)
		if !periodEnd.Before(end) {
			periodEnd = end.AddDate(0, 0, -1)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO accounting_periods
				(tenant_id, company_id, financial_year_id, period_number, name, start_date, end_date, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'open')`,
			tenantID, companyID, fyID, number, periodStart.Format("Jan 2006"), periodStart, periodEnd); err != nil {
			return fmt.Errorf("seed accounting period %d: %w", number, err)
		}
		periodStart = next
	}
	return nil
}

func (i *Initializer) seedTaxConfig(ctx context.Context, tx pgx.Tx, tenantID, companyID string, validFrom time.Time) error {
	for _, slab := range gstSlabs {
		var taxCodeID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO tax_codes (tenant_id, company_id, code, name, tax_family, status)
			VALUES ($1,$2,$3,$4,'gst','active') RETURNING id`,
			tenantID, companyID, slab.Code, slab.Name).Scan(&taxCodeID); err != nil {
			return fmt.Errorf("seed tax code %s: %w", slab.Code, err)
		}
		var versionID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO tax_code_versions
				(tenant_id, company_id, tax_code_id, version_number, tax_treatment, valid_from, description)
			VALUES ($1,$2,$3,1,$4,$5,$6) RETURNING id`,
			tenantID, companyID, taxCodeID, taxTreatment(slab.Rate), validFrom, slab.Name).Scan(&versionID); err != nil {
			return fmt.Errorf("seed tax version %s: %w", slab.Code, err)
		}
		components := []struct {
			scope string
			code  string
			name  string
			// rateHundredths stores the percentage in hundredths (2.5% = 250)
			// so seeding never relies on binary floating point.
			rateHundredths int
		}{
			{"intra_state", "CGST", "Central GST", slab.Rate * 50},
			{"intra_state", "SGST", "State GST", slab.Rate * 50},
			{"inter_state", "IGST", "Integrated GST", slab.Rate * 100},
		}
		for order, component := range components {
			rate := fmt.Sprintf("%d.%02d", component.rateHundredths/100, component.rateHundredths%100)
			if _, err := tx.Exec(ctx, `
				INSERT INTO tax_code_components
					(tenant_id, company_id, tax_code_version_id, supply_scope, component_code,
					 component_name, calculation_method, rate_percentage, calculation_order)
				VALUES ($1,$2,$3,$4,$5,$6,'percentage',$7,$8)`,
				tenantID, companyID, versionID, component.scope, component.code, component.name,
				rate, order+1); err != nil {
				return fmt.Errorf("seed tax component %s: %w", component.code, err)
			}
		}
	}
	return nil
}

func taxTreatment(rate int) string {
	if rate == 0 {
		return "nil_rated"
	}
	return "taxable"
}

func registrationID(registration *TaxRegistration) any {
	if registration == nil {
		return nil
	}
	return registration.ID
}
