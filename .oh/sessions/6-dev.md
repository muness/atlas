# Session: Issue #6 — Wire SQLObject into diff and migrate pipeline

**Branch:** `6-sqlobject-diff-migrate`
**Issue:** https://github.com/muness/atlas/issues/6

---

## Phase 1: Problem Statement

Issue #6: Wire SQLObject into diff and migrate pipeline
- SchemaObjectDiff(): compare SQLObject bodies by checksum (sha256 of trimmed body); emit ModifyObject when changed
- addObject(): emit Body SQL verbatim for AddObject and ModifyObject
- dropObject(): emit DROP FUNCTION / DROP TRIGGER as appropriate
- No-op diff when body unchanged
- ModifyObject for functions = CREATE OR REPLACE FUNCTION; for triggers = DROP + CREATE (no OR REPLACE in PG)
- End-to-end test: modify a function body, run diff, assert correct migration

---

## Phase 2: Branch + Draft PR — Solution Space

**Solution:** Minimal extension of existing addObject/dropObject/modifyObject and SchemaObjectDiff functions in driver.go to handle *schema.SQLObject alongside *schema.EnumType.

Key decisions:
- Use crypto/sha256 for body checksum comparison
- strings.TrimSpace() before hashing
- DROP FUNCTION/TRIGGER uses IF EXISTS
- ModifyObject for trigger = DROP + CREATE sequence

---

## Phase 3: Execute

Files modified:
- `sql/postgres/driver.go`: extend SchemaObjectDiff, addObject, dropObject, modifyObject
- `sql/postgres/driver_test.go`: unit tests for new functionality
- `sql/postgres/postgres_oss_test.go`: end-to-end integration test

---

## RNA Tool Friction Log

| Step | Tool Used | Query | Verdict |
|------|-----------|-------|---------|
| Read addObject/dropObject/modifyObject | Read tool | driver.go:629-666 | needed for pattern |
| Read SchemaObjectDiff | Read tool | driver.go:708-744 | needed for pattern |
