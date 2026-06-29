CREATE TABLE IF NOT EXISTS meal_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    custom_meal_item_id UUID NULL REFERENCES custom_meal_items(id) ON DELETE SET NULL,
    prebuilt_meal_id UUID NULL REFERENCES prebuilt_meals(id) ON DELETE SET NULL,

    meal_name TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    eaten_at TIMESTAMPTZ NOT NULL,
    meal_category TEXT[] NOT NULL DEFAULT '{}',
    calories NUMERIC(10, 2) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT chk_meal_logs_one_source CHECK (
        (custom_meal_item_id IS NOT NULL AND prebuilt_meal_id IS NULL)
        OR
        (custom_meal_item_id IS NULL AND prebuilt_meal_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_meal_logs_user_eaten_at
ON meal_logs(user_id, eaten_at)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meal_logs_custom_meal_item_id
ON meal_logs(custom_meal_item_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meal_logs_prebuilt_meal_id
ON meal_logs(prebuilt_meal_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meal_logs_meal_category
ON meal_logs USING GIN (meal_category)
WHERE deleted_at IS NULL;