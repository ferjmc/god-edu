# Plataforma educativa católica — Manifiesto del proyecto

Este archivo es el contexto persistente del proyecto. Léelo por completo antes de escribir código. Si algo en una tarea puntual contradice este documento, este documento gana salvo que el usuario diga explícitamente lo contrario.

## Qué estamos construyendo

Una plataforma educativa online, similar en función a Thinkific, para publicar cursos sobre la religión católica (video, PDF, markdown). La mantiene y desarrolla **una sola persona**, sin equipo. Es para una organización sin fines de lucro con ~500 usuarios.

**Principio rector: simplicidad sobre elegancia.** Cada decisión técnica se evalúa primero por "¿puede una sola persona mantener esto en un año, con poco tiempo libre?" antes que por "¿es esto lo más moderno/escalable?". Ante la duda, elegí la opción más aburrida y predecible.

## Restricciones duras (no negociables)

- **Presupuesto de hosting: máximo 30 USD/mes**, objetivo real ~10 USD/mes.
- **Un solo desarrollador.** No introducir arquitecturas que requieran coordinación de equipo (no microservicios, no colas de mensajes complejas, no Kubernetes).
- **Mínimas dependencias externas de pago.** Preferir self-hosted o capas gratuitas generosas sobre SaaS con cuota mensual, salvo que el ahorro de tiempo sea muy claro.
- Videos se alojan en **YouTube (no listados)**, nunca en storage propio. No implementar upload/transcodificación de video.
- Los PDFs se guardan en **Cloudflare R2** (compatible con S3, sin costo de egreso, capa gratis de 10GB).

## Stack técnico (decidido, no reabrir esta discusión sin razón fuerte)

- **Frontend**: Astro + TypeScript. Sitio mayormente estático. Desplegado en **Cloudflare Pages** (gratis, sin límite práctico de ancho de banda, sin restricción de uso comercial).
- **Backend**: Go. API REST/JSON. Desplegado como contenedor Docker en un **VPS Hetzner** (2 vCPU/4GB). Deploy vía GitHub Actions: build de la imagen en el runner de GitHub (no en el VPS, que tiene poco recursos) → push a GitHub Container Registry → el VPS solo hace `pull` + reinicia el contenedor por SSH. Se evaluó Coolify/Dokploy (mencionados en una versión anterior de este documento) pero se optó por manejarlo a mano con GitHub Actions — decisión explícita del usuario, no hay que reabrirla sin razón fuerte tampoco. Ver `.github/workflows/deploy.yml` y `docker-compose.prod.yml`.
- **Base de datos**: PostgreSQL, corriendo en el mismo VPS (o Postgres gestionado barato si el usuario lo prefiere más adelante).
- **Reverse proxy / HTTPS**: **Cloudflare Tunnel** (`cloudflared`), no Caddy. El túnel abre la conexión desde el VPS hacia Cloudflare (no hay puertos entrantes expuestos en el VPS para la API), Cloudflare termina el TLS y gestiona el DNS del subdominio automáticamente. Se descartó Caddy a propósito: un componente menos para mantener a mano (sin renovación de certificados, sin firewall que administrar para el puerto de la API).
- **Autenticación**: implementación propia en Go usando `markbates/goth` para OAuth (Google, Facebook) + email/contraseña con bcrypt. Sesión vía JWT en cookie `httpOnly`, `Secure`, `SameSite=Lax`. Nunca tokens en `localStorage`.
- **Storage de archivos**: Cloudflare R2 (API compatible S3).
- **Email transaccional**: Resend (capa gratis) para verificación de cuenta y reseteo de contraseña.

## Estructura del repositorio (monorepo)

```
/web            → Astro + TypeScript (frontend)
/api            → Go (backend, API REST)
/migrations     → SQL versionado (golang-migrate)
docker-compose.yml   → levanta postgres + api + (opcional) proxy local
CLAUDE.md       → este archivo
```

No usar un monorepo con herramientas tipo Nx/Turborepo — son complejidad innecesaria para dos proyectos (web/api) mantenidos por una persona. `/web` y `/api` con sus propios `package.json`/`go.mod` alcanza.

## Modelo de datos (fase 1, sujeto a extender, no a rehacer)

Tablas mínimas para el MVP:

- `users`: id, email, password_hash (nullable si vino por OAuth), name, auth_provider (email/google/facebook), created_at.
- `courses`: id, title, slug, description, published (bool), created_at.
- `lessons`: id, course_id (FK), title, order_index, created_at.
- `lesson_content`: id, lesson_id (FK), title, order_index, content_type, youtube_url, pdf_url (R2), created_at
- `enrollments`: id, user_id (FK), course_id (FK), enrolled_at. Se usa de verdad desde la etapa de certificados: se crea con una acción explícita del alumno ("Inscribirme" en el detalle del curso), no automáticamente. Sigue sin gatear acceso — el acceso a un curso sigue siendo 100% por rol (`course_visible_roles`); enrollment es tracking (para saber quién toma qué curso y desde cuándo) y la base para poder sumar precio/pasarela de pago en una etapa futura.
- `lesson_progress`: id, user_id (FK), lesson_id (FK), completed (bool), completed_at.
- `certificates`: id, user_id (FK), course_id (FK), code (único, viaja en la URL pública de verificación), recipient_name y course_title (desnormalizados: el certificado dice lo que decía cuando se emitió, no lo que el curso/usuario se llama hoy), issued_at. Se emite automáticamente al completar el 100% de un curso, solo si el curso tiene `certificate_enabled = true` (columna en `courses`, configurable por el admin). Verificación pública sin login vía `GET /certificates/{code}`.

Diseñá estas tablas pensando en que en fases futuras se agregan: newsletter (tabla `subscribers` separada), noticias (tabla `posts`, reutiliza el mismo patrón markdown que las lecciones), y un feed social (tabla `feed_posts` con `user_id`, `content`, `created_at` — no la implementes ahora, pero no bloquees su llegada con decisiones de esquema difíciles de revertir).

## Fases del proyecto

- **Fase 0 — ✅ completa**: monorepo (`/web` Astro+TS, `/api` Go+chi, `/migrations`), Docker Compose con Postgres 16, esquema inicial migrado, `GET /health` respondiendo.
- **Fase 1 — ✅ completa**: registro/login por email (bcrypt + JWT en cookie `httpOnly`/`Secure`/`SameSite=Lax`), verificación de cuenta y reset de password vía Resend (con fallback a `email.NoopSender` si no hay `RESEND_API_KEY`, para poder desarrollar sin cuenta de Resend), login con Google y Facebook vía goth (con reclamo seguro de cuentas registradas-pero-no-verificadas, ver `UserRepo.ClaimByOAuth` — cierra un hueco de pre-account-takeover), middleware `auth.RequireAuth` + `GET /auth/me`. Todo cableado en `api/main.go` + `api/config.go`.
- **Fase 2 (MVP) — EN CURSO**: landing page, páginas institucionales, 3 cursos con 8 lecciones cada uno (video de YouTube + PDF), progreso básico de usuario. (Registro/login ya está resuelto desde Fase 1, no es parte del alcance de Fase 2.)
  - Etapa "Enrollment + Certificados": inscripción real (con acción explícita del alumno), panel de usuario con inscripciones/progreso, rediseño del detalle de curso, y certificados HTML con QR verificables públicamente. Ver decisión de arquitectura abajo.
- **Fase 3**: noticias/blog, newsletter (Listmonk self-hosted o Resend/Buttondown).
- **Fase 4**: comunidad / feed social.

No implementes nada de Fase 3 o 4 salvo que el usuario lo pida explícitamente. El objetivo de cada sesión de trabajo es avanzar la fase actual, no anticipar features futuras con código.

Fase 2 se trabaja **en pasos chicos**: proponé un paso concreto y acotado, mostrá el resultado (build, verificación funcional contra Postgres/Docker real, capturas si aplica a frontend) antes de pasar al siguiente paso. No avances varios pasos de una sola vez sin mostrar el resultado intermedio — así se viene trabajando desde Fase 0 y funcionó bien.

## Arquitectura del backend

Se evaluó migrar todo `/api` a hexagonal/DDD y se descartó por sobre-ingeniería para este tamaño de proyecto (quedó pendiente de decisión durante Fase 1, se retomó y se resolvió en la etapa "Enrollment + Certificados"). La migración es **incremental por dominio**, no un big-bang:

- Los dominios existentes (`User`, `Course`, `Lesson`, `Auth`) quedan con la estructura plana actual (`handlers`, `models`, `db`, `auth`) — no se tocan salvo extensiones puntuales y aditivas (un campo nuevo, una columna más en un SELECT).
- Los dominios **nuevos** se construyen desde cero con capas `domain/application/infrastructure` bajo `api/internal/<dominio>/`: los puertos (interfaces de repositorio) se definen en `domain`, las implementaciones concretas contra Postgres en `infrastructure/postgres`. `domain`/`application` no conocen SQL ni HTTP.
- El transporte HTTP de los dominios nuevos sigue viviendo en `api/handlers/` junto con todo lo demás — no se creó `infrastructure/http` por dominio, para no fragmentar el único lugar donde hoy se ve toda la superficie HTTP de la API.
- Primeros dominios construidos así: `enrollment` y `certificate`.

## Convenciones de código

- Go: estructura simple por paquetes (`handlers`, `models`, `db`, `auth`) pero con diseño DDD, sin frameworks pesados — `net/http` estándar + `chi` o `gorilla/mux` para routing está bien; evitá frameworks tipo Gin si `chi` alcanza.
- TypeScript: strict mode activado.
- Astro: usar islands de React/Vue solo donde haya interactividad real (login, reproductor, formulario de progreso); todo lo demás estático.
- Commits pequeños y descriptivos. Sin dependencias añadidas "por si acaso".
- Cada nueva dependencia (librería) debe justificarse: ¿resuelve algo que tomaría más de ~30 min escribir a mano? Si no, no se agrega.

## Cómo trabajar en este proyecto

- Antes de generar código para una tarea, confirmá en qué fase estamos y qué parte de esa fase se está atacando.
- Si una tarea implica una decisión de arquitectura no cubierta en este documento, preguntá antes de asumir.
- Priorizá que el código sea legible y modificable por una sola persona sin contexto reciente, por sobre abstracciones "reusables" prematuras.
