package authorization

import "testing"

func TestDefaultRolesAreConsistent(t *testing.T) {
	known := make(map[string]bool, len(AllPermissions))
	for _, permission := range AllPermissions {
		if permission == "" {
			t.Fatal("permission catalogue contains an empty code")
		}
		if known[permission] {
			t.Fatalf("duplicate permission %q", permission)
		}
		known[permission] = true
	}

	codes := make(map[string]bool, len(DefaultRoles))
	ownerPermissions := 0
	for _, role := range DefaultRoles {
		if role.Code == "" || role.Name == "" {
			t.Fatal("system role is missing a code or name")
		}
		if codes[role.Code] {
			t.Fatalf("duplicate system role %q", role.Code)
		}
		codes[role.Code] = true
		for _, permission := range role.Permissions {
			if !known[permission] {
				t.Fatalf("role %q references unknown permission %q", role.Code, permission)
			}
		}
		if role.Code == RoleOwner {
			ownerPermissions = len(role.Permissions)
		}
	}
	if ownerPermissions != len(AllPermissions) {
		t.Fatalf("owner must hold every permission: got %d of %d", ownerPermissions, len(AllPermissions))
	}
	if !codes[RoleAdmin] || !codes[RoleAccountant] {
		t.Fatal("expected baseline admin and accountant roles")
	}
}

func TestAdminCannotManageRoles(t *testing.T) {
	for _, role := range DefaultRoles {
		if role.Code != RoleAdmin {
			continue
		}
		for _, permission := range role.Permissions {
			if permission == WorkspaceRolesManage {
				t.Fatal("admin must not manage roles")
			}
		}
		return
	}
	t.Fatal("admin role not found")
}
