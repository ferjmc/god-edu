package domain

import "testing"

func TestUserRole_Valid(t *testing.T) {
	cases := []struct {
		role UserRole
		want bool
	}{
		{RoleAdmin, true},
		{RoleCommunityMember, true},
		{RolePublicMember, true},
		{RolePaidMember, true},
		{"", false},
		{"SUPERADMIN", false},
		{"admin", false}, // los valores válidos son mayúsculas, esto no debe colarse
	}

	for _, c := range cases {
		if got := c.role.Valid(); got != c.want {
			t.Errorf("UserRole(%q).Valid() = %v, quería %v", c.role, got, c.want)
		}
	}
}
