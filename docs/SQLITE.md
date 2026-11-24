# SQLite Backend: Feasibility Research

Research for [#236](https://github.com/Turbootzz/Nimbus/issues/236): can Nimbus run on SQLite behind the existing repository layer, so that a single container needs no database server? This is a decision document with an estimate. Nothing here is a product commitment.

Measured on 2026-11-18 against `develop` (`aa3657f`) plus the Next.js bump, commit `8a7ae2e` (34 migrations, 14 repositories). For what was added since, see [Since the audit](#since-the-audit).

---

## Summary

- **It works.** A prototype with `DB_DRIVER=sqlite` (pure Go driver, no cgo) needed changes in five Go files. A 60-step API scenario then gives the same answers as Postgres except for two small differences, and the metrics queries return identical rows on both.
- **The queries are not the problem.** 116 of the 120 SQL statements in the repository layer run unchanged. Two needed a change, and three places already had a SQLite fallback.
- **The schema and the tests are the problem.** The 34 Postgres migrations do not translate, so SQLite needs its own migration set forever. The test suite already "runs on SQLite", but on 77 hand-written table definitions that have drifted from the real schema, and through a different driver than the one production would use.
- **Performance is fine at homelab scale.** With 50 services and 30 days of history, SQLite handles roughly 3,000 to 4,000 health checks per second against a real load of under 1 per second. The 30-day metrics query is slower than on Postgres (about 200 ms against 77 ms) and the database file is twice as large.
- **Estimate: 12 to 16 engineer-days** for an opt-in SQLite mode, plus roughly 15 to 25 percent extra on every future change that touches the database.
- **Recommendation: do it, but only after the tests run against the real migrations on both databases.** That first step (4 to 6 days) is worth doing even if SQLite never ships. See [Recommendation](#recommendation).

---

## Context

The Postgres requirement is the recurring objection when Nimbus is compared with Glance or Homepage, which are one container with no database. The unified image reduced the setup to two containers; a SQLite mode would reduce it to one.

Two facts shape everything below:

1. The backend talks to the database with `database/sql` and hand-written SQL. There is no ORM and no query builder.
2. Every Go test that touches a database already uses in-memory SQLite, through `github.com/mattn/go-sqlite3`. CI starts a Postgres service for the backend job, but no Go test connects to it.

---

## Audit

### Repository layer

All 15 non-test files in `backend/internal/repository/` at `8a7ae2e` (14 repositories and `interface.go`; 3,742 lines, 120 SQL statements, 333 `$n` placeholders). Counts are SQL occurrences; comments and Go code are excluded.

Statements are classified by the constructs they use, and each construct was checked on `modernc.org/sqlite` 1.62.0 (SQLite 3.54.1) for behaviour, not just syntax. Not every statement was executed. As a second check, the 99 statements that are a single string literal were prepared against the SQLite schema of the prototype: 97 succeed, and the 2 that fail are the `unnest` queries that already have a fallback. All 99 prepare on Postgres. The other 21 statements are assembled from fragments in Go and were not prepared; the scenario under [Prototype](#prototype) executes some of them.

| Construct | Count | Where | On SQLite |
|---|---|---|---|
| `$n` placeholders | 333 in 120 statements | everywhere | Unchanged. Reused (`$1 ... $1`) and out-of-order placeholders bind correctly. |
| `RETURNING` | 14 | `api_token:50`, `group:62`, `integration:107,235`, `invitation:35`, `password_reset:31`, `preferences:63`, `service:46`, `status_log:46`, `user:54`, `webhook:30,319`, `widget:200,295` | Unchanged. 8 of them return an id the database generates, so the SQLite schema needs a UUID default expression. |
| `ON CONFLICT` | 5 | `group:206`, `preferences:127`, `settings:94,120`, `snapshot:26` | Unchanged, including the partial-index target `ON CONFLICT (user_id) WHERE is_default = TRUE`. |
| `CURRENT_TIMESTAMP` | 30 | 11 files | Unchanged syntax. Correct only with the driver settings under [Time handling](#time-handling). |
| Time compared with a Go parameter | 7 | `status_log:78,156,204,245`, `password_reset:52,101`, `webhook:388` | Same condition as above. |
| Transactions (`BeginTx`) | 12 | 7 files | Unchanged code. Need `_txlock=immediate`, see [Write concurrency](#write-concurrency). |
| Boolean literals `TRUE` / `FALSE` | 9 | `group`, `service`, `integration`, `webhook`, `widget` | Unchanged. |
| Row lock by no-op update (`UPDATE users SET id = id`) | 1 | `widget:72` | Unchanged. It takes the write lock, which is what it is for. |
| `unnest($1::uuid[])` + `pq.Array` + `UPDATE ... FROM` | 2 statements (4 `unnest`, 4 casts, 2 `pq.Array`) | `service:322-334`, `group:369-381` | Already branched. SQLite uses the existing loop fallback. |
| `SELECT ... FOR UPDATE` | 1 | `group:46` | **Needs a branch.** Syntax error on SQLite. |
| `date_trunc` + `EXTRACT(...)::int` + `interval` | 1 statement | `status_log:184-185` | **Needs a rewrite** with `strftime`. |
| `LOWER(col) LIKE` | 1 | `user:229` | Runs, but differs: SQLite lowercases ASCII only. |
| `ORDER BY name` | 1 | `integration:174` | Runs. Postgres sorts by locale, SQLite by bytes. |
| JSONB operators (`->`, `->>`, `@>`) | 0 | | Not used. The 6 JSONB columns are read and written whole. |
| `ILIKE`, advisory locks, `LATERAL`, window functions, CTEs | 0 | | Not used. |
| `NOW()`, `gen_random_uuid()` in queries | 0 | | Only in migrations. 6 repositories generate UUIDs in Go. |
| Postgres error codes (`pq.Error`, `23505`) | 0 | | Not used. Duplicates are checked before the insert. |

Result by statement:

| Class | Statements |
|---|---|
| Works unchanged | 116 |
| Already has a dialect branch (`isPostgreSQL`) | 2 queries, plus `EnsureDefaultGroup` at `group:201` |
| Needs a new branch | 1 (`FOR UPDATE`) |
| Needs a rewrite | 1 (time buckets) |

### SQL outside the repositories

| Place | SQL | On SQLite |
|---|---|---|
| `internal/db/migrate.go:81` | `DEFAULT NOW()` in `schema_migrations` | Needs a change. `CURRENT_TIMESTAMP` works on both. |
| `internal/db/migrate.go:44,67` | two bookkeeping statements | Unchanged. |
| `internal/db/utils.go:16` | `SELECT version()` | Postgres only. This is the existing dialect probe: it fails on SQLite and so returns "not Postgres". |
| `internal/db/db.go:41,69` | `sql.Open("postgres", ...)`, `SELECT 1` | Needs the driver switch. |
| `internal/config/config.go:86-118` | requires `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | Needs to skip these for SQLite. |
| `internal/seeds/users.go:163` | one `UPDATE` | Unchanged. |
| `internal/workers/`, `cmd/` | none (they go through services and repositories) | |
| `docker/entrypoint.sh:98` | `psql ... "SELECT COUNT(*) FROM users"` data-loss check | Needs a SQLite branch. Not done in the prototype. |
| `Dockerfile:71` | copies `internal/db/migrations` | Must also copy the SQLite set. Not done. |
| `docs/MIGRATION.md` | backup with `pg_dump` | Needs a SQLite section. |

### Migrations

All 34 `*.up.sql` files (480 lines). The cheap route was tried first: run each file on SQLite as is, then again after 8 regex rules (drop `CREATE EXTENSION` and `COMMENT ON`, replace `gen_random_uuid()` and `NOW()`, strip `::jsonb`, rename timestamp types, drop `IF NOT EXISTS` from `ADD COLUMN`, split multi-column `ALTER TABLE`, map enum types to `TEXT`).

| Result | Files | Which |
|---|---|---|
| Runs unchanged | 5 | 19, 20, 21, 23, 28 |
| Runs after the regex rules | 22 | |
| Needs a hand rewrite | 7 | 10, 11, 12, 13, 22, 30, 34 |

The 7 that fail use things SQLite cannot do in place: `DO $$ ... $$` blocks, `CREATE TYPE` / `ALTER TYPE ... ADD VALUE`, `ALTER COLUMN ... SET DEFAULT` / `DROP NOT NULL`, and `ADD` / `DROP CONSTRAINT`. On SQLite each of these means rebuilding the table (create new, copy, drop, rename). So a translator does not converge, and "runs after regex" is not the same as "equivalent": for example the translated file 4 keeps a `CHECK` that file 12 can then no longer change.

Constructs in the migrations:

| Construct | Lines | Files | On SQLite |
|---|---|---|---|
| `UUID` columns | 34 | 13 | `TEXT`. No format check. |
| `VARCHAR(n)` | 36 | 15 | `TEXT`. **Length is not enforced.** |
| `DEFAULT gen_random_uuid()` | 14 | 11 | Expression built from `randomblob()`. It must produce a real UUID: the routes use Fiber's `<guid>` constraint. |
| `TIMESTAMP WITH TIME ZONE` / `TIMESTAMPTZ` | 17 | 8 | Must be declared `TIMESTAMP`. The driver only parses `DATE`, `DATETIME` and `TIMESTAMP`; with the Postgres spelling every scan into `time.Time` fails. |
| `ADD COLUMN IF NOT EXISTS` | 17 | 13 | Syntax error. Plain `ADD COLUMN` works. |
| `CHECK (...)` | 14 | 10 | Works. |
| `BOOLEAN` | 13 | 11 | Works (stored as 0/1). |
| `COMMENT ON` | 9 | 5 | Drop. |
| `JSONB`, `::jsonb` | 6, 3 | 6, 3 | `TEXT`. Postgres reorders keys, SQLite keeps the text as written. |
| Enum types | 6 | 4 | `TEXT` + `CHECK`. |
| `ALTER COLUMN` | 6 | 5 | Table rebuild. |
| Partial indexes | 5 | 5 | Work. |
| `DO $$` blocks | 4 | 2 | Not supported. |
| `ADD` / `DROP CONSTRAINT` | 4 | 2 | Table rebuild. |
| `NOW()` | 2 | 1 | `CURRENT_TIMESTAMP`. |
| `INET`, `BYTEA`, `CREATE EXTENSION` | 1 each | 1 each | `TEXT`, `BLOB`, drop. |

### The existing SQLite tests

The Go tests are the reason this looked cheap, and the main reason it is not.

| Fact | Number |
|---|---|
| Test files that open SQLite | 31 |
| Driver | `mattn/go-sqlite3` (cgo). Production images build with `CGO_ENABLED=0`, so that driver cannot ship. |
| Hand-written `CREATE TABLE` statements | 77, in 34 test files. No shared schema, nothing reads the migrations. |
| Definitions with the same columns as the real table | 60 |
| Definitions missing columns | 17 (165 columns in total; `services` has 6 variants, the smallest lacks 15 of 17 columns) |
| Definitions with fewer foreign keys than the real table | 50 of 62 |
| Definitions with fewer `CHECK`s than the real table | 42 of 48 (word count, approximate) |
| Definitions with an `INTEGER` id where the real one is a UUID | 4 |
| Indexes created in tests | 2. The real schema has 37, including the partial unique index on default groups. |
| Test files that turn foreign keys on | 1 |
| Tests that run against Postgres | 0 |

What follows from this:

- **Postgres-only SQL is not tested anywhere.** No test executes `bulkUpdatePositionsPostgreSQL` (twice), the `ON CONFLICT` branch of `EnsureDefaultGroup`, `GroupRepository.Create` (tests insert rows by hand to avoid `FOR UPDATE`) or `GetAggregatedByInterval`. Two tests in `services/metrics_test.go` are skipped with "tested in integration tests with real PostgreSQL". Those integration tests do not exist.
- **The fallbacks have drifted.** On Postgres the default group is created at position 0. The SQLite fallback goes through `Create` and puts it after the existing groups. The end-to-end comparison found this.
- **The fallbacks cannot simply be removed.** The `ON CONFLICT` statement runs fine on real SQLite, so the branch is unnecessary. Removing it made `TestGroupHandler_GetGroups` fail, because that test's table has no partial unique index. The hand-written schemas hold the branches in place.
- **A green suite says little about production SQLite.** The two drivers differ. `mattn` binds `$n` by order of appearance, so the one out-of-order statement (`status_log:182`, `$4` before `$1`) would bind wrongly there. Time handling differs too.

### Since the audit

Checked by reading and grep on this branch, not executed like the rest of this section.

- **New repository files:** `board_repository.go`, `dashboard_repository.go`, `dashboard_sync.go` and `discovery_repository.go` (19 non-test files now). None of the Postgres-only constructs from the table above appear in them (`FOR UPDATE`, `unnest`, `::` casts, `date_trunc` / `EXTRACT` / `interval`, `pq.*`, JSONB operators). They use `ON CONFLICT ... DO NOTHING` (2), `RETURNING` (2, timestamps only), transactions (7), `TRUE` / `FALSE`, `COALESCE` and `CAST(... AS TEXT)`.
- **Changed repositories:** `GroupRepository.Create` now adds `FOR UPDATE` only on Postgres, so the one "needs a new branch" item is done. The new `GetHourlyUptimeByGroup` has a SQLite expression (`julianday`) next to the Postgres one (`EXTRACT(EPOCH ...)`). `GetAggregatedByInterval` still needs the rewrite.
- **Migrations 035 to 039, 041 and 042** (040 is not on this branch; 41 in total). By inspection, not run: 036, 037, 038, 039 and 042 need only the regex rules, and 035 (`ALTER COLUMN ... SET DEFAULT`) and 041 (`ALTER COLUMN board_id SET NOT NULL` on `groups`, `services` and `widgets`) need a hand rewrite. That makes 9 of 41: the 7 measured above plus these 2. The prototype's SQLite baseline stops at migration 34.
- **Tests:** `board_postgres_test.go` runs migration 041 and the board queries on a real Postgres, but only when `NIMBUS_TEST_POSTGRES_DSN` is set. CI does not set it.

The recommendation does not change. The new queries are portable; the schema side, already the main cost, has two more migrations to rewrite by hand.

---

## Prototype

Branch `docs/sqlite-research`, commits after this document. Driver: `modernc.org/sqlite` (pure Go). It was not in `go.mod`; `mattn/go-sqlite3` was, but needs cgo.

### What was changed

| File | Change |
|---|---|
| `internal/db/db.go` | `DB_DRIVER` (`postgres` default, `sqlite`; anything else is an error) and `DB_PATH` (default `data/nimbus.db`). SQLite DSN with WAL, `busy_timeout`, `synchronous=NORMAL`, foreign keys, UTC time format, immediate transactions. |
| `internal/db/migrate.go` | Reads `migrations_sqlite/` when the database is not Postgres. `NOW()` became `CURRENT_TIMESTAMP`. |
| `internal/db/migrations_sqlite/000001_baseline.up.sql` | New. The schema after migration 34, generated from `pg_dump` of a migrated Postgres 18 and checked: 15 tables, 37 indexes, 40 constraints, same as Postgres. One hand fix was needed, see below. |
| `internal/config/config.go` | `DB_HOST` and friends are only required for Postgres. |
| `internal/repository/group_repository.go` | `FOR UPDATE` only on Postgres. |
| `internal/repository/status_log_repository.go` | Time bucket expression per dialect. Same semantics, including the existing quirk that a bucket never crosses the hour. |
| `internal/repository/sqlite_driver_test.go` | New. One test on the real driver and the real SQLite migrations. |
| `cmd/dbbench/` | New. The benchmark tool used below. |
| `go.mod` | `modernc.org/sqlite` and 6 indirect modules. Linux binary grows from 24.0 MB to 30.1 MB. |

That is 5 existing Go files and about 160 changed lines (a third of it re-indentation in `config.go`), plus the new files. No existing test was modified.

### What worked

Server built with `CGO_ENABLED=0`, started on a high port with a temporary SQLite file and `TZ=Europe/Amsterdam`, driven with a 60-step scenario over HTTP. The same scenario ran against a throwaway Postgres 18 container, and the two outputs were compared after replacing ids, timestamps and response times.

Covered and identical on both (full response bodies): setup status, register, duplicate register, login, wrong password, `/auth/me`, groups (create, list, reorder), services (create, list, reorder, update, validation error, unknown id), manual health check, preferences (read, upsert, validation error), a widget, tile reorder, a webhook, an API token, admin user search, user stats, settings upsert, closed registration, forgot password, deleting a group with its services, cascade to status logs, deleting a service, role checks, account deletion.

Compared only as a summary, because the content depends on timing: the health monitor writing status logs for an online and an offline service, the status log list, and metrics for 1 hour and 30 days (status code, statuses seen, uptime, number of buckets). The Prometheus endpoint and the webhook log list were called but their output was not compared.

The metrics queries were therefore compared separately on fixed data: 103 status rows at fixed times across midnight, written from two time zones (one with a half-hour offset), with the process in `TZ=Europe/Amsterdam`. `GetUptimeStats`, `GetAggregatedByInterval` at 1, 5, 7, 15, 60 and 90 minutes, `GetLatestByServiceID`, `GetByServiceID` and `DeleteOlderThan` print 224 lines of results. The output is byte-identical on SQLite and Postgres.

Test suite: 1,305 pass and 2 skip before; 1,306 pass and 2 skip after (`make ci-check`, race detector on).

### What differed

From the scenario:

| Difference | Cause | Fix |
|---|---|---|
| Default group at position 2 instead of 0 | The drifted fallback in `EnsureDefaultGroup` | Delete the fallback. Blocked by a test schema without the partial index. |
| Admin search for `ÜBUNG` finds "Bob Übung" on Postgres, nothing on SQLite | SQLite `LOWER()` is ASCII only | Register a Unicode `lower` function, or store a lowercased column. |

From preparing the statements: the generated baseline wrote the default-group index as `WHERE (is_default = true)`, the way `pg_dump` prints it. SQLite then rejects `ON CONFLICT (user_id) WHERE is_default = TRUE` with "does not match any PRIMARY KEY or UNIQUE constraint": it matches the predicate text, and `true` is not `TRUE`. Nothing failed at runtime only because the SQLite fallback never runs that statement. The baseline now uses `TRUE`. It is a small example of how the two schemas can disagree without any test noticing.

### What was not done or not exercised

- Docker: `entrypoint.sh` still calls `psql`, the Dockerfile does not copy `migrations_sqlite/`, there is no compose file. No image was built.
- The regular test suite still uses `mattn` and hand-written tables. Only the one new test uses the production driver and schema.
- Not exercised over HTTP: OAuth, completing a password reset, integrations with a real app, the widget poller and snapshots, the SSE stream, uploads, bulk user actions, webhook delivery to a reachable target.
- `ActivityLogRepository` and `InvitationRepository` are not wired into `main.go` and have no tests on either database.
- No data migration between Postgres and SQLite.
- No down migrations for SQLite (the runner does not use them on Postgres either).

### Time handling

SQLite has no timestamp type; times are text and are compared as text. This is where a naive port breaks silently.

With driver defaults and `TZ=Europe/Amsterdam`, a row written by Go and a row written by `CURRENT_TIMESTAMP` at the same moment were stored as `2026-11-18 20:34:52.03 +0100 CET m=+0.005` and `2026-11-18 19:34:52`. A one-minute window around "now" matched one of the two rows. In a UTC container the bug hides; many homelab users set `TZ`.

Three DSN settings fix it, verified with a non-UTC zone: `_time_format=sqlite`, `_timezone=UTC`, `_texttotime=1`. The last one also makes computed columns such as the time bucket scan into `time.Time`. The existing tests do not have these settings; they pass because each test writes and reads times the same way.

### Write concurrency

SQLite allows one writer at a time. Nimbus writes from the health monitor (10 workers), the widget poller, API requests and cleanup jobs, and reads from SSE and the dashboard at the same time. Measured with 10 concurrent writers:

| Settings | Result |
|---|---|
| WAL + `busy_timeout(5000)` + `_txlock=immediate` (prototype) | 0 errors in 6,000 statements and 200 transactions |
| Default deferred transactions | 92 of 200 concurrent group creates fail with `SQLITE_BUSY`. A transaction that reads first cannot upgrade to a write once someone else has written, and `busy_timeout` does not retry that. |
| No `busy_timeout` | 3,461 of 6,000 health check statements fail |

So the settings are mandatory, not tuning. With them, writers queue for up to 5 seconds. The longest write measured was the daily retention cleanup: 1.4 to 1.9 seconds for one day of logs (72,000 rows), during which other writes wait. That is inside the timeout but it is the thing to watch if the dataset grows.

One side finding on Postgres: 200 concurrent group creates produced 205 groups with only 54 to 56 distinct positions. `FOR UPDATE` on the last row does not stop two inserts from picking the same next position. SQLite, serialized, produced 205 distinct positions. This is an existing bug, harmless in practice.

---

## Benchmarks

**Method.** `cmd/dbbench`, one binary, through the repository methods the server uses. SQLite ran with its default page cache (`cache_size`, 2 MB); a larger cache could move the seed and cleanup rows. Dataset: 1 user, 5 groups, 50 services, 30 days of status logs at the default 60-second interval, which is 2,160,000 rows, then `ANALYZE`. Reads: median of 30 runs. Each backend ran twice on a fresh database; both runs are shown. Apple M3 Pro, macOS 27.2, Go 1.27.3.

- SQLite 3.54.1 through `modernc.org/sqlite` 1.62.0, file on the local SSD, the prototype DSN.
- Postgres 18.6 (`postgres:18-alpine`, default configuration) in Docker Desktop, reached over a forwarded TCP port.

**This is not like for like.** Postgres runs in a VM behind a port forward, so every statement pays a network round trip: the single-statement rows take 0.4 to 0.8 ms on Postgres here. That probably explains most of the small-query and single-writer gap. On Linux with both in containers, expect Postgres to be closer on those rows. The aggregation, cleanup and size rows are dominated by the engines.

| Measurement | SQLite | Postgres |
|---|---|---|
| Connect + migrate, empty database | 24 / 29 ms | 204 / 183 ms |
| Bulk seed 2.16 M rows | 156 / 142 s (14-15 k rows/s) | 46 / 46 s (47 k rows/s) |
| Dashboard load (services, groups, preferences, widgets) | 0.22 / 0.22 ms | 2.1 / 2.2 ms |
| Services to monitor | 0.15 / 0.15 ms | 0.38 / 0.44 ms |
| Last 100 status logs of a service | 0.14 / 0.14 ms | 0.72 / 0.78 ms |
| Metrics 24 h (stats + 5-minute buckets) | 6.1 / 8.7 ms | 7.7 / 8.5 ms |
| Metrics 7 days | 44 / 60 ms | 29 / 28 ms |
| Metrics 30 days | 192 / 211 ms | 75 / 79 ms |
| 24 h uptime for all 50 services | 100 / 98 ms | 221 / 191 ms |
| Health check writes, 1 writer | 3,336 / 4,380 per s | 527 / 566 per s |
| Health check writes, 10 writers | 2,881 / 4,124 per s | 2,302 / 1,973 per s |
| Same, with 4 readers running the 30-day query | 2,432 / 4,109 per s | 1,437 / 1,410 per s |
| 30-day query during those writes | 285 / 317 ms | 84 / 89 ms |
| Retention cleanup, oldest day (72,000 rows) | 1.9 / 1.4 s | 0.03 / 0.03 s |
| Database size | 935 / 930 MB | 465 / 468 MB |

A "health check write" is what one check does: read the service, update its status, insert a log row, as three autocommit statements.

**Reading the numbers.**

- The real write load is 50 checks per 60 seconds: 0.83 per second. SQLite has more than 3,000 times that. A large install with 500 services at a 10-second interval needs 50 per second.
- The 30-day metrics query reads 43,200 rows and groups them. SQLite uses the right index (`idx_status_logs_service_time`) and then a temporary B-tree for the `GROUP BY`. About 200 ms is acceptable for a page a user opens now and then, but it is 2.6 times slower than Postgres and grows with history.
- The file is twice the size because UUIDs are 36-byte text in the table and in four indexes: 433 bytes per log row against 215 on Postgres.
- An `INTEGER` primary key on `service_status_logs` (SQLite only) was tried as a variant: seed 64 s instead of about 150 s, cleanup 0.3 s instead of 1.4 to 1.9 s, size 733 MB (single run). The read queries stayed within noise, except the 24-hour uptime for all 50 services: 157 ms against about 99 ms, which was not investigated. The API would return numeric log ids on SQLite, so it is not in the prototype.

**Startup and footprint.**

| Measurement | SQLite | Postgres |
|---|---|---|
| Server start to first `200` on `/api/v1/health`, empty database (median of 10) | 24 ms | 165 ms |
| Same, existing database with 2.16 M rows | 17 ms | 41 ms |
| Database server restart to ready | none | 0.3 to 0.6 s (3 runs, warm) |
| Server memory after two check cycles (RSS) | 34 MB | 27 MB |
| Database server memory, idle | none | 22 to 31 MiB |
| Backend binary, linux/amd64 | 30.1 MB | 24.0 MB |

Not measured: first initialisation of a new Postgres data directory, anything on a Raspberry Pi or an SD card, anything on a network filesystem, and long-running behaviour (WAL growth, fragmentation after months of deletes).

---

## Options

### A. No SQLite

Keep Postgres as the only database. Zero cost. The friction stays.

A cheaper way to reach "one container" that this research did not prototype: run Postgres inside the unified image under the supervisord that is already there. No second SQL dialect, but a larger image and Postgres major-version upgrades inside the app container. Worth a separate look before choosing B.

### B. Two dialects behind the repositories

What the prototype does: one codebase, `DB_DRIVER` picks the driver, a handful of branches, a second migration set.

### C. Query builder or abstraction layer

Rewrite the 120 statements with a builder (squirrel, goqu, bun) so that dialect differences are generated. It does not pay: 4 or 5 statements differ, a builder does not cover DDL, time buckets or locking, and the migrations and tests are still needed twice. All of B's cost plus the rewrite and a new dependency.

### D. SQLite only, in a separate "lite" image

Two readings:

- **D1: a second image that only speaks SQLite, next to the Postgres image.** The code is still option B. It adds an image to build and release, and buys a clear support boundary ("lite is for small installs").
- **D2: drop Postgres.** One dialect again, but every existing install has to be migrated, which makes the data migration tool mandatory and risky, and the larger installs lose the faster metrics path.

### Effort

Engineer-days for one developer who knows the codebase. Assumptions: the prototype is the starting point, SQLite is opt-in, no data migration tool unless listed.

| Work | A | B | C | D1 | D2 |
|---|---|---|---|---|---|
| Driver switch, config, DSN, embed both migration sets | | 1 | 1 | 1 | 1 |
| SQLite baseline, paired-migration convention, schema parity check in CI | | 2 | 2 | 2 | 1 |
| Tests: replace 77 hand-written tables with a helper that runs the real migrations on the real driver; fix what breaks | | 4 | 4 | 4 | 4 |
| CI: run repository and handler tests on Postgres too (today: none) | | 1.5 | 1.5 | 1.5 | 0 |
| Dialect fixes (buckets, locking, default group, Unicode search), remove dead fallbacks | | 1 | 1 | 1 | 1 |
| Rewrite 120 statements with a builder | | | 8 | | |
| Docker: entrypoint check, volume, compose example, backup instructions | | 1.5 | 1.5 | 2.5 | 2 |
| Docs: configuration, README, migration guide, development | | 1 | 1 | 1 | 1 |
| Soak test (weeks of synthetic checks), tuning, release | | 1.5 | 1.5 | 1.5 | 2 |
| Data migration Postgres to SQLite | | (2.5, optional) | (2.5) | (2.5) | 4 |
| Remove Postgres code, images and docs | | | | | 1.5 |
| **Total** | **0** | **13.5** (16 with migration tool) | **21.5** | **14.5** | **17.5** |

Realistic range for B: 12 to 16 days. The uncertain line is the test conversion. 77 table definitions go away, but tests that relied on missing foreign keys, missing `CHECK`s or integer ids will start failing, and how many is not known until it is tried.

### Ongoing cost of B and D1

- **Every migration twice.** 34 migrations in 12 months (2025-10 to 2026-10). 7 of the 34 (21 percent) would have needed a table rebuild on SQLite, which is the kind of migration that loses data when done wrong.
- **Every new query on two engines.** Cheap if CI runs both, since 116 of 120 statements needed nothing. Without that CI matrix it is a source of bugs that only one group of users sees.
- **CI.** The backend job runs twice, or the database tests do. Probably a minute more per run (not measured).
- **Support.** Two backup procedures, two sets of "database is locked" or "connection refused" reports, and users who want to move between the two.
- **A parity check is required.** Something in CI must compare the Postgres schema after all migrations with the SQLite one (tables, columns, nullability, defaults, indexes). Without it the two sets drift exactly like the test schemas did.

Rough guess: 15 to 25 percent more time on work that touches the database. Nine commits touched the repositories in the last 6 months, so this is hours per month, not days.

---

## Risks

| Risk | Severity | Note |
|---|---|---|
| The two schemas drift | High without a parity check | Already happened with the test schemas: 17 of 77 definitions are missing columns. |
| Time bugs outside UTC | High if the DSN settings are ever lost | Silent wrong results in metrics and token expiry. Needs a test that runs in a non-UTC zone. |
| `SQLITE_BUSY` under concurrent writes | Low with the measured settings | Mandatory: WAL, `busy_timeout`, immediate transactions. |
| Data directory on NFS or SMB | High for those users | SQLite locking is unreliable on network filesystems, and NAS-mounted app data is common in homelabs. Must be documented; cannot be fixed. |
| Backups by copying the file | Medium | A copy of `nimbus.db` without the `-wal` file can be stale or corrupt. Document `VACUUM INTO` or `sqlite3 .backup`. |
| Weaker schema guarantees | Low | No `VARCHAR` length, no UUID format, no JSON validation. The API validates these today; a future endpoint that forgets will only fail on Postgres. |
| Behaviour differences | Low | Unicode search, sort order of integration names, JSON key order. Two found, more are possible in the paths not exercised. |
| Growth | Low to medium | 930 MB for 50 services and 30 days. The 30-day query and the cleanup get slower with more services or longer retention. |
| Dependency | Low | `modernc.org/sqlite` is a machine translation of the C code. It is widely used and tracks upstream, but it is 6 MB of generated Go. |
| Tests pass on a different driver | Medium until fixed | `mattn` in tests, `modernc` in production. The conversion in the estimate removes this. |

---

## Recommendation

**Go with B, in two steps, and decide on step 2 after step 1.**

**Step 1 (4 to 6 days, worth doing regardless):** make the tests run the real migrations instead of 77 hand-written tables, and run the repository tests against the Postgres service CI already starts. Today no test executes the Postgres-only SQL, the skipped metrics tests point at integration tests that do not exist, and one fallback has already drifted. This fixes a real gap in the current product and is also the largest part of the SQLite cost.

**Step 2 (8 to 10 days):** ship `DB_DRIVER=sqlite` as an opt-in, labelled experimental, for new installs only. No data migration tool at first. Postgres stays the default and the recommendation for larger installs.

**Do not do it** if step 1 is not acceptable. SQLite on top of the current tests would be a second dialect that nothing verifies, and the audit shows how that ends.

**Not recommended:** C (pays for 120 rewrites to save 5 branches) and D2 (forces a migration on every existing user for a problem only new users have).

**Before starting step 2,** spend half a day on the alternative under option A: Postgres inside the unified image. If one container is the only goal, that may reach it with no second dialect.

The honest summary: SQLite is technically easy here and fast enough. The price is not the 13 days; it is writing every migration twice from then on. That price is acceptable only with the parity check and the two-database CI in place.
