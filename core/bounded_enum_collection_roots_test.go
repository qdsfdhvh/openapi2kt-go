package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func boundedEnumRootsGenerate(t *testing.T, doc []byte) {
	t.Helper()
	rq := request(doc, "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal(err)
	}
}

func TestBoundedEnumCollectionRootsAndIndependentDefaults(t *testing.T) {
	leaf := `{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}`
	for _, tc := range []struct{ shape, value, other string }{
		{`{"type":"array","items":LEAF}`, `["one"]`, `["two"]`},
		{`{"additionalProperties":LEAF}`, `{"a":"one"}`, `{"b":"two"}`},
		{`{"type":"array","items":{"additionalProperties":LEAF}}`, `[{"a":"one"}]`, `[{"b":"two"}]`},
		{`{"additionalProperties":{"type":"array","items":LEAF}}`, `{"a":["one"]}`, `{"b":["two"]}`},
	} {
		for _, targetDefault := range []bool{false, true} {
			shape := strings.Replace(tc.shape, "LEAF", leaf, 1)
			if targetDefault {
				shape = strings.TrimSuffix(shape, "}") + `,"default":` + tc.value + `}`
			}
			body := `{"Tree":` + shape + `,"Alias":{"$ref":"#/components/schemas/Tree","default":` + tc.other + `},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":` + tc.value + `},"unannotated":{"$ref":"#/components/schemas/Tree"}},"default":{"value":` + tc.other + `}}}`
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("bounded root/alias/consuming annotation rejected", targetDefault, err, *err.(core.Diagnostics)[0].Pointer)
			}
			if !strings.Contains(source(t, got, "TreeDto"), "public typealias") || !strings.Contains(source(t, got, "UseDto"), "serialName.let") || !strings.Contains(source(t, got, "AliasDto"), "default") {
				t.Fatal("bounded root lost actual alias/check/factory")
			}
			if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("bounded collection root is nondeterministic", err)
			}
			rq.Options.Validation = false
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("bounded collection root bypassed validation gate", err)
			}
		}
	}
}

func TestBoundedEnumCollectionRootsFormerGuards(t *testing.T) {
	tree := `{"type":"array","items":{"additionalProperties":{"enum":["one"],"minLength":2}}}`
	for _, body := range []string{
		`{"Tree":` + tree + `}`,
		`{"Tree":` + tree + `,"Use":{"properties":{"value":{"$ref":"#/components/schemas/Tree"}}}}`,
		`{"Root":{"type":"array","items":{"enum":["one","x"],"minLength":2},"default":["one"]}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original bounded root refusal did not graduate", body, err)
		}
	}
}

func TestBoundedEnumCollectionRootsFormerFactoryGuards(t *testing.T) {
	for _, body := range []string{
		`{"Statuses":{"type":"array","items":{"type":"string","enum":["on"]},"default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Statuses","default":[]}}}}`,
		`{"Statuses":{"type":"array","items":{"type":"string","enum":["on"]}},"Alias":{"$ref":"#/components/schemas/Statuses","default":[]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":[]}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("original independent collection factory refusal did not graduate", checked, body, err)
			}
		}
	}
}

func TestBoundedEnumCollectionRootsFrozenCheckIdentity(t *testing.T) {
	shape := `{"properties":{"value":{"$ref":"#/components/schemas/Tree","default":[{"b":"two","a":"one"}]}}}`
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}`
	}
	doc := `{"openapi":"3.1.0","components":{"schemas":{"Tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2,"pattern":"^[a-z]+$"}}}}},"paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`
	rq := request([]byte(doc), "Dto")
	rq.Options.Validation, rq.Options.HoistSharedShapes, rq.Options.LockHoistedNames = true, true, true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(got.Artifacts.NameLock) == 0 {
		t.Fatal("root reference lost nonempty name lock", err)
	}
	rq.PreviousNameLock = got.Artifacts.NameLock
	rq.Document = []byte(strings.ReplaceAll(doc, `"b":"two","a":"one"`, `"a":"one","b":"two"`))
	canonical, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(got, canonical) {
		t.Fatal("root reference changed canonical default identity", err)
	}
	for _, change := range []struct{ from, to string }{
		{`"minLength":2`, `"minLength":3`},
		{`"pattern":"^[a-z]+$"`, `"pattern":"[a-z]"`},
		{`"b":"two","a":"one"`, `"b":"one","a":"two"`},
	} {
		rq.Document = []byte(strings.ReplaceAll(doc, change.from, change.to))
		changed, err := core.GenerateWithDiagnostics(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
			t.Fatal("root ref reused changed frozen check/default", change, err)
		}
	}
}
