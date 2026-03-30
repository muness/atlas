# Session: Issue #3 — vector(N) type dimension changes not detected by atlas schema diff

**Branch:** `3-vector-dim-diff`
**PR:** TBD
**Issue:** https://github.com/muness/atlas/issues/3

---

## Phase 1: Problem Statement

Issue #3 reopened with a specific bug: vector(N) type dimension changes are not detected by `atlas schema diff`.

**Root Cause:**
`sql/postgres/diff.go` line 363-368, the `*UserDefinedType` case in `typeChanged()`:

```go
case *UserDefinedType:
    toT := toT.(*UserDefinedType)
    changed = toT.T != fromT.T &&
        ns != "" && trimSchema(toT.T, ns) != trimSchema(toT.T, ns)
```

Two bugs:
1. The last condition compares `toT.T` with itself (`trimSchema(toT.T, ns) != trimSchema(toT.T, ns)`) — always false
2. Logic uses AND where it should be: "changed if strings differ AND that difference is not merely schema qualification"

Correct logic: `changed = fromT.T != toT.T && (ns == "" || trimSchema(fromT.T, ns) != trimSchema(toT.T, ns))`

**Acceptance Criteria:**
- [x] `vector(768)` vs `vector(1536)` is detected as a column type change
- [x] `atlas migrate diff` emits `ALTER TABLE t ALTER COLUMN c TYPE vector(N)` for dimension changes
- [x] No-op diff still produces no changes when dimensions match
- [x] Test against real DBs (localhost:5432 vs localhost:5431) confirms migration generated

---

## Phase 2: Branch + Draft PR — Solution Space

**Branch:** `3-vector-dim-diff`

### Solution

Minimal fix to `sql/postgres/diff.go` `typeChanged()` `*UserDefinedType` case.

Fix the logic from AND to the correct form:
```go
changed = fromT.T != toT.T && (ns == "" || trimSchema(fromT.T, ns) != trimSchema(toT.T, ns))
```

This correctly handles:
- `vector(768)` vs `vector(1536)` → different (no ns stripping needed, changed=true)
- `public.vector(768)` vs `vector(768)` with `ns="public"` → same after stripping, changed=false
- `public.mytype` vs `public.othertype` → different after stripping, changed=true

---

## Phase 3: Execute

### Changes

- `sql/postgres/diff.go`: Fix `*UserDefinedType` case in `typeChanged()` (one-line fix)
- `sql/postgres/diff_test.go`: Add test for vector dimension change detection

---

## Phase 4: Ship

TBD

---

## RNA Tool Friction Log

| Step | Tool Used | Query | Verdict |
|------|-----------|-------|---------|
| Find vector handling | Grep | vector in convert.go | ok |
| Find typeChanged | Grep | typeChanged in diff.go | ok |
| Read typeChanged | Read | diff.go:337-416 | ok — needed to read the buggy logic |
| Read columnType | Read | convert.go:220-319 | ok — confirm inspect path |
| Read inspect scan | Read | inspect.go:256-332 | ok — confirm fmtype usage |
