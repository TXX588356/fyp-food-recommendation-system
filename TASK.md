# Phase 2 XGBoost TODO

Source of truth: [docs/PHASE2_XGBOOST_REFERENCE.md](docs/PHASE2_XGBOOST_REFERENCE.md)

Primary rule: preserve the existing candidate retrieval and hard health, allergy, and dietary filtering. Phase 2 changes only the preference-scoring/ranking stage by introducing a self-trained XGBoost model with Phase 1 rule-based scoring retained as the baseline and fallback.

## 1. Baseline Discovery

- [x] Locate the current recommendation orchestration path in Go/Echo.
- [x] Locate existing candidate retrieval/generation logic.
- [x] Locate hard dietary, allergy, health, and medical filtering logic.
- [x] Locate the current manual preference score calculation.
- [x] Locate budget, recency, and diversity adjustments.
- [x] Locate meal-log/history repositories and budget repositories used for feature generation.
- [x] Document the current Phase 1 scoring formula and final ordering flow before changing it.

Discovery notes:
- Go/Echo entrypoint: `internal/endpoint/recommendations.go` registers `POST /api/recommendations` and calls `RecommendationService.GenerateRecommendationResult`.
- Service orchestration: `internal/service/recommendation/service.go` loads saved preferences, derives budget/location prompt context, loads 90 days of meal logs, asks Gemini for generated meals, matches generated meals to backend-owned custom/catalog records, scores candidates, hard-filters them, ranks them, then persists impressions.
- Candidate retrieval/generation: Gemini generation is via `interfaces.MealGenerator.GenerateMeals`; backend candidate retrieval is `internal/service/mealsearch.CandidateSearcher`, which merges visible custom meals and catalog candidates. Catalog candidate retrieval is `internal/service/catalog.CandidateSearcher`. This must remain unchanged for Phase 2.
- Hard filtering: `internal/service/recommendation/filtering.go` removes candidates before final response when dietary tags conflict (`halal`, `non_beef`, `vegetarian`, `vegan`, `seafood_free`, `nut_free`) or generated health flags mark a user health concern as `AVOID`.
- Phase 1 scoring formula: `scoreCandidate` in `internal/service/recommendation/ranking.go` computes `total = goal_alignment + budget_fit + recency_penalty + preference`.
- Goal alignment: for `muscle_gain`, `protein*20 + fat*5 + carbs*5`; for `quick_recommendation`, constant `25`; otherwise `fat*10 + protein*12 + carbs*8`, where fat/carbs are inverse normalized and protein is capped.
- Budget fit: `budgetFitScore` compares the expected generated price against `PerMealBudget`; it returns `20` when budget/price is missing or within budget, tapers down between 1.0x and 1.5x budget, and returns `0` above 1.5x budget.
- Recency/diversity: `recencyPenaltyScore` returns the lower of same-meal recency and fatigued-category recency. Same meal gets `0/8/14/20` for eaten within 1 day, 2 days, 3 days, or otherwise. Category fatigue gets `4/10/16/20` for category counts of `>=3`, `2`, `1`, or `0` in the recent fatigue window.
- Preference score: `preferenceScore` returns `30` when the user has no preferred tags; otherwise it returns matched preferred-tag ratio times `30`.
- History source: `MealLogRepository.ListByUserAndRange` loads logs from Postgres. `buildMealHistoryContext` derives 30-day recent meals, 7-day fatigued category counts, and 90-day learned category counts from meal logs only.
- Budget source: recommendation generation receives `currentMonthSpent` from the frontend, combines it with saved `MonthlyMealBudget`, and calculates `PerMealBudget` dynamically when the request does not provide one. There is no separate budget repository; budget data lives in preferences and current-month spending from meal logs.
- Final ordering: candidates are first scored in generated-meal order, hard filtered, then `rankCandidates` sorts descending by score with stable original index as the tie-breaker. Phase 2 must only replace the preference-scoring component after hard filtering and before existing final adjustments.

## 2. Interaction Storage

- [x] Create or confirm storage for recommendation interactions/impressions.
- [x] Add a reversible migration using the repo's existing migration convention.
- [x] Store at minimum:
  - [x] `recommendation_id`
  - [x] `user_id`
  - [x] `meal_id`
  - [x] `position`
  - [x] `shown_at`
  - [x] `clicked_at`
  - [x] `selected_at`
  - [x] `logged_at`
  - [x] `rating`
  - [x] `created_at`
- [x] Persist recommendation impressions whenever Top-N recommendations are returned.
- [x] Record selection/log outcomes so shown meals can become positive or negative training observations.
- [x] Define and document the observation window for "shown but not selected" negatives.

Result:
- Added `db/migrations/000007_create_recommendation_interactions.{up,down}.sql`, `model.RecommendationInteraction`, and a Postgres repository for batch impression creation plus clicked/selected/logged/rating updates.
- Recommendation generation now generates one UUID `recommendation_id` per returned recommendation set and stores one impression per returned ranked candidate after hard filtering and ranking. Persistence failures are logged and do not fail recommendation responses.
- Returned recommendation candidates now include `recommendation_id`; opening a recommendation detail records `selected_at`, and logging that recommended meal records `logged_at` when the ID is available.

Decisions:
- The implementation uses UUIDs for `recommendation_id`, `user_id`, and `meal_id` to match the existing users, prebuilt meals, and custom meals schema instead of the reference document's generic `BIGINT` example.
- `meal_source` is stored as `prebuilt` or `custom` so the shared `meal_id` can identify either catalog source without changing candidate retrieval.
- Initial negative label observation window: shown meals with no `selected_at` or `logged_at` after 7 days from `shown_at` are eligible negative observations in the dataset pipeline.
- Interaction outcome updates are best-effort: persistence failures are logged but do not fail meal-detail loading or meal-log creation.

Remaining work:
- `clicked_at` and `rating` repository methods exist but are not wired to UI flows yet; the current product flow has no separate click tracking endpoint or recommendation rating control.
- Older recommendation entries cached before this change do not contain `recommendation_id`, so they will not produce selected/logged outcomes until recommendations are regenerated.

## 3. Feature Schema

- [x] Centralize the feature list and order so offline training and online inference cannot drift silently.
- [x] Implement only reliable features from available data; defer any feature that cannot be computed honestly.
- [x] Initial numeric features:
  - [x] `meal_price`
  - [x] `calories`
  - [x] `protein`
  - [x] `carbs`
  - [x] `fat`
  - [x] `calorie_difference`
  - [x] `price_difference_from_avg`
  - [x] `times_meal_eaten`
  - [x] `times_category_eaten`
  - [x] `days_since_last_eaten`
  - [x] `remaining_monthly_budget`
  - [x] `average_meal_spending`
- [x] Initial binary features:
  - [x] `category_match`
  - [x] `cuisine_match`, only if cuisine data is reliable.
- [x] Initial categorical features:
  - [x] `meal_type`, only if reliably available.
- [x] Define missing-value and sentinel policies, especially for never-eaten meals.
- [x] Ensure all history-derived features use only data available at or before `shown_at`.
- [x] Do not encode allergy conflicts, prohibited ingredients, hard dietary restrictions, or medical exclusions as model-overridable features.

Result:
- Added a single machine-readable initial feature schema at `internal/service/recommendation/features/schema.json`, with Go loader helpers in `internal/service/recommendation/features/schema.go`.
- Added tests that lock feature order, active/deferred feature decisions, the 7-day observation window, missing-value policies, and forbidden safety features.

Decisions:
- Active feature order is: `meal_price`, `calories`, `protein`, `carbs`, `fat`, `calorie_difference`, `price_difference_from_avg`, `times_meal_eaten`, `times_category_eaten`, `days_since_last_eaten`, `remaining_monthly_budget`, `average_meal_spending`, `category_match`, `meal_type`.
- `cuisine_match` is explicitly deferred because current meal categories mix cuisine and dish taxonomy without a dedicated cuisine field.
- Missing numeric values default to `0` except `days_since_last_eaten`, which uses sentinel `999` for never-eaten meals.
- `meal_type` uses recommendation request meal category and allows `breakfast`, `lunch`, `dinner`, `snack`, and `other`; unsupported/missing values should map to `other`.
- All history-derived features must use only data available at or before `shown_at`.
- Allergy conflicts, prohibited ingredients, hard dietary restrictions, medical exclusions, and health `AVOID` flags are forbidden model features and must stay hard filters.

## 4. Dataset Pipeline

- [ ] Create `ml/` project structure:
  - [ ] `ml/data/`
  - [ ] `ml/src/prepare_dataset.py`
  - [ ] `ml/src/features.py`
  - [ ] `ml/src/train.py`
  - [ ] `ml/src/evaluate.py`
  - [ ] `ml/src/predict.py`
  - [ ] `ml/service/app.py`
  - [ ] `ml/models/`
  - [ ] `ml/tests/`
- [ ] Implement dataset extraction from users, preferences, meals, meal logs, budgets, and recommendation interactions.
- [ ] Produce one row per observed recommendation interaction.
- [ ] Preserve metadata columns such as `user_id`, `meal_id`, `recommendation_id`, and `shown_at`.
- [ ] Label rows according to the initial policy:
  - [ ] shown and later selected/logged = `1`
  - [ ] shown but not selected within the observation window = `0`
  - [ ] clicked but not selected = `0` initially
  - [ ] not shown = no label
- [ ] Implement cleaning, encoding, and imputation.
- [ ] Prefer time-based train/validation/test splits.
- [ ] If data is too small for a stable time split, document the limitation and use a leakage-safe grouped/random split.

## 5. Training

- [ ] Train a Logistic Regression baseline before XGBoost.
- [ ] Train an initial `XGBClassifier` with binary logistic objective.
- [ ] Use deterministic seeds where practical.
- [ ] Save the XGBoost model artifact to `ml/models/xgboost_meal_ranker.json`.
- [ ] Save model metadata to `ml/models/metadata.json`.
- [ ] Metadata must include:
  - [ ] model version
  - [ ] training timestamp
  - [ ] feature list and order
  - [ ] data range
  - [ ] library versions
  - [ ] evaluation summary
- [ ] Fail fast when an artifact and requested feature schema are incompatible.

## 6. Evaluation

- [ ] Implement classifier metrics:
  - [ ] ROC-AUC
  - [ ] Precision
  - [ ] Recall
  - [ ] F1
  - [ ] Log Loss
- [ ] Implement ranking metrics:
  - [ ] Precision@K
  - [ ] Recall@K
  - [ ] HitRate@K
  - [ ] NDCG@K
- [ ] Compare:
  - [ ] Phase 1 rule-based baseline
  - [ ] Logistic Regression
  - [ ] XGBoost
- [ ] Produce a reproducible results table for Precision@5, Recall@5, and NDCG@5.
- [ ] Do not fabricate metric values; populate results only from reproducible runs.

## 7. ML Prediction Service

- [ ] Implement a small Python prediction service.
- [ ] Load the XGBoost model once at startup.
- [ ] Add `POST /v1/predict/batch`.
- [ ] Use batch scoring; do not make one request per candidate meal.
- [ ] Validate required fields and feature types.
- [ ] Preserve `meal_id` to prediction mapping exactly.
- [ ] Return `preference_score` values in the `0-1` range.
- [ ] Include `model_version` in responses.
- [ ] Add a health/readiness endpoint.
- [ ] Keep inference deterministic for identical model/version/features.
- [ ] Log model version and inference errors without logging sensitive profile data unnecessarily.

## 8. Go/Echo Integration

- [ ] Add configuration for:
  - [ ] ML service URL
  - [ ] ML request timeout
  - [ ] ML enable/disable flag
  - [ ] minimum interaction threshold for cold start
  - [ ] rollout/model switches if needed
- [ ] Refactor the existing rule-based scorer behind a clear interface.
- [ ] Keep the rule-based scorer available for baseline comparison and fallback.
- [ ] After candidate retrieval and hard filtering, compute feature vectors for eligible candidates.
- [ ] Send all candidate feature vectors to the ML service in one batch request.
- [ ] Attach returned `preference_score` values to the correct candidate meals.
- [ ] Replace only the manual preference component with ML `preference_score`.
- [ ] Retain budget, recency, diversity, and final re-ranking behavior unless a documented scoring review requires changes.
- [ ] If a hybrid formula remains, define weights and score ranges in one configurable, testable location.
- [ ] Preserve existing API response contracts where possible.
- [ ] Do not expose internal ML scores in the UI unless explicitly requested.

## 9. Fallbacks

- [ ] Add cold-start fallback when `user_interaction_count < MIN_INTERACTIONS`.
- [ ] Make `MIN_INTERACTIONS` configurable.
- [ ] Add ML-service failure fallback on:
  - [ ] connection error
  - [ ] timeout
  - [ ] invalid response
  - [ ] model unavailable
- [ ] Ensure the recommendation endpoint still returns recommendations when ML is unavailable.
- [ ] Emit structured logs/metrics whenever fallback is used.

## 10. Tests

- [ ] Dataset tests:
  - [ ] label correctness
  - [ ] timestamp-safe feature generation
  - [ ] missing-value handling
  - [ ] no duplicate interaction rows
- [ ] Feature tests:
  - [ ] known input fixtures produce expected values
  - [ ] no post-event leakage
- [ ] Training tests:
  - [ ] training completes
  - [ ] model artifact is produced
  - [ ] metadata is produced
  - [ ] deterministic seed behavior where practical
- [ ] Inference service tests:
  - [ ] valid batch request
  - [ ] missing feature
  - [ ] invalid type
  - [ ] unknown category handling
  - [ ] model unavailable
  - [ ] health endpoint
- [ ] Go integration tests:
  - [ ] batch request mapping
  - [ ] score-to-meal mapping
  - [ ] timeout/error fallback
  - [ ] cold-start fallback
  - [ ] final ordering
- [ ] End-to-end tests:
  - [ ] existing candidate retrieval
  - [ ] hard filtering
  - [ ] ML scoring
  - [ ] re-ranking
  - [ ] Top-N response
  - [ ] interaction persistence

## 11. Documentation

- [ ] Document the final feature list and feature order.
- [ ] Document missing-value/default policies.
- [ ] Document the target-label policy and observation window.
- [ ] Document split strategy and known data limitations.
- [ ] Document model version, artifact path, and metadata schema.
- [ ] Document evaluation metrics and Phase 1 vs Phase 2 comparison results.
- [ ] Document cold-start and ML-failure fallback behavior.
- [ ] Document known limitations and deferred enhancements.

## 12. Acceptance Criteria

- [ ] Existing candidate retrieval continues to work without semantic changes.
- [ ] Hard dietary/health filtering occurs before ML scoring.
- [ ] Recommendation impressions and outcomes are stored for training.
- [ ] A reproducible script can create the ML training dataset from stored data.
- [ ] A reproducible training command produces a versioned XGBoost model artifact.
- [ ] Offline evaluation compares XGBoost with at least the Phase 1 rule-based baseline.
- [ ] The ML service can score multiple candidate meals in one request.
- [ ] Go can obtain XGBoost scores and rank eligible candidates using them.
- [ ] New/low-history users can still receive recommendations through the existing fallback.
- [ ] Recommendations continue when the ML service is unavailable.
- [ ] Automated tests cover feature calculation, inference contract, score mapping, and fallback paths.
- [ ] Phase 1 and Phase 2 can be compared using reproducible test/evaluation data.
- [ ] Documentation records the final feature list, target-label policy, split strategy, model version, evaluation results, and known limitations.

## Deferred Enhancements

- [ ] XGBoost learning-to-rank objective.
- [ ] SHAP-based model interpretation and per-recommendation ML explanations.
- [ ] Automated periodic retraining and model promotion.
- [ ] Keras/TensorFlow neural ranking comparison.
- [ ] Self-trained candidate retrieval using two-tower embeddings.
- [ ] Approximate nearest-neighbour/vector retrieval.
- [ ] Multi-task learning from click, save, selection, and rating signals.
- [ ] Online A/B testing between ranking strategies.

## Frontend Build Artifact Handling

- [x] Removed the backend dependency on embedded frontend assets so generated distributables do not need to be committed for Go builds.
- [x] Kept runtime serving from `static/`, matching the existing Docker runtime layout and the sibling project.
- [x] Updated the frontend build script to clear `static/` and copy the latest Vite `dist/` output into it.
- [x] Ignored and untracked `static/` so generated hashed assets stay local build output instead of commit content.

Decision:
- Frontend assets are runtime files, not Go embed inputs. `pnpm -C client build` is responsible for producing a matched `static/index.html` and `static/assets/` pair. Production images can continue copying build output into `/app/static`, and local runs can use `static/` after building.

