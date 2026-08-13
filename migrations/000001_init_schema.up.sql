-- users
CREATE TABLE users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         TEXT NOT NULL,
    password_hash TEXT,
    name          TEXT NOT NULL,
    auth_provider TEXT NOT NULL CHECK (auth_provider IN ('email', 'google', 'facebook')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_key ON users (email);

-- courses
CREATE TABLE courses (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title       TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT,
    published   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX courses_slug_key ON courses (slug);

-- lessons
CREATE TABLE lessons (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id   BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    order_index INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lessons_course_id_idx ON lessons (course_id);
CREATE UNIQUE INDEX lessons_course_id_order_index_key ON lessons (course_id, order_index);

-- lesson_content (video de YouTube, PDF o markdown de cada lección)
CREATE TABLE lesson_content (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lesson_id    BIGINT NOT NULL REFERENCES lessons (id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    order_index  INTEGER NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('video', 'pdf', 'markdown')),
    youtube_url  TEXT,
    pdf_url      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lesson_content_lesson_id_idx ON lesson_content (lesson_id);
CREATE UNIQUE INDEX lesson_content_lesson_id_order_index_key ON lesson_content (lesson_id, order_index);

-- enrollments
CREATE TABLE enrollments (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    course_id   BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX enrollments_user_id_idx ON enrollments (user_id);
CREATE INDEX enrollments_course_id_idx ON enrollments (course_id);
CREATE UNIQUE INDEX enrollments_user_id_course_id_key ON enrollments (user_id, course_id);

-- lesson_progress
CREATE TABLE lesson_progress (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    lesson_id    BIGINT NOT NULL REFERENCES lessons (id) ON DELETE CASCADE,
    completed    BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ
);

CREATE INDEX lesson_progress_user_id_idx ON lesson_progress (user_id);
CREATE INDEX lesson_progress_lesson_id_idx ON lesson_progress (lesson_id);
CREATE UNIQUE INDEX lesson_progress_user_id_lesson_id_key ON lesson_progress (user_id, lesson_id);
