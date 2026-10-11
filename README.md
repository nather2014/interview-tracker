# Interview Tracker

Plain HTML/CSS/JavaScript → Go API → PostgreSQL. Anonymous interview-question reports, combined search and filters, pagination and token-protected moderation. No frontend build step or user accounts.

## Structure

- `backend/`: Go API, embedded idempotent PostgreSQL schema, unit and integration tests.
- `web/`: existing posting, browse and moderation frontend.
- `deploy/install-vps.py`: native Ubuntu service installer for a prepared release.
- `compose.yaml`: local PostgreSQL database.
- `Dockerfile`: optional application container build.

The retired Spring Boot source, build outputs and local H2 database are preserved outside the project at `../interview-questions-archives/spring-20261004/`. Existing PostgreSQL interview/question tables remain compatible; startup adds a field column with a blank default for older reports (shown as Unspecified); H2 data is not automatically imported.

## Local use

```bash
docker compose up -d
export DATABASE_URL='postgres://interviews:interviews@127.0.0.1:5432/interviews?sslmode=disable'
# Optionally set ADMIN_TOKEN to enable moderation.
cd backend
go run .
```

Open http://localhost:8091. Default credentials in Compose are only for local development. DATABASE_URL uses PostgreSQL URI format, not JDBC. Set LISTEN_ADDR and WEB_DIR to override defaults. Schema initialization creates missing tables/indexes without dropping data.

## Features and limits

Reports contain 1–50 topic-labelled questions, field, company, role, experience band, format, date and optional location. Search is case-insensitive across questions/topics/company/role; field/role/company/topic/experience/year filters combine. Literal percent and underscore searches are escaped. Twelve reports appear per page, newest interview date first. Invalid and future-dated submissions are rejected.

Moderation lives at `/admin.html`; ADMIN_TOKEN stays in frontend memory. Deleting a report cascades to its questions. With no server token moderation is disabled. Reports publish immediately; public hosting still needs HTTPS and proxy-level submission rate limits. No fabricated seed data is included.

The Go database pool is limited to four open and one idle connection. Report questions are fetched without per-report database round trips. Browser pages load only the current results page. Request body, HTTP header and request-duration limits bound individual requests.

## VPS installation

The existing VPS deployment still uses the legacy `interview-questions` service, paths, and `interview-notes` binary. Renaming this repository does not migrate that running deployment. Before redeploying under the new name, migrate its environment configuration and stop the legacy service to avoid a port conflict.

The installer configures the service as `interview-tracker`, listening on `127.0.0.1:8091`. It shares the existing PostgreSQL server but uses its own `interviews` database and role. Runtime files are in `/opt/interview-tracker`; configuration and moderation token are in root-only `/etc/interview-tracker.env`. No domain or public routing is configured yet.

The installer expects `/root/interview-tracker-release.tar.gz` containing `interview-tracker` and `web/`. Build with `CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o interview-tracker .` inside backend. Upload the release and run `python3 deploy/install-vps.py` as root on the VPS. Existing environment credentials are reused. GOMEMLIMIT is 64 MiB (a soft runtime target); systemd MemoryMax is 128 MiB. This is not a claim that actual RAM use is always below the soft target.

Preview before configuring a domain:

```bash
ssh -L 8091:127.0.0.1:8091 root@51.77.117.66 -p 20021
```

Then open http://localhost:8091 on that computer. Logs: `journalctl -u interview-tracker`. Health: `/api/health`.

Daily native database backups run at 02:45 server time to `/var/backups/interview-tracker`; `/usr/local/sbin/interview-tracker-backup` makes an immediate dump. Keep off-server copies. Do not restore over live data.

## Verify

```bash
cd backend
go test ./...
go vet ./...
TEST_DATABASE_URL='postgres://USER:PASSWORD@HOST:5432/DB' go test -count=1 -v ./...
```

PostgreSQL integration tests create and remove a uniquely named schema; the test database user must be allowed to create schemas. Tests cover posting, combined/literal search, pagination, validation, moderation, cascading deletion and static routes.

API: GET/POST `/api/interviews`; GET `/api/filters`; GET `/api/admin/session`; DELETE `/api/admin/interviews/{id}`. Experience bands: `0–2`, `3–5`, `6–10`, `10+`; formats: Online, On-site, Walk-in, Phone.

Browser verification: `scripts/test-general-browser.cjs` uses Playwright at widths 320 and 1280. Set TEST_REPORT_IDS_FILE to record its temporary report IDs and companies; delete only those exact reports after running it. It checks non-technology posting, combined field/role filters, search, reset and overflow.
