# Session: Issue #12 — Replace extension block with inspection handler

**Branch:** `12-extension-inspection-handler`
**PR:** https://github.com/muness/atlas/pull/13
**Issue:** https://github.com/muness/atlas/issues/12
**Outcome:** `.oh/outcomes/sqlalchemy-postgres-python-subset.md`

---

## Phase 1: Problem Statement

Issue #12 created. Problem: `convertExtensions()` in `sql/postgres/driver.go:738` returns a hard error when any extension block is present. This blocks OSS users with schemas using extensions like `pgvector`.

**Acceptance Criteria:**
1. `convertExtensions()` converts HCL `extension {}` blocks into `Extension` objects on `schema.Realm`
2. New `inspectExtensions()` queries `pg_extension` and populates realm, following `inspectEnums()` pattern
3. Extensions round-trip through `MarshalSpec` / `EvalHCL`
4. Unit tests: inspect (mocked DB) + convert paths
5. No regression in existing tests

---

## Phase 2: Solution Space

**Chosen approach:** Follow the `inspectEnums()` / `convertEnums()` pattern exactly.

Key design decisions:
- Extensions are realm-level objects (stored in `schema.Realm.Objects`, not schema-level)
- Introduce postgres-specific `Extension` struct in `sql/postgres/inspect.go` implementing `schema.Object`
- `inspectExtensions()` queries `pg_extension` catalog (extname, extversion, nspname, obj_description)
- `convertExtensions()` converts `[]*extension` HCL specs into `Extension` objects on realm
- `objectSpec` / `MarshalSpec` updated to emit `extension {}` blocks for `Extension` realm objects
- `RealmObjectDiff()` updated to diff extensions (add/drop/modify version)

**Key files to change:**
- `sql/postgres/inspect.go` — add `Extension` type + `inspectExtensions()` function + query constant
- `sql/postgres/driver.go` — replace `convertExtensions()` stub, update `objectSpec`, `RealmObjectDiff`
- `sql/postgres/sqlspec.go` — wire `inspectExtensions()` into `InspectRealm` + `InspectSchema` calls
- `sql/postgres/inspect_test.go` — add `TestInspect_Extensions`
- `sql/postgres/sqlspec_test.go` — add HCL round-trip test

---

## Phase 3: Execute

### Changes

#### 1. `sql/postgres/inspect.go`
- Add `Extension` struct (name, version, schema, comment) implementing `schema.Object`
- Add `extensionsQuery` constant querying `pg_extension` joined to `pg_namespace`
- Add `inspectExtensions(ctx, r *schema.Realm) error` method on `*inspect`
- Wire into `InspectRealm()` alongside `inspectEnums()`

#### 2. `sql/postgres/driver.go`
- Replace `convertExtensions()` stub with actual conversion
- Update `objectSpec()` to emit extension specs for realm-level `Extension` objects
- Update `RealmObjectDiff()` to diff extensions

#### 3. `sql/postgres/sqlspec.go`
- Wire `inspectExtensions()` call in `EvalOptions` realm path

---

## RNA Tool Friction Log

| Step | Tool Used | Query | Verdict |
|------|-----------|-------|---------|
| Find convertExtensions | Bash grep (fallback) | `convertExtensions\|extension` in driver.go | skipped — RNA search returned no match for "convertExtensions" |
| Find extension struct | Bash grep (fallback) | `type extension` in postgres/*.go | skipped |
| Find enum query | Bash grep (fallback) | `enumsQuery` in inspect.go | skipped |
| Find Realm struct | Bash grep (fallback) | `Realm` in schema.go | skipped |
