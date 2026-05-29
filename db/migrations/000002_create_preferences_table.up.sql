CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    main_goal TEXT NOT NULL,
    monthly_meal_budget NUMERIC(10, 2) NOT NULL,
    data_sharing_consent BOOLEAN, 
    home_location TEXT NOT NULL,
    work_school_location TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS user_health_concerns (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    concern TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, concern)
);

CREATE TABLE IF NOT EXISTS user_dietary_restrictions (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    restriction TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, restriction)
);

CREATE TABLE IF NOT EXISTS user_meal_preferences (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    preference_tag TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, preference_tag)
);