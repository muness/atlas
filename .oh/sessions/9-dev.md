# Issue #9 — Add CI workflow for sql/postgres package with pgvector

## Phase 1: Problem Statement

GitHub issue #9 is well-formed with clear acceptance criteria:
- New workflow `.github/workflows/ci-postgres.yml`
- Uses `pgvector/pgvector:pg16` service image
- Runs `go test ./sql/postgres/...` and `./sql/schema/...`
- No `ATLAS_TOKEN` / `ariga/setup-atlas` dependency
- Triggers on push to any branch and on PRs
- Green on master before Phase 1 work begins

## Phase 2: Branch + Draft PR

- Branch: `9-ci-postgres-workflow`
- Draft PR: https://github.com/muness/atlas/pull/11
- Empty commit pushed before solution exploration

## Solution Space

### Key findings from code exploration:
1. `sql/postgres` and `sql/schema` are in the **root `go.mod`** (not separate modules)
2. All existing tests in `sql/postgres/` use `sqlmock` — pure unit tests, no live DB required
3. `ci-go_oss.yaml` already runs `go test -race ./...` from `sql/` working directory
   - But it uses Go `1.22` and does NOT spin up pgvector
4. `ci-sdk.yml` uses `atlasexec`/`sdk` as separate working directories — but those are NOT separate go.mod modules; they're subdirectories of the root module
5. The e2e postgres DSN pattern: `postgres://postgres:pass@localhost:5432/postgres?search_path=public&sslmode=disable`
6. The pgvector service env var will be `TEST_DATABASE_URL` (standard pattern for future integration tests in #7/#8)

### Solution chosen:
New standalone workflow `ci-postgres.yml`:
- Two jobs: `unit-tests` (no DB service) + `integration-tests` (with pgvector service, skipped if no DB url set)
- Actually: simpler is better — one job with pgvector service, run unit tests; when integration tests land they already have a DB
- `go-version-file: go.mod` (root module)
- No `ariga/setup-atlas`, no `ATLAS_TOKEN`
- Triggers: push to any branch + pull_request

## Phase 3: Execute

Created `.github/workflows/ci-postgres.yml`

## Phase 4: Ship

PR #11 merged.

---

## RNA Tool Friction Log

| Step | Tool Used | Should Have Used | Reason | Status |
|------|-----------|-----------------|--------|--------|
| Find postgres test DB env var | Bash grep | mcp__rna-mcp__search | Falling back to grep for env var patterns | skipped |
| Find DB connection pattern | Bash grep | mcp__rna-mcp__search | Searching for integration test patterns | skipped |
