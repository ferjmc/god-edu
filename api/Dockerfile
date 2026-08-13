# Build de producción: multi-stage, binario compilado y estático, imagen
# final mínima. Distinto de Dockerfile.dev (que corre "go run ." con el
# código montado como volumen) — ese sigue siendo el que usa
# docker-compose.yml para desarrollo local; este es el que apunta Coolify.
#
# OJO — build context es la RAÍZ del repo, no api/: necesita copiar
# /migrations (hermana de /api, no una subcarpeta) para que el binario
# pueda migrar la base solo al arrancar (ver api/db/migrate.go). Construir
# con:
#   docker build -f api/Dockerfile -t god-edu-api .
# (desde la raíz del repo — no "cd api && docker build ."). En Coolify:
# Dockerfile Location = api/Dockerfile, Base Directory = raíz del repo.

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /app

# Cache de módulos separada del código: si go.mod/go.sum no cambian, esta
# capa no se vuelve a bajar aunque cambie el código fuente.
COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/. .

# CGO_ENABLED=0: pgx es pure Go, no hace falta cgo. Un binario estático no
# depende de ninguna libc del sistema en el runtime, así que puede correr
# en una imagen final que no tiene absolutamente nada más que él.
# -trimpath + -ldflags="-s -w": saca paths absolutos del build y símbolos
# de debug — imagen más chica, nada que el runtime necesite.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

# ---- runtime stage ----
# distroless/static: ni shell, ni package manager, ni nada que no sea el
# binario, sus certificados CA (los necesita para hablar HTTPS con R2,
# Resend, Google/Facebook OAuth) y un usuario "nonroot" ya armado. Superficie
# de ataque mínima — no hay nada para explotar más allá del binario mismo.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app

COPY --from=build /out/api ./api
# Migraciones al lado del binario — RunMigrations las lee de "migrations"
# (relativo al WORKDIR) por default, ver config.go MIGRATIONS_PATH.
COPY migrations ./migrations

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/api"]
