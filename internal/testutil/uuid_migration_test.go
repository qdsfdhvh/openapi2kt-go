package testutil

import (
	"path/filepath"
	"testing"
)

func TestUUIDMigrationAcceptsNativeSnapshotPaths(t *testing.T) {
	for _, tc := range []struct{ id, file, property string }{
		{"scalars-30", "Record.kt", "uuid"},
		{"catalog-graph", "Account.kt", "id"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			legacy := []byte("public val " + tc.property + ": kotlin.String,\npublic val text: kotlin.String,\n")
			want := "public val " + tc.property + ": kotlin.uuid.Uuid,\npublic val text: kotlin.String,\n"
			for _, path := range []string{"org/example/models/" + tc.file, filepath.Join("org", "example", "models", tc.file)} {
				if got := string(CurrentUUIDSnapshot(t, Case{ID: tc.id}, path, legacy)); got != want {
					t.Fatalf("native snapshot path %q: got %q, want %q", path, got, want)
				}
			}
			if got := string(CurrentUUIDSnapshot(t, Case{ID: tc.id}, filepath.Join("org", "example", "models", "Unchanged.kt"), legacy)); got != string(legacy) {
				t.Fatal("unrelated snapshot changed")
			}
		})
	}
}
