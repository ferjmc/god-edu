package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ferjmc/god-edu/api/models"
)

// LessonRepo implementa el acceso a lessons y lesson_content. Van juntas en
// un mismo repo porque casi nunca se consultan por separado: el contenido
// de una lección no tiene sentido sin la lección, y viceversa.
type LessonRepo struct {
	pool *pgxpool.Pool
}

func NewLessonRepo(pool *pgxpool.Pool) *LessonRepo {
	return &LessonRepo{pool: pool}
}

// ListByCourse devuelve la currícula de un curso: todas sus lecciones en
// orden, con el estado de completado del usuario dado. El LEFT JOIN es a
// propósito: un usuario sin ninguna fila en lesson_progress todavía tiene
// que ver la lista completa, solo que con completed = false en todas.
func (r *LessonRepo) ListByCourse(ctx context.Context, courseID, userID int64) ([]models.LessonSummary, error) {
	const q = `
		SELECT l.order_index, l.title, COALESCE(lp.completed, false)
		FROM lessons l
		LEFT JOIN lesson_progress lp ON lp.lesson_id = l.id AND lp.user_id = $2
		WHERE l.course_id = $1
		ORDER BY l.order_index
	`
	rows, err := r.pool.Query(ctx, q, courseID, userID)
	if err != nil {
		return nil, fmt.Errorf("db: listando lecciones del curso %d: %w", courseID, err)
	}
	defer rows.Close()

	var lessons []models.LessonSummary
	for rows.Next() {
		var l models.LessonSummary
		if err := rows.Scan(&l.Order, &l.Title, &l.Completed); err != nil {
			return nil, fmt.Errorf("db: leyendo lección del curso %d: %w", courseID, err)
		}
		lessons = append(lessons, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterando lecciones del curso %d: %w", courseID, err)
	}

	return lessons, nil
}

// GetByCourseAndOrder busca una lección por curso + order_index. El
// order_index (1-based) es el identificador público en la URL — ver
// /cursos/{slug}/{leccion} en el frontend — para no exponer ids internos
// ni depender de que cada lección tenga un slug propio.
func (r *LessonRepo) GetByCourseAndOrder(ctx context.Context, courseID int64, order int) (models.Lesson, error) {
	const q = `
		SELECT id, course_id, title, order_index, created_at
		FROM lessons
		WHERE course_id = $1 AND order_index = $2
	`
	var l models.Lesson
	err := r.pool.QueryRow(ctx, q, courseID, order).
		Scan(&l.ID, &l.CourseID, &l.Title, &l.OrderIndex, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Lesson{}, ErrNotFound
		}
		return models.Lesson{}, fmt.Errorf("db: leyendo lección %d/%d: %w", courseID, order, err)
	}
	return l, nil
}

// IsCompleted indica si el usuario dado ya completó la lección dada.
func (r *LessonRepo) IsCompleted(ctx context.Context, lessonID, userID int64) (bool, error) {
	const q = `SELECT completed FROM lesson_progress WHERE lesson_id = $1 AND user_id = $2`
	var completed bool
	err := r.pool.QueryRow(ctx, q, lessonID, userID).Scan(&completed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // sin fila todavía = no completada
		}
		return false, fmt.Errorf("db: leyendo progreso de lección %d: %w", lessonID, err)
	}
	return completed, nil
}

// MarkCompleted registra que el usuario dado completó la lección dada.
// Upsert sobre la UNIQUE(user_id, lesson_id): la primera vez inserta la
// fila, las siguientes (por ejemplo si el usuario vuelve a tocar el botón)
// solo refrescan completed_at — nunca hay más de una fila de progreso por
// usuario y lección. No hay "desmarcar": una vez completada, se asume que
// consumir la lección de nuevo no le resta valor a haberla terminado antes.
func (r *LessonRepo) MarkCompleted(ctx context.Context, lessonID, userID int64) error {
	const q = `
		INSERT INTO lesson_progress (user_id, lesson_id, completed, completed_at)
		VALUES ($1, $2, true, now())
		ON CONFLICT (user_id, lesson_id)
		DO UPDATE SET completed = true, completed_at = now()
	`
	if _, err := r.pool.Exec(ctx, q, userID, lessonID); err != nil {
		return fmt.Errorf("db: marcando lección %d completada para usuario %d: %w", lessonID, userID, err)
	}
	return nil
}

// CreateLesson inserta una lección nueva bajo un curso. Devuelve
// ErrConflict si ya existe una lección con ese order_index en el curso
// (la restricción UNIQUE(course_id, order_index) es a propósito: dos
// lecciones no pueden pelear por el mismo lugar en la currícula).
func (r *LessonRepo) CreateLesson(ctx context.Context, l models.Lesson) (models.Lesson, error) {
	const q = `
		INSERT INTO lessons (course_id, title, order_index)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, q, l.CourseID, l.Title, l.OrderIndex).Scan(&l.ID, &l.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return models.Lesson{}, ErrConflict
		}
		return models.Lesson{}, fmt.Errorf("db: creando lección: %w", err)
	}
	return l, nil
}

// CreateContent inserta una pieza de contenido bajo una lección. Igual que
// CreateLesson, devuelve ErrConflict si el order_index ya está usado
// dentro de esa lección.
func (r *LessonRepo) CreateContent(ctx context.Context, c models.LessonContent) (models.LessonContent, error) {
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
			return models.LessonContent{}, ErrConflict
		}
		return models.LessonContent{}, fmt.Errorf("db: creando contenido de lección %d: %w", c.LessonID, err)
	}
	return c, nil
}

// NextContentOrder devuelve el próximo order_index libre para el contenido
// de una lección (max existente + 1, o 1 si todavía no tiene ninguno). El
// admin no arma el orden a mano: cada pieza de contenido se agrega al
// final, en el orden en que se cargó.
func (r *LessonRepo) NextContentOrder(ctx context.Context, lessonID int64) (int, error) {
	const q = `SELECT COALESCE(MAX(order_index), 0) + 1 FROM lesson_content WHERE lesson_id = $1`
	var next int
	if err := r.pool.QueryRow(ctx, q, lessonID).Scan(&next); err != nil {
		return 0, fmt.Errorf("db: calculando próximo orden de contenido para lección %d: %w", lessonID, err)
	}
	return next, nil
}

// ListContent devuelve el contenido de una lección (video, PDFs y/o texto
// markdown), en el orden en que se cargó.
func (r *LessonRepo) ListContent(ctx context.Context, lessonID int64) ([]models.LessonContent, error) {
	const q = `
		SELECT id, lesson_id, title, order_index, content_type, youtube_url, pdf_url, body, created_at
		FROM lesson_content
		WHERE lesson_id = $1
		ORDER BY order_index
	`
	rows, err := r.pool.Query(ctx, q, lessonID)
	if err != nil {
		return nil, fmt.Errorf("db: listando contenido de lección %d: %w", lessonID, err)
	}
	defer rows.Close()

	var content []models.LessonContent
	for rows.Next() {
		var c models.LessonContent
		if err := rows.Scan(&c.ID, &c.LessonID, &c.Title, &c.OrderIndex, &c.Type, &c.YoutubeURL, &c.PDFURL, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: leyendo contenido de lección %d: %w", lessonID, err)
		}
		content = append(content, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterando contenido de lección %d: %w", lessonID, err)
	}

	return content, nil
}
