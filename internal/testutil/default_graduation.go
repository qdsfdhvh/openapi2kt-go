package testutil

import (
	"strings"
	"testing"
)

// Keep the original rejected fixture and manifest bytes; prove its new output
// through both core and the actual native CLI instead of skipping the case.
func DefaultGraduation(c Case) bool { return c.ID == "unsupported-default" }

func CheckDefaultGraduation(t testing.TB, c Case, files map[string]string) {
	t.Helper()
	if !DefaultGraduation(c) || c.Expect.Kind != "error" || c.Expect.Code != "E_SCHEMA_UNSUPPORTED" || c.Input != "invalid/unsupported-default.json" {
		t.Fatal("invalid original default graduation", c)
	}
	path := strings.ReplaceAll(c.PackageName, ".", "/") + "/models/Record" + c.ModelNameSuffix + ".kt"
	if len(files) != 1 || !strings.Contains(files[path], `public val value: kotlin.String? = "ready",`) {
		t.Fatal("original default fixture output mismatch", files)
	}
}
