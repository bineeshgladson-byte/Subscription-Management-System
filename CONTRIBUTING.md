# Contributing

## Branches

| Branch | Purpose | Rules |
|--------|---------|-------|
| `main` | Always demo-able | Protected. Only receives merges from `develop` at milestones (tag `v0.1`, `v0.2`…) |
| `develop` | Integration | Protected. PR + 1 approval + green CI |
| `feature/be-<topic>` | Backend work | e.g. `feature/be-products` |
| `feature/fe-<topic>` | Flutter work | e.g. `feature/fe-invoice-list` |
| `db/<topic>` | Migrations / seeds | Reviewed by the DB owner |
| `fix/<topic>` | Bug fixes | |

Branches live for hours, not days. Rebase on `develop` at least twice a day:

```bash
git fetch origin && git rebase origin/develop
```

## Commits

Conventional commits: `feat(billing): catch up missed periods`, `fix(auth): ...`,
`chore: ...`, `docs: ...`, `test: ...`.

## Pull requests

- Open early as a **draft**; squash-merge into `develop`.
- Fill in the PR template (what, why, how tested, screenshots for UI).
- **API change?** Update `docs/api/openapi.yaml` in a separate PR first, reviewed by
  one backend AND one Flutter developer.
- **Migration?** Timestamped file names, never edit a merged migration, include `.down.sql`.
- CI must be green: `go vet`, `go test`, `flutter analyze`, `flutter test`.

## Who owns what (edit names in `.github/CODEOWNERS`)

| Person | Area |
|--------|------|
| Backend 1 | auth, staff, customers, products, plans, templates → `routes_auth/staff/customers/catalog/plans.go` |
| Backend 2 | quotations, subscriptions, billing, invoices, discounts, payments, reports |
| Flutter 1 | `lib/core`, auth, customers, staff, products, plans, settings |
| Flutter 2 | quotations, subscriptions, invoices, payments, discounts, dashboards, reports |
| Shared | `database/`, `docker-compose.yml`, `docs/` |

## Avoiding merge conflicts

- Each backend module has its own `internal/router/routes_<module>.go`: edit yours only.
- Each Flutter feature has its own folder; `core/router/app_router.dart` changes by one line per screen.
- Add dependencies (`go.mod`, `pubspec.yaml`) in small, separate PRs.

## Before you push

```bash
make be-fmt be-vet be-test      # backend
make fe-analyze fe-test         # frontend
```
