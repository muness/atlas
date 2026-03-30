# Session: Issue #5 — Add inspectFunctions() and inspectTriggers() to postgres driver

**Branch:** `5-inspect-functions-triggers`
**Issue:** https://github.com/muness/atlas/issues/5

---

## Phase 1: Problem Statement

Issue #5: Add inspectFunctions() and inspectTriggers() to postgres driver
- Depends on #4 (SQLObject type) - now merged
- inspectFunctions(): query pg_proc, use pg_get_functiondef(oid) for body
- inspectTriggers(): query pg_trigger + pg_proc, use pg_get_triggerdef(oid)
- Filter to user-defined only (exclude pg_catalog schema)
- Called from InspectSchema()
- Integration test: inspect DB with at least one function and one trigger

---

## Phase 2: Branch + Draft PR — Solution Space

**Solution:** Minimal — add two new inspect methods in inspect.go following the pattern of inspectExtensions(). Both methods query the DB, build SQLObject values, and append to s.Objects. Call both from InspectSchema() under InspectTypes mode.

---

## Phase 3: Execute

Files modified:
- `sql/postgres/inspect.go`: add inspectFunctions(), inspectTriggers(), queries, call from InspectSchema
- `internal/integration/postgres_test.go`: integration test

---

## RNA Tool Friction Log

| Step | Tool Used | Query | Verdict |
|------|-----------|-------|---------|
| Read inspect.go inspectExtensions | Read tool | inspect.go:427-457 | needed for pattern |
| Read InspectSchema | Read tool | inspect.go:98-140 | needed for call site |
