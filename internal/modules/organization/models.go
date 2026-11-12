package organization

import "time"

// Company is the primary business profile for a workspace.
type Company struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	TradeName    string    `json:"trade_name"`
	LegalName    *string   `json:"legal_name,omitempty"`
	PAN          *string   `json:"pan,omitempty"`
	BusinessType *string   `json:"business_type,omitempty"`
	GSTProfile   string    `json:"gst_profile"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	Website      *string   `json:"website,omitempty"`
	AddressLine1 *string   `json:"address_line1,omitempty"`
	AddressLine2 *string   `json:"address_line2,omitempty"`
	City         *string   `json:"city,omitempty"`
	District     *string   `json:"district,omitempty"`
	StateCode    *string   `json:"state_code,omitempty"`
	PostalCode   *string   `json:"postal_code,omitempty"`
	CountryCode  string    `json:"country_code"`
	BaseCurrency string    `json:"base_currency"`
	Timezone     string    `json:"timezone"`
	IsPrimary    bool      `json:"is_primary"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	RowVersion   int64     `json:"row_version"`
}

// TaxRegistration is a company GST registration.
type TaxRegistration struct {
	ID               string     `json:"id"`
	GSTIN            string     `json:"gstin"`
	StateCode        string     `json:"state_code"`
	RegistrationType string     `json:"registration_type"`
	LegalName        *string    `json:"legal_name,omitempty"`
	TradeName        *string    `json:"trade_name,omitempty"`
	AddressLine1     *string    `json:"address_line1,omitempty"`
	AddressLine2     *string    `json:"address_line2,omitempty"`
	City             *string    `json:"city,omitempty"`
	District         *string    `json:"district,omitempty"`
	PostalCode       *string    `json:"postal_code,omitempty"`
	CountryCode      string     `json:"country_code"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidTo          *time.Time `json:"valid_to,omitempty"`
	IsDefault        bool       `json:"is_default"`
	Status           string     `json:"status"`
}

// Settings is company configuration.
type Settings struct {
	DefaultBranchID          string  `json:"default_branch_id"`
	DefaultWarehouseID       string  `json:"default_warehouse_id"`
	DefaultCashBankAccountID *string `json:"default_cash_bank_account_id,omitempty"`
	DefaultPriceIncludesTax  bool    `json:"default_price_includes_tax"`
	NegativeStockPolicy      string  `json:"negative_stock_policy"`
	InventoryCostMethod      string  `json:"inventory_cost_method"`
	DefaultCreditDays        int     `json:"default_credit_days"`
	CreditLimitAction        string  `json:"credit_limit_action"`
	AmountDisplayScale       int16   `json:"amount_display_scale"`
	QuantityDisplayScale     int16   `json:"quantity_display_scale"`
	DateFormat               string  `json:"date_format"`
	Locale                   string  `json:"locale"`
	RowVersion               int64   `json:"row_version"`
}

// ChecklistItem is one onboarding readiness check.
type ChecklistItem struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Mandatory  bool   `json:"mandatory"`
	Complete   bool   `json:"complete"`
	Message    string `json:"message"`
	ResourceID string `json:"resource_id,omitempty"`
}

// OnboardingStatus is the authoritative readiness response.
type OnboardingStatus struct {
	Company   *Company        `json:"company"`
	Ready     bool            `json:"ready"`
	Checklist []ChecklistItem `json:"checklist"`
	Completed int             `json:"completed"`
	Mandatory int             `json:"mandatory"`
}

// Overview is the configuration-only business overview for the next frontend phase.
type Overview struct {
	User            map[string]any    `json:"user"`
	Workspace       map[string]any    `json:"workspace"`
	Membership      map[string]any    `json:"membership"`
	PrimaryCompany  *Company          `json:"primary_company"`
	CompanySettings *Settings         `json:"company_settings,omitempty"`
	Readiness       *OnboardingStatus `json:"readiness"`
	Permissions     map[string]bool   `json:"permissions"`
}

// CreateCompanyInput is the validated company onboarding request.
type CreateCompanyInput struct {
	TradeName    string
	LegalName    string
	BusinessType string
	PAN          string
	Email        string
	Phone        string
	Website      string

	AddressLine1 string
	AddressLine2 string
	City         string
	District     string
	StateCode    string
	PostalCode   string
	CountryCode  string

	GSTProfile            string
	RegistrationType      string
	GSTIN                 string
	RegistrationState     string
	RegistrationLegalName string
	RegistrationTradeName string

	BaseCurrency       string
	Timezone           string
	FinancialYearStart string
	FinancialYearEnd   string
	DateFormat         string

	InvoicePrefix  string
	NumberPadding  int
	StartingNumber int64
}
