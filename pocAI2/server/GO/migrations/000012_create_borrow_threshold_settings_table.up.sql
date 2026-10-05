-- 000012_create_borrow_threshold_settings_table.up.sql
-- Table: public.borrow_threshold_settings (Per-tenant Borrow Anomaly Thresholds)

CREATE TABLE IF NOT EXISTS public.borrow_threshold_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(125) NOT NULL,
    max_duration_days INT NOT NULL DEFAULT 28,
    max_active_loans INT NOT NULL DEFAULT 3,
    max_late_returns INT NOT NULL DEFAULT 2,
    max_extensions INT NOT NULL DEFAULT 2,
    max_asset_count INT NOT NULL DEFAULT 5,
    audit_window_days INT NOT NULL DEFAULT 7,
    repeat_borrow_cycles INT NOT NULL DEFAULT 2,
    min_signals_for_review INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_borrow_threshold_company ON public.borrow_threshold_settings (LOWER(TRIM(company_name)));

-- Function & Trigger to automatically create default threshold settings whenever a new row is added to public.perusahaan
CREATE OR REPLACE FUNCTION trg_auto_create_borrow_thresholds()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.borrow_threshold_settings (company_name)
    VALUES (NEW.nama_perusahaan)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_perusahaan_insert_thresholds ON public.perusahaan;
CREATE TRIGGER trg_perusahaan_insert_thresholds
AFTER INSERT ON public.perusahaan
FOR EACH ROW
EXECUTE FUNCTION trg_auto_create_borrow_thresholds();

-- Seed existing perusahaan into borrow_threshold_settings
INSERT INTO public.borrow_threshold_settings (company_name)
SELECT DISTINCT nama_perusahaan FROM public.perusahaan 
WHERE nama_perusahaan IS NOT NULL AND TRIM(nama_perusahaan) != ''
ON CONFLICT DO NOTHING;

-- Seed default company 'qtera mandiri'
INSERT INTO public.borrow_threshold_settings (company_name)
VALUES ('qtera mandiri')
ON CONFLICT DO NOTHING;
