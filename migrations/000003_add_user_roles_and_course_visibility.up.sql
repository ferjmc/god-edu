-- roles de usuario
ALTER TABLE users
    ADD COLUMN role TEXT NOT NULL DEFAULT 'PUBLIC_MEMBER'
        CHECK (role IN ('ADMIN', 'COMMUNITY_MEMBER', 'PUBLIC_MEMBER', 'PAID_MEMBER'));

-- visibilidad de cursos por rol: un curso sin filas acá es visible para
-- cualquier usuario logueado (opt-in por curso, no opt-out — no rompe los
-- cursos existentes sin restricción configurada).
CREATE TABLE course_visible_roles (
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    role      TEXT NOT NULL CHECK (role IN ('ADMIN', 'COMMUNITY_MEMBER', 'PUBLIC_MEMBER', 'PAID_MEMBER')),
    PRIMARY KEY (course_id, role)
);
