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

func TestMergePrimitiveUnionsNativeLifecycle(t *testing.T) {
	input := read(t, filepath.Join(repo, "tests/fixtures/http/merge-primitive-unions.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			p := newProject(t)
			put(t, p.Input, input)
			original := objectDefaultReferencedPropertyArgs(p.Input, p.Output, checked, style)
			original = append(original, "--hoist-shared-shapes", "--lock-hoisted-names")
			result(t, invoke(t, p.Root, original...), 0)
			lock := read(t, filepath.Join(p.Output, ".openapi-kt-names.json"))
			args := append(append([]string{}, original...), "--merge-identical-models")
			result(t, invoke(t, p.Root, append(append([]string{}, args...), "--check")...), 3)
			result(t, invoke(t, p.Root, args...), 0)
			if !bytes.Equal(lock, read(t, filepath.Join(p.Output, ".openapi-kt-names.json"))) {
				t.Fatal("union merge changed pre-merge naming lock")
			}
			options := core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto", ModelStyle: style, Validation: checked, HoistSharedShapes: true, LockHoistedNames: true, MergeIdenticalModels: true}
			expected, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: input, Options: options, PreviousNameLock: lock})
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range expected.Artifacts.Files {
				if string(read(t, filepath.Join(p.Output, file.Path))) != file.Content {
					t.Fatal("native/core alias group parity", file.Path)
				}
			}
			code := string(read(t, filepath.Join(p.Output, "org/example/catalog/api/DefaultApi.kt")))
			if !strings.Contains(code, "public suspend fun submit3(") || strings.Count(code, "public suspend fun submit(") != 1 || !strings.Contains(code, "public suspend fun submit2(") {
				t.Fatal("canonical union overload allocation lost")
			}
			before := snapshot(t, p.Output)
			validate := append([]string{}, args...)
			validate[0] = "validate"
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
				result(t, invoke(t, p.Root, command...), 0)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("union alias replay changed bytes/attributes")
			}
			var bad map[string]any
			_ = json.Unmarshal(input, &bad)
			bad["components"].(map[string]any)["schemas"].(map[string]any)["Z"].(map[string]any)["oneOf"].([]any)[0].(map[string]any)["nullable"] = "invalid"
			data, _ := json.Marshal(bad)
			put(t, p.Input, data)
			for _, command := range [][]string{args, append(append([]string{}, args...), "--check"), validate} {
				fails(t, invoke(t, p.Root, command...), 1, "E_SCHEMA_INVALID")
				if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("union merge failure lost old output")
				}
			}
			put(t, p.Input, input)
			result(t, invoke(t, p.Root, append(append([]string{}, original...), "--check")...), 3)
			result(t, invoke(t, p.Root, original...), 0)
			if !bytes.Equal(lock, read(t, filepath.Join(p.Output, ".openapi-kt-names.json"))) {
				t.Fatal("disabling union merge changed hoist lock")
			}
		}
	}
}
