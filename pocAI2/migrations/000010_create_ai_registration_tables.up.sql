-- 000010_create_ai_registration_tables.up.sql
-- Tables: public.ai_registration_sessions & public.ai_registration_drafts

CREATE TABLE IF NOT EXISTS public.ai_registration_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    source_file_url TEXT,
    raw_text TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'processing',
    reject_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_reg_sessions_user_id ON public.ai_registration_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_ai_reg_sessions_status ON public.ai_registration_sessions (status);

CREATE TABLE IF NOT EXISTS public.ai_registration_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES public.ai_registration_sessions(id) ON DELETE CASCADE,
    sequence_no INT NOT NULL DEFAULT 1,
    fields JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_asset_id VARCHAR(125),
    low_confidence BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_reg_drafts_session_id ON public.ai_registration_drafts (session_id);
CREATE INDEX IF NOT EXISTS idx_ai_reg_drafts_status ON public.ai_registration_drafts (status);
