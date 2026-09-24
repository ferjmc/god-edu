-- certificate_enabled: el admin decide, por curso, si completarlo al 100%
-- emite un certificado. Arranca en false — un curso existente no empieza a
-- emitir certificados solo por esta migración.
ALTER TABLE courses ADD COLUMN certificate_enabled BOOLEAN NOT NULL DEFAULT false;

-- certificates: un certificado por usuario+curso, emitido automáticamente
-- al completar el 100% de un curso con certificate_enabled = true.
-- recipient_name y course_title van desnormalizados a propósito: un
-- certificado dice lo que decía cuando se emitió, no lo que el curso o el
-- usuario se llaman hoy.
CREATE TABLE certificates (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    course_id      BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    code           TEXT NOT NULL,
    recipient_name TEXT NOT NULL,
    course_title   TEXT NOT NULL,
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX certificates_code_key ON certificates (code);
CREATE UNIQUE INDEX certificates_user_id_course_id_key ON certificates (user_id, course_id);
CREATE INDEX certificates_user_id_idx ON certificates (user_id);
