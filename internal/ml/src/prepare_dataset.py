"""
Prepare Phase 2 recommendation training dataset.

Dataset rule:
- one row per recommendation_interactions row
- labels come only from shown recommendations
- no synthetic negatives from unseen meals
- history-derived features only use data at or before shown_at
"""

from __future__ import annotations

import argparse
import os
from pathlib import Path

import pandas as pd
import psycopg

from features import features_names, metadata_columns, missing_value_by_feature, load_feature_schema

def build_dataset_query() -> str:
    """
    SQL extraction query.

    Notes:
    - recommendation_interactions is the anchor table.
    - prebuilt/custom meals are joined based on meal_source.
    - meal logs are only counted when eaten_at <= shown_at.
    - positive labels use selected_at/logged_at.
    - negative labels require the observation window to have passed.
    - unresolved recent impressions are excluded.
    """
    return """
    WITH interaction_rows AS (
        SELECT
            ri.recommendation_id,
            ri.user_id,
            ri.meal_source,
            ri.meal_id,
            ri.position,
            ri.shown_at,
            ri.clicked_at,
            ri.selected_at,
            ri.logged_at,
            ri.rating,

            CASE
                WHEN ri.selected_at IS NOT NULL OR ri.logged_at IS NOT NULL THEN 1
                WHEN ri.shown_at <= (now() - (%(observation_window_days)s || ' days')::interval) THEN 0
                ELSE NULL
            END AS label
        FROM recommendation_interactions ri
    ),

    meal_rows AS (
        SELECT
            i.*,

            COALESCE(cm.price, 0) AS custom_price,

            CASE
                WHEN i.meal_source = 'custom' THEN cm.name
                ELSE pm.name
            END AS meal_name,

            CASE
                WHEN i.meal_source = 'custom' THEN cm.calories
                ELSE COALESCE(pm.calories, 0)
            END AS calories,

            CASE
                WHEN i.meal_source = 'custom' THEN cm.protein_g
                ELSE COALESCE(pm.protein_g, 0)
            END AS protein,

            CASE
                WHEN i.meal_source = 'custom' THEN cm.carbs_g
                ELSE COALESCE(pm.carbs_g, 0)
            END AS carbs,

            CASE
                WHEN i.meal_source = 'custom' THEN cm.fat_g
                ELSE COALESCE(pm.fat_g, 0)
            END AS fat,

            CASE
                WHEN i.meal_source = 'custom' THEN ARRAY(
                    SELECT c.meal_category
                    FROM custom_meal_categories c
                    WHERE c.custom_meal_item_id = cm.id
                )
                ELSE pm.category_codes
            END AS meal_categories

        FROM interaction_rows i
        LEFT JOIN prebuilt_meals pm
            ON i.meal_source = 'prebuilt'
            AND i.meal_id = pm.id
        LEFT JOIN custom_meal_items cm
            ON i.meal_source = 'custom'
            AND i.meal_id = cm.id
    ),

    user_preferred_tags AS (
        SELECT
            user_id,
            array_agg(preference_tag) AS preferred_tags
        FROM user_meal_preferences
        WHERE deleted_at IS NULL
        GROUP BY user_id
    )

    SELECT
        m.recommendation_id,
        m.user_id,
        m.meal_source,
        m.meal_id,
        m.shown_at,
        m.label,

        -- meal_price:
        -- For custom meals, use stored custom meal price.
        -- For prebuilt meals, current schema has no persisted shown price,
        -- so default to 0 until price snapshotting exists.
        CASE
            WHEN m.meal_source = 'custom' THEN COALESCE(m.custom_price, 0)
            ELSE 0
        END AS meal_price,

        COALESCE(m.calories, 0) AS calories,
        COALESCE(m.protein, 0) AS protein,
        COALESCE(m.carbs, 0) AS carbs,
        COALESCE(m.fat, 0) AS fat,

        -- calorie_difference:
        -- No explicit calorie target exists yet, so follow schema policy:
        -- use 0 rather than fabricating a target.
        0 AS calorie_difference,

        -- Historical spending features use only logs before shown_at.
        COALESCE(hist.meal_price - hist.average_meal_spending, 0) AS price_difference_from_avg,
        COALESCE(hist.times_meal_eaten, 0) AS times_meal_eaten,
        COALESCE(hist.times_category_eaten, 0) AS times_category_eaten,
        COALESCE(hist.days_since_last_eaten, 999) AS days_since_last_eaten,
        COALESCE(hist.remaining_monthly_budget, 0) AS remaining_monthly_budget,
        COALESCE(hist.average_meal_spending, 0) AS average_meal_spending,

        CASE
            WHEN COALESCE(array_length(m.meal_categories, 1), 0) = 0 THEN 0
            WHEN COALESCE(array_length(u.preferred_tags, 1), 0) = 0 THEN 0
            WHEN m.meal_categories && u.preferred_tags THEN 1
            ELSE 0
        END AS category_match,

        -- Current interactions do not store request meal category.
        -- Use schema default until meal_type is captured at impression time.
        'other' AS meal_type

    FROM meal_rows m
    LEFT JOIN user_preferences p
        ON p.user_id = m.user_id
        AND p.deleted_at IS NULL
    LEFT JOIN user_preferred_tags u
        ON u.user_id = m.user_id
    LEFT JOIN LATERAL (
        SELECT
            CASE
                WHEN m.meal_source = 'custom' THEN COALESCE(m.custom_price, 0)
                ELSE 0
            END AS meal_price,

            AVG(ml.price) AS average_meal_spending,

            COUNT(*) FILTER (
                WHERE
                    (m.meal_source = 'prebuilt' AND ml.prebuilt_meal_id = m.meal_id)
                    OR
                    (m.meal_source = 'custom' AND ml.custom_meal_item_id = m.meal_id)
            ) AS times_meal_eaten,

            COUNT(*) FILTER (
                WHERE ml.meal_category && COALESCE(m.meal_categories, ARRAY[]::text[])
            ) AS times_category_eaten,

            EXTRACT(
                DAY FROM m.shown_at - MAX(ml.eaten_at) FILTER (
                    WHERE
                        (m.meal_source = 'prebuilt' AND ml.prebuilt_meal_id = m.meal_id)
                        OR
                        (m.meal_source = 'custom' AND ml.custom_meal_item_id = m.meal_id)
                )
            ) AS days_since_last_eaten,

            GREATEST(
                COALESCE(p.monthly_meal_budget, 0)
                - COALESCE(SUM(ml.price) FILTER (
                    WHERE date_trunc('month', ml.eaten_at) = date_trunc('month', m.shown_at)
                ), 0),
                0
            ) AS remaining_monthly_budget

        FROM meal_logs ml
        WHERE ml.user_id = m.user_id
          AND ml.deleted_at IS NULL
          AND ml.eaten_at <= m.shown_at
    ) hist ON TRUE

    WHERE m.label IS NOT NULL
    ORDER BY m.shown_at ASC, m.recommendation_id, m.position;
    """

def clean_and_order_dataset(df: pd.DataFrame) -> pd.DataFrame:
    """Apply schema-defined imputation and column ordering"""
    defaults = missing_value_by_feature()
    features = features_names()
    metadata = metadata_columns()

    for column, default in defaults.items():
        if column not in df.columns:
            df[column] = default
        df[column] = df[column].fillna(default)
