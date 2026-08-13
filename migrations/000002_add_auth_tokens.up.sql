ALTER TABLE users
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;

-- Tokens de un solo uso para verificación de email y reset de password.
-- Se guarda el hash del token, nunca el valor en texto plano: si la tabla
-- se filtra, los tokens ya emitidos no sirven para nada.
CREATE TABLE auth_tokens (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    purpose    TEXT NOT NULL CHECK (purpose IN ('email_verification', 'password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX auth_tokens_token_hash_key ON auth_tokens (token_hash);
CREATE INDEX auth_tokens_user_id_purpose_idx ON auth_tokens (user_id, purpose);
