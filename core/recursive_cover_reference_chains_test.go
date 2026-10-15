package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestRecursiveCoverReferenceChainsOriginalJSONBacking(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/recursive-cover-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("recursive cover chain", checked, err)
		}
		for _, name := range []string{"CoverBridgeDto", "AgainBridgeDto", "SelfBridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlinx.serialization.json.JsonElement?\n") {
				t.Fatal(name, source(t, got, name))
			}
		}
		if !strings.Contains(source(t, got, "CoverFalseBridgeDto"), "= kotlinx.serialization.json.JsonElement\n") {
			t.Fatal(source(t, got, "CoverFalseBridgeDto"))
		}
		count := 0
		for _, w := range got.Warnings {
			if w.Code == "W_RECURSIVE_JSON_PROJECTION" {
				count++
				if w.Pointer == nil || (*w.Pointer != "/components/schemas/ZAny" && *w.Pointer != "/components/schemas/ZRaw" && *w.Pointer != "/components/schemas/ZEmpty") {
					t.Fatal(w)
				}
			}
		}
		if count != 3 {
			t.Fatal("cover root warnings lost/repeated", got.Warnings)
		}
	}
}
