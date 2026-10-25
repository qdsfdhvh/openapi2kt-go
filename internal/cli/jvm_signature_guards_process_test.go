package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCertainJVMSignatureNativeDiagnosticsAndOutputPreservation(t *testing.T) {
	valid := read(t, filepath.Join(repo, "tests/fixtures/http/wide-deep-validation.json"))
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		p := newProject(t)
		put(t, p.Input, valid)
		args := objectDefaultReferencedPropertyArgs(p.Input, p.Output, true, style)
		result(t, invoke(t, p.Root, args...), 0)
		before := snapshot(t, p.Output)
		for _, scope := range []string{"component", "request"} {
			var d map[string]any
			decoder := json.NewDecoder(bytes.NewReader(valid))
			decoder.UseNumber()
			if e := decoder.Decode(&d); e != nil {
				t.Fatal(e)
			}
			var leaf any = map[string]any{"type": "integer", "minimum": 1}
			for range 16 {
				leaf = map[string]any{"type": "array", "items": leaf}
			}
			fields := map[string]any{}
			for i := range 240 {
				fields[fmt.Sprintf("p%d", i)] = leaf
			}
			schema := map[string]any{"type": "object", "properties": fields}
			pointer := "/components/schemas/Overflow"
			if scope == "component" {
				d["components"].(map[string]any)["schemas"].(map[string]any)["Overflow"] = schema
			} else {
				body := d["paths"].(map[string]any)["/send"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)
				body["content"].(map[string]any)["application/json"].(map[string]any)["schema"] = schema
				pointer = "/paths/~1send/post/requestBody/content/application~1json/schema"
			}
			data, e := json.Marshal(d)
			if e != nil {
				t.Fatal(e)
			}
			put(t, p.Input, data)
			for _, cmd := range [][]string{args, append(append([]string{}, args...), "--check"), {"validate", "--input", p.Input, "--validation", "--diagnostic-format=json"}} {
				failure := fails(t, invoke(t, p.Root, cmd...), 1, "E_MODEL_LIMIT")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != pointer || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("signature error lost sourcePointer or old output", scope, failure)
				}
			}
		}
	}
}
