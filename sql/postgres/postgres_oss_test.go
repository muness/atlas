// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package postgres

import (
	"context"
	"database/sql"
	"os"
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
