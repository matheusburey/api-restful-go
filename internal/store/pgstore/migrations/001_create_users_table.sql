-- This is a sample migration.
CREATE TABLE IF NOT EXISTS users (
    id UUID primary key NOT NULL DEFAULT gen_random_uuid(),
    name VARCHAR(80) UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash BYTEA NOT NULL,
    bio TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );

---- create above / drop below ----
DROP TABLE IF EXISTS users;