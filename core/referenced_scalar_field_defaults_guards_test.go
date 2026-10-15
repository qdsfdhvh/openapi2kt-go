package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func scalarRefDoc(terminal, value, override string) []byte {
	// The nullable root path is already accepted with validation off. Nearest
	// internal false, followed by consumer override, exercises the real resolver.
	return schemas(`{"AHolder":{"properties":{"a/~":{"$ref":"#/components/schemas/Hops/properties/narrow","default":` + value + override + `}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Alias","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},"Alias":{"$ref":"#/components/schemas/ZValue"},"ZValue":` + terminal + `}`)
}

func TestReferencedScalarFieldDefaultsKindsChecksAndOriginalPointers(t *testing.T) {
	for _, tc := range []struct{ terminal, value, want string }{
		{`{"type":"string","nullable":true}`, `"a\"\\\n$"`, `= "a\"\\\n\$"`},
		{`{"type":"boolean","nullable":true}`, `false`, `= false`},
		{`{"type":"integer","nullable":true}`, `-2147483648`, `= (-2147483647 - 1)`},
		{`{"type":"integer","format":"int64","nullable":true}`, `-9223372036854775808`, `= (-9223372036854775807L - 1L)`},
		{`{"type":"number","format":"float","nullable":true}`, `1.000000001`, `= 1.0f`},
		{`{"type":"number","nullable":true}`, `1.25`, `= 1.25`},
	} {
		for _, checked := range []bool{false, true} {
			r := request(scalarRefDoc(tc.terminal, tc.value, ""), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(tc, checked, err)
			}
			text := source(t, got, "AHolderDto")
			if !strings.Contains(text, tc.want) || strings.Contains(text, "default tree") {
				t.Fatal(tc.want, text)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
		}
	}
	for _, tc := range []struct {
		terminal, value string
		checked         bool
	}{
		{`{"type":"integer","nullable":true}`, `1.0`, false},
		{`{"type":"integer","nullable":true}`, `-0`, false},
		{`{"type":"integer","nullable":true}`, `2147483648`, false},
		{`{"type":"integer","format":"int64","nullable":true}`, `9223372036854775808`, false},
		{`{"type":"number","format":"float","nullable":true}`, `1e300`, false},
		{`{"type":"boolean","nullable":true}`, `"true"`, false},
		{`{"type":"string","nullable":true}`, `true`, false},
		{`{"type":"integer","nullable":true}`, `null`, false},
		{`{"type":"string","nullable":true,"minLength":2}`, `"x"`, true},
		{`{"type":"string","nullable":true,"maxLength":1}`, `"😀a"`, true},
		{`{"type":"string","nullable":true,"pattern":"ab"}`, `"x"`, true},
		{`{"type":"integer","nullable":true,"minimum":1}`, `0`, true},
		{`{"type":"integer","format":"int64","nullable":true,"multipleOf":2}`, `3`, true},
		{`{"type":"number","format":"float","nullable":true,"exclusiveMinimum":1}`, `1.000000001`, true},
		{`{"type":"number","nullable":true,"maximum":2}`, `3`, true},
	} {
		r := request(scalarRefDoc(tc.terminal, tc.value, ""), "Dto")
		r.Options.Validation = tc.checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc, err)
		}
		d := err.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || *d.Pointer != "/components/schemas/AHolder/properties/a~1~0/default" {
			t.Fatal(tc, err)
		}
	}
	// Codepoints and substring matching retain the direct scalar contract.
	for _, tc := range []struct{ schema, value string }{
		{`{"type":"string","nullable":true,"minLength":1,"maxLength":1}`, `"😀"`},
		{`{"type":"string","nullable":true,"pattern":"ab"}`, `"zabz"`},
	} {
		r := request(scalarRefDoc(tc.schema, tc.value, ""), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReferencedScalarFieldDefaultsNoNewSourceRoles(t *testing.T) {
	for _, terminal := range []string{
		`{"type":"string","const":"a"}`,
		`{"type":"string","x-native-type":"kotlin.String"}`, `{"type":"string","x-kotlin-value-class":true}`,
		`{"properties":{"id":{"type":"integer"}}}`, `{"oneOf":[{"type":"string"},{"type":"null"}]}`,
		`{"type":"integer","nullable":true,"default":1}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(scalarRefDoc(terminal, "null", `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("source role", terminal, checked, err)
			}
		}
	}
	for _, body := range []string{
		`{"Value":{"type":"integer"},"Alias":{"$ref":"#/components/schemas/Value"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":2}}}}`,
		`{"Value":{"type":"integer"},"Root":{"$ref":"#/components/schemas/Value","default":2}}`,
		`{"Value":{"type":"integer"},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Value"}],"default":2}}}}`,
		`{"Holder":{"properties":{"target":{"type":"integer","default":1},"v":{"$ref":"#/components/schemas/Holder/properties/target","default":2}}}}`,
		`{"Value":{"type":"integer","nullable":true},"Alias":{"$ref":"#/components/schemas/Value","default":2},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":2}}}}`,
	} {
		if got, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto")); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("ungranted alias/default", body, err)
		}
	}
	// Schema errors run before null can skip value checks, with source pointers.
	for _, tc := range []struct{ schema, code, tail string }{
		{`{"type":"integer","nullable":true,"minimum":"bad"}`, "E_SCHEMA_INVALID", "minimum"},
		{`{"type":"number","nullable":true,"multipleOf":0.5}`, "E_SCHEMA_UNSUPPORTED", "multipleOf"},
	} {
		r := request(scalarRefDoc(tc.schema, "null", `,"nullable":true`), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/ZValue/"+tc.tail {
			t.Fatal(err)
		}
	}
}

func TestReferencedScalarFieldDefaultsPresenceAndDiscarded(t *testing.T) {
	for _, role := range []string{"optional", "required", "patch", "discarded"} {
		for _, valid := range []bool{false, true} {
			value := "0"
			if valid {
				value = "2"
			}
			field := `{"$ref":"#/components/schemas/Value","default":` + value + `}`
			holder := `{"properties":{"v":` + field + `}}`
			if role == "required" {
				holder = `{"required":["v"],"properties":{"v":` + field + `}}`
			}
			if role == "discarded" {
				holder = `{"allOf":[` + holder + `,{"properties":{"v":{"type":"string"}}}]}`
			}
			doc := schemas(`{"Value":{"type":"integer","nullable":true,"minimum":1},"Holder":` + holder + `}`)
			if role == "patch" {
				doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":{"type":"integer","nullable":true,"minimum":1}}},"paths":{"/p":{"patch":{"operationId":"patch","requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
			}
			r := request(doc, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			// Existing noDrop rejects discarded constrained sources even when valid.
			if !valid || role == "discarded" {
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal(role, valid, err)
				}
				continue
			}
			if err != nil {
				t.Fatal(role, err)
			}
			name := "HolderDto"
			if role == "patch" {
				name = "PatchInputDto"
			}
			text := source(t, got, name)
			if role == "required" && strings.Contains(text, "= 2") || role == "patch" && (!strings.Contains(text, "Patch.Absent") || strings.Contains(text, "= 2")) {
				t.Fatal(role, text)
			}
		}
	}
	// Unconstrained discarded own annotations still must be valid.
	for _, value := range []string{"1", "false"} {
		doc := schemas(`{"Value":{"type":"integer"},"Holder":{"allOf":[{"properties":{"v":{"$ref":"#/components/schemas/Value","default":` + value + `}}},{"properties":{"v":{"type":"string"}}}]}}`)
		_, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if (err == nil) != (value == "1") {
			t.Fatal(value, err)
		}
	}
}

func TestReferencedScalarFieldDefaultsAnnotationAndSlotBudgets(t *testing.T) {
	for _, size := range []int{32766, 32767} {
		for _, role := range []string{"optional", "required", "discarded"} {
			holder := `{"properties":{"v":{"$ref":"#/components/schemas/Value","default":"` + strings.Repeat("a", size) + `"}}}`
			if role == "required" {
				holder = strings.TrimSuffix(holder, "}") + `,"required":["v"]}`
			}
			if role == "discarded" {
				holder = `{"allOf":[` + holder + `,{"properties":{"v":{"type":"integer"}}}]}`
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":{"type":"string"},"Holder":`+holder+`}`), "Dto"))
			if size == 32766 && err != nil || size == 32767 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
				t.Fatal(size, role, err)
			}
		}
	}
	for _, n := range []int{240, 241} {
		props := []string{}
		for i := 0; i < n; i++ {
			props = append(props, fmt.Sprintf(`"v%d":{"$ref":"#/components/schemas/Value","default":2}`, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":{"type":"integer","format":"int64"},"Holder":{"properties":{`+strings.Join(props, ",")+`}}}`), "Dto"))
		if n == 240 && err != nil || n == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, err)
		}
	}
}
