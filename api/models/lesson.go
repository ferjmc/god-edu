package models

import "time"

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
// LessonRepo.ListByCourse con el join a lesson_progress.
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
