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
