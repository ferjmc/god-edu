// Package db contiene la conexión a Postgres y los repositorios que
// traducen entre filas de la base y los structs de models.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound indica que una consulta no encontró la fila buscada.
// Los handlers chequean esto para devolver 404 sin conocer detalles de pgx.
var ErrNotFound = errors.New("db: not found")

// ErrConflict indica que una inserción violó una restricción única
// (por ejemplo, un email que ya existe).
var ErrConflict = errors.New("db: already exists")

// NewPool abre un pool de conexiones a Postgres a partir de una cadena de
// conexión (ver DATABASE_URL en .env.example) y confirma que responde.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: creando pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping inicial falló: %w", err)
	}

	return pool, nil
}
