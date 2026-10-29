package core_test

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"openapi-kt/core"
)

func nativeScalarDefaultDoc(leaf, value string) []byte {
	return schemas(`{"Code":` + leaf + `,"Holder":{"properties":{"a/~":{"$ref":"#/components/schemas/Code","default":` + value + `}}}}`)
}

func TestReferencedNativeEnumScalarFieldDefaults(t *testing.T) {
	for _, tc := range []struct{ leaf, value, scalar, literal string }{
		{`{"type":"integer","format":"int32","enum":[-2147483648,1,2]}`, `-2147483648`, "Int", "(-2147483647 - 1)"},
		{`{"type":"integer","format":"int64","enum":[-9223372036854775808,1,2]}`, `-9223372036854775808`, "Long", "(-9223372036854775807L - 1L)"},
		{`{"type":"number","format":"float","enum":[1,1.5]}`, `1.5`, "Float", "1.5f"},
		{`{"type":"number","format":"double","enum":[1.25,2]}`, `1.25`, "Double", "1.25"},
		{`{"type":"boolean","enum":[true,false]}`, `false`, "Boolean", "false"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				input := nativeScalarDefaultDoc(tc.leaf, tc.value)
				r := request(bytes.Replace(input, []byte(`,"default":`+tc.value), nil, 1), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal("no-default control", tc, checked, style, err)
				}
				if !strings.Contains(source(t, baseline, "HolderDto"), "kotlin."+tc.scalar+"? = null") {
					t.Fatal("baseline scalar backing")
				}
				r.Document = input
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(tc, checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "kotlin."+tc.scalar+"? = "+tc.literal) || strings.Contains(text, "default tree") {
					t.Fatal("literal/ABI", text)
				}
				if source(t, got, "CodeDto") != source(t, baseline, "CodeDto") || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("declaration/warning drift")
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("determinism", err)
				}
			}
		}
	}
}

func TestReferencedNativeEnumScalarFieldDefaultsValuesAndSourceRoles(t *testing.T) {
	at := "/components/schemas/Holder/properties/a~1~0/default"
	for _, checked := range []bool{false, true} {
		for _, value := range []string{`9`, `true`, `"1"`, `1.0`, `1e0`, `null`, `[]`, `{}`} {
			nativeEnumDefaultFailure(t, nativeScalarDefaultDoc(`{"type":"integer","format":"int32","enum":[1,2]}`, value), checked, "E_SCHEMA_INVALID", at)
		}
		nativeEnumDefaultFailure(t, nativeScalarDefaultDoc(`{"type":"number","format":"float","enum":[1.5]}`, `1.5000000000000001`), checked, "E_SCHEMA_INVALID", at)
		for _, value := range []string{`1`, `1.0`, `1e0`} {
			r := request(nativeScalarDefaultDoc(`{"type":"number","format":"double","enum":[1]}`, value), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("numeric equality before mapping", value, err)
			}
		}
		// Explicit field nullability works in both modes, independently of optional presence.
		rNullable := request(bytes.Replace(nativeScalarDefaultDoc(`{"type":"integer","enum":[1,2]}`, `null`), []byte(`"default":null`), []byte(`"nullable":true,"default":null`), 1), "Dto")
		rNullable.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(rNullable); err != nil {
			t.Fatal("field nullable", err)
		}
		// Internal native enum references retain their existing validation opt-in.
		r := request(scalarRefDoc(`{"type":"integer","nullable":true,"enum":[1,2]}`, `null`, `,"nullable":true`), "Dto")
		r.Options.Validation = checked
		internal, internalErr := core.GenerateWithDiagnostics(r)
		if checked && internalErr != nil || !checked && (internalErr == nil || !reflect.DeepEqual(internal, core.GenerationReport{})) {
			t.Fatal("internal admission drift", checked, internalErr)
		}
		if checked {
			r.Document = scalarRefDoc(`{"type":"integer","nullable":true,"enum":[1,2]}`, `1`, "")
			if got, err := core.GenerateWithDiagnostics(r); err != nil || !strings.Contains(source(t, got, "AHolderDto"), "kotlin.Int? = 1") {
				t.Fatal("nearest false/source chain", err)
			}
		}
		for _, leaf := range []string{
			`{"type":"string","enum":["a"]}`, `{"enum":[1,2]}`,
			`{"type":"integer","enum":[1],"const":1}`, `{"type":"integer","enum":[1],"x-native-type":"kotlin.Int"}`,
			`{"type":"integer","enum":[1],"x-kotlin-value-class":true}`, `{"type":"integer","enum":[]}`,
			`{"type":"integer","enum":[1],"default":1}`,
		} {
			r.Document = nativeScalarDefaultDoc(leaf, `null`)
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("ungranted source", leaf, err)
			}
		}
		for _, body := range []string{
			`{"Code":{"type":"integer","enum":[1]},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Code"}],"default":1}}}}`,
			`{"Code":{"type":"integer","enum":[1]},"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"v":{"$ref":"#/components/schemas/Code","default":1}}},{"properties":{"tag":{"const":"b"}}}]}}`,
		} {
			r.Document = schemas(body)
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("ungranted role", body, err)
			}
		}
	}
	// Verbatim native terminal from the former builtin-scalar default guard.
	graduation := request(scalarRefDoc(`{"type":"integer","nullable":true,"enum":[1]}`, `null`, `,"nullable":true`), "Dto")
	graduation.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(graduation); err != nil {
		t.Fatal("former native guard", err)
	}
	// Null cannot bypass unsupported floating factors, nor invalid retained checks.
	r := request(nativeScalarDefaultDoc(`{"type":"integer","enum":[1,2],"minimum":2}`, `2`), "Dto")
	r.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(r); err != nil {
		t.Fatal(err)
	}
	nativeEnumDefaultFailure(t, nativeScalarDefaultDoc(`{"type":"integer","enum":[1,2],"minimum":2}`, `1`), true, "E_SCHEMA_INVALID", at)
	nativeEnumDefaultFailure(t, nativeScalarDefaultDoc(`{"type":"integer","enum":[1,2],"minimum":2}`, `2`), false, "E_SCHEMA_UNSUPPORTED", "")
	nativeEnumDefaultFailure(t, nativeScalarDefaultDoc(`{"type":"number","enum":[1.5],"multipleOf":0.5}`, `1.5`), true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Code/multipleOf")
}

func TestReferencedNativeEnumScalarFieldDefaultsPresenceAndIdentity(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, role := range []string{"required", "patch", "discarded"} {
			field := `{"$ref":"#/components/schemas/Code","nullable":true,"default":1}`
			holder := `{"required":["v"],"properties":{"v":` + field + `}}`
			if role == "patch" {
				holder = `{"properties":{"v":` + field + `}}`
			}
			if role == "discarded" {
				holder = `{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`
			}
			doc := schemas(`{"Code":{"type":"integer","enum":[1,2]},"Holder":` + holder + `}`)
			if role == "patch" {
				doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2]}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
			}
			r := request(doc, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			// Existing noDrop rejects discarded checks; no new aggregation permission.
			if role == "discarded" && checked {
				if err == nil {
					t.Fatal("discarded native checks dropped")
				}
				continue
			}
			if err != nil {
				t.Fatal(role, checked, err)
			}
			r.Document = bytes.Replace(doc, []byte(`"default":1`), []byte(`"default":2`), 1)
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("suppressed annotation identity", role, checked, err)
			}
			r.Document = bytes.Replace(doc, []byte(`"default":1`), []byte(`"default":9`), 1)
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("suppression skipped validation", role, err)
			}
		}
	}
	// Effective literal participates in merged shape identity.
	for _, different := range []bool{false, true} {
		other := "1"
		if different {
			other = "2"
		}
		r := request(schemas(`{"Code":{"type":"integer","enum":[1,2]},"Alpha":{"properties":{"v":{"$ref":"#/components/schemas/Code","default":1}}},"Zeta":{"properties":{"v":{"$ref":"#/components/schemas/Code","default":`+other+`}}}}`), "Dto")
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		alias := strings.Contains(source(t, got, "ZetaDto"), "typealias ZetaDto =")
		if alias == different {
			t.Fatal("default merge identity", different)
		}
	}
}

func TestReferencedNativeEnumScalarFieldDefaultsIndependentLimits(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			members, fields := []string{}, []string{}
			for i := 1; i <= count; i++ {
				members = append(members, strconv.Itoa(i))
				fields = append(fields, `"v`+strconv.Itoa(i)+`":{"$ref":"#/components/schemas/Code","default":1}`)
			}
			for _, input := range [][]byte{
				nativeScalarDefaultDoc(`{"type":"integer","enum":[`+strings.Join(members, ",")+`]}`, `1`),
				schemas(`{"Code":{"type":"integer","enum":[1,2]},"Holder":{"properties":{` + strings.Join(fields, ",") + `}}}`),
			} {
				r := request(input, "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				if count == 240 && err != nil {
					t.Fatal("independent boundary", checked, err)
				}
				if count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
					t.Fatal("limit widened", checked, err)
				}
			}
		}
	}
}
