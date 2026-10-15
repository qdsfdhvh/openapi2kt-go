package core_test

import (
	"openapi-kt/core"
	"os"
	"strings"
	"testing"
)

func TestRawTrueReferenceChainsOriginalJSONBacking(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/raw-true-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("raw/true chains", checked, err)
		}
		for _, name := range []string{"RawBridgeDto", "AgainBridgeDto", "TrueBridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlinx.serialization.json.JsonElement?\n") {
				t.Fatal(name, source(t, got, name))
			}
		}
		for _, name := range []string{"RawFalseBridgeDto", "TrueFalseBridgeDto"} {
			if !strings.Contains(source(t, got, name), "= kotlinx.serialization.json.JsonElement\n") {
				t.Fatal(name, source(t, got, name))
			}
		}
		raw, trueCount := 0, 0
		for _, w := range got.Warnings {
			if w.Code == "W_JSON_ELEMENT_APPROXIMATION" {
				raw++
				if w.Pointer == nil || (*w.Pointer != "/components/schemas/ZRaw" && *w.Pointer != "/components/schemas/ZEmpty") {
					t.Fatal(w)
				}
			}
			if w.Code == "W_BOOLEAN_SCHEMA_PROJECTION" {
				trueCount++
				if w.Pointer == nil || *w.Pointer != "/components/schemas/ZAny" {
					t.Fatal(w)
				}
			}
		}
		if raw != 2 || trueCount != 1 {
			t.Fatal("projection warning duplicated or lost original-root owner", got.Warnings)
		}
	}
}
