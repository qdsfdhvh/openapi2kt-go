package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func implicitStringEnumInput(t *testing.T, input []byte) []byte {
	t.Helper()
	var root any
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		t.Fatal(err)
	}
	var visit func(any)
	changed := 0
	visit = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			values, _ := v["enum"].([]any)
			null, text := false, false
			for _, value := range values {
				null = null || value == nil
				_, isText := value.(string)
				text = text || isText
			}
			if null && text {
				delete(v, "type")
				delete(v, "nullable")
				changed++
			}
			for key, child := range v {
				if key != "enum" && key != "default" && key != "const" {
					visit(child)
				}
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(root)
	if changed == 0 {
		t.Fatal("missing nullable String enum controls")
	}
	out, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func implicitStringEnumGraduation(t *testing.T, input []byte, validation bool) {
	t.Helper()
	r := request(input, "Dto")
	r.Options.Validation = validation
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal("original implicit vocabulary did not graduate", err)
	}
	r.Document = []byte(strings.ReplaceAll(string(input), `"enum":`, `"nullable":true,"enum":`))
	control, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, control) {
		t.Fatal("original input changed explicit nullable projection", err)
	}
	r.Document = []byte(strings.ReplaceAll(string(input), `"enum":`, `"nullable":false,"enum":`))
	failed, err := core.GenerateWithDiagnostics(r)
	if err == nil || !reflect.DeepEqual(failed, core.GenerationReport{}) {
		t.Fatal("explicit false lost its original refusal", err)
	}
}

func TestImplicitStringEnumNullKeepsExplicitNullableProjection(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, optimized := range []bool{false, true} {
				body := `{"State":{"enum":[null,"ready","done","null"]},"Alias":{"$ref":"#/components/schemas/State"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Owner":{"properties":{"state":{"enum":["open",null,"closed"]}}},"Nested":{"$ref":"#/components/schemas/Owner/properties/state"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Nested"}},"Map":{"additionalProperties":{"enum":["open",null,"closed"]}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias"},"narrow":{"$ref":"#/components/schemas/False"},"rows":{"$ref":"#/components/schemas/Rows"},"map":{"$ref":"#/components/schemas/Map"},"defaulted":{"enum":["ready",null,"done","null"],"default":"ready"},"clear":{"enum":[null,"on"],"default":null}},"required":["value","narrow"]}}`
				r := request(schemas(body), "Dto")
				r.Options.Validation, r.Options.ModelStyle = validation, style
				r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = optimized, optimized, optimized
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(validation, style, optimized, err)
				}
				r.Document = schemas(strings.ReplaceAll(body, `"enum":`, `"nullable":true,"enum":`))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, control) {
					t.Fatal("inference changed the explicit nullable API, codec, defaults or diagnostics", validation, style, optimized, err)
				}
			}
		}
	}
}

func TestImplicitStringEnumNullKeepsExistingContextsAndOptions(t *testing.T) {
	for _, fixture := range []string{
		"string-enum-null-members", "referenced-string-enum-null-member-defaults", "nested-string-enum-collection-defaults",
		"dto-allof-unique", "dto-collection-reference-leaves-unique", "dto-string-collection-reference-unique",
		"nested-object-defaults", "string-enum-collection-bounds", "string-enum-collection-defaults",
		"string-enum-unique", "tuple-enum-references",
	} {
		explicit, err := os.ReadFile("../tests/fixtures/http/" + fixture + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var tree any
		decoder := json.NewDecoder(bytes.NewReader(explicit))
		decoder.UseNumber()
		if err := decoder.Decode(&tree); err != nil {
			t.Fatal(err)
		}
		explicit, _ = json.Marshal(tree)
		implicit := implicitStringEnumInput(t, explicit)
		accepted := 0
		for _, validation := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, optimized := range []bool{false, true} {
					r := request(implicit, "Dto")
					r.Options.Validation, r.Options.ModelStyle = validation, style
					r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = optimized, optimized, optimized
					got, err := core.GenerateWithDiagnostics(r)
					r.Document = explicit
					control, controlErr := core.GenerateWithDiagnostics(r)
					if !reflect.DeepEqual(err, controlErr) || !reflect.DeepEqual(got, control) {
						t.Fatal("implicit source changed complete explicit context report", fixture, validation, style, optimized, err, controlErr)
					}
					if err != nil {
						continue
					}
					accepted++
					r.Document = implicit
					replay, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, replay) {
						t.Fatal("implicit generation is not deterministic", fixture, err)
					}
				}
			}
		}
		if accepted == 0 {
			t.Fatal("no successful context control", fixture)
		}
	}
}

func TestImplicitStringEnumNullKeepsRejectedRolesAndRawLimits(t *testing.T) {
	for _, tc := range []struct{ leaf, tail string }{
		{`{"enum":["a",null,null]}`, "enum/2"},
		{`{"enum":[null,"a","a"]}`, "enum/2"},
		{`{"enum":[null,"a",1]}`, "enum/0"},
		{`{"enum":["a",null],"nullable":false}`, "enum/1"},
		{`{"enum":["a",null],"type":"string"}`, "enum/1"},
		{`{"enum":["a",null],"default":"future"}`, "default"},
		{`{"enum":["a",null],"nullable":"bad"}`, "nullable"},
		{`{"enum":["a",null],"format":"uuid"}`, "enum/1"},
		{`{"enum":["a",null],"x-native-type":"bigint"}`, "enum/1"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.leaf), "Dto")
			r.Options.Validation = validation
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !strings.HasSuffix(*err.(core.Diagnostics)[0].Pointer, "/"+tc.tail) || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, err)
			}
		}
	}
	for _, count := range []int{239, 240} {
		values := []string{"null"}
		for i := range count {
			values = append(values, fmt.Sprintf(`"v%d"`, i))
		}
		r := request(schemas(`{"State":{"enum":[`+strings.Join(values, ",")+`]}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (count == 239) || err != nil && (err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(count, err)
		}
	}
}

func TestImplicitStringEnumNullKeepsSourceHeightsAndMappedHierarchyTag(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, named := range []bool{false, true} {
			for _, typed := range []bool{false, true} {
				limit := 47
				if typed {
					limit = 45
				}
				for _, count := range []int{limit, limit + 1} {
					fields := []string{}
					for i := range count {
						target := "Owner/properties/state"
						if named {
							target = "Named"
						}
						if i > 0 {
							target = fmt.Sprint("Hops/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					body := `"Owner":{"properties":{"state":{"enum":["on",null,"off"]}}},"Named":{"enum":["on",null,"off"]},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(count-1) + `","nullable":false}`
					if typed {
						body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["value"]}`
					}
					r := request(schemas(`{`+body+`}`), "Dto")
					r.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(r)
					if count == limit {
						if err != nil {
							t.Fatal(checked, named, typed, count, err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal(checked, named, typed, count, err)
					}
				}
			}
		}
	}
	body := `{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"fixed":"Leaf"}}},"Leaf":{"properties":{"kind":{"enum":["fixed",null]}}}}`
	r := request(schemas(body), "Dto")
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Document = schemas(strings.ReplaceAll(body, `"enum":`, `"nullable":true,"enum":`))
	control, err := core.GenerateWithDiagnostics(r)
	leaf := source(t, got, "LeafDto")
	if err != nil || !reflect.DeepEqual(got, control) || strings.Contains(leaf, "RootDtoKind?") || !strings.Contains(leaf, "public override val kind: org.example.models.models.RootDtoKind,") {
		t.Fatal("inferred tag changed actual non-null mapped hierarchy body", err)
	}
}

func TestImplicitStringEnumNullKeepsIndependentRootFactories(t *testing.T) {
	for _, value := range []string{`null`, `"on"`, `"future"`} {
		body := `{"State":{"enum":["on",null,"off"],"default":` + value + `},"Alias":{"$ref":"#/components/schemas/State","default":null},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/State"}}}}`
		for _, checked := range []bool{false, true} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			r.Document = schemas(strings.ReplaceAll(body, `"enum":`, `"nullable":true,"enum":`))
			control, controlErr := core.GenerateWithDiagnostics(r)
			if !reflect.DeepEqual(err, controlErr) || !reflect.DeepEqual(got, control) {
				t.Fatal("inferred root factory changed explicit nullable control", value, checked, err, controlErr)
			}
			if value == `"future"` {
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("unknown own factory default admitted")
				}
			} else if err != nil {
				t.Fatal("existing nullable factory refused", value, checked, err)
			}
		}
	}
	// Retain the original narrowed factory input as an acceptance contract.
	// Source null members do not widen a non-null alias or its own annotation.
	body := `{"State":{"enum":["on",null,"off"],"default":null},"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":"on"}}`
	r := request(schemas(body), "Dto")
	got, err := core.GenerateWithDiagnostics(r)
	r.Document = schemas(strings.ReplaceAll(body, `"enum":`, `"nullable":true,"enum":`))
	control, controlErr := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(err, controlErr) || !reflect.DeepEqual(got, control) {
		t.Fatal("narrowed factory role changed its original explicit control", err, controlErr)
	}
	r.Document = schemas(strings.ReplaceAll(body, `"default":"on"`, `"default":null`))
	if failed, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(failed, core.GenerationReport{}) || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Narrow/default" {
		t.Fatal("source null widened narrowed alias annotation", err)
	}
}
