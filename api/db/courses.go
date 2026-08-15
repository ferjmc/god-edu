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

// CourseRepo implementa el acceso a la tabla courses.
type CourseRepo struct {
	pool *pgxpool.Pool
}

func NewCourseRepo(pool *pgxpool.Pool) *CourseRepo {
	return &CourseRepo{pool: pool}
}

// ListPublished devuelve los cursos publicados, más nuevos primero. Es el
// único listado que ve un usuario sin permisos de administración — los
// cursos en borrador (published = false) nunca salen de acá.
func (r *CourseRepo) ListPublished(ctx context.Context) ([]models.Course, error) {
	const q = `
		SELECT id, title, slug, description, published, created_at
		FROM courses
		WHERE published = true
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db: listando cursos publicados: %w", err)
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.Title, &c.Slug, &c.Description, &c.Published, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: leyendo curso: %w", err)
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterando cursos: %w", err)
	}

	return courses, nil
}

// ListAll devuelve todos los cursos, publicados y en borrador, más nuevos
// primero. La usa el panel admin (ListPublished es para la vidriera
// pública) — necesita ver los borradores para poder seguir cargándolos.
func (r *CourseRepo) ListAll(ctx context.Context) ([]models.Course, error) {
	const q = `
		SELECT id, title, slug, description, published, created_at
		FROM courses
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db: listando todos los cursos: %w", err)
	}
	defer rows.Close()

	var courses []models.Course
	for rows.Next() {
		var c models.Course
		if err := rows.Scan(&c.ID, &c.Title, &c.Slug, &c.Description, &c.Published, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("db: leyendo curso: %w", err)
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterando cursos: %w", err)
	}

	return courses, nil
}

// GetBySlug busca un curso publicado por slug. Devuelve ErrNotFound tanto si
// no existe como si existe pero todavía no está publicado — desde afuera
// ambos casos son indistinguibles a propósito, para no filtrar qué slugs
// existen en borrador.
func (r *CourseRepo) GetBySlug(ctx context.Context, slug string) (models.Course, error) {
	const q = `
		SELECT id, title, slug, description, published, created_at
		FROM courses
		WHERE slug = $1 AND published = true
	`
	var c models.Course
	err := r.pool.QueryRow(ctx, q, slug).
		Scan(&c.ID, &c.Title, &c.Slug, &c.Description, &c.Published, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Course{}, ErrNotFound
		}
		return models.Course{}, fmt.Errorf("db: leyendo curso %q: %w", slug, err)
	}

	visibleRoles, err := r.visibleRoles(ctx, c.ID)
	if err != nil {
		return models.Course{}, err
	}
	c.VisibleRoles = visibleRoles

	return c, nil
}

// GetBySlugAny busca un curso por slug sin filtrar por published — a
// diferencia de GetBySlug, la usa el panel admin: mientras se carga un
// curso nuevo (lecciones, contenido) todavía está en borrador, y el admin
// necesita poder seguir viéndolo y editándolo igual.
func (r *CourseRepo) GetBySlugAny(ctx context.Context, slug string) (models.Course, error) {
	const q = `
		SELECT id, title, slug, description, published, created_at
		FROM courses
		WHERE slug = $1
	`
	var c models.Course
	err := r.pool.QueryRow(ctx, q, slug).
		Scan(&c.ID, &c.Title, &c.Slug, &c.Description, &c.Published, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Course{}, ErrNotFound
		}
		return models.Course{}, fmt.Errorf("db: leyendo curso %q: %w", slug, err)
	}

	visibleRoles, err := r.visibleRoles(ctx, c.ID)
	if err != nil {
		return models.Course{}, err
	}
	c.VisibleRoles = visibleRoles

	return c, nil
}

// Create inserta un curso nuevo. Arranca siempre en borrador
// (published = false) — publicarlo es un paso aparte (ver SetPublished),
// a propósito: nadie debería ver un curso a medio cargar solo porque se
// creó la fila.
func (r *CourseRepo) Create(ctx context.Context, c models.Course) (models.Course, error) {
	const q = `
		INSERT INTO courses (title, slug, description, published)
		VALUES ($1, $2, $3, false)
		RETURNING id, published, created_at
	`
	err := r.pool.QueryRow(ctx, q, c.Title, c.Slug, c.Description).
		Scan(&c.ID, &c.Published, &c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return models.Course{}, ErrConflict
		}
		return models.Course{}, fmt.Errorf("db: creando curso: %w", err)
	}
	return c, nil
}

// UpdateDetails actualiza título y descripción de un curso. El slug no se
// puede tocar acá a propósito: cambiarlo rompería cualquier link ya
// compartido a /cursos/{slug} — si hace falta renombrar la URL de un curso,
// es una decisión aparte, no un campo más de este formulario.
func (r *CourseRepo) UpdateDetails(ctx context.Context, slug, title string, description *string) error {
	const q = `UPDATE courses SET title = $1, description = $2 WHERE slug = $3`
	tag, err := r.pool.Exec(ctx, q, title, description, slug)
	if err != nil {
		return fmt.Errorf("db: actualizando curso %q: %w", slug, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete borra un curso y, en cascada (ver migrations), sus lecciones,
// contenido, inscripciones y progreso. No borra los PDFs correspondientes
// de R2 — eso lo resuelve el caller (AdminHandler.DeleteCourse) antes de
// llamar acá, mientras todavía puede leer qué lecciones tenía.
func (r *CourseRepo) Delete(ctx context.Context, slug string) error {
	const q = `DELETE FROM courses WHERE slug = $1`
	tag, err := r.pool.Exec(ctx, q, slug)
	if err != nil {
		return fmt.Errorf("db: borrando curso %q: %w", slug, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPublished cambia el estado de publicación de un curso.
func (r *CourseRepo) SetPublished(ctx context.Context, slug string, published bool) error {
	const q = `UPDATE courses SET published = $1 WHERE slug = $2`
	tag, err := r.pool.Exec(ctx, q, published, slug)
	if err != nil {
		return fmt.Errorf("db: publicando curso %q: %w", slug, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// visibleRoles devuelve los roles configurados en course_visible_roles para
// un curso. Vacío (no error) si el curso no tiene restricción configurada.
func (r *CourseRepo) visibleRoles(ctx context.Context, courseID int64) ([]models.UserRole, error) {
	const q = `SELECT role FROM course_visible_roles WHERE course_id = $1`
	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("db: leyendo roles visibles del curso %d: %w", courseID, err)
	}
	defer rows.Close()

	var roles []models.UserRole
	for rows.Next() {
		var role models.UserRole
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("db: leyendo rol visible del curso %d: %w", courseID, err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterando roles visibles del curso %d: %w", courseID, err)
	}

	return roles, nil
}
