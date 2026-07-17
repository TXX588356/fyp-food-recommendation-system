CREATE TABLE IF NOT EXISTS prebuilt_meals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_code TEXT NOT NULL,
    source_record_id TEXT NOT NULL,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL CHECK (btrim(normalized_name) <> ''),
    category_codes TEXT[] NOT NULL DEFAULT '{}'::text[],
    serving_description TEXT NOT NULL CHECK (btrim(serving_description) <> ''),
    calories NUMERIC CHECK (calories >= 0),
    protein_g NUMERIC CHECK (protein_g >= 0),
    carbs_g NUMERIC CHECK (carbs_g >= 0),
    fat_g NUMERIC CHECK (fat_g >= 0),
    fiber_g NUMERIC CHECK (fiber_g >= 0),
    sugar_g NUMERIC CHECK (sugar_g >= 0),
    sodium_mg NUMERIC CHECK (sodium_mg >= 0),
    cholesterol_mg NUMERIC CHECK (cholesterol_mg >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_code, source_record_id)
);

CREATE INDEX IF NOT EXISTS prebuilt_meals_normalized_name_idx ON prebuilt_meals(normalized_name, id);
CREATE INDEX IF NOT EXISTS prebuilt_meals_source_idx ON prebuilt_meals(source_code);
CREATE INDEX IF NOT EXISTS prebuilt_meals_category_codes_idx ON prebuilt_meals USING GIN(category_codes);

CREATE TABLE IF NOT EXISTS meal_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    display_order INTEGER NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS prebuilt_meal_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prebuilt_meal_id UUID NOT NULL REFERENCES prebuilt_meals(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT 'wikimedia_commons' CHECK (provider = 'wikimedia_commons'),
    provider_file_id TEXT NOT NULL,
    commons_file_title TEXT,
    commons_page_url TEXT,
    original_file_url TEXT,
    minio_object_key TEXT,
    mime_type TEXT,
    width INTEGER CHECK (width > 0),
    height INTEGER CHECK (height > 0),
    file_size_bytes BIGINT CHECK (file_size_bytes > 0),
    sha256 TEXT,
    author_name TEXT,
    author_url TEXT,
    license_name TEXT,
    license_url TEXT,
    attribution_text TEXT,
    search_query TEXT,
    match_score NUMERIC CHECK (match_score >= 0 AND match_score <= 1),
    scoring_policy_version TEXT NOT NULL,
    model_version TEXT,
    match_status TEXT NOT NULL CHECK (match_status IN ('auto_accepted', 'needs_review', 'approved', 'rejected', 'no_match')),
    validation_details JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    review_reason TEXT,
    reviewed_at TIMESTAMPTZ,
    rematch_requested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (prebuilt_meal_id, provider, provider_file_id),
    CHECK (NOT is_primary OR (
        match_status IN ('auto_accepted', 'approved')
        AND minio_object_key IS NOT NULL
        AND mime_type IS NOT NULL
        AND width IS NOT NULL
        AND height IS NOT NULL
        AND sha256 IS NOT NULL
        AND license_name IS NOT NULL
        AND attribution_text IS NOT NULL
    ))
);
CREATE UNIQUE INDEX IF NOT EXISTS prebuilt_meal_one_primary_image_idx ON prebuilt_meal_images(prebuilt_meal_id) WHERE is_primary;
CREATE INDEX IF NOT EXISTS prebuilt_meal_images_review_idx ON prebuilt_meal_images(match_status, match_score, id);
CREATE INDEX IF NOT EXISTS prebuilt_meal_images_rematch_idx ON prebuilt_meal_images(rematch_requested_at) WHERE rematch_requested_at IS NOT NULL;

INSERT INTO meal_categories (code, label, display_order) VALUES
('malaysian','Malaysian',1),('singaporean','Singaporean',2),('indonesian','Indonesian',3),('chinese','Chinese',4),('indian','Indian',5),('thai','Thai',6),('vietnamese','Vietnamese',7),('japanese','Japanese',8),('korean','Korean',9),('middle_eastern','Middle Eastern',10),('american','American',11),('mexican','Mexican',12),('italian','Italian',13),('french','French',14),('greek','Greek',15),('spanish','Spanish',16),('western','Western',17),
('breakfast','Breakfast',18),('rice_dishes','Rice Dishes',19),('noodle_dishes','Noodle Dishes',20),('soups','Soups',21),('stews','Stews',22),('curries','Curries',23),('stir_fries','Stir-Fries',24),('grilled_roasted','Grilled & Roasted',25),('fried_foods','Fried Foods',26),('salads','Salads',27),('sandwiches_wraps','Sandwiches & Wraps',28),('breads_flatbreads','Breads & Flatbreads',29),('porridge','Porridge',30),('dumplings','Dumplings',31),('snacks','Snacks',32),('desserts','Desserts',33),('kuih','Kuih',34),('beverages','Beverages',35),('condiments_sauces','Condiments & Sauces',36),
('poultry','Poultry',37),('beef','Beef',38),('pork','Pork',39),('lamb','Lamb',40),('seafood','Seafood',41),('eggs','Eggs',42),('tofu_soy','Tofu & Soy',43),('legumes','Legumes',44),('vegetables','Vegetables',45),('fruits','Fruits',46),('grains','Grains',47),('dairy','Dairy',48),('nuts_seeds','Nuts & Seeds',49)
ON CONFLICT (code) DO NOTHING;
