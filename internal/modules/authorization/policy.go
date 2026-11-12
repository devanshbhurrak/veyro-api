// Package authorization implements server-side role-based access control,
// trusted request context and permission middleware.
package authorization

// Permission codes from the platform-managed catalogue.
const (
	WorkspaceView          = "workspace.view"
	WorkspaceUpdate        = "workspace.update"
	WorkspaceMembersView   = "workspace.members.view"
	WorkspaceMembersInvite = "workspace.members.invite"
	WorkspaceMembersManage = "workspace.members.manage"
	WorkspaceRolesView     = "workspace.roles.view"
	WorkspaceRolesManage   = "workspace.roles.manage"

	CompanyView           = "company.view"
	CompanyCreate         = "company.create"
	CompanyUpdate         = "company.update"
	CompanySettingsManage = "company.settings.manage"
	CompanyTaxManage      = "company.tax.manage"

	AccountingAccountsView   = "accounting.accounts.view"
	AccountingSettingsManage = "accounting.settings.manage"

	SalesInvoiceView    = "sales.invoice.view"
	SalesInvoiceCreate  = "sales.invoice.create"
	SalesInvoicePost    = "sales.invoice.post"
	SalesInvoiceReverse = "sales.invoice.reverse"

	PurchaseBillView   = "purchase.bill.view"
	PurchaseBillCreate = "purchase.bill.create"
	PurchaseBillPost   = "purchase.bill.post"

	InventoryStockView   = "inventory.stock.view"
	InventoryStockAdjust = "inventory.stock.adjust"

	ReportsView   = "reports.view"
	ReportsExport = "reports.export"
)

// AllPermissions is the full, approved catalogue.
var AllPermissions = []string{
	WorkspaceView, WorkspaceUpdate, WorkspaceMembersView, WorkspaceMembersInvite,
	WorkspaceMembersManage, WorkspaceRolesView, WorkspaceRolesManage,
	CompanyView, CompanyCreate, CompanyUpdate, CompanySettingsManage, CompanyTaxManage,
	AccountingAccountsView, AccountingSettingsManage,
	SalesInvoiceView, SalesInvoiceCreate, SalesInvoicePost, SalesInvoiceReverse,
	PurchaseBillView, PurchaseBillCreate, PurchaseBillPost,
	InventoryStockView, InventoryStockAdjust,
	ReportsView, ReportsExport,
}

// RoleCodes are the stable system role codes seeded for every workspace.
const (
	RoleOwner            = "owner"
	RoleAdmin            = "admin"
	RoleAccountant       = "accountant"
	RoleSalesperson      = "salesperson"
	RoleInventoryManager = "inventory_manager"
	RolePurchaseStaff    = "purchase_staff"
)

// SystemRole describes a default role and its permission set.
type SystemRole struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

// DefaultRoles is the controlled baseline role set created per workspace.
var DefaultRoles = []SystemRole{
	{Code: RoleOwner, Name: "Owner", Description: "Full control of the workspace", Permissions: AllPermissions},
	{Code: RoleAdmin, Name: "Admin", Description: "Administrative access without role administration", Permissions: []string{
		WorkspaceView, WorkspaceUpdate, WorkspaceMembersView, WorkspaceMembersInvite, WorkspaceMembersManage,
		WorkspaceRolesView, CompanyView, CompanyCreate, CompanyUpdate, CompanySettingsManage, CompanyTaxManage,
		AccountingAccountsView, AccountingSettingsManage,
		SalesInvoiceView, SalesInvoiceCreate, SalesInvoicePost, SalesInvoiceReverse,
		PurchaseBillView, PurchaseBillCreate, PurchaseBillPost,
		InventoryStockView, InventoryStockAdjust, ReportsView, ReportsExport,
	}},
	{Code: RoleAccountant, Name: "Accountant", Description: "Accounting and reporting access", Permissions: []string{
		WorkspaceView, CompanyView, AccountingAccountsView, AccountingSettingsManage,
		SalesInvoiceView, SalesInvoiceCreate, SalesInvoicePost,
		PurchaseBillView, PurchaseBillCreate, PurchaseBillPost, ReportsView, ReportsExport,
	}},
	{Code: RoleSalesperson, Name: "Salesperson / Cashier", Description: "Sales and customer billing access", Permissions: []string{
		WorkspaceView, CompanyView, SalesInvoiceView, SalesInvoiceCreate, InventoryStockView, ReportsView,
	}},
	{Code: RoleInventoryManager, Name: "Inventory Manager", Description: "Inventory and stock access", Permissions: []string{
		WorkspaceView, CompanyView, InventoryStockView, InventoryStockAdjust, ReportsView,
	}},
	{Code: RolePurchaseStaff, Name: "Purchase Staff", Description: "Purchase and supplier access", Permissions: []string{
		WorkspaceView, CompanyView, PurchaseBillView, PurchaseBillCreate, InventoryStockView, ReportsView,
	}},
}
