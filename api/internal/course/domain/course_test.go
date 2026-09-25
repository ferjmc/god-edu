package domain

import (
	"testing"

	userdomain "github.com/ferjmc/god-edu/api/internal/user/domain"
)

func TestCourse_VisibleTo(t *testing.T) {
	cases := []struct {
		name         string
		visibleRoles []userdomain.UserRole
		role         userdomain.UserRole
		want         bool
	}{
		{
			name:         "admin siempre puede, aunque el rol no esté en la lista",
			visibleRoles: []userdomain.UserRole{userdomain.RolePaidMember},
			role:         userdomain.RoleAdmin,
			want:         true,
		},
		{
			name:         "sin restricción configurada, cualquier rol puede",
			visibleRoles: nil,
			role:         userdomain.RolePublicMember,
			want:         true,
		},
		{
			name:         "rol incluido en la restricción",
			visibleRoles: []userdomain.UserRole{userdomain.RolePaidMember},
			role:         userdomain.RolePaidMember,
			want:         true,
		},
		{
			name:         "rol no incluido en la restricción",
			visibleRoles: []userdomain.UserRole{userdomain.RolePaidMember},
			role:         userdomain.RolePublicMember,
			want:         false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			course := Course{VisibleRoles: c.visibleRoles}
			if got := course.VisibleTo(c.role); got != c.want {
				t.Errorf("VisibleTo(%v) con VisibleRoles=%v = %v, quería %v", c.role, c.visibleRoles, got, c.want)
			}
		})
	}
}
