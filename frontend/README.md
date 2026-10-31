# Brightloop Flutter app

```bash
flutter pub get
# Against the real API (make up at repo root):
flutter run -d chrome --web-port 5000 --dart-define=API_BASE_URL=http://localhost:8080/api/v1
# Against the mock API (make mock-api):
flutter run -d chrome --web-port 5000 --dart-define=API_BASE_URL=http://localhost:4010
# Android emulator:
flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080/api/v1
```

Port 5000 matters on web: it's in the API's `CORS_ALLOWED_ORIGINS`.

## Structure

```
lib/
  main.dart, app.dart
  core/
    config/env.dart           API_BASE_URL from --dart-define
    network/api_client.dart   Dio + auth header + 401 → logout; ApiClient unwraps the envelope
    network/api_exception.dart one error type for all API failures (with per-field details)
    storage/token_storage.dart JWT in flutter_secure_storage
    router/app_router.dart    GoRouter + redirect (logged out → /login, wrong role → /forbidden)
    router/nav_items.dart     sidebar entries + which roles see them
    utils/                    validators (mirror backend rules), Fmt.inr / Fmt.date
    widgets/                  AppShell, AsyncView, StatusChip, PlaceholderView
  features/<module>/
    data/          models (fromJson) + repository (calls ApiClient)
    application/   Riverpod providers / notifiers
    presentation/  screens and widgets
```

`features/auth` is the complete reference implementation. Every other feature has a
placeholder screen listing the endpoints it will use.

## Rules

- Screens never use Dio directly: screen → provider → repository → ApiClient.
- Money is a `String` from the API; parse with `Decimal`, display with `Fmt.inr`. Never `double`.
- One screen per feature serves all roles; use `role.scope` (`Scope.all` vs `Scope.mine`)
  to choose `/invoices` vs `/me/invoices`.
- Hiding a button is UX only. The backend enforces permissions.
