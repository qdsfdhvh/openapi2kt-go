package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHTTPReportsCheckAndFailuresPreserveOutput(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		files   int
	}{{"basic.json", 6}, {"collections.json", 3}, {"bodies.json", 3}, {"references.json", 3}, {"operation-models.json", 13}, {"operation-collisions.json", 6}, {"request-presence.json", 10}, {"request-presence-collisions.json", 5}, {"request-unions.json", 6}, {"deep-object.json", 5}, {"deep-bindings.json", 6}, {"errors.json", 6}, {"projections.json", 4}, {"response-kinds.json", 2}, {"structural-values.json", 3}, {"nullable-request-unions.json", 4}, {"json-kind-request-unions.json", 10}, {"operation-scalar-unions.json", 3}, {"empty-response-unions.json", 4}, {"request-field-unions.json", 5}, {"parameter-values.json", 6}, {"special-scalars.json", 7}, {"special-scalar-values.json", 4}, {"native-enum-values.json", 7}, {"sse.json", 12}, {"value-classes.json", 16}, {"value-constraints.json", 12}} {
		t.Run(tc.fixture, func(t *testing.T) { testHTTPProcess(t, tc.fixture, tc.files) })
	}
}

func TestConstraintFailureAndOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "value-constraints.json"))
	put(t, p.Input, input)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	bad := bytes.Replace(input, []byte(`"minLength":1,"maxLength":2`), []byte(`"minLength":3,"maxLength":2`), 1)
	if bytes.Equal(bad, input) {
		t.Fatal("injection unchanged")
	}
	put(t, p.Input, bad)
	for _, args := range [][]string{nil, {"--check"}, {"--exclude-api", "ChecksApi"}} {
		r := result(t, p.generate(t, args...), 1)
		if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid bounds changed output")
		}
	}
	put(t, p.Input, input)
	r := result(t, p.generate(t, "--exclude-api", "ChecksApi"), 0)
	if len(r.Files) != 8 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	if !bytes.Contains(read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "Name.kt")), []byte("require(length <= 2)")) {
		t.Fatal("excluded API dropped wrapper checks")
	}
	result(t, p.generate(t, "--exclude-api", "ChecksApi", "--check"), 0)
}

func TestValueClassFailureAndExclusionOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "value-classes.json"))
	put(t, p.Input, input)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	bad := bytes.Replace(input, []byte(`"x-kotlin-value-class":true`), []byte(`"x-kotlin-value-class":"yes"`), 1)
	if bytes.Equal(bad, input) {
		t.Fatal("injection unchanged")
	}
	put(t, p.Input, bad)
	for _, args := range [][]string{nil, {"--check"}, {"--exclude-api", "TypesApi"}} {
		r := result(t, p.generate(t, args...), 1)
		if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid wrapper flag changed output")
		}
	}
	put(t, p.Input, input)
	r := result(t, p.generate(t, "--exclude-api", "TypesApi"), 0)
	if len(r.Files) != 11 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	for _, pattern := range []string{"api/models/*.kt", "api/TypesApi.kt", "models/Patch.kt"} {
		matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", pattern))
		if err != nil || len(matches) != 0 {
			t.Fatal("stale wrapper API helper", matches, err)
		}
	}
	if !bytes.Contains(read(t, filepath.Join(p.Output, "org", "example", "catalog", "models", "UserId.kt")), []byte("public value class UserId(")) {
		t.Fatal("excluded API removed component wrapper")
	}
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--exclude-api", "TypesApi", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("excluded check mutated wrapper output")
	}
}

func TestSSEFailureAndExclusionOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "sse.json"))
	put(t, p.Input, input)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	bad := bytes.Replace(input, []byte(`"data":{"type":"integer"}`), []byte(`"data":{"type":"integer","minimum":0}`), 1)
	if bytes.Equal(bad, input) {
		t.Fatal("failure injection did not change input")
	}
	put(t, p.Input, bad)
	for _, args := range [][]string{nil, {"--check"}, {"--exclude-api", "StreamApi"}} {
		r := result(t, p.generate(t, args...), 1)
		if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_UNSUPPORTED" {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid stream payload changed output")
		}
	}
	put(t, p.Input, input)
	r := result(t, p.generate(t, "--exclude-api", "StreamApi"), 0)
	if len(r.Files) != 3 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	for _, pattern := range []string{"api/models/*.kt", "api/StreamApi.kt", "models/Patch.kt"} {
		matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", pattern))
		if err != nil || len(matches) != 0 {
			t.Fatal("stale stream model/runtime", matches, err)
		}
	}
	before = snapshot(t, p.Output)
	result(t, p.generate(t, "--exclude-api", "StreamApi", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("excluded stream check changed output")
	}
}

func TestNativeEnumFailureAndExclusionOwnership(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "native-enum-values.json"))
	put(t, p.Input, input)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	put(t, p.Input, bytes.ReplaceAll(input, []byte(`"x-native-type":"bigint"`), []byte(`"x-native-type":"org.example.Bad<Type>"`)))
	for _, args := range [][]string{nil, {"--check"}, {"--exclude-api", "DefaultApi"}} {
		r := result(t, p.generate(t, args...), 1)
		if len(r.Files) != 0 || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "E_SCHEMA_INVALID" {
			t.Fatal(r)
		}
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid native metadata changed output")
		}
	}
	put(t, p.Input, input)
	r := result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	if len(r.Files) != 3 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	for _, pattern := range []string{"api/models/*.kt", "models/Patch.kt", "models/LongAsStringSerializer.kt"} {
		matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", pattern))
		if err != nil || len(matches) != 0 {
			t.Fatal("unneeded runtime/local model", matches, err)
		}
	}
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
}

func TestHTTPRequestKindExclusionOwnership(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "json-kind-request-unions.json")))
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	excluded := result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	if len(excluded.Files) != 3 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(excluded)
	}
	if matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", "api", "models", "*.kt")); err != nil || len(matches) != 0 {
		t.Fatal("stale union files were not removed", matches, err)
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("excluded check changed output")
	}
}

func TestHTTPEmptyResponseExclusionOwnership(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "empty-response-unions.json")))
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("unowned file removed")
	}
	if matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", "api", "models", "*.kt")); err != nil || len(matches) != 0 {
		t.Fatal("stale borrowed response remained", matches, err)
	}
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
}

func TestHTTPRequestFieldExclusionOwnership(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "request-field-unions.json")))
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	r := result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	if len(r.Files) != 2 || string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal(r)
	}
	for _, pattern := range []string{"api/models/*.kt", "models/Patch.kt"} {
		matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", pattern))
		if err != nil || len(matches) != 0 {
			t.Fatal("stale request model/runtime remained", matches, err)
		}
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("excluded check changed output")
	}
}

func TestSpecialScalarHelperExclusionOwnership(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","paths":{"/only":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"big":{"type":["string","null"],"x-native-type":"bigint"}}}}}},"responses":{"204":{}}}}}}`))
	generated := result(t, p.generate(t), 0)
	if len(generated.Files) != 5 {
		t.Fatal(generated)
	}
	put(t, filepath.Join(p.Output, "keep.txt"), []byte("unowned"))
	before := snapshot(t, p.Output)
	put(t, p.Input, []byte(`{"openapi":"3.1.0","paths":{"/only":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"bad":{"type":"string","x-native-type":"bad; code"}}}}}},"responses":{"204":{}}}}}}`))
	fails(t, p.generate(t), 1, "E_SCHEMA_INVALID")
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("native failure altered owned/unowned output")
	}
	put(t, p.Input, []byte(`{"openapi":"3.1.0","paths":{"/only":{"get":{"responses":{"204":{}}}}}}`))
	result(t, p.generate(t, "--exclude-api", "DefaultApi"), 0)
	for _, pattern := range []string{"api/models/*.kt", "models/Patch.kt", "models/LongAsStringSerializer.kt"} {
		matches, err := filepath.Glob(filepath.Join(p.Output, "org", "example", "catalog", pattern))
		if err != nil || len(matches) != 0 {
			t.Fatal("unused owned scalar runtime remained", matches, err)
		}
	}
	if string(read(t, filepath.Join(p.Output, "keep.txt"))) != "unowned" {
		t.Fatal("unowned file removed")
	}
	result(t, p.generate(t, "--exclude-api", "DefaultApi", "--check"), 0)
}

func testHTTPProcess(t *testing.T, fixture string, files int) {
	t.Helper()
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", fixture)))
	generated := result(t, p.generate(t), 0)
	if len(generated.Files) != files {
		t.Fatal(generated)
	}
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	result(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
	result(t, p.generate(t), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("HTTP check/noop changed output")
	}
	for _, tc := range []struct{ document, code string }{
		{`{"openapi":"3.1.0","paths":{"/ping":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"payload":{"anyOf":[{"type":"string","pattern":"x"},{"type":"boolean"}]}}}}}},"responses":{"204":{}}}}}}`, "E_SCHEMA_UNSUPPORTED"},
		{`{"openapi":"3.1.0","paths":{"/ping":{"get":{"parameters":[{"$ref":"#/pool/0"}],"responses":{"204":{}}}}},"pool":[{"$ref":"#/pool/0"}]}`, "E_REF_CYCLE"},
		{`{"openapi":"3.1.0","paths":{"/ping":{"get":{"responses":{"200":{"$ref":"#/missing"}}}}}}`, "E_REF_MISSING"},
	} {
		put(t, p.Input, []byte(tc.document))
		for _, flags := range [][]string{nil, {"--check"}} {
			fails(t, p.generate(t, flags...), 1, tc.code)
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("HTTP failure changed previous output")
			}
		}
	}
}
