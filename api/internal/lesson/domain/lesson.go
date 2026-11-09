// Package domain contiene las entidades Lesson, LessonSummary y
// LessonContent y el puerto (Repository) que infrastructure/postgres
// implementa. No conoce SQL ni HTTP.
package domain

import (
	"context"
	"time"
)

// Lesson es una lección dentro de un curso — en la interfaz se llama
// "sesión". El identificador público que usa la URL (/cursos/{slug}/{orden},
// ver frontend) es OrderIndex, no ID — nadie necesita saber el id interno
// de una lección. OrderIndex es correlativo en todo el curso, no se
// reinicia por clase.
//
// ClassID es nil si la lección no está asignada a ninguna clase.
// ClassOrder/ClassTitle son de solo lectura (vienen del join con
// course_classes); para cambiar de clase se usa Service.SetLessonClass.
//
// Hidden también es de solo lectura: true si la lección está en una clase
// sin publicar. Una lección sin clase nunca está oculta.
type Lesson struct {
	ID         int64
	CourseID   int64
	ClassID    *int64
	ClassOrder *int
	ClassTitle *string
	Hidden     bool
	Title      string
	OrderIndex int
	CreatedAt  time.Time
}

// Class agrupa lecciones de un curso. No tiene progreso propio: el avance
// de una clase se calcula a partir de las lecciones que contiene.
//
// Published = false oculta sus lecciones al alumno (currícula, detalle y
// progreso), pero siguen contando para el certificado: nadie se certifica
// antes de que el curso esté publicado completo.
type Class struct {
	ID         int64
	CourseID   int64
	Title      string
	OrderIndex int
	Published  bool
	CreatedAt  time.Time
}

// LessonSummary es cómo se ve una lección desde la currícula de un curso:
// lo necesario para listarla, más si el usuario que pidió el listado ya la
// completó. No es Lesson porque "completado" es relativo a un usuario, no
// una propiedad de la lección en sí — por eso vive aparte y lo arma
// Repository.ListByCourse con el join a lesson_progress.
type LessonSummary struct {
	Order      int
	Title      string
	Completed  bool
	ClassOrder *int
	ClassTitle *string
}

// UserLessonActivity es una lección que un usuario puntual abrió y/o
// completó, con el curso al que pertenece. La usa el panel admin para ver
// el recorrido de un usuario. FirstViewedAt/LastViewedAt pueden ser nil en
// una lección completada si nunca se registró la vista (no debería pasar
// después de la migración 000007, pero el esquema lo permite).
type UserLessonActivity struct {
	CourseSlug    string
	CourseTitle   string
	LessonOrder   int
	LessonTitle   string
	FirstViewedAt *time.Time
	LastViewedAt  *time.Time
	Completed     bool
	CompletedAt   *time.Time
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
	// ListByCourse devuelve la currícula de un curso: sus lecciones visibles
	// (sin clase o en una clase publicada) en orden, con el estado de
	// completado del usuario dado.
	ListByCourse(ctx context.Context, courseID, userID int64) ([]LessonSummary, error)
	// HasCompletedAll indica si el usuario completó todas las lecciones del
	// curso, incluidas las de clases sin publicar. False si el curso no
	// tiene lecciones.
	HasCompletedAll(ctx context.Context, courseID, userID int64) (bool, error)
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
	// RecordView registra que el usuario dado abrió la lección dada: la
	// primera vez fija first_viewed_at, las siguientes solo last_viewed_at.
	RecordView(ctx context.Context, lessonID, userID int64) error
	// ListActivityByUser devuelve las lecciones que el usuario dado abrió
	// o completó, ordenadas por curso y orden de lección.
	ListActivityByUser(ctx context.Context, userID int64) ([]UserLessonActivity, error)
	// CreateLesson inserta una lección nueva bajo un curso (con su ClassID,
	// si tiene).
	CreateLesson(ctx context.Context, l Lesson) (Lesson, error)
	// SetLessonClass asigna una lección a una clase, o la deja sin clase si
	// classID es nil.
	SetLessonClass(ctx context.Context, lessonID int64, classID *int64) error
	// ListClasses devuelve las clases de un curso en orden.
	ListClasses(ctx context.Context, courseID int64) ([]Class, error)
	// GetClassByCourseAndOrder busca una clase por curso + order_index.
	GetClassByCourseAndOrder(ctx context.Context, courseID int64, order int) (Class, error)
	// CreateClass inserta una clase nueva bajo un curso.
	CreateClass(ctx context.Context, c Class) (Class, error)
	// UpdateClassTitle renombra una clase.
	UpdateClassTitle(ctx context.Context, classID int64, title string) error
	// SetClassPublished publica u oculta una clase.
	SetClassPublished(ctx context.Context, classID int64, published bool) error
	// DeleteClass borra una clase; sus lecciones quedan sin clase.
	DeleteClass(ctx context.Context, classID int64) error
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
