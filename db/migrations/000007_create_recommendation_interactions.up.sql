CREATE TABLE IF NOT EXISTS recommendation_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    meal_id UUID NOT NULL,
    meal_source TEXT NOT NULL DEFAULT 'prebuilt',
    position INT NOT NULL CHECK (position >= 1),
    shown_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    clicked_at TIMESTAMPTZ NULL,
    selected_at TIMESTAMPTZ NULL,
    logged_at TIMESTAMPTZ NULL,
    rating INT NULL CHECK (rating BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_recommendation_interactions_meal_source
    CHECK (meal_source IN ('prebuilt', 'custom'))
);

CREATE INDEX IF NOT EXISTS idx_recommendation_interactions_user_shown_at
ON recommendation_interactions(user_id, shown_at DESC);

CREATE INDEX IF NOT EXISTS idx_recommendation_interactions_recommendation
ON recommendation_interactions(recommendation_id, position);

CREATE INDEX IF NOT EXISTS idx_recommendation_interactions_user_meal
ON recommendation_interactions(user_id, meal_source, meal_id, shown_at DESC);
