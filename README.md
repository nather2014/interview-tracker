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

The Add interview form supports pasting a whole numbered list (including WhatsApp message headers), or one question per line. Set a shared topic, add the batch, then review and edit each question/topic before saving. Existing filled questions are preserved; batches exceeding the 50-question or 2,000-character-per-question limits are rejected without changing the form.

Only company name is required. Reports may include up to 50 questions, topics, field, role, experience band, format, date and location. Omitted dates remain unspecified. Search is case-insensitive across questions/topics/company/role/field, supports word-order-independent terms and tolerates typos in words of four or more characters using PostgreSQL pg_trgm. Search shows matching questions and combines with topic filters. pg_trgm must be available on the PostgreSQL server; startup installs the trusted extension when needed. field/role/company/topic/experience/year filters combine. Literal percent and underscore searches are escaped. Twelve reports appear per page, newest interview date first. Invalid and future-dated submissions are rejected.

Moderation lives at `/admin.html`; ADMIN_TOKEN stays in frontend memory. Admins can edit report metadata and add, remove or revise questions, including bulk paste. Updates keep the report ID and replace its questions in one transaction. Reports with an unspecified field remain editable. Browse filters refresh before loading results and when returning to the page; edits in another tab trigger a refresh. Removed selections reset to All. Deleting a report cascades to its questions. With no server token moderation is disabled. Reports publish immediately; public hosting still needs HTTPS and proxy-level submission rate limits. No fabricated seed data is included.

The Go database pool is limited to four open and one idle connection. Report questions are fetched without per-report database round trips. Browser pages load only the current results page. Request body, HTTP header and request-duration limits bound individual requests.

## VPS installation

The VPS runs the `interview-tracker` service at `/opt/interview-tracker`, with configuration in `/etc/interview-tracker.env`. The legacy `interview-questions` service is disabled; its files are retained for rollback.

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

API: GET/POST `/api/interviews`; PUT `/api/admin/interviews/{id}` (admin token required); GET `/api/filters`; GET `/api/admin/session`; DELETE `/api/admin/interviews/{id}`. Experience bands: `0–2`, `3–5`, `6–10`, `10+`; formats: Online, On-site, Walk-in, Phone.

Browser verification: `scripts/test-general-browser.cjs` uses Playwright at widths 320 and 1280. Set TEST_REPORT_IDS_FILE to record its temporary report IDs and companies; delete only those exact reports after running it. It checks non-technology posting, combined field/role filters, search, reset and overflow.

## Visitor counts

Sign in at `/admin.html` to see Online now (visible public pages reporting within the last 5 minutes) and Visitors today (unique browsers since midnight UTC). Counts refresh every 15 seconds. Public pages send a heartbeat every minute while visible. A random HttpOnly same-site cookie expires at midnight UTC; PostgreSQL stores only its daily hash and last-seen time. Older days are removed on the next heartbeat. No IP addresses or fingerprints are stored by this feature. Different browsers, blocked cookies, and automated traffic can affect these approximate counts. Collection starts when this release is deployed.

## Automatic topics

The optional local classifier uses a quantized MiniLM model with rules for common technical terms. It runs on loopback port 8092, with one inference thread; question text stays on the VPS. Set `TOPIC_SERVICE_URL=http://127.0.0.1:8092` for the Go API. Suggestions run while typing and on bulk import, preserve manual overrides, and flag uncertain matches for review. All topics remain optional and editable.

Releases containing `topic-ai/` (including its downloaded `model/`) are installed by `deploy/install-vps.py` along with the `interview-topics` systemd service and a Python virtual environment. The VPS needs Python venv support and access to download the pinned Python requirements. The installer checks classifier health before switching the API. The classifier is limited to 256 MiB and half a CPU. Logs: `journalctl -u interview-topics`.

The reviewed PwC topic updates are in `imports/pwc-topic-review-plan.json`. On the VPS, run `python3 imports/apply-reviewed-topics.py pwc-topic-review-plan.json` from the extracted release. This backs up the database and verifies question IDs, original text and topics before making the update in a transaction.

## One-command deployment

From this project directory, run:

```bash
python3 scripts/deploy.py
```

The command uses `~/.ssh/interview_tracker_deploy` to connect to `root@51.77.117.66:20021`. The private key stays outside the repository and outside temporary folders. It runs Go tests/vet, JavaScript syntax checks and classifier checks; builds a static release; uploads it; runs PostgreSQL integration tests in an isolated schema; backs up the current application, configuration and database; installs/restarts the services; and verifies company-only submissions and AI suggestions. Temporary verification reports are deleted. Failed installation or verification restores the previous application/configuration; database schema migrations are additive and are retained.

To apply an explicitly reviewed topic plan after a successful deployment:

```bash
python3 scripts/deploy.py --topic-plan pwc-topic-review-plan.json
```

Topic plans verify the original IDs, text and topics before applying updates, so a plan is intended to run once. A topic-plan failure leaves the verified application deployment in place. Override the key, host, port or Go executable with `INTERVIEW_DEPLOY_KEY`, `INTERVIEW_DEPLOY_HOST`, `INTERVIEW_DEPLOY_PORT` or `GO_BINARY` if needed.

## Interview insights

Open `/insights.html` from Browse to explore submitted reports. Company, field, role, experience and interview-year filters apply to report, question and company totals, the top ten topics, top ten companies, and the interview-month timeline. Counts describe this collection, not hiring probabilities. Topics show both question and distinct report counts; empty questions are excluded. Company and topic names group case-insensitively. Undated reports and questions without topics are counted separately; the timeline excludes undated reports and fills intervening months with zero. No additional visitor tracking or schema changes are needed. API: GET `/api/insights`.
