# Deployment (Phase 10)

Planned contents:

- `docker-compose.prod.yml` — api + postgres + nginx serving the Flutter web build
- `nginx/default.conf` — serves `/` from the Flutter build, proxies `/api` to the Go API (same origin, no CORS)
- `frontend.Dockerfile` — `flutter build web` then copy into nginx
- `scripts/backup_db.sh` — `pg_dump` of the database

Not needed for local development; use the root `docker-compose.yml`.
