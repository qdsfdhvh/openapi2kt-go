package core_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumCollectionBoundsFixturePresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/string-enum-collection-bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{"data_class", "plain_class"} {
		r := request(input, "Dto")
		r.Options.Validation, r.Options.ModelStyle = true, style
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(style, err)
		}
		text := source(t, got, "HolderDto")
		for _, marker := range []string{"List<RequiredInner>,", "List<CodesInner?>? =", "Map<kotlin.String, LabelsValue>? =", "private fun openapiKtValidatePrimitive", "HolderDto.CodesInner", "HolderDto.LabelsValue"} {
			if !strings.Contains(text, marker) {
				t.Fatal(marker, text)
			}
		}
		patch := source(t, got, "PatchInputDto")
		for _, marker := range []string{"Patch<kotlin.collections.List<CodesInner>>", "Patch.Absent", "Map<kotlin.String, LabelsValue?>", "public val required: kotlin.collections.List<RequiredInner>?,"} {
			if !strings.Contains(patch, marker) {
				t.Fatal(marker, patch)
			}
		}
		// A valid runtime unknown remains an invalid default, including Patch.
		var tree map[string]any
		decoder := json.NewDecoder(bytes.NewReader(input))
		decoder.UseNumber()
		if err := decoder.Decode(&tree); err != nil {
			t.Fatal(err)
		}
		patchSchema := tree["paths"].(map[string]any)["/patch"].(map[string]any)["patch"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		patchSchema["properties"].(map[string]any)["codes"].(map[string]any)["default"] = []string{"future"}
		r.Document, err = json.Marshal(tree)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("Unknown default accepted")
		}
	}
}

func TestStringEnumCollectionBoundsDirectAdmission(t *testing.T) {
	for _, leaf := range []string{`{"enum":["one","x","toolong"],"minLength":2,"maxLength":4,"pattern":"^[a-z]+$"}`, `{"type":["string","null"],"enum":[null,"one","x"],"minLength":2,"maxLength":4,"pattern":"^[a-z]+$"}`} {
		for _, container := range []string{`{"type":"array","items":LEAF}`, `{"additionalProperties":LEAF}`} {
			for _, def := range []string{"", `,"default":EMPTY`, `,"default":VALUE`} {
				field := strings.ReplaceAll(container, "LEAF", leaf)
				d := strings.ReplaceAll(def, "EMPTY", "[]")
				d = strings.ReplaceAll(d, "VALUE", `["one"]`)
				if strings.Contains(container, "additionalProperties") {
					d = strings.ReplaceAll(def, "EMPTY", "{}")
					d = strings.ReplaceAll(d, "VALUE", `{"a/~":"one"}`)
				}
				field = strings.TrimSuffix(field, "}") + d + "}"
				r := request(fieldDoc(field), "Dto")
				r.Options.Validation = true
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(field, err)
				}
				text := source(t, got, "RecordDto")
				for _, marker := range []string{"public sealed class", "class Unrecognized", ".serialName", "value.indices.count", "containsMatchIn", "minLength", "maxLength"} {
					if !strings.Contains(text, marker) {
						t.Fatal(marker, text)
					}
				}
				if strings.Contains(text, "must be a declared enum") {
					t.Fatal("Closed runtime vocabulary", text)
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Nondeterministic", err)
				}
				r.Options.Validation = false
				if _, err := core.GenerateWithDiagnostics(r); err == nil {
					t.Fatal("Validation off dropped constraints")
				}
			}
		}
	}
}
