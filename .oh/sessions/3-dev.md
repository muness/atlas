# Session: Issue #3 — Register vector(N) type in TypeRegistry and columnType()

**Branch:** `3-vector-type`
**PR:** https://github.com/muness/atlas/pull/15
**Issue:** https://github.com/muness/atlas/issues/3

---

## Phase 1: Problem Statement

Issue #3 is open with clear acceptance criteria.

**Acceptance Criteria:**
- [ ] `vector(N)` parses correctly via `ParseType()`
- [ ] `vector(N)` round-trips through inspect → diff → migrate without data loss
- [ ] TypeRegistry includes a `vector` spec with dimension parameter
- [ ] `columnType()` handles the `vector` case
- [ ] Test: column with `vector(768)` type survives a no-op diff

---

## Phase 2: Branch + Draft PR — Solution Space

**Branch:** `3-vector-type`
**PR:** #15

### Solution

The approach: Add minimal support following the `bit(N)` pattern.

1. **`driver.go`**: Add `TypeVector = "vector"` constant.

2. **`convert.go`** — `parseColumn()`: Add a `"vector"` case that strips schema qualifier and extracts dimension into `c.size`. Also handle the `"public.vector"` prefix.

3. **`convert.go`** — `columnType()`: Add `"vector"` case that returns a `UserDefinedType` with formatted name `"vector(N)"` when N > 0, or `"vector"` when N == 0.

4. **`sqlspec.go`** — `TypeRegistry`: Add `vector` spec with a `dim` int64 attribute (optional).

5. **Tests**: `convert_test.go` — test `ParseType("vector(768)")`, `ParseType("public.vector(768)")`, round-trip `FormatType`. Mock-based no-op diff test.

**Key design decision**: Use `UserDefinedType` to represent `vector(N)` - this fits since vector is a user-defined extension type. Store the full `"vector(N)"` string in `T` field.

---

## Phase 3: Execute

### Changes

- `sql/postgres/driver.go`: `TypeVector = "vector"` constant
- `sql/postgres/convert.go`: `parseColumn` vector case, `columnType` vector case
- `sql/postgres/sqlspec.go`: TypeRegistry vector spec
- `sql/postgres/convert_test.go`: tests

---

## Phase 4: Ship

TBD

---

## RNA Tool Friction Log

| Step | Tool Used | Query | Verdict |
|------|-----------|-------|---------|
| Read parseColumn | Read tool | convert.go:359-428 | ok |
| Read columnType | Read tool | convert.go:220-314 | ok |
| Read TypeRegistry | Read tool | sqlspec.go:888-975 | ok |
