// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"io"
	"testing"
	"time"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/internal/sqltest"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/schema"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestDriver_LockAcquired(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	name, hash := "name", 797654004

	t.Run("NoTimeout", func(t *testing.T) {
		m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_lock"}).AddRow(1)).
			RowsWillBeClosed()
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(1)).
			RowsWillBeClosed()

		d := &Driver{conn: &conn{ExecQuerier: db}}
		unlock, err := d.Lock(context.Background(), name, 0)
		require.NoError(t, err)
		require.NoError(t, unlock())
		require.NoError(t, m.ExpectationsWereMet())
	})

	t.Run("WithTimeout", func(t *testing.T) {
		m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_lock"}).AddRow(0)).
			RowsWillBeClosed()
		m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_lock"}).AddRow(1)).
			RowsWillBeClosed()
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(1)).
			RowsWillBeClosed()

		d := &Driver{conn: &conn{ExecQuerier: db}}
		unlock, err := d.Lock(context.Background(), name, time.Second)
		require.NoError(t, err)
		require.NoError(t, unlock())
		require.NoError(t, m.ExpectationsWereMet())
	})
}

func TestDriver_LockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{conn: &conn{ExecQuerier: db}}
	name, hash := "migrate", 979249972

	t.Run("Internal", func(t *testing.T) {
		m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnError(io.EOF).
			RowsWillBeClosed()
		unlock, err := d.Lock(context.Background(), name, time.Minute)
		require.Equal(t, io.EOF, err)
		require.Nil(t, unlock)
	})
}

func TestDriver_UnlockError(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	d := &Driver{conn: &conn{ExecQuerier: db}}
	name, hash := "up", 1551306158
	acquired := func() {
		m.ExpectQuery(sqltest.Escape("SELECT pg_try_advisory_lock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(1)).
			RowsWillBeClosed()
	}

	t.Run("NotHeld", func(t *testing.T) {
		acquired()
		unlock, err := d.Lock(context.Background(), name, 0)
		require.NoError(t, err)
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(0)).
			RowsWillBeClosed()
		require.Error(t, unlock())
	})

	t.Run("Internal", func(t *testing.T) {
		acquired()
		unlock, err := d.Lock(context.Background(), name, 0)
		require.NoError(t, err)
		m.ExpectQuery(sqltest.Escape("SELECT pg_advisory_unlock($1)")).
			WithArgs(hash).
			WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(nil)).
			RowsWillBeClosed()
		require.Error(t, unlock())
	})
}

func TestDriver_CheckClean(t *testing.T) {
	s := schema.New("test")
	drv := &Driver{Inspector: &mockInspector{schema: s}, conn: &conn{schema: "test"}}
	// Empty schema.
	err := drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Revisions table found.
	s.AddTables(schema.NewTable("revisions"))
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)
	// Multiple tables.
	s.Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found table "a" in schema "test"`)

	r := schema.NewRealm()
	drv.schema = ""
	drv.Inspector = &mockInspector{realm: r}
	// Empty realm.
	err = drv.CheckClean(context.Background(), nil)
	require.NoError(t, err)
	// Revisions table found.
	s.Tables = []*schema.Table{schema.NewTable("revisions").SetSchema(s)}
	r.AddSchemas(s)
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Name: "revisions", Schema: "test"})
	require.NoError(t, err)
	// Unknown table.
	s.Tables[0].Name = "unknown"
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found table "unknown" in schema "test"`)
	// Multiple tables.
	s.Tables = []*schema.Table{schema.NewTable("a"), schema.NewTable("revisions")}
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.EqualError(t, err, `sql/migrate: connected database is not clean: found 2 tables in schema "test"`)
	// With auto created public schema.
	s.Tables = []*schema.Table{schema.NewTable("revisions")}
	r.AddSchemas(schema.New("public"))
	err = drv.CheckClean(context.Background(), &migrate.TableIdent{Schema: "test", Name: "revisions"})
	require.NoError(t, err)
}

func TestDriver_Version(t *testing.T) {
	db, m, err := sqlmock.New()
	require.NoError(t, err)
	mock{m}.version("130000")
	drv, err := Open(db)
	require.NoError(t, err)

	type vr interface{ Version() string }
	require.Implements(t, (*vr)(nil), drv)
	require.Equal(t, "130000", drv.(vr).Version())
}

func TestDriver_RealmRestoreFunc(t *testing.T) {
	var (
		apply   = &mockPlanApplier{}
		inspect = &mockInspector{}
		drv     = &Driver{
			Inspector:   inspect,
			Differ:      DefaultDiff,
			conn:        &conn{schema: "test"},
			PlanApplier: apply,
		}
	)
	f := drv.RealmRestoreFunc(schema.NewRealm().AddSchemas(schema.New("public")))

	// No changes.
	inspect.realm = schema.NewRealm().AddSchemas(schema.New("public"))
	err := f(context.Background())
	require.NoError(t, err)
	require.Empty(t, apply.applied)

	// Schema changes.
	inspect.realm = schema.NewRealm().AddSchemas(schema.New("public").AddTables(schema.NewTable("t1")))
	err = f(context.Background())
	require.NoError(t, err)
	require.Len(t, apply.applied, 2)
	drop, ok := apply.applied[0].(*schema.DropSchema)
	require.True(t, ok)
	require.Equal(t, "public", drop.S.Name)
	create, ok := apply.applied[1].(*schema.AddSchema)
	require.True(t, ok)
	require.Equal(t, "public", create.S.Name)

	// Recreate the public schema.
	apply.applied = nil
	inspect.realm = schema.NewRealm().AddSchemas(schema.New("test").AddTables(schema.NewTable("t1")))
	err = f(context.Background())
	require.NoError(t, err)
	require.Len(t, apply.applied, 2)
	drop, ok = apply.applied[0].(*schema.DropSchema)
	require.True(t, ok)
	require.Equal(t, "test", drop.S.Name)
	create, ok = apply.applied[1].(*schema.AddSchema)
	require.True(t, ok)
	require.Equal(t, "public", create.S.Name)

	// Non-public changes.
	apply.applied = nil
	f = drv.RealmRestoreFunc(schema.NewRealm().AddSchemas(schema.New("test")))
	inspect.realm = schema.NewRealm().AddSchemas(schema.New("test").AddTables(schema.NewTable("t1")))
	err = f(context.Background())
	require.NoError(t, err)
	require.Len(t, apply.applied, 1)
	dropT, ok := apply.applied[0].(*schema.DropTable)
	require.True(t, ok)
	require.Equal(t, "t1", dropT.T.Name)
}

type mockInspector struct {
	schema.Inspector
	realm  *schema.Realm
	schema *schema.Schema
}

func (m *mockInspector) InspectSchema(context.Context, string, *schema.InspectOptions) (*schema.Schema, error) {
	if m.schema == nil {
		return nil, &schema.NotExistError{}
	}
	return m.schema, nil
}

func (m *mockInspector) InspectRealm(context.Context, *schema.InspectRealmOption) (*schema.Realm, error) {
	return m.realm, nil
}

func TestConvertExtensions(t *testing.T) {
	r := &schema.Realm{}
	exs := []*extension{
		{Name: "pgvector"},
		{Name: "uuid-ossp"},
	}
	exs[0].Extra.SetAttr(&schemahcl.Attr{K: "version", V: cty.StringVal("0.7.0")})
	exs[0].Extra.SetAttr(&schemahcl.Attr{K: "schema", V: cty.StringVal("public")})
	exs[1].Extra.SetAttr(&schemahcl.Attr{K: "comment", V: cty.StringVal("UUID generation functions")})
	require.NoError(t, convertExtensions(exs, r))
	require.Len(t, r.Objects, 2)

	ext1, ok := r.Objects[0].(*Extension)
	require.True(t, ok)
	require.Equal(t, "pgvector", ext1.T)
	require.Equal(t, "0.7.0", ext1.Version)
	require.Equal(t, "public", ext1.Schema)
	require.Empty(t, ext1.Attrs)

	ext2, ok := r.Objects[1].(*Extension)
	require.True(t, ok)
	require.Equal(t, "uuid-ossp", ext2.T)
	require.Empty(t, ext2.Version)
	require.Empty(t, ext2.Schema)
	require.Len(t, ext2.Attrs, 1)
	c, ok := ext2.Attrs[0].(*schema.Comment)
	require.True(t, ok)
	require.Equal(t, "UUID generation functions", c.Text)
}

func TestRealmObjectDiff_Extensions(t *testing.T) {
	d := &diff{}
	from := &schema.Realm{
		Objects: []schema.Object{
			&Extension{T: "pgvector", Version: "0.6.0"},
			&Extension{T: "uuid-ossp"},
		},
	}
	to := &schema.Realm{
		Objects: []schema.Object{
			&Extension{T: "pgvector", Version: "0.7.0"}, // version changed
			&Extension{T: "postgis"},                     // new
			// uuid-ossp dropped
		},
	}
	changes, err := d.RealmObjectDiff(from, to)
	require.NoError(t, err)
	require.Len(t, changes, 3)
	// from loop: pgvector version changed, uuid-ossp dropped.
	mod, ok := changes[0].(*schema.ModifyObject)
	require.True(t, ok, "expected ModifyObject for pgvector")
	require.Equal(t, "pgvector", mod.From.(*Extension).T)
	require.Equal(t, "0.7.0", mod.To.(*Extension).Version)
	drop, ok := changes[1].(*schema.DropObject)
	require.True(t, ok, "expected DropObject for uuid-ossp")
	require.Equal(t, "uuid-ossp", drop.O.(*Extension).T)
	// to loop: postgis is new.
	add, ok := changes[2].(*schema.AddObject)
	require.True(t, ok, "expected AddObject for postgis")
	require.Equal(t, "postgis", add.O.(*Extension).T)
}

type mockPlanApplier struct {
	planned []schema.Change
	applied []schema.Change
}

func (m *mockPlanApplier) PlanChanges(_ context.Context, _ string, planned []schema.Change, _ ...migrate.PlanOption) (*migrate.Plan, error) {
	m.planned = append(m.planned, planned...)
	return nil, nil
}

func (m *mockPlanApplier) ApplyChanges(_ context.Context, applied []schema.Change, _ ...migrate.PlanOption) error {
	m.applied = append(m.applied, applied...)
	return nil
}
