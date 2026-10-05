-- 000012_create_borrow_threshold_settings_table.down.sql
DROP TRIGGER IF EXISTS trg_perusahaan_insert_thresholds ON public.perusahaan;
DROP FUNCTION IF EXISTS trg_auto_create_borrow_thresholds();
DROP TABLE IF EXISTS public.borrow_threshold_settings CASCADE;
