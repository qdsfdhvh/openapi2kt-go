package testutil

import (
	"strings"
	"testing"
)

// ScalarGraduation documents source-keyed feature graduations without editing
// the historical manifest, inputs or golden bytes. This is not an error skip:
// both core and native CLI must prove the explicit new artifact contract.
func ScalarGraduation(c Case) bool {
	switch c.ID {
	case "format-date", "format-date-time", "format-byte", "format-binary", "unsupported-native-date", "unsupported-native-bigint", "null-type":
		return true
	}
	return false
}

func CheckScalarGraduation(t testing.TB, c Case, files map[string]string) {
	t.Helper()
	types := map[string]string{
		"format-date":               "kotlinx.datetime.LocalDate",
		"format-date-time":          "kotlin.time.Instant",
		"format-byte":               "kotlin.ByteArray",
		"format-binary":             "kotlin.ByteArray",
		"unsupported-native-date":   "kotlin.time.Instant",
		"unsupported-native-bigint": "kotlin.Long",
		"null-type":                 "kotlinx.serialization.json.JsonNull",
	}
	ty, ok := types[c.ID]
	if !ok || c.Expect.Kind != "error" || c.Expect.Code != "E_SCHEMA_UNSUPPORTED" || c.Input != "invalid/"+c.ID+".json" {
		t.Fatal("invalid historical scalar graduation identity", c)
	}
	base := strings.ReplaceAll(c.PackageName, ".", "/") + "/models/"
	wantType := ty + "? = null"
	wantFiles := 1
	if c.ID == "unsupported-native-bigint" {
		wantFiles = 2
		wantType = "@kotlinx.serialization.Serializable(with = " + c.PackageName + ".models.LongAsStringSerializer::class) " + wantType
		helper := files[base+"LongAsStringSerializer.kt"]
		for _, token := range []string{"internal object LongAsStringSerializer", "PrimitiveKind.STRING", "encoder.encodeString(value.toString())", "decoder.decodeString().toLong()"} {
			if !strings.Contains(helper, token) {
				t.Fatal("graduated helper contract missing", token)
			}
		}
	}
	if len(files) != wantFiles || !strings.Contains(files[base+"Record"+c.ModelNameSuffix+".kt"], "public val value: "+wantType) {
		t.Fatal("graduated original artifact contract mismatch", c.ID, files)
	}
}
