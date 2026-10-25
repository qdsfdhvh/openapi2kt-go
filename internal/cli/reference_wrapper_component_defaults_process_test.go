package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferenceWrapperComponentDefaultsNativeParity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			t.Run(string(style)+"/checked="+map[bool]string{false: "0", true: "1"}[checked], func(t *testing.T) {
				p := newProject(t)
				document := read(t, filepath.Join(repo, "tests/fixtures/http/reference-wrapper-component-defaults.json"))
				put(t, p.Input, document)
				args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
				result(t, invoke(t, p.Root, args...), 0)
				report, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: document, Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked}})
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range report.Artifacts.Files {
					if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
						t.Fatal("native/core chain factory mismatch", file.Path)
					}
				}
				for _, name := range []string{"WrappedObject", "WrappedNested", "WrappedRaw", "WrappedNilRaw", "WrappedNilObject", "NullableObject", "NullObject", "NullableCombined", "NullCombined", "NullableNested", "NullNested", "NullableMode", "NullMode", "NullableDay", "NullDay", "NullableBlob", "NullBlob", "NullableRaw", "NullRaw", "NullableOpen", "NullOpen"} {
					content := string(read(t, filepath.Join(p.Output, "org/example/catalog/models/"+name+"Dto.kt")))
					if !strings.Contains(content, "public fun `"+name+"Dto default`()") {
						t.Fatal("native chain factory missing", name)
					}
				}
				before := snapshot(t, p.Output)
				result(t, invoke(t, p.Root, args...), 0)
				result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 0)
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("chain no-op/check changed bytes or mtimes")
				}
				for _, tc := range []struct {
					name, component, key, code, pointer string
					value                               any
				}{
					{"nullable deep value", "NullableNested", "default", "E_SCHEMA_INVALID", "/components/schemas/NullableNested/default/0/a~1~0", []any{map[string]any{"a/~": false}}},
					{"nullable enum value", "NullableMode", "default", "E_SCHEMA_INVALID", "/components/schemas/NullableMode/default", "future"},
					{"narrowed wrapper", "NullObject", "nullable", "E_SCHEMA_INVALID", "/components/schemas/NullObject/default", false},
					{"deep value", "WrappedNested", "default", "E_SCHEMA_INVALID", "/components/schemas/WrappedNested/default/0/a~1~0", []any{map[string]any{"a/~": false}}},
					{"invalid middle default", "ObjectBridge", "default", "E_SCHEMA_INVALID", "/components/schemas/ObjectBridge/default", map[string]any{}},
					{"middle constraint", "ObjectBridge", "minLength", "E_SCHEMA_UNSUPPORTED", "/components/schemas/NullObject/default", 0},
				} {
					t.Run(tc.name, func(t *testing.T) {
						var bad map[string]any
						decoder := json.NewDecoder(bytes.NewReader(document))
						decoder.UseNumber()
						if err := decoder.Decode(&bad); err != nil {
							t.Fatal(err)
						}
						schemas := bad["components"].(map[string]any)["schemas"].(map[string]any)
						schemas[tc.component].(map[string]any)[tc.key] = tc.value
						encoded, err := json.Marshal(bad)
						if err != nil {
							t.Fatal(err)
						}
						put(t, p.Input, encoded)
						for _, command := range [][]string{args, append(append([]string{}, args...), "--check")} {
							failure := fails(t, invoke(t, p.Root, command...), 1, tc.code)
							if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
								details, _ := json.Marshal(failure.Diagnostics)
								t.Fatalf("invalid chain did not retain original Pointer/output: got=%s want=%s", details, tc.pointer)
							}
						}
					})
				}
			})
		}
	}
}
