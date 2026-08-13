package db

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registra el driver "pgx5://" — mismo pgx que usa el resto de la API, no lib/pq
	_ "github.com/golang-migrate/migrate/v4/source/file"     // registra el driver "file://"
)

// RunMigrations aplica las migraciones pendientes de path contra
// databaseURL. Se llama al arrancar la API (ver main.go), antes de
// empezar a servir tráfico — así cada deploy queda con el schema al día
// sin un paso manual aparte, sin importar qué plataforma lo despliegue.
//
// Si path no existe, no es un error: loguea y no hace nada. Cubre el caso
// de desarrollo local corriendo el binario suelto (sin Docker) desde un
// directorio que no tiene la carpeta migrations al lado — ahí se sigue
// aplicando a mano con el CLI, como hasta ahora.
//
// golang-migrate toma un advisory lock de Postgres mientras migra, así que
// es seguro aunque el proceso arranque más de una vez en simultáneo (no es
// el caso de este despliegue de una sola instancia, pero no está de más).
func RunMigrations(databaseURL, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("db: %q no existe, se omiten las migraciones automáticas", path)
		return nil
	}

	pgx5URL, err := toPgx5URL(databaseURL)
	if err != nil {
		return fmt.Errorf("db: parseando DATABASE_URL para migraciones: %w", err)
	}

	m, err := migrate.New("file://"+path, pgx5URL)
	if err != nil {
		return fmt.Errorf("db: abriendo migraciones desde %q: %w", path, err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			log.Printf("db: cerrando migrador (fuente: %v, conexión: %v)", srcErr, dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: aplicando migraciones: %w", err)
	}

	return nil
}

// toPgx5URL cambia el scheme de la connection string ("postgres://" o
// "postgresql://") a "pgx5://", el que espera el driver pgx de
// golang-migrate — el resto de la URL (host, user, password, query
// params como sslmode) queda igual.
func toPgx5URL(databaseURL string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	u.Scheme = "pgx5"
	return u.String(), nil
}
