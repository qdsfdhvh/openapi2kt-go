package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMutualRecursiveJSONNativeLifecycle(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests/fixtures/http/mutual-recursive-json.json"))
	var d map[string]any
	if err := json.Unmarshal(input, &d); err != nil {
		t.Fatal(err)
	}
	schemas(d)["ZAny"], schemas(d)["ZOther"] = true, true
	control, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p.Input, control)
	result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	put(t, p.Input, input)
	got := result(t, p.generate(t, "--model-name-suffix=Dto"), 0)
	if len(got.Warnings) != 2 {
		t.Fatal(got)
	}
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned file"))
	before := snapshot(t, p.Output)
	for _, extra := range [][]string{nil, {"--check"}, {"--deny-warnings", "--allow-warning=W_RECURSIVE_JSON_PROJECTION", "--check"}} {
		result(t, p.generate(t, append([]string{"--model-name-suffix=Dto"}, extra...)...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("check/noop changed output")
		}
	}
	warningDenied(t, p.generate(t, "--model-name-suffix=Dto", "--deny-warnings"))
	p.change(t, func(d map[string]any) {
		schemas(d)["ZOther"].(map[string]any)["anyOf"].([]any)[3].(map[string]any)["minimum"] = 1
	})
	for _, extra := range [][]string{nil, {"--check"}} {
		fails(t, p.generate(t, extra...), 1, "E_SCHEMA_UNSUPPORTED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("failed graph changed output")
		}
	}
}
