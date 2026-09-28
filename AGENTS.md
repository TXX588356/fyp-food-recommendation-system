# Project Instructions

## General

- Inspect existing code before modifying it.
- Do not assume filenames or architecture.
- Make the smallest change necessary.
- Do not rewrite unrelated code.
- Follow existing project conventions.
- Do not fabricate missing data.
- Run relevant tests after changes.

## Phase 2

- Do not change candidate retrieval.
- Hard safety/dietary filtering happens before ML.
- XGBoost is preference scoring only.
- Never allow ML to override safety constraints.
- Preserve the existing rule-based scorer.
- Use batch inference.
- ML failure must fall back to rule-based scoring.

## Task Management

After completing a task:
1. Run relevant tests.
2. Update TASK.md.
3. Mark completed work.
4. Record important decisions.
5. Record blockers or remaining work.
