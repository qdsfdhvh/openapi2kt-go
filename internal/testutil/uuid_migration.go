package testutil

import (
	"path/filepath"
	"strings"
	"testing"
)

// CurrentUUIDSnapshot applies the two intentional UUID API changes to immutable
// historical goldens. All other bytes and the complete file inventory remain
// checked by both the core corpus and native CLI corpus.
func CurrentUUIDSnapshot(t testing.TB, c Case, path string, legacy []byte) []byte {
	t.Helper()
	path = filepath.ToSlash(path)
	property := ""
	if c.ID == "scalars-30" && strings.HasSuffix(path, "/Record.kt") {
		property = "uuid"
	}
	if c.ID == "catalog-graph" && strings.HasSuffix(path, "/Account.kt") {
		property = "id"
	}
	if property == "" {
		return legacy
	}
	old := "public val " + property + ": kotlin.String,"
	if strings.Count(string(legacy), old) != 1 {
		t.Fatal("historical UUID migration source drift", c.ID, path)
	}
	return []byte(strings.Replace(string(legacy), old, "public val "+property+": kotlin.uuid.Uuid,", 1))
}
