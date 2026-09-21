-- 000005_create_assets_table.up.sql
-- Table: public.assets (IT Assets & Inventory Management)

CREATE TABLE IF NOT EXISTS public.assets (
    asset_id VARCHAR(125) CONSTRAINT "assets_Asset_ID_not_null" NOT NULL,
    name VARCHAR(125),
    category VARCHAR(125),
    brand VARCHAR(125),
    model_type VARCHAR(125),
    purchase_date VARCHAR(125),
    purchase_price VARCHAR(125),
    location VARCHAR(125),
    created_at VARCHAR(125),
    perusahaan VARCHAR(125),
    status VARCHAR(125),
    CONSTRAINT assets_pkey PRIMARY KEY (asset_id)
);

CREATE INDEX IF NOT EXISTS idx_assets_perusahaan ON public.assets (perusahaan);
CREATE INDEX IF NOT EXISTS idx_assets_category ON public.assets (category);
CREATE INDEX IF NOT EXISTS idx_assets_brand ON public.assets (brand);
CREATE INDEX IF NOT EXISTS idx_assets_location ON public.assets (location);
CREATE INDEX IF NOT EXISTS idx_assets_status ON public.assets (status);
