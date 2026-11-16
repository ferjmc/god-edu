-- course_classes: agrupa las lecciones de un curso en "clases". En la
-- interfaz, cada fila de lessons es una "sesión" y sigue siendo la unidad
-- de progreso y la que cuenta para el certificado — una clase no tiene
-- progreso propio, se calcula a partir de sus sesiones.
--
-- published nace en false, igual que un curso en borrador: el admin carga
-- las sesiones de una clase oculta y la publica cuando está lista. Las
-- sesiones de una clase oculta no se le muestran al alumno ni cuentan en
-- su progreso, pero sí cuentan para el certificado.
CREATE TABLE course_classes (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id   BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    order_index INTEGER NOT NULL,
    published   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX course_classes_course_id_order_index_key ON course_classes (course_id, order_index);

-- Nullable a propósito: los cursos existentes no necesitan backfill (sus
-- sesiones quedan "sin clase" hasta que el admin las asigne). SET NULL:
-- borrar una clase nunca borra sesiones ni el progreso de los alumnos.
-- Que la clase sea del mismo curso que la lección lo valida
-- lessonapp.Service, no un constraint.
ALTER TABLE lessons
    ADD COLUMN class_id BIGINT NULL REFERENCES course_classes (id) ON DELETE SET NULL;

CREATE INDEX lessons_class_id_idx ON lessons (class_id);
