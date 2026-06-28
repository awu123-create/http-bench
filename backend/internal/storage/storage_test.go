package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigrationsAndPragmas(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "storage-test.db"), filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// PRAGMA busy_timeout 应为 5000
	var busyTimeout int
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}

	// 三张业务表应存在
	tables := []string{"test_tasks", "metrics_buckets", "test_results"}
	for _, tbl := range tables {
		var name string
		err := db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		if err != nil {
			t.Fatalf("table %q not found: %v", tbl, err)
		}
	}

	// goose 迁移版本应为 3（001/002/003 全部应用）
	var version int64
	if err := db.QueryRowContext(ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied=1").Scan(&version); err != nil {
		t.Fatalf("query goose version: %v", err)
	}
	if version != 3 {
		t.Fatalf("goose version = %d, want 3", version)
	}

	// timeout_millis 列应存在（migration 003 已应用），timeout_seconds 应已删除
	cols, err := db.QueryContext(ctx, "PRAGMA table_info(test_tasks)")
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	defer cols.Close()
	hasTimeoutMillis := false
	hasTimeoutSeconds := false
	for cols.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := cols.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		switch name {
		case "timeout_millis":
			hasTimeoutMillis = true
		case "timeout_seconds":
			hasTimeoutSeconds = true
		}
	}
	if !hasTimeoutMillis {
		t.Fatal("timeout_millis column missing (migration 003 not applied)")
	}
	if hasTimeoutSeconds {
		t.Fatal("timeout_seconds column still exists (should have been dropped by migration 003)")
	}
}
