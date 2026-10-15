package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestHTTPRequestOnlyPresence(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "request-presence.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := report(t, input, "Dto")
	model := source(t, r, "ModifyInputDto")
	for _, want := range []string{
		"token: kotlin.String?,",
		"org.example.models.models.Patch<kotlin.String>",
		"org.example.models.models.Patch<kotlin.Boolean>",
		"org.example.models.models.Patch<org.example.models.models.StateDto>",
		"org.example.models.models.Patch.Absent",
		"kotlinx.serialization.EncodeDefault.Mode.NEVER",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("missing %q in %s", want, model)
		}
	}
	if strings.Contains(model, "token: kotlin.String? = null") {
		t.Fatal("required request key gained a default", model)
	}
	shared := source(t, r, "SharedDto")
	if strings.Contains(shared, "Patch") || !strings.Contains(shared, "title: kotlin.String? = null") {
		t.Fatal("shared component presence changed", shared)
	}
	_ = source(t, r, "Patch")
	api := source(t, r, "DefaultApi")
	if !strings.Contains(api, "input: org.example.models.api.models.ModifyInputDto,") {
		t.Fatal(api)
	}
}
