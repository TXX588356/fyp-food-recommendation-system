CREATE TABLE IF NOT EXISTS custom_meal_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    calories NUMERIC(10, 2) NOT NULL,
    fat_g NUMERIC(10, 2) NOT NULL,
    protein_g NUMERIC(10, 2) NOT NULL,
    carbs_g NUMERIC(10, 2) NOT NULL,
    state TEXT NOT NULL,
    district TEXT NOT NULL,
    restaurant_name TEXT NOT NULL,
    image_url TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS custom_meal_dietary_restriction_tags (
    custom_meal_item_id UUID REFERENCES custom_meal_items(id) ON DELETE CASCADE,
    dietary_restriction_tag TEXT NOT NULL,
    PRIMARY KEY (custom_meal_item_id, dietary_restriction_tag)
);

CREATE TABLE IF NOT EXISTS custom_meal_categories (
    custom_meal_item_id UUID REFERENCES custom_meal_items(id) ON DELETE CASCADE,
    meal_category TEXT NOT NULL,
    PRIMARY KEY (custom_meal_item_id, meal_category)
);