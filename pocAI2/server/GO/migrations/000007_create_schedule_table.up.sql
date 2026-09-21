-- 000007_create_schedule_table.up.sql
-- Table: public.schedule (Asset Inspection & Audit Scheduling)

CREATE TABLE IF NOT EXISTS public.schedule (
    id_schedule UUID DEFAULT gen_random_uuid() NOT NULL,
    category VARCHAR(125),
    start VARCHAR(125),
    frequency VARCHAR(125),
    CONSTRAINT schedule_pkey PRIMARY KEY (id_schedule)
);

CREATE INDEX IF NOT EXISTS idx_schedule_category ON public.schedule (category);
CREATE INDEX IF NOT EXISTS idx_schedule_start ON public.schedule (start);
