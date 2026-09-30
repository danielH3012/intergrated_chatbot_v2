-- 1. Asset condition
ALTER TABLE public.assets ADD COLUMN IF NOT EXISTS condition VARCHAR(50) DEFAULT 'Normal';
UPDATE public.assets SET condition = 'Normal' WHERE condition IS NULL;

-- 2. Schedule group link
ALTER TABLE public.schedule ADD COLUMN IF NOT EXISTS group_name VARCHAR(125) DEFAULT 'IT Department';

-- 3. Simple Borrow Transactions table
CREATE TABLE IF NOT EXISTS public.borrow_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_code VARCHAR(50) UNIQUE NOT NULL,
    borrower_id TEXT NOT NULL,
    borrower_name VARCHAR(125) NOT NULL,
    manager_id TEXT,
    manager_name VARCHAR(125) NOT NULL,
    group_name VARCHAR(125) NOT NULL,
    request_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending_approval',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Simple Borrow Items table
CREATE TABLE IF NOT EXISTS public.borrow_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES public.borrow_transactions(id) ON DELETE CASCADE,
    asset_id VARCHAR(125) NOT NULL REFERENCES public.assets(asset_id),
    duration_days INT NOT NULL DEFAULT 7,
    approved_duration_days INT,
    note TEXT,
    is_approved BOOLEAN,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    borrowed_at TIMESTAMPTZ,
    due_date TIMESTAMPTZ,
    returned_at TIMESTAMPTZ
);

-- 5. Simple Extensions table
CREATE TABLE IF NOT EXISTS public.borrow_extensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES public.borrow_transactions(id) ON DELETE SET NULL,
    borrower_id TEXT NOT NULL,
    request_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    days_requested INT NOT NULL DEFAULT 7
);
