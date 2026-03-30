# Session: Issue #4 — Add SQLObject type to schema model

**Branch:** `4-sqlobject-type`
**PR:** https://github.com/muness/atlas/pull/16
**Issue:** https://github.com/muness/atlas/issues/4

---

## Phase 1: Problem Statement

Issue #4 is open with clear acceptance criteria.

**Acceptance Criteria:**
- [ ] `SQLObject` type added to `sql/schema/schema.go` alongside `View`
- [ ] Fields: `Name string`, `Type string`, `Body string`, `Schema *Schema`
- [ ] Implements the `Object` interface (`obj()` marker)
- [ ] `AddObject`/`DropObject`/`ModifyObject` work with `SQLObject`

---

## Phase 2: Branch + Draft PR — Solution Space

**Branch:** `4-sqlobject-type`
**PR:** #16

### Solution

Minimal: add `SQLObject` struct alongside `View` in schema.go, add `obj()` marker. The AddObject/DropObject/ModifyObject types use the `Object` interface, so they already work.

---

## Phase 3: Execute

- `sql/schema/schema.go`: Added `SQLObject` struct after `View`, added `func (*SQLObject) obj() {}`
- `sql/schema/schema_test.go` (or existing test file): Compile-time interface check

---

## RNA Tool Friction Log

| Step | Tool Used | Query | Verdict |
|------|-----------|-------|---------|
| Read schema.go View struct | Read tool | schema.go:52-60 | ok |
| Read migrate.go AddObject | Read tool | migrate.go:85-112 | ok |
