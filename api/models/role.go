package models

// UserRole clasifica el nivel de acceso de un usuario. Un usuario tiene
// exactamente un rol (columna users.role); un curso puede restringir su
// detalle a un subconjunto arbitrario de roles (tabla
// course_visible_roles) — ver Course.VisibleTo.
type UserRole string

const (
	RoleAdmin           UserRole = "ADMIN"
	RoleCommunityMember UserRole = "COMMUNITY_MEMBER"
	RolePublicMember    UserRole = "PUBLIC_MEMBER"
	RolePaidMember      UserRole = "PAID_MEMBER"
)

// Valid indica si role es uno de los cuatro roles conocidos. La usa
// AdminHandler.UpdateUserRole para rechazar un rol inventado antes de
// tocar la base (el CHECK constraint de la columna igual lo frenaría, pero
// devolver 400 con mensaje claro es mejor que un 500 de Postgres).
func (r UserRole) Valid() bool {
	switch r {
	case RoleAdmin, RoleCommunityMember, RolePublicMember, RolePaidMember:
		return true
	}
	return false
}
