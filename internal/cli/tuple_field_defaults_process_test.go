package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTupleFieldDefaultsNativeReadOnlyAndOriginalMemberFailurePreservation(t *testing.T) {
	for _, validation := range []bool{false, true} {
		p := newProject(t)
		original := read(t, filepath.Join(repo, "tests/fixtures/http/tuple-field-defaults.json"))
		flags := []string{}
		if validation {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, original)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Array check changed bytes/mtime")
		}
		for _, tc := range []struct{ old, bad, pointer string }{
			{"-9223372036854775808", "-9223372036854775809", "/components/schemas/Holder/properties/longs/default/0"},
			{"2024-02-29", "2023-02-29", "/components/schemas/Holder/properties/dates/default/0"},
			{"2024-02-29T12:34:56.123456789+02:30", "2024-02-29T12:34:56.1234567891Z", "/components/schemas/Holder/properties/times/default/0"},
		} {
			changed := bytes.Replace(original, []byte(tc.old), []byte(tc.bad), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("Missing original array member", tc.old)
			}
			put(t, p.Input, changed)
			failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("Bad original member changed owned outputs", failure)
			}
		}
		for _, kind := range []string{"required", "Patch", "prefix"} {
			var d map[string]any
			decoder := json.NewDecoder(bytes.NewReader(original))
			decoder.UseNumber()
			if err := decoder.Decode(&d); err != nil {
				t.Fatal(err)
			}
			properties := d["components"].(map[string]any)["schemas"].(map[string]any)["Holder"].(map[string]any)["properties"].(map[string]any)
			code := "E_SCHEMA_INVALID"
			pointer := "/components/schemas/Holder/properties/required/default/0"
			switch kind {
			case "required":
				properties["required"].(map[string]any)["default"] = []any{nil}
			case "Patch":
				body := d["paths"].(map[string]any)["/patch"].(map[string]any)["patch"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
				body["properties"].(map[string]any)["values"].(map[string]any)["default"] = []any{"bad"}
				pointer = "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/values/default/0"
			case "prefix":
				properties["empty"].(map[string]any)["prefixItems"].([]any)[0].(map[string]any)["format"] = "unsupported"
				pointer = "/components/schemas/Holder/properties/empty/prefixItems/0"
				code = "E_SCHEMA_UNSUPPORTED"
			}
			bad, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p.Input, bad)
			apply := append([]string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", "org.example.catalog", "--diagnostic-format=json"}, flags...)
			validate := append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)
			for _, command := range [][]string{apply, append(append([]string{}, apply...), "--check"), validate} {
				failure := fails(t, invoke(t, p.Root, command...), 1, code)
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("tuple default required/Patch/discarded-source validation lost old output", kind, failure.Diagnostics[0].Pointer, failure)
				}
			}
		}
		put(t, p.Input, original)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Restored array check mutated existing state")
		}
	}
}
