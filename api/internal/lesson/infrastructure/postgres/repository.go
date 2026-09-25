// Package postgres implementa domain.Repository contra las tablas lessons
// y lesson_content. Única capa del dominio lesson que sabe que existe
// Postgres/pgx — mismo estilo (SQL crudo, sin ORM) que api/db.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ferjmc/god-edu/api/db"
	"github.com/ferjmc/god-edu/api/internal/lesson/domain"
)

// uniqueViolation es el código de error de Postgres para restricciones
// UNIQUE — mismo valor que db/users.go, duplicado acá a propósito: es un
// código fijo del protocolo de Postgres, no algo que dependa del resto de
// la API.
const uniqueViolation = "23505"

// Repository implementa domain.Repository contra lessons y lesson_content.
// Van juntas en la misma implementación porque casi nunca se consultan por
// separado: el contenido de una lección no tiene sentido sin la lección, y
// viceversa. Los errores "no encontrado"/"ya existe" siguen siendo
// db.ErrNotFound y db.ErrConflict (no sentinels propios): son vocabulario
// compartido que ya usan handlers/respond.go (handleNotFound) y el resto de
// los handlers vía errors.Is — mantenerlos evita tocar esa capa solo por
// moverse de paquete.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListByCourse devuelve la currícula de un curso: todas sus lecciones en
// orden, con el estado de completado del usuario dado. El LEFT JOIN es a
// propósito: un usuario sin ninguna fila en lesson_progress todavía tiene
// que ver la lista completa, solo que con completed = false en todas.
func (r *Repository) ListByCourse(ctx context.Context, courseID, userID int64) ([]domain.LessonSummary, error) {
	const q = `
		SELECT l.order_index, l.title, COALESCE(lp.completed, false)
		FROM lessons l
		LEFT JOIN lesson_progress lp ON lp.lesson_id = l.id AND lp.user_id = $2
		WHERE l.course_id = $1
		ORDER BY l.order_index
	`
	rows, err := r.pool.Query(ctx, q, courseID, userID)
	if err != nil {
		return nil, fmt.Errorf("lesson: listando lecciones del curso %d: %w", courseID, err)
	}
	defer rows.Close()

	var lessons []domain.LessonSummary
	for rows.Next() {
		var l domain.LessonSummary
		if err := rows.Scan(&l.Order, &l.Title, &l.Completed); err != nil {
			return nil, fmt.Errorf("lesson: leyendo lección del curso %d: %w", courseID, err)
		}
		lessons = append(lessons, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lesson: iterando lecciones del curso %d: %w", courseID, err)
	}

	return lessons, nil
}

// ListAllByCourse devuelve todas las lecciones de un curso en orden, sin el
// estado de completado de ningún usuario en particular — a diferencia de
// ListByCourse, que arma la currícula para un usuario logueado. La usa el
// panel admin, donde "completado" no tiene sentido (no hay un usuario al
// que preguntarle).
func (r *Repository) ListAllByCourse(ctx context.Context, courseID int64) ([]domain.Lesson, error) {
	const q = `
		SELECT id, course_id, title, order_index, created_at
		FROM lessons
		WHERE course_id = $1
		ORDER BY order_index
	`
	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("lesson: listando lecciones del curso %d: %w", courseID, err)
	}
	defer rows.Close()

	var lessons []domain.Lesson
	for rows.Next() {
		var l domain.Lesson
		if err := rows.Scan(&l.ID, &l.CourseID, &l.Title, &l.OrderIndex, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("lesson: leyendo lección del curso %d: %w", courseID, err)
		}
		lessons = append(lessons, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lesson: iterando lecciones del curso %d: %w", courseID, err)
	}

	return lessons, nil
}

// GetByCourseAndOrder busca una lección por curso + order_index. El
// order_index (1-based) es el identificador público en la URL — ver
// /cursos/{slug}/{leccion} en el frontend — para no exponer ids internos
// ni depender de que cada lección tenga un slug propio.
func (r *Repository) GetByCourseAndOrder(ctx context.Context, courseID int64, order int) (domain.Lesson, error) {
	const q = `
		SELECT id, course_id, title, order_index, created_at
		FROM lessons
		WHERE course_id = $1 AND order_index = $2
	`
	var l domain.Lesson
	err := r.pool.QueryRow(ctx, q, courseID, order).
		Scan(&l.ID, &l.CourseID, &l.Title, &l.OrderIndex, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Lesson{}, db.ErrNotFound
		}
		return domain.Lesson{}, fmt.Errorf("lesson: leyendo lección %d/%d: %w", courseID, order, err)
	}
	return l, nil
}

// UpdateTitle renombra una lección. El order_index no se toca acá:
// reordenar lecciones es una operación aparte (swap de dos índices a la
// vez, contra la constraint UNIQUE(course_id, order_index)) que todavía no
// tiene endpoint — no hace falta para cargar contenido en el orden
// correcto desde el principio.
func (r *Repository) UpdateTitle(ctx context.Context, lessonID int64, title string) error {
	const q = `UPDATE lessons SET title = $1 WHERE id = $2`
	tag, err := r.pool.Exec(ctx, q, title, lessonID)
	if err != nil {
		return fmt.Errorf("lesson: renombrando lección %d: %w", lessonID, err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// DeleteLesson borra una lección y, en cascada, su contenido y el progreso
// de los usuarios sobre ella. No borra los PDFs de R2 — eso lo resuelve el
// caller (AdminHandler.DeleteLesson) antes de llamar acá.
func (r *Repository) DeleteLesson(ctx context.Context, lessonID int64) error {
	const q = `DELETE FROM lessons WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, lessonID)
	if err != nil {
		return fmt.Errorf("lesson: borrando lección %d: %w", lessonID, err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// IsCompleted indica si el usuario dado ya completó la lección dada.
func (r *Repository) IsCompleted(ctx context.Context, lessonID, userID int64) (bool, error) {
	const q = `SELECT completed FROM lesson_progress WHERE lesson_id = $1 AND user_id = $2`
	var completed bool
	err := r.pool.QueryRow(ctx, q, lessonID, userID).Scan(&completed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // sin fila todavía = no completada
		}
		return false, fmt.Errorf("lesson: leyendo progreso de lección %d: %w", lessonID, err)
	}
	return completed, nil
}

// MarkCompleted registra que el usuario dado completó la lección dada.
// Upsert sobre la UNIQUE(user_id, lesson_id): la primera vez inserta la
// fila, las siguientes (por ejemplo si el usuario vuelve a tocar el botón)
// solo refrescan completed_at — nunca hay más de una fila de progreso por
// usuario y lección. No hay "desmarcar": una vez completada, se asume que
// consumir la lección de nuevo no le resta valor a haberla terminado antes.
func (r *Repository) MarkCompleted(ctx context.Context, lessonID, userID int64) error {
	const q = `
		INSERT INTO lesson_progress (user_id, lesson_id, completed, completed_at)
		VALUES ($1, $2, true, now())
		ON CONFLICT (user_id, lesson_id)
		DO UPDATE SET completed = true, completed_at = now()
	`
	if _, err := r.pool.Exec(ctx, q, userID, lessonID); err != nil {
		return fmt.Errorf("lesson: marcando lección %d completada para usuario %d: %w", lessonID, userID, err)
	}
	return nil
}

// CreateLesson inserta una lección nueva bajo un curso. Devuelve
// db.ErrConflict si ya existe una lección con ese order_index en el curso
// (la restricción UNIQUE(course_id, order_index) es a propósito: dos
// lecciones no pueden pelear por el mismo lugar en la currícula).
func (r *Repository) CreateLesson(ctx context.Context, l domain.Lesson) (domain.Lesson, error) {
	const q = `
		INSERT INTO lessons (course_id, title, order_index)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, q, l.CourseID, l.Title, l.OrderIndex).Scan(&l.ID, &l.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.Lesson{}, db.ErrConflict
		}
		return domain.Lesson{}, fmt.Errorf("lesson: creando lección: %w", err)
	}
	return l, nil
}

// CreateContent inserta una pieza de contenido bajo una lección. Igual que
// CreateLesson, devuelve db.ErrConflict si el order_index ya está usado
// dentro de esa lección.
func (r *Repository) CreateContent(ctx context.Context, c domain.LessonContent) (domain.LessonContent, error) {
	const q = `
		INSERT INTO lesson_content (lesson_id, title, order_index, content_type, youtube_url, pdf_url, body)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, q, c.LessonID, c.Title, c.OrderIndex, c.Type, c.YoutubeURL, c.PDFURL, c.Body).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.LessonContent{}, db.ErrConflict
		}
		return domain.LessonContent{}, fmt.Errorf("lesson: creando contenido de lección %d: %w", c.LessonID, err)
	}
	return c, nil
}

// GetContentByLessonAndOrder busca una pieza de contenido puntual dentro de
// una lección, por su order_index — mismo esquema identificador que
// GetByCourseAndOrder usa para lecciones dentro de un curso.
func (r *Repository) GetContentByLessonAndOrder(ctx context.Context, lessonID int64, order int) (domain.LessonContent, error) {
	const q = `
		SELECT id, lesson_id, title, order_index, content_type, youtube_url, pdf_url, body, created_at
		FROM lesson_content
		WHERE lesson_id = $1 AND order_index = $2
	`
	var c domain.LessonContent
	err := r.pool.QueryRow(ctx, q, lessonID, order).
		Scan(&c.ID, &c.LessonID, &c.Title, &c.OrderIndex, &c.Type, &c.YoutubeURL, &c.PDFURL, &c.Body, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.LessonContent{}, db.ErrNotFound
		}
		return domain.LessonContent{}, fmt.Errorf("lesson: leyendo contenido %d/%d: %w", lessonID, order, err)
	}
	return c, nil
}

// UpdateContent actualiza el título y el campo específico de tipo (video:
// youtube_url, markdown: body) de una pieza de contenido. pdf_url nunca se
// toca acá: si hay que reemplazar el archivo de un PDF, se borra esa pieza
// de contenido y se sube una nueva (ver UploadPDFContent) — editar un
// archivo ya subido no es un caso que valga la pena resolver aparte.
func (r *Repository) UpdateContent(ctx context.Context, id int64, title string, youtubeURL, body *string) error {
	const q = `UPDATE lesson_content SET title = $1, youtube_url = $2, body = $3 WHERE id = $4`
	tag, err := r.pool.Exec(ctx, q, title, youtubeURL, body, id)
	if err != nil {
		return fmt.Errorf("lesson: actualizando contenido %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// DeleteContent borra una pieza de contenido. No borra el PDF de R2 si
// corresponde — eso lo resuelve el caller (AdminHandler.DeleteContent)
// antes de llamar acá.
func (r *Repository) DeleteContent(ctx context.Context, id int64) error {
	const q = `DELETE FROM lesson_content WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("lesson: borrando contenido %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}
	return nil
}

// NextContentOrder devuelve el próximo order_index libre para el contenido
// de una lección (max existente + 1, o 1 si todavía no tiene ninguno). El
// admin no arma el orden a mano: cada pieza de contenido se agrega al
// final, en el orden en que se cargó.
func (r *Repository) NextContentOrder(ctx context.Context, lessonID int64) (int, error) {
	const q = `SELECT COALESCE(MAX(order_index), 0) + 1 FROM lesson_content WHERE lesson_id = $1`
	var next int
	if err := r.pool.QueryRow(ctx, q, lessonID).Scan(&next); err != nil {
		return 0, fmt.Errorf("lesson: calculando próximo orden de contenido para lección %d: %w", lessonID, err)
	}
	return next, nil
}

// ListContent devuelve el contenido de una lección (video, PDFs y/o texto
// markdown), en el orden en que se cargó.
func (r *Repository) ListContent(ctx context.Context, lessonID int64) ([]domain.LessonContent, error) {
	const q = `
		SELECT id, lesson_id, title, order_index, content_type, youtube_url, pdf_url, body, created_at
		FROM lesson_content
		WHERE lesson_id = $1
		ORDER BY order_index
	`
	rows, err := r.pool.Query(ctx, q, lessonID)
	if err != nil {
		return nil, fmt.Errorf("lesson: listando contenido de lección %d: %w", lessonID, err)
	}
	defer rows.Close()

	var content []domain.LessonContent
	for rows.Next() {
		var c domain.LessonContent
		if err := rows.Scan(&c.ID, &c.LessonID, &c.Title, &c.OrderIndex, &c.Type, &c.YoutubeURL, &c.PDFURL, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("lesson: leyendo contenido de lección %d: %w", lessonID, err)
		}
		content = append(content, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lesson: iterando contenido de lección %d: %w", lessonID, err)
	}

	return content, nil
}
