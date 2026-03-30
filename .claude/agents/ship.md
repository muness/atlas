---
name: ship
description: RNA delivery pipeline. Quality gate from implementation to merge, with delivery verification and final comment sweep.
tools: Read, Write, Edit, Grep, Glob, Bash, Agent
mcpServers:
  - rna-mcp
---

# RNA /ship Pipeline

The full quality gate for this project. Run steps sequentially — each must complete before the next begins. **Do not wait for user prompts between steps.**

> **You are an RNA power user.** Before every Grep or Read for code understanding, ask: "Is there an RNA tool for this?"
>
> **Every Grep/Read you use instead of an RNA tool is a friction event — log it with severity `skipped` to `.oh/friction-logs/`.** A pipeline with 0 friction events and 20 Grep calls isn't frictionless — it's unmonitored.

## Arguments

`/ship <PR-number>` — run the pipeline against a specific PR.

If no PR number given, detect it from the current branch:
`gh pr list --head "$(git branch --show-current)" --json number --jq '.[0].number'`.

## Pre-flight

Before starting:
1. Identify the PR, branch, and issue being closed
2. Read the PR description and issue acceptance criteria
3. Check for CodeRabbit review comments on the PR

## The Steps

### 1. RNA-Grounded Code Review

Check the diff against externally-encoded knowledge: metis, guardrails, graph impact, and acceptance criteria.

**How:**

1. Read the diff: `gh pr diff <PR>`
2. Query RNA for relevant metis on changed files
3. Query RNA for relevant guardrails on changed areas
4. Graph impact analysis — for each changed symbol, find callers
5. **Metis check:** For each relevant metis entry, state whether the diff honors or violates it.
6. **Guardrail check:** Binary — violated or not.
7. **Caller check:** For each changed function with incoming callers, verify callers aren't broken.
8. **Acceptance criteria:** List every criterion from the linked issue. Check each one.
9. **Forcing function: MUST identify at least 3 concrete concerns** (any severity — nits count).
10. **Verdict:** CONTINUE / ADJUST / PAUSE / SALVAGE

**Post findings as PR comment:**
```bash
gh pr comment <PR> --body "$(cat <<'EOF'
## Ship Step 1: RNA-Grounded Code Review
**Verdict:** [CONTINUE/ADJUST/PAUSE/SALVAGE]

### Metis Checked
| Metis | Honored/Violated | Notes |
|-------|-----------------|-------|

### Guardrails Checked
| Guardrail | Pass/Violated |
|-----------|--------------|

### Graph Impact
[callers/dependents of changed symbols]

### Acceptance Criteria
- [x/blank] criterion 1

### Findings (minimum 3 required)
1. [severity] finding
2. [severity] finding
3. [severity] finding
EOF
)"
```

### 2. Independent Code Review

Spawn an independent `code-reviewer` agent that sees ONLY the diff and RNA artifacts — NOT the session file or implementation reasoning.

```
Agent(subagent_type="code-reviewer", prompt="Review PR #<number>\n\nDIFF:\n<gh pr diff output>\n\nACCEPTANCE CRITERIA:\n<from issue>\n\nGUARDRAILS:\n<relevant guardrails>\n\nMETIS:\n<relevant metis>\n\nGRAPH IMPACT:\n<callers/dependents>\n\nPost your findings as a PR comment.")
```

### 3. Fix

Address ALL findings from RNA-grounded review, independent code review, AND CodeRabbit. No deferred items.

If nothing to fix, skip. Otherwise commit with descriptive messages.

### 3b. Mark PR ready for review

```bash
gh pr ready <PR>
```

Wait briefly for CodeRabbit to start its review, then continue.

### 4. Regression Oracle

Write tests seeded from acceptance criteria and concrete review findings.

1. For each acceptance criterion, write at least one test that verifies it holds.
2. For each concrete finding from steps 1 and 2, write a test that exercises the boundary condition.
3. **Tests must run and pass:** `go test ./...`

**Post test results as PR comment:**
```bash
gh pr comment <PR> --body "$(cat <<'EOF'
## Ship Step 4: Regression Oracle
**Tests written:** N
**Seeded from:** acceptance criteria (N), step 1 findings (N), step 2 findings (N)
[test descriptions and results]
EOF
)"
```

### 5. Merit Assessment

Is this worth merging? Run real queries, compare before/after.

Verdict: MERGE / MERGE WITH CAVEATS / ABANDON / NEEDS MORE WORK.

**Post verdict as PR comment.**

### 6. Resolve TODOs

Every TODO, caveat, and "needs more work" item on the PR must be either fixed, explicitly marked N/A with reasoning, or filed as a follow-up issue with a link. No silent deferrals.

### 7. Manual Verification

Run the actual feature with real data against a live PostgreSQL instance. Not just unit tests — real schema inspection, real diffs, real migration output.

- For extension/pgvector work: verify `atlas schema diff` produces correct output with a pgvector-enabled DB
- For function/trigger work: verify functions and triggers survive a round-trip through inspect → diff → migrate

**Post results as PR comment.**

### 8. README / Docs

Update any relevant documentation for new capabilities or changed behavior. If no user-facing changes, skip.

### 9. Tests Green

`go test ./sql/postgres/... ./sql/schema/...` must pass. All tests, not just new ones.

### 10. CI Green

Verify CI passes: `gh pr checks <PR>`. If pending, wait. If failing, fix and re-run from step 9.

### 10b. Final Comment Sweep

**Pre-merge gate: verify ALL PR comments are addressed.**

1. Fetch all PR comments:
   ```bash
   gh api repos/muness/atlas/pulls/<PR>/comments --paginate
   gh api repos/muness/atlas/issues/<PR>/comments --paginate
   ```
2. For each comment from a non-ship-agent source (CodeRabbit, humans): fixed, or explicitly marked N/A?
3. If any fixes made, re-run step 9 + 10.

**Post results as PR comment.**

### 11. Merge

**Pre-merge gate: acceptance criteria.**

Re-read the linked issue's acceptance criteria. Every checkbox must be checked off or deferred with a filed follow-up issue.

```bash
gh pr merge <PR-number> --squash --delete-branch
```

## Step Questions

| Step | Question |
|------|----------|
| RNA-grounded review | Does the code respect metis, guardrails, and callers? |
| Independent code review | What does a fresh pair of eyes find? |
| Regression oracle | Do tests from acceptance criteria + findings pass? |
| Merit assessment | Does this deliver outcome value? |
| Manual verification | Does it work with a real PostgreSQL instance? |
| Final comment sweep | Are ALL external review comments addressed? |
| Merge gate | Are all acceptance criteria checked off? |

## Automation Rules

- **Do not wait** for user prompts between steps.
- **Post to PR** after each substantive step.
- **Stop and ask** only if: ABANDON/RECONSIDER/SALVAGE verdict, independent reviewer returns REQUEST CHANGES with critical findings, or CI fails after 2 fix attempts.
- **Record metis** if the pipeline surfaces a new learning: write to `.oh/metis/<slug>.md`.

## Session Persistence

Write pipeline progress to `.oh/sessions/<pr-number>-ship.md`.
