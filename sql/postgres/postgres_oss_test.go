// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"ariga.io/atlas/sql/schema"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// testDB opens a real PostgreSQL connection using TEST_DATABASE_URL.
// The test is skipped if the environment variable is not set.
func testDB(t *testing.T) *Driver {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	drv, err := Open(db)
	require.NoError(t, err)
	return drv.(*Driver)
}

// TestIntegration_InspectExtensions verifies that inspectExtensions() is
// wired into both InspectSchema() and InspectRealm() and that extensions
// installed in the database are visible in the returned model.
// plpgsql is always present in PostgreSQL; the pgvector service image also
// has the vector extension available.
func TestIntegration_InspectExtensions(t *testing.T) {
	drv := testDB(t)
	ctx := context.Background()
	opts := &schema.InspectOptions{
		Mode: schema.InspectSchemas | schema.InspectTypes,
	}

	t.Run("InspectSchema", func(t *testing.T) {
		s, err := drv.InspectSchema(ctx, "public", opts)
		require.NoError(t, err)
		require.NotNil(t, s.Realm, "schema must be attached to a realm")
		var names []string
		for _, obj := range s.Realm.Objects {
			if ext, ok := obj.(*Extension); ok {
				names = append(names, ext.T)
			}
		}
		require.Contains(t, names, "plpgsql", "plpgsql extension must be present via InspectSchema")
	})

	t.Run("InspectRealm", func(t *testing.T) {
		r, err := drv.InspectRealm(ctx, &schema.InspectRealmOption{
			Mode: schema.InspectSchemas | schema.InspectTypes,
		})
		require.NoError(t, err)
		var names []string
		for _, obj := range r.Objects {
			if ext, ok := obj.(*Extension); ok {
				names = append(names, ext.T)
			}
		}
		require.Contains(t, names, "plpgsql", "plpgsql extension must be present via InspectRealm")
	})
}

// TestIntegration_InspectFunctionsAndTriggers verifies that inspectFunctions() and
// inspectTriggers() are wired into InspectSchema() and correctly capture user-defined
// functions and triggers in the schema.
func TestIntegration_InspectFunctionsAndTriggers(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	rawDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { rawDB.Close() })
	rawDrv, err := Open(rawDB)
	require.NoError(t, err)
	drv := rawDrv.(*Driver)
	ctx := context.Background()

	// Create a scratch schema to isolate test objects.
	schemaName := "atlas_fn_trigger_test"
	_, err = rawDB.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	require.NoError(t, err)
	t.Cleanup(func() {
		rawDB.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
	})

	// Create a simple table (needed for the trigger).
	_, err = rawDB.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.events (
			id serial PRIMARY KEY,
			payload text
		)
	`, schemaName))
	require.NoError(t, err)

	// Create a user-defined function.
	_, err = rawDB.ExecContext(ctx, fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION %s.log_event()
		RETURNS trigger AS $$
		BEGIN
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`, schemaName))
	require.NoError(t, err)

	// Create a trigger that uses the function.
	_, err = rawDB.ExecContext(ctx, fmt.Sprintf(`
		CREATE TRIGGER events_log
		AFTER INSERT ON %s.events
		FOR EACH ROW EXECUTE FUNCTION %s.log_event()
	`, schemaName, schemaName))
	require.NoError(t, err)

	// Inspect the schema with types mode to capture functions and triggers.
	opts := &schema.InspectOptions{
		Mode: schema.InspectSchemas | schema.InspectTypes,
	}
	s, err := drv.InspectSchema(ctx, schemaName, opts)
	require.NoError(t, err)

	// Collect function and trigger names from schema objects.
	var funcNames, trigNames []string
	for _, obj := range s.Objects {
		so, ok := obj.(*schema.SQLObject)
		if !ok {
			continue
		}
		switch so.Type {
		case "function":
			funcNames = append(funcNames, so.Name)
			require.NotEmpty(t, so.Body, "function body must not be empty")
			require.True(t, strings.Contains(so.Body, "log_event"), "function body should contain function name")
		case "trigger":
			trigNames = append(trigNames, so.Name)
			require.NotEmpty(t, so.Body, "trigger body must not be empty")
		}
		require.Equal(t, s, so.Schema, "SQLObject.Schema must point to the inspected schema")
	}

	require.Contains(t, funcNames, "log_event", "log_event function must be inspected")
	require.Contains(t, trigNames, "events_log", "events_log trigger must be inspected")
}
