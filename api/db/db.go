// Package db contiene el pool de conexión a Postgres, las migraciones, y
// los sentinels de error compartidos (ErrNotFound, ErrConflict) que usan
// los repositorios de cada dominio migrado (ver api/internal/<dominio>/
// infrastructure/postgres) — ya no aloja repositorios propios: el último
// (UserRepo) se movió a internal/user en esta misma serie de migraciones.
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
