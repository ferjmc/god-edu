package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ferjmc/god-edu/api/auth"
	"github.com/ferjmc/god-edu/api/db"
)

// LessonHandler agrupa los endpoints de lecciones dentro de un curso.
// Ambos requieren sesión y el mismo chequeo de rol que el detalle del
// curso (ver resolveVisibleCourse): el contenido de una lección nunca es
// más accesible que el curso que la contiene.
//
// No hay bloqueo secuencial acá adentro a propósito: cualquier usuario con
// acceso al curso puede pedir cualquier lección, complete o no las
// anteriores. El orden es una guía de navegación, no un control de acceso
// — mantiene el handler simple y no le pone un techo artificial a quien
// quiere repasar o adelantarse.
type LessonHandler struct {
	Courses *db.CourseRepo
	Users   *db.UserRepo
	Lessons *db.LessonRepo
}

type lessonSummaryResponse struct {
	Order     int    `json:"order"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// List devuelve la currícula completa del curso: todas sus lecciones en
// orden, con el estado de completado del usuario actual. La usan tanto la
// página de detalle del curso como el sidebar de currícula de cada lección.
func (h *LessonHandler) List(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	course, err := resolveVisibleCourse(r.Context(), h.Courses, h.Users, slug)
	if err != nil {
		writeCourseAccessError(w, err)
		return
	}

	// resolveVisibleCourse ya validó que hay un usuario autenticado.
	userID, _ := auth.UserIDFromContext(r.Context())

	lessons, err := h.Lessons.ListByCourse(r.Context(), course.ID, userID)
	if err != nil {
		log.Printf("lessons: listando curso %d: %v", course.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudieron obtener las lecciones")
		return
	}

	response := make([]lessonSummaryResponse, len(lessons))
	for i, l := range lessons {
		response[i] = lessonSummaryResponse{Order: l.Order, Title: l.Title, Completed: l.Completed}
	}

	writeJSON(w, http.StatusOK, response)
}

type lessonContentResponse struct {
	// Order es el order_index de la fila — lo necesita el panel admin para
	// armar la URL de editar/borrar una pieza puntual (.../content/{order}).
	// No alcanza con la posición en el array: si se borra un contenido del
	// medio, el order_index deja de ser contiguo, pero sigue siendo el
	// identificador real.
	Order      int     `json:"order"`
	Title      string  `json:"title"`
	Type       string  `json:"type"`
	YoutubeURL *string `json:"youtubeUrl,omitempty"`
	PDFURL     *string `json:"pdfUrl,omitempty"`
	Body       *string `json:"body,omitempty"`
}

type lessonDetailResponse struct {
	Order     int                     `json:"order"`
	Title     string                  `json:"title"`
	Completed bool                    `json:"completed"`
	Content   []lessonContentResponse `json:"content"`
}

// Detail devuelve una lección puntual (identificada por su orden dentro
// del curso, no por id — ver LessonRepo.GetByCourseAndOrder) con todo su
// contenido: video, PDFs y/o texto markdown, en el orden en que se cargó.
func (h *LessonHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	order, err := strconv.Atoi(chi.URLParam(r, "order"))
	if err != nil || order < 1 {
		writeError(w, http.StatusBadRequest, "número de lección inválido")
		return
	}

	course, err := resolveVisibleCourse(r.Context(), h.Courses, h.Users, slug)
	if err != nil {
		writeCourseAccessError(w, err)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	lesson, err := h.Lessons.GetByCourseAndOrder(r.Context(), course.ID, order)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lección no encontrada")
			return
		}
		log.Printf("lessons: obteniendo lección %d/%d: %v", course.ID, order, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener la lección")
		return
	}

	completed, err := h.Lessons.IsCompleted(r.Context(), lesson.ID, userID)
	if err != nil {
		log.Printf("lessons: leyendo progreso de lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el progreso")
		return
	}

	content, err := h.Lessons.ListContent(r.Context(), lesson.ID)
	if err != nil {
		log.Printf("lessons: listando contenido de lección %d: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener el contenido")
		return
	}

	contentResponse := make([]lessonContentResponse, len(content))
	for i, c := range content {
		contentResponse[i] = lessonContentResponse{
			Order:      c.OrderIndex,
			Title:      c.Title,
			Type:       string(c.Type),
			YoutubeURL: c.YoutubeURL,
			PDFURL:     c.PDFURL,
			Body:       c.Body,
		}
	}

	writeJSON(w, http.StatusOK, lessonDetailResponse{
		Order:     lesson.OrderIndex,
		Title:     lesson.Title,
		Completed: completed,
		Content:   contentResponse,
	})
}

type lessonCompleteResponse struct {
	Completed bool `json:"completed"`
}

// MarkComplete registra que el usuario actual completó la lección. Manual
// a propósito — ver decisión en CLAUDE.md/memoria del proyecto: marcar
// automático con solo entrar a la lección no distingue "abrió la página" de
// "consumió el contenido", y para un video de YouTube embebido de forma
// simple (sin la IFrame Player API) no hay señal confiable de "lo terminó
// de ver". El botón manual es la superficie mínima que deja el dato
// utilizable para reportes futuros; detectar automáticamente el fin del
// video puede sumarse después sin tocar este endpoint.
func (h *LessonHandler) MarkComplete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	order, err := strconv.Atoi(chi.URLParam(r, "order"))
	if err != nil || order < 1 {
		writeError(w, http.StatusBadRequest, "número de lección inválido")
		return
	}

	course, err := resolveVisibleCourse(r.Context(), h.Courses, h.Users, slug)
	if err != nil {
		writeCourseAccessError(w, err)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())

	lesson, err := h.Lessons.GetByCourseAndOrder(r.Context(), course.ID, order)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lección no encontrada")
			return
		}
		log.Printf("lessons: obteniendo lección %d/%d: %v", course.ID, order, err)
		writeError(w, http.StatusInternalServerError, "no se pudo obtener la lección")
		return
	}

	if err := h.Lessons.MarkCompleted(r.Context(), lesson.ID, userID); err != nil {
		log.Printf("lessons: marcando lección %d completada: %v", lesson.ID, err)
		writeError(w, http.StatusInternalServerError, "no se pudo registrar el progreso")
		return
	}

	writeJSON(w, http.StatusOK, lessonCompleteResponse{Completed: true})
}
