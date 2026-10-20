-- Las tres quedan nullable: hay cuentas OAuth existentes que no van a tener
-- estos valores desde el día uno. terms_accepted_at y signup_ip se completan
-- en el alta (email/password o primera vez por OAuth); birth_date se pide
-- después si falta, incluso para cuentas OAuth viejas (ver oauth.go).
ALTER TABLE users
    ADD COLUMN terms_accepted_at TIMESTAMPTZ NULL,
    ADD COLUMN birth_date        DATE        NULL,
    ADD COLUMN signup_ip         TEXT        NULL;
