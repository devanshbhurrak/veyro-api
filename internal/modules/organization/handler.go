package organization

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/modules/identity"
	"backend/internal/modules/tenancy"
	"backend/internal/platform/auth"
	"backend/internal/platform/middleware"
	"backend/internal/shared/apperr"
	"backend/internal/shared/response"
)

// Handler exposes company, onboarding, and overview endpoints.
type Handler struct {
	svc     *Service
	tenancy *tenancy.Service
	users   *identity.Service
}

func NewHandler(svc *Service, tenancySvc *tenancy.Service, users *identity.Service) *Handler {
	return &Handler{svc: svc, tenancy: tenancySvc, users: users}
}

type createCompanyRequest struct {
	TradeName    string `json:"trade_name"`
	LegalName    string `json:"legal_name"`
	BusinessType string `json:"business_type"`
	PAN          string `json:"pan"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Website      string `json:"website"`

	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	District     string `json:"district"`
	StateCode    string `json:"state_code"`
	PostalCode   string `json:"postal_code"`
	CountryCode  string `json:"country_code"`

	GSTProfile            string `json:"gst_profile"`
	RegistrationType      string `json:"registration_type"`
	GSTIN                 string `json:"gstin"`
	RegistrationState     string `json:"registration_state"`
	RegistrationLegalName string `json:"registration_legal_name"`
	RegistrationTradeName string `json:"registration_trade_name"`

	BaseCurrency       string `json:"base_currency"`
	Timezone           string `json:"timezone"`
	FinancialYearStart string `json:"financial_year_start"`
	FinancialYearEnd   string `json:"financial_year_end"`
	DateFormat         string `json:"date_format"`

	InvoicePrefix  string `json:"invoice_prefix"`
	NumberPadding  int    `json:"number_padding"`
	StartingNumber int64  `json:"starting_number"`
}

// CreateCompany onboards the workspace's primary company.
func (h *Handler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req createCompanyRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	result, err := h.svc.CreateCompany(r.Context(), identity.TenantID, identity.UserID,
		r.Header.Get("Idempotency-Key"), CreateCompanyInput{
			TradeName: req.TradeName, LegalName: req.LegalName, BusinessType: req.BusinessType,
			PAN: req.PAN, Email: req.Email, Phone: req.Phone, Website: req.Website,
			AddressLine1: req.AddressLine1, AddressLine2: req.AddressLine2, City: req.City,
			District: req.District, StateCode: req.StateCode, PostalCode: req.PostalCode,
			CountryCode: req.CountryCode, GSTProfile: req.GSTProfile,
			RegistrationType: req.RegistrationType, GSTIN: req.GSTIN,
			RegistrationState: req.RegistrationState, RegistrationLegalName: req.RegistrationLegalName,
			RegistrationTradeName: req.RegistrationTradeName, BaseCurrency: req.BaseCurrency,
			Timezone: req.Timezone, FinancialYearStart: req.FinancialYearStart,
			FinancialYearEnd: req.FinancialYearEnd, DateFormat: req.DateFormat,
			InvoicePrefix: req.InvoicePrefix, NumberPadding: req.NumberPadding,
			StartingNumber: req.StartingNumber,
		})
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	status := http.StatusCreated
	if result.AlreadyExisted {
		status = http.StatusOK
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), status, map[string]any{
		"company": result.Company, "tax_registration": result.Registration,
	})
}

// GetCompany returns the workspace's primary company.
func (h *Handler) GetCompany(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if company == nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.NotFound("company_not_found", "Complete company onboarding to continue."))
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"company": company})
}

type updateCompanyRequest struct {
	TradeName    *string `json:"trade_name"`
	LegalName    *string `json:"legal_name"`
	BusinessType *string `json:"business_type"`
	PAN          *string `json:"pan"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
	Website      *string `json:"website"`
	AddressLine1 *string `json:"address_line1"`
	AddressLine2 *string `json:"address_line2"`
	City         *string `json:"city"`
	District     *string `json:"district"`
	StateCode    *string `json:"state_code"`
	PostalCode   *string `json:"postal_code"`
	CountryCode  *string `json:"country_code"`
	RowVersion   int64   `json:"row_version"`
}

// UpdateCompany applies a partial company update.
func (h *Handler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	var req updateCompanyRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if company == nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.NotFound("company_not_found", "Complete company onboarding to continue."))
		return
	}
	updated, err := h.svc.UpdateCompany(r.Context(), identity.TenantID, company.ID, identity.UserID,
		UpdateCompanyInput{
			TradeName: req.TradeName, LegalName: req.LegalName, BusinessType: req.BusinessType,
			PAN: req.PAN, Email: req.Email, Phone: req.Phone, Website: req.Website,
			AddressLine1: req.AddressLine1, AddressLine2: req.AddressLine2, City: req.City,
			District: req.District, StateCode: req.StateCode, PostalCode: req.PostalCode,
			CountryCode: req.CountryCode, RowVersion: req.RowVersion,
		})
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"company": updated})
}

// GetSettings returns company settings.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil || company == nil {
		if err == nil {
			err = apperr.NotFound("company_not_found", "Complete company onboarding to continue.")
		}
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	settings, err := h.svc.Settings(r.Context(), identity.TenantID, company.ID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"settings": settings})
}

type updateSettingsRequest struct {
	DefaultPriceIncludesTax *bool   `json:"default_price_includes_tax"`
	NegativeStockPolicy     *string `json:"negative_stock_policy"`
	InventoryCostMethod     *string `json:"inventory_cost_method"`
	DefaultCreditDays       *int    `json:"default_credit_days"`
	CreditLimitAction       *string `json:"credit_limit_action"`
	AmountDisplayScale      *int16  `json:"amount_display_scale"`
	QuantityDisplayScale    *int16  `json:"quantity_display_scale"`
	DateFormat              *string `json:"date_format"`
	Locale                  *string `json:"locale"`
	RowVersion              int64   `json:"row_version"`
}

// UpdateSettings applies a version-checked settings update.
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil || company == nil {
		if err == nil {
			err = apperr.NotFound("company_not_found", "Complete company onboarding to continue.")
		}
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	settings, err := h.svc.UpdateSettings(r.Context(), identity.TenantID, company.ID, identity.UserID,
		UpdateSettingsInput{
			DefaultPriceIncludesTax: req.DefaultPriceIncludesTax,
			NegativeStockPolicy:     req.NegativeStockPolicy,
			InventoryCostMethod:     req.InventoryCostMethod,
			DefaultCreditDays:       req.DefaultCreditDays,
			CreditLimitAction:       req.CreditLimitAction,
			AmountDisplayScale:      req.AmountDisplayScale,
			QuantityDisplayScale:    req.QuantityDisplayScale,
			DateFormat:              req.DateFormat,
			Locale:                  req.Locale,
			RowVersion:              req.RowVersion,
		})
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"settings": settings})
}

// ListTaxRegistrations returns all GST registrations for the primary company.
func (h *Handler) ListTaxRegistrations(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil || company == nil {
		if err == nil {
			err = apperr.NotFound("company_not_found", "Complete company onboarding to continue.")
		}
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	registrations, err := h.svc.TaxRegistrations(r.Context(), identity.TenantID, company.ID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"tax_registrations": registrations})
}

type taxRegistrationRequest struct {
	GSTIN            string `json:"gstin"`
	StateCode        string `json:"state_code"`
	RegistrationType string `json:"registration_type"`
	LegalName        string `json:"legal_name"`
	TradeName        string `json:"trade_name"`
	AddressLine1     string `json:"address_line1"`
	AddressLine2     string `json:"address_line2"`
	City             string `json:"city"`
	District         string `json:"district"`
	PostalCode       string `json:"postal_code"`
	CountryCode      string `json:"country_code"`
	ValidFrom        string `json:"valid_from"`
	ValidTo          string `json:"valid_to"`
	IsDefault        bool   `json:"is_default"`
}

func (req taxRegistrationRequest) input() TaxRegistrationInput {
	return TaxRegistrationInput{
		GSTIN: req.GSTIN, StateCode: req.StateCode, RegistrationType: req.RegistrationType,
		LegalName: req.LegalName, TradeName: req.TradeName, AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2, City: req.City, District: req.District,
		PostalCode: req.PostalCode, CountryCode: req.CountryCode, ValidFrom: req.ValidFrom,
		ValidTo: req.ValidTo, IsDefault: req.IsDefault,
	}
}

// CreateTaxRegistration adds a GST registration.
func (h *Handler) CreateTaxRegistration(w http.ResponseWriter, r *http.Request) {
	var req taxRegistrationRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil || company == nil {
		if err == nil {
			err = apperr.NotFound("company_not_found", "Complete company onboarding to continue.")
		}
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	registration, err := h.svc.CreateTaxRegistration(r.Context(), identity.TenantID, company.ID,
		identity.UserID, req.input())
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusCreated,
		map[string]any{"tax_registration": registration})
}

// UpdateTaxRegistration updates a GST registration.
func (h *Handler) UpdateTaxRegistration(w http.ResponseWriter, r *http.Request) {
	var req taxRegistrationRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil || company == nil {
		if err == nil {
			err = apperr.NotFound("company_not_found", "Complete company onboarding to continue.")
		}
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	registration, err := h.svc.UpdateTaxRegistration(r.Context(), identity.TenantID, company.ID,
		chi.URLParam(r, "registrationID"), identity.UserID, req.input())
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"tax_registration": registration})
}

// OnboardingStatus returns the readiness checklist.
func (h *Handler) OnboardingStatus(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	status, err := h.svc.OnboardingStatus(r.Context(), identity.TenantID, "")
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, status)
}

// Overview returns configuration-only context for the authenticated session.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	workspace, err := h.tenancy.GetWorkspace(r.Context(), identity.TenantID, identity.UserID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	member, err := h.tenancy.GetMember(r.Context(), identity.TenantID, identity.MembershipID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	company, err := h.svc.PrimaryCompany(r.Context(), identity.TenantID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	overview := Overview{
		User: map[string]any{
			"id": identity.UserID, "email": identity.Email, "full_name": identity.FullName,
		},
		Workspace: map[string]any{
			"id": workspace.ID, "name": workspace.Name, "slug": workspace.Slug,
			"status": workspace.Status, "timezone": workspace.Timezone, "locale": workspace.Locale,
			"roles": workspace.Roles,
		},
		Membership: map[string]any{
			"id": member.ID, "status": member.Status, "roles": member.Roles,
		},
		PrimaryCompany: company,
		Permissions:    permissionMap(identity),
	}
	if company != nil {
		settings, err := h.svc.Settings(r.Context(), identity.TenantID, company.ID)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
			return
		}
		readiness, err := h.svc.OnboardingStatus(r.Context(), identity.TenantID, company.ID)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
			return
		}
		overview.CompanySettings = settings
		overview.Readiness = readiness
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, overview)
}

func permissionMap(identity *auth.Identity) map[string]bool {
	permissions := make(map[string]bool, len(identity.Permissions))
	for code := range identity.Permissions {
		permissions[code] = true
	}
	return permissions
}
