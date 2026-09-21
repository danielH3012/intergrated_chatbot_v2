-- 000009_create_groups_table.up.sql
-- Table: public.groups (User/Agent Groups and Teams)

CREATE TABLE IF NOT EXISTS public.groups (
    id TEXT PRIMARY KEY,
    perusahaan TEXT DEFAULT '',
    group_name TEXT NOT NULL,
    application TEXT DEFAULT '',
    agent_id TEXT DEFAULT '',
    created_by_user_id TEXT DEFAULT '',
    created_by_username TEXT DEFAULT '',
    created_by_role TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_groups_perusahaan ON public.groups (perusahaan);
CREATE INDEX IF NOT EXISTS idx_groups_name ON public.groups (group_name);
