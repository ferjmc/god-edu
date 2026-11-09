package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/httpkit"
	"github.com/ferjmc/god-edu/api/internal/lesson/domain"
)

// Clases: agrupan las lecciones ("sesiones") de un curso. Mismo esquema de
// identificador público que las lecciones: order_index dentro del curso,
// nunca el id interno.

type classResponse struct {
	Order int    `json:"order"`
	Title string `json:"title"`
}

type createClassRequest struct {
	Title string `json:"title"`
	Order int    `json:"order"`
}

// CreateClass agrega una clase a un curso.
func (h *AdminHandler) CreateClass(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return
	}

	var req createClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}
	if req.Order < 1 {
		httpkit.WriteError(w, http.StatusBadRequest, "order debe ser 1 o mayor")
		return
	}

	class, err := h.Lessons.CreateClass(r.Context(), domain.Class{
		CourseID:   course.ID,
		Title:      title,
		OrderIndex: req.Order,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			httpkit.WriteError(w, http.StatusConflict, "ya existe una clase con ese order en este curso")
			return
		}
		log.Printf("admin: creando clase en curso %d: %v", course.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo crear la clase")
		return
	}

	httpkit.WriteJSON(w, http.StatusCreated, classResponse{Order: class.OrderIndex, Title: class.Title})
}

type updateClassRequest struct {
	Title string `json:"title"`
}

// UpdateClass renombra una clase.
func (h *AdminHandler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	class, ok := h.resolveClass(w, r)
	if !ok {
		return
	}

	var req updateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpkit.WriteError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	title, ok := httpkit.RequireNonEmpty(w, req.Title, "título")
	if !ok {
		return
	}

	if err := h.Lessons.UpdateClassTitle(r.Context(), class.ID, title); err != nil {
		if httpkit.HandleNotFound(w, err, "clase no encontrada") {
			return
		}
		log.Printf("admin: renombrando clase %d: %v", class.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo actualizar la clase")
		return
	}

	httpkit.WriteJSON(w, http.StatusOK, classResponse{Order: class.OrderIndex, Title: title})
}

// DeleteClass borra una clase. Sus lecciones, su contenido y el progreso de
// los alumnos no se tocan: las lecciones quedan sin clase.
func (h *AdminHandler) DeleteClass(w http.ResponseWriter, r *http.Request) {
	class, ok := h.resolveClass(w, r)
	if !ok {
		return
	}

	if err := h.Lessons.DeleteClass(r.Context(), class.ID); err != nil {
		if httpkit.HandleNotFound(w, err, "clase no encontrada") {
			return
		}
		log.Printf("admin: borrando clase %d: %v", class.ID, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo borrar la clase")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// resolveClass lee slug + classOrder de la URL
// (.../courses/{slug}/classes/{classOrder}) y devuelve la clase, o escribe
// la respuesta de error y devuelve ok=false.
func (h *AdminHandler) resolveClass(w http.ResponseWriter, r *http.Request) (domain.Class, bool) {
	slug := chi.URLParam(r, "slug")
	order, err := strconv.Atoi(chi.URLParam(r, "classOrder"))
	if err != nil || order < 1 {
		httpkit.WriteError(w, http.StatusBadRequest, "número de clase inválido")
		return domain.Class{}, false
	}

	course, err := h.Courses.GetBySlugAny(r.Context(), slug)
	if err != nil {
		if httpkit.HandleNotFound(w, err, "curso no encontrado") {
			return domain.Class{}, false
		}
		log.Printf("admin: obteniendo curso %q: %v", slug, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener el curso")
		return domain.Class{}, false
	}

	class, err := h.Lessons.GetClassByCourseAndOrder(r.Context(), course.ID, order)
	if err != nil {
		if httpkit.HandleNotFound(w, err, "clase no encontrada") {
			return domain.Class{}, false
		}
		log.Printf("admin: obteniendo clase %d/%d: %v", course.ID, order, err)
		httpkit.WriteError(w, http.StatusInternalServerError, "no se pudo obtener la clase")
		return domain.Class{}, false
	}

	return class, true
}
