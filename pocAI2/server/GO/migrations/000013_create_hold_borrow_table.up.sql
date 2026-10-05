-- 000013_create_hold_borrow_table.up.sql
-- Table: public.hold_borrow
-- Holds AI-generated borrow transactions before manual user confirmation/approval

CREATE TABLE IF NOT EXISTS public.hold_borrow (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hold_code VARCHAR(50) NOT NULL UNIQUE,
    borrower_id TEXT NOT NULL,
    borrower_name TEXT NOT NULL,
    manager_id TEXT NOT NULL,
    manager_name TEXT NOT NULL,
    group_name TEXT NOT NULL,
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'held',
    notes TEXT DEFAULT 'Created via AI Assistant',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hold_borrow_status ON public.hold_borrow (status);
CREATE INDEX IF NOT EXISTS idx_hold_borrow_group ON public.hold_borrow (group_name);
CREATE INDEX IF NOT EXISTS idx_hold_borrow_borrower ON public.hold_borrow (borrower_id);
CREATE INDEX IF NOT EXISTS idx_hold_borrow_created_at ON public.hold_borrow (created_at DESC);
