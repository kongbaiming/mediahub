package db

import (
	"strings"
	"testing"
)

func TestSplitSQL_mediaFilesLayer(t *testing.T) {
	content, err := migrationsFS.ReadFile("sql/000008_media_files_layer.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	stmts := splitSQL(string(content))
	if len(stmts) < 10 {
		t.Fatalf("expected >= 10 statements, got %d", len(stmts))
	}
	if !strings.HasPrefix(stmts[0], "CREATE TABLE IF NOT EXISTS media_files") {
		t.Fatalf("first statement unexpected: %q", stmts[0][:min(60, len(stmts[0]))])
	}
	for i, s := range stmts {
		if strings.Contains(s, "CREATE TABLE") && strings.Contains(s, "CREATE INDEX") {
			t.Fatalf("statement %d contains multiple DDL blocks (missing semicolons?)", i+1)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
