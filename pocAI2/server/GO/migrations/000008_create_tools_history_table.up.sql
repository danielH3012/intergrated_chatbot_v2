-- 000008_create_tools_history_table.up.sql
-- Table: public.tools_history (AI Tools Execution Audit Trail)

CREATE TABLE IF NOT EXISTS public.tools_history (
    id TEXT NOT NULL,
    perusahaan TEXT DEFAULT '' NOT NULL,
    tools TEXT DEFAULT '' NOT NULL,
    created_by_user_id TEXT DEFAULT '',
    created_by_username TEXT DEFAULT '',
    created_by_role TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT tools_history_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_tools_history_perusahaan ON public.tools_history (perusahaan);
CREATE INDEX IF NOT EXISTS idx_tools_history_created_at ON public.tools_history (created_at);
