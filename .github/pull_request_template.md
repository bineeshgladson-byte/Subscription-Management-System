## What

## Why

## How I tested
- [ ] `make be-test` / `make fe-test`
- [ ] Tried it manually (steps):

## Checklist
- [ ] API contract changed? `docs/api/openapi.yaml` updated (separate PR, reviewed by BE + FE)
- [ ] New migration has a `.down.sql`; no merged migration was edited
- [ ] Role checks enforced on the backend, not only hidden in Flutter
- [ ] Money uses decimal types (no float/double)
- [ ] No secrets committed

## Screenshots (UI changes)
