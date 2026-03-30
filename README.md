# Atlas — PostgreSQL + SQLAlchemy Fork

> **This is a fork of [ariga/atlas](https://github.com/ariga/atlas)** focused on supporting the PostgreSQL features needed for Python/SQLAlchemy applications in Atlas OSS.
>
> **Goal:** A SQLAlchemy-managed Postgres schema can run `atlas migrate diff` and get a correct, complete migration plan — including extension-backed types and function/trigger definitions — without Atlas Pro or raw SQL escape hatching.
>
> **What this fork adds:**
> - **pgvector extension + `vector(N)` types** — unblocks embedding columns (previously Pro-only)
> - **PL/pgSQL functions and DML triggers** — tracked as checksum-verified SQL objects; detected and replayed when bodies change
> - **Custom ENUMs** — adding values supported; drop/reorder not possible (PostgreSQL constraint)
>
> See the upstream project at [ariga/atlas](https://github.com/ariga/atlas) for the full feature set.

## Build from Source

```bash
git clone https://github.com/muness/atlas.git
cd atlas/cmd/atlas
go build -o ~/go/bin/atlas .
```

Verify:

```bash
atlas version
```

## SQLAlchemy + PostgreSQL Usage

This fork is designed for Python apps using SQLAlchemy with PostgreSQL. It supports `vector(N)` columns (pgvector), PL/pgSQL functions, DML triggers, and custom ENUMs — all detected automatically by `atlas migrate diff`.

### Prerequisites

```bash
# Install the SQLAlchemy schema provider in your project
pip install atlas-provider-sqlalchemy
```

### Schema loader

Create `atlas_loader.py` in your project root. This generates the desired schema SQL from your SQLAlchemy models:

```python
"""Atlas schema loader — run via: poetry run python atlas_loader.py > desired_schema.sql"""
from pathlib import Path
from sqlalchemy import event
from atlas_provider_sqlalchemy.ddl import dump_ddl
from myapp.models import Base  # adjust to your models import

# atlas_provider_sqlalchemy strips newlines from DDL, which breaks PL/pgSQL bodies.
# Remove after_create listeners (functions/triggers) and emit raw SQL files instead.
for table in Base.metadata.tables.values():
    for fn in list(table.dispatch.after_create):
        event.remove(table, "after_create", fn)

print("CREATE EXTENSION IF NOT EXISTS vector;")  # remove if not using pgvector
print()

dump_ddl("postgresql", [Base.metadata], [])

# Emit raw DDL files (functions + triggers) with proper formatting.
# Must come after tables since triggers reference them.
ddl_dir = Path(__file__).parent / "src" / "data" / "ddl"  # adjust path to your DDL files
for sql_file in sorted(ddl_dir.glob("*.sql")):
    print(sql_file.read_text())
```

> **Why the loader?** `atlas_provider_sqlalchemy` strips newlines from DDL output, which breaks multi-line PL/pgSQL function bodies. The loader removes SQLAlchemy's `after_create` event listeners (which would emit broken one-line SQL) and appends your raw `.sql` DDL files instead. `data "external_schema"` (the upstream clean solution) is Pro-only; this is the OSS workaround.

### `atlas.hcl`

```hcl
env "dev" {
  src = "file://desired_schema.sql"
  url = "postgres://user:password@localhost:5432/mydb?sslmode=disable"
  dev = "docker://pgvector/pgvector/pg16/dev"  # or "docker://postgres/16/dev" if no pgvector

  migration {
    dir    = "file://migrations"
    format = atlas
  }
}
```

### Day-to-day workflow

```bash
# 1. Edit a SQLAlchemy model, DDL file, or enum
# 2. Regenerate the desired schema
poetry run python atlas_loader.py > desired_schema.sql

# 3. Generate the migration
atlas migrate diff <name> --env dev

# 4. Review migrations/<timestamp>_<name>.sql, then apply
atlas migrate apply --env dev
```

Atlas detects and generates migrations for:

| Change | Migration output |
|--------|-----------------|
| Column type (e.g. `vector(768)` → `vector(1536)`) | `ALTER TABLE ... ALTER COLUMN ... TYPE vector(1536)` |
| Function body change | `CREATE OR REPLACE FUNCTION ...` |
| Trigger definition change | `DROP TRIGGER IF EXISTS ... ; CREATE TRIGGER ...` |
| Enum value added | `ALTER TYPE ... ADD VALUE '...'` |
| Table added/modified | Standard `CREATE TABLE` / `ALTER TABLE` |

### Migrating from Alembic

If your schema was previously managed by Alembic, create a baseline migration so Atlas doesn't try to recreate the existing schema:

```bash
# 1. Generate a baseline migration from the current live DB
atlas migrate diff baseline --env dev

# 2. Review and edit migrations/<timestamp>_baseline.sql:
#    - Prepend: CREATE EXTENSION IF NOT EXISTS vector;  (if using pgvector)
#    - Remove any statements that would fail on first run

# 3. Mark the live DB as already at the baseline (don't re-run the SQL)
atlas migrate set <timestamp> --env dev

# 4. Verify status
atlas migrate status --env dev
# Expected: Migration Status: OK
```

After this, normal `atlas migrate diff` → `atlas migrate apply` replaces Alembic going forward.

### Fresh database

For a brand-new database with no prior migration history:

```bash
# 1. Start with an empty database (or create one)
# 2. Generate the initial migration from your SQLAlchemy models
poetry run python atlas_loader.py > desired_schema.sql
atlas migrate diff initial --env dev

# 3. Apply to the target database
atlas migrate apply --env dev
```

### Caveats

- **`desired_schema.sql` is generated** — add it to `.gitignore`; regenerate before every `atlas migrate diff`
- **Enum values can be added but not removed** — PostgreSQL does not support dropping enum values without recreating the type
- **Functions are matched by name** — overloaded functions (same name, different signatures) are tracked as separate objects; avoid overloads if you want clean diffs
- **Triggers depend on functions** — if you change both a function and its trigger in one edit, both appear in the same migration (functions first, then triggers)

## Upstream

This fork tracks [ariga/atlas](https://github.com/ariga/atlas). For databases other than PostgreSQL, CI/CD integrations, schema testing, migration linting, and the full feature set, see the upstream project and [atlasgo.io](https://atlasgo.io).
