package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/httpkit"
	"github.com/ferjmc/god-edu/api/internal/user/application"
	"github.com/ferjmc/god-edu/api/internal/user/domain"
)

// AdminHandler agrupa los endpoints de administración de usuarios: listar
// y cambiar roles. Detrás de auth.RequireAuth + RequireAdmin (ver
// newRouter en main.go) — no hay bloqueo adicional acá adentro, el
// middleware ya filtró.
type AdminHandler struct {
	Users *application.Service
}

// --- Listar / ver usuarios ---

// adminUserResponse es la representación de un usuario para el panel
// admin: lleva todo lo que un admin necesita para decidir qué tocar — estado de
// usuario, roles y fecha de alta.
type adminUserResponse struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	AuthProvider  string    `json:"authProvider"`
	EmailVerified bool      `json:"emailVerified"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toAdminUserResponse(u domain.User) adminUserResponse {
	return adminUserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		AuthProvider:  string(u.AuthProvider),
		EmailVerified: u.EmailVerified,
		Role:          string(u.Role),
		CreatedAt:     u.CreatedAt,
	}
}

// ListUsers devuelve todos los usuarios (activos e inactivos) para el
// panel admin.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.ListAll(r.Context())
	if err != nil {
		log.Printf("admin: listando usuarios: %v", err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudieron obtener los usuarios")
		return
	}

	response := make([]adminUserResponse, len(users))
	for i, u := range users {
		response[i] = toAdminUserResponse(u)
	}

	httpkit.WriteJSON(w, http.StatusOK, response)
}

type updateUserRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserRole cambia el rol de un usuario. Las dos protecciones para no
// dejar la plataforma sin ningún ADMIN (no auto-cambiarse el rol, no
// degradar al último admin) viven consolidadas en application.Service.UpdateRole
// — este handler solo traduce sus sentinels de error a la respuesta HTTP.
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "id de usuario inválido")
		return
	}

	// Chequeo temprano y barato: application.Service.UpdateRole igual lo
	// repite (es la protección real, la que vale para cualquier caller
	// futuro), pero hacerlo acá antes también preserva el orden de
	// validación de siempre — un intento de auto-cambio con body inválido
	// sigue devolviendo el error de auto-cambio, no un genérico "cuerpo
	// inválido" por haber cambiado el orden de los chequeos en esta
	// migración.
	callerID, _ := auth.UserIDFromContext(r.Context()) // RequireAuth ya lo garantiza
	if callerID == id {
		httpkit.WriteError(w, http.StatusBadRequest, "no podés cambiar tu propio rol")
		return
	}

	var req updateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	role := domain.UserRole(req.Role)
	if !role.Valid() {
		httpkit.WriteError(w, http.StatusBadRequest, "rol inválido")
		return
	}

	if err := h.Users.UpdateRole(r.Context(), callerID, id, role); err != nil {
		if httpkit.HandleNotFound(w, err, "usuario no encontrado") {
			return
		}
		if errors.Is(err, domain.ErrSelfRoleChange) {
			// Inalcanzable hoy por acá (el chequeo temprano de arriba ya
			// cortó el caso callerID == id) — se deja de todos modos porque
			// Service.UpdateRole puede devolver este error a CUALQUIER
			// caller, no solo a este handler; si el chequeo temprano de
			// arriba se sacara alguna vez, esta rama sigue traduciendo el
			// error correctamente en vez de caer al 500 genérico de abajo.
			httpkit.WriteError(w, http.StatusBadRequest, "no podés cambiar tu propio rol")
			return
		}
		if errors.Is(err, domain.ErrLastAdmin) {
			httpkit.WriteError(w, http.StatusBadRequest, "no podés quitarle el rol de admin al último administrador")
			return
		}
		log.Printf("admin: actualizando rol de usuario %d: %v", id, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar el rol")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
