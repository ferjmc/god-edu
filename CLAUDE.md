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
- **Backend**: Go. API REST/JSON. Desplegado como contenedor Docker en un **VPS Hetzner** (2 vCPU/4GB), administrado con **Coolify o Dokploy** para tener despliegues tipo git-push.
- **Base de datos**: PostgreSQL, corriendo en el mismo VPS (o Postgres gestionado barato si el usuario lo prefiere más adelante).
- **Reverse proxy / HTTPS**: Caddy (o lo que gestione Coolify/Dokploy internamente).
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
- `enrollments`: id, user_id (FK), course_id (FK), enrolled_at. (En fase 1 puede ser "todo usuario logueado tiene acceso a todo", pero modelá la tabla desde ya para no migrar después.)
- `lesson_progress`: id, user_id (FK), lesson_id (FK), completed (bool), completed_at.

Diseñá estas tablas pensando en que en fases futuras se agregan: newsletter (tabla `subscribers` separada), noticias (tabla `posts`, reutiliza el mismo patrón markdown que las lecciones), y un feed social (tabla `feed_posts` con `user_id`, `content`, `created_at` — no la implementes ahora, pero no bloquees su llegada con decisiones de esquema difíciles de revertir).

## Fases del proyecto

- **Fase 0 — ✅ completa**: monorepo (`/web` Astro+TS, `/api` Go+chi, `/migrations`), Docker Compose con Postgres 16, esquema inicial migrado, `GET /health` respondiendo.
- **Fase 1 — ✅ completa**: registro/login por email (bcrypt + JWT en cookie `httpOnly`/`Secure`/`SameSite=Lax`), verificación de cuenta y reset de password vía Resend (con fallback a `email.NoopSender` si no hay `RESEND_API_KEY`, para poder desarrollar sin cuenta de Resend), login con Google y Facebook vía goth (con reclamo seguro de cuentas registradas-pero-no-verificadas, ver `UserRepo.ClaimByOAuth` — cierra un hueco de pre-account-takeover), middleware `auth.RequireAuth` + `GET /auth/me`. Todo cableado en `api/main.go` + `api/config.go`.
  - Pendiente de decisión (no bloquea Fase 2): se evaluó migrar `/api` a arquitectura hexagonal con capas Domain/Application/Infrastructure por dominio; se decidió no hacerlo sin evaluarlo con calma porque contradice la estructura simple por paquetes de este documento. Queda la estructura plana (`handlers`, `models`, `db`, `auth`, `email`). Retomar esta conversación antes de que el código crezca mucho más, si se quiere.
- **Fase 2 (MVP) — EN CURSO**: landing page, páginas institucionales, 3 cursos con 8 lecciones cada uno (video de YouTube + PDF), progreso básico de usuario. (Registro/login ya está resuelto desde Fase 1, no es parte del alcance de Fase 2.)
- **Fase 3**: noticias/blog, newsletter (Listmonk self-hosted o Resend/Buttondown).
- **Fase 4**: comunidad / feed social.

No implementes nada de Fase 3 o 4 salvo que el usuario lo pida explícitamente. El objetivo de cada sesión de trabajo es avanzar la fase actual, no anticipar features futuras con código.

Fase 2 se trabaja **en pasos chicos**: proponé un paso concreto y acotado, mostrá el resultado (build, verificación funcional contra Postgres/Docker real, capturas si aplica a frontend) antes de pasar al siguiente paso. No avances varios pasos de una sola vez sin mostrar el resultado intermedio — así se viene trabajando desde Fase 0 y funcionó bien.

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
