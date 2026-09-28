"""
Feature schema loader 

This file intentionally reads the existing schema owned by the Go backend:

    internal/service/recommendation/features/schema.json

That prevents the Python training pipeline from inventing a different
feature order than the Go inference path will later use.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

# Resolve repo root from this file: 
# ml/src/features.py -> ml/src -> ml -> repo root
REPO_ROOT = Path(__file__).resolve().parents[2]

# Single source of truth created in Section 3
SCHEMA_PATH = REPO_ROOT / "internal/service/recommendation/features/schema.json"

def load_feature_schema() -> dict[str, any]:
    """
    Load the shared feature schema.

    The schema defines:
    - metadata columns to preserve
    - feature names
    - feature order
    - feature types
    - missing-value policies
    - deferred features
    - forbidden safety features
    """
    with SCHEMA_PATH.open("r", encoding="utf-8") as file:
        return json.load(file)

def feature_names() -> list[str]:
    """
    Return active model feature names in exact model order.

    This order must be used for:
    - dataset creation
    - training
    - evaluation
    - online prediction

    Changing this order changes the model input contract.
    """
    schema = load_feature_schema()
    return [feature["name"] for feature in schema["features"]]

def metadata_columns() -> list[str]:
    """Return columns preserved for traceability but not model input."""
    schema = load_feature_schema()
    return schema["metadata_columns"]

def missing_value_by_feature() -> dict[str, any]:
    """Return per-feature imputation defaults from the schema"""
    schema = load_feature_schema()
    return {
        feature["name"]: feature["missing_value"]
        for feature in schema["features"]
    }
