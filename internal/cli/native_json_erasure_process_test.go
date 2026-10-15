package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNativeJSONMethodErasureNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "native-json-erasure.json")))
	for _, validation := range []bool{false, true} {
		args := []string{"--model-name-suffix", "Dto"}
		if validation {
			args = append(args, "--validation")
		}
		result(t, p.generate(t, args...), 0)
		api := string(read(t, filepath.Join(p.Output, "org", "example", "catalog", "api", "DefaultApi.kt")))
		for _, name := range []string{"send", "send2", "send3", "putPut", "putPut2", "putPut3"} {
			if strings.Count(api, "public suspend fun "+name+"(") != 1 {
				t.Fatal("uncompilable native allocation", name, api)
			}
		}
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(args, "--check")...), 0)
		result(t, p.generate(t, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("check/no-op")
		}
	}
	put(t, filepath.Join(p.Output, "custom.txt"), []byte("synthetic handwritten file"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) { schemas(doc)["NativeElement"].(map[string]any)["x-native-type"] = true })
	for _, args := range [][]string{{"--model-name-suffix", "Dto"}, {"--model-name-suffix", "Dto", "--check"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/NativeElement/x-native-type" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("source/output preservation", r)
		}
	}
}
