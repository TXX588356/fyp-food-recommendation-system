package features

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed schema.json
var schemaJSON []byte

type Schema struct {
	Version               string            `json:"version"`
	ObservationWindowDays int               `json:"observation_window_days"`
	MetadataColumns       []string          `json:"metadata_columns"`
	Features              []Feature         `json:"features"`
	DeferredFeatures      []DeferredFeature `json:"deferred_features"`
	ForbiddenFeatures     []string          `json:"forbidden_features"`
	HistoryCutoffPolicy   string            `json:"history_cutoff_policy"`
}

type Feature struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Source        string   `json:"source"`
	MissingValue  any      `json:"missing_value"`
	MissingPolicy string   `json:"missing_policy"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Reliable      bool     `json:"reliable"`
}

type DeferredFeature struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func LoadSchema() (Schema, error) {
	var schema Schema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return Schema{}, fmt.Errorf("parse recommendation feature schema: %w", err)
	}

	return schema, nil
}

func MustLoadSchema() Schema {
	schema, err := LoadSchema()
	if err != nil {
		panic(err)
	}

	return schema
}

func FeatureNames() []string {
	schema := MustLoadSchema()
	names := make([]string, 0, len(schema.Features))

	for _, feature := range schema.Features {
		names = append(names, feature.Name)
	}

	return names
}
