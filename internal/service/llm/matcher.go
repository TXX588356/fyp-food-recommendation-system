package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

// Adds a prompt builder for ambiguous meal matching
// a prompt to gemini's MATCH / NO_MATCH decisions
// Client.ResolveMatches
// parser-lvl validation before backend allowlist validation
const (
	// MatchDecisionMatch means Gemini selected one candidate from the supplied
	// candidate list
	MatchDecisionMatch = "MATCH"

	// MatchDecisionNoMatch means Gemini decided that none of the supplied
	// candidates are semantically equivalent to the generated meal.
	MatchDecisionNoMatch = "NO_MATCH"
)

// matcherPayload is the top-level JSON payload embedded in the Gemini prompt.
//
// It contains all ambiguous meal matching tasks in one request so the
// recommendation service needs at most one additional Gemini call.
type matcherPayload struct {
	Tasks []matcherTask `json:"tasks"`
}

// matcherTask is the compact prompt shape for one ambiguous generated meal.
//
// It intentionally uses only identity-relevant fields. Nutrition is not included
// because calories/macros do not prove that two names refer to the same food.
type matcherTask struct {
	MealIndex     int                  `json:"meal_index"`
	GeneratedMeal matcherGeneratedMeal `json:"generated_meal"`
	Candidates    []matcherCandidate   `json:"candidates"`
}

// matcherGeneratedMeal is the generated meal identity information sent to Gemini.
//
// The generated meal name and alternative search terms help Gemini understand
// what food identity was intended by the recommendation generator.
type matcherGeneratedMeal struct {
	Name                   string   `json:"name"`
	AlternativeSearchTerms []string `json:"alternative_search_terms,omitempty"`
}

// matcherCandidate is the compact candidate shape sent to Gemini.
//
// It omits nutrition values because the final accepted nutrition must come from
// backend-owned records, not from LLM reasoning.
type matcherCandidate struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Tags []string `json:"tags,omitempty"`
}

// mealMatchResponse is the JSON shape Gemini must return.
//
// The parser validates this shape before the recommendation package validates
// candidate IDs against the backend allowlist.
type mealMatchResponse struct {
	Decisions []interfaces.MealMatchDecision `json:"decisions"`
}

func (c Client) ResolveMatches(ctx context.Context, tasks []interfaces.MealMatchTask) ([]interfaces.MealMatchDecision, error) {
	if len(tasks) == 0 {
		return []interfaces.MealMatchDecision{}, nil
	}

	// Build one prompt containing every ambiguous task
	prompt, err := BuildMealMatchingPrompt(tasks)
	if err != nil {
		return nil, err
	}
	slog.Info("gemini prompt", "operation", "resolve_matches", "model", c.model, "prompt", prompt)

	// Use the same configured Gemini model as meal generation/autocomplete
	result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), nil)
	if err != nil {
		return nil, err
	}

	// Parse and validate the response shape. Candidate ID membership is checked
	// later by the recommendation package against backend-owned candidate lists.
	return ParseMealMatchDecisions(result.Text())
}

// ParseMealMatchDecisions removes optional Markdown fencing, decodes Gemini's
// JSON response, and validates the response shape.
//
// This parser validates only generic shape rules. It intentionally does not
// check whether candidate_id belongs to the supplied candidate list; that is done
// later by recommendation.validateDecisions using backend-owned allowlists.
func ParseMealMatchDecisions(text string) ([]interfaces.MealMatchDecision, error) {
	cleaned := cleanJSONText(text)

	var response mealMatchResponse
	decoder := json.NewDecoder(bytes.NewReader([]byte(cleaned)))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("parse Gemini meal match response: %w", err)
	}

	if err := validateMealMatchDecisions(response.Decisions); err != nil {
		return nil, fmt.Errorf("validate Gemini meal match response: %w", err)
	}

	return response.Decisions, nil
}

// validateMealMatchDecisions checks parser-level constraints for Gemini match
// decisions.
//
// It rejects malformed decision shapes, but does not validate candidate
// membership. Backend allowlist validation handles that later.
func validateMealMatchDecisions(decisions []interfaces.MealMatchDecision) error {
	for index, decision := range decisions {
		if decision.MealIndex < 0 {
			return fmt.Errorf("decision %d: meal_index cannot be negative", index)
		}

		switch decision.Decision {
		case MatchDecisionMatch:
			// MATCH must include the selected candidate ID.
			if strings.TrimSpace(decision.CandidateID) == "" {
				return fmt.Errorf("decision %d: MATCH requires candidate_id", index)
			}

		case MatchDecisionNoMatch:
			// NO_MATCH must not include a candidate ID.
			if strings.TrimSpace(decision.CandidateID) != "" {
				return fmt.Errorf("decision %d: NO_MATCH cannot include candidate_id", index)
			}

		default:
			return fmt.Errorf("decision %d: unsupported decision %q", index, decision.Decision)
		}
	}

	return nil
}

func BuildMealMatchingPrompt(tasks []interfaces.MealMatchTask) (string, error) {
	payload := matcherPayload{
		Tasks: make([]matcherTask, 0, len(tasks)),
	}

	for _, task := range tasks {
		matcherTask := matcherTask{
			MealIndex: task.MealIndex,
			GeneratedMeal: matcherGeneratedMeal{
				Name:                   task.GeneratedMeal.Name,
				AlternativeSearchTerms: task.GeneratedMeal.AlternativeSearchTerms,
			},
			Candidates: make([]matcherCandidate, 0, len(task.Candidates)),
		}
		// Copy only identity-relevant candidate fields into the prompt payload
		for _, candidate := range task.Candidates {
			matcherTask.Candidates = append(matcherTask.Candidates, matcherCandidate{
				ID:   candidate.Food.ID,
				Name: candidate.Food.Name,
				Tags: candidate.Food.Tags,
			})
		}

		payload.Tasks = append(payload.Tasks, matcherTask)
	}

	payloadJSON, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		return "", fmt.Errorf("build meal matching prompt payload: %w", err)
	}

	return fmt.Sprintf(`You are matching generated meal recommendations to backend food records.
						IMPORTANT RULES:
						- Compare semantic food identity, not just shared words.
						- Consider food category, food form, preparation, target audience, and serving context when available.
						- Return MATCH only when a supplied candidate is equivalent to the generated meal.
						- Return NO_MATCH when no supplied candidate is equivalent.
						- Return only candidate IDs that appear in the candidates list for that same meal_index.
						- Return exactly one decision for every meal_index.
						- Return only valid JSON. Do not include markdown or explanation.

						RESPONSE SHAPE:
						{
						"decisions": [
							{
							"meal_index": 0,
							"decision": "MATCH",
							"candidate_id": "candidate-id"
							},
							{
							"meal_index": 1,
							"decision": "NO_MATCH"
							}
						]
						}

						MATCHING TASKS:
						%s`, string(payloadJSON)), nil
}

// cleanJSONText strips optional Markdown JSON fences that Gemini may return even
// when prompted to return raw JSON.
func cleanJSONText(text string) string {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
