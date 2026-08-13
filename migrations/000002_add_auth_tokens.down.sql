DROP TABLE IF EXISTS auth_tokens;

ALTER TABLE users
    DROP COLUMN email_verified;
