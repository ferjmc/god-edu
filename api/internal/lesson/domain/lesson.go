// Package domain contiene las entidades Lesson, LessonSummary y
// LessonContent y el puerto (Repository) que infrastructure/postgres
// implementa. No conoce SQL ni HTTP.
package domain

import (
	"context"
	"time"
)

// Lesson es una lección dentro de un curso. El identificador público que
// usa la URL (/cursos/{slug}/{orden}, ver frontend) es OrderIndex, no ID —
// nadie necesita saber el id interno de una lección.
type Lesson struct {
	ID         int64
	CourseID   int64
	Title      string
	OrderIndex int
	CreatedAt  time.Time
}

// LessonSummary es cómo se ve una lección desde la currícula de un curso:
// lo necesario para listarla, más si el usuario que pidió el listado ya la
// completó. No es Lesson porque "completado" es relativo a un usuario, no
// una propiedad de la lección en sí — por eso vive aparte y lo arma
// Repository.ListByCourse con el join a lesson_progress.
type LessonSummary struct {
	Order     int
	Title     string
	Completed bool
}

// ContentType identifica qué tipo de contenido lleva una fila de
// lesson_content. Coincide 1:1 con el CHECK de la columna en la base.
type ContentType string

const (
	ContentTypeVideo    ContentType = "video"
	ContentTypePDF      ContentType = "pdf"
	ContentTypeMarkdown ContentType = "markdown"
)

// LessonContent es una pieza de contenido de una lección. Una lección
// puede tener varias piezas (por ejemplo: un video + dos PDFs), cada una
// con su propio orden. Solo uno de YoutubeURL/PDFURL/Body está poblado
// según Type — los otros dos quedan nil.
type LessonContent struct {
	ID         int64
	LessonID   int64
	Title      string
	OrderIndex int
	Type       ContentType
	YoutubeURL *string
	PDFURL     *string
	Body       *string // markdown; solo poblado cuando Type == ContentTypeMarkdown
	CreatedAt  time.Time
}

// Repository es el puerto que expone infrastructure/postgres.
type Repository interface {
	// ListByCourse devuelve la currícula de un curso: todas sus lecciones en
	// orden, con el estado de completado del usuario dado.
	ListByCourse(ctx context.Context, courseID, userID int64) ([]LessonSummary, error)
	// ListAllByCourse devuelve todas las lecciones de un curso en orden, sin
	// el estado de completado de ningún usuario en particular — la usa el
	// panel admin.
	ListAllByCourse(ctx context.Context, courseID int64) ([]Lesson, error)
	// GetByCourseAndOrder busca una lección por curso + order_index.
	GetByCourseAndOrder(ctx context.Context, courseID int64, order int) (Lesson, error)
	// UpdateTitle renombra una lección.
	UpdateTitle(ctx context.Context, lessonID int64, title string) error
	// DeleteLesson borra una lección y, en cascada, su contenido y el
	// progreso de los usuarios sobre ella.
	DeleteLesson(ctx context.Context, lessonID int64) error
	// IsCompleted indica si el usuario dado ya completó la lección dada.
	IsCompleted(ctx context.Context, lessonID, userID int64) (bool, error)
	// MarkCompleted registra que el usuario dado completó la lección dada.
	MarkCompleted(ctx context.Context, lessonID, userID int64) error
	// CreateLesson inserta una lección nueva bajo un curso.
	CreateLesson(ctx context.Context, l Lesson) (Lesson, error)
	// CreateContent inserta una pieza de contenido bajo una lección.
	CreateContent(ctx context.Context, c LessonContent) (LessonContent, error)
	// GetContentByLessonAndOrder busca una pieza de contenido puntual dentro
	// de una lección, por su order_index.
	GetContentByLessonAndOrder(ctx context.Context, lessonID int64, order int) (LessonContent, error)
	// UpdateContent actualiza el título y el campo específico de tipo
	// (video: youtube_url, markdown: body) de una pieza de contenido.
	UpdateContent(ctx context.Context, id int64, title string, youtubeURL, body *string) error
	// DeleteContent borra una pieza de contenido.
	DeleteContent(ctx context.Context, id int64) error
	// NextContentOrder devuelve el próximo order_index libre para el
	// contenido de una lección.
	NextContentOrder(ctx context.Context, lessonID int64) (int, error)
	// ListContent devuelve el contenido de una lección (video, PDFs y/o
	// texto markdown), en el orden en que se cargó.
	ListContent(ctx context.Context, lessonID int64) ([]LessonContent, error)
}
