-- 000004_create_users_table.up.sql
-- Table: public.users (Core user accounts, credentials, and tenancy)

CREATE TABLE IF NOT EXISTS public.users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL CONSTRAINT users_username_key UNIQUE,
    email TEXT NOT NULL CONSTRAINT users_email_key UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'operator',
    company TEXT DEFAULT '',
    id_perusahaan BIGINT REFERENCES public.perusahaan(id_perusahaan) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON public.users (username);
CREATE INDEX IF NOT EXISTS idx_users_email ON public.users (email);
CREATE INDEX IF NOT EXISTS idx_users_id_perusahaan ON public.users (id_perusahaan);
