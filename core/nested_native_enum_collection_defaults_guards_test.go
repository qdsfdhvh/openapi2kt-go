package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedNativeEnumCollectionDefaultsOriginalMembershipGrammarAndNullability(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{nestedDefaultChain(2, `{"type":"number","format":"float","enum":[1]}`, `[[1.000000001]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"number","format":"double","enum":[1]}`, `[[1.00000000000000001]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `[[1.0]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `[[1e0]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[0]}`, `[[-0]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `[[2147483648]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","format":"int64","enum":[1]}`, `[[9223372036854775808]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"number","format":"float","enum":[1]}`, `[[1e39]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"boolean","enum":[true]}`, `[["true"]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"boolean","enum":[true]}`, `[[1]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{`{"type":"array","items":{"additionalProperties":{"type":"boolean","enum":[true]}},"default":[{"a/~":false}]}`, "E_SCHEMA_INVALID", "/default/0/a~1~0"},
		{`{"additionalProperties":{"additionalProperties":{"type":"integer","enum":[1]}},"default":{"z":{"z":2,"a":3},"a":4}}`, "E_SCHEMA_INVALID", "/default/z/z"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `null`), "E_SCHEMA_INVALID", "/default"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `[null]`), "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","nullable":true,"items":{"type":"array","items":{"type":"integer","enum":[1]}} ,"default":[null]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":"integer","enum":[1]}},"default":[[null]]}`, "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","nullable":true,"enum":[1]}`, `[null]`), "E_SCHEMA_INVALID", "/default/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1,1]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum/1"},
		{nestedDefaultChain(2, `{"type":"boolean","enum":[true,false,true]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum/2"},
		{nestedDefaultChain(2, `{"type":"number","enum":[1.0,1e0]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum/1"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1,1.0]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum/1"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1,"bad"]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum/1"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[null,1]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum/0"},
		{nestedDefaultChain(2, `{"type":"number","enum":[1]}`, `[[0.`+strings.Repeat("0", 256)+`1]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"number","enum":[1]}`, `[[1e-513]]`), "E_SCHEMA_INVALID", "/default/0/0"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				// Optional DTO nullability must not grant an annotation null.
				r := nestedNativeEnumDefaultRequest(tc.field, checked, style)
				nestedNativeEnumDefaultFailure(t, r, tc.code, "/components/schemas/Holder/properties/value"+tc.tail)
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsFullPrevalidationBeforeSuppression(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `[[3]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[]}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum"},
		{`{"type":"array","nullable":true,"items":{"type":"array","items":{"type":"integer","enum":[1,"bad"]}},"default":null}`, "E_SCHEMA_INVALID", "/items/items/enum/1"},
		{nestedDefaultChain(2, `{"type":"number","enum":[1,2],"multipleOf":0.5}`, `[]`), "E_SCHEMA_UNSUPPORTED", "/items/items/multipleOf"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1]}`, `[[`+strings.TrimSuffix(strings.Repeat("0,", 16383), ",")+`]]`), "E_MODEL_LIMIT", ""},
	} {
		patchField := strings.Replace(tc.field, `"type":"array"`, `"type":"array","nullable":true`, 1)
		if strings.Contains(tc.field, `"nullable":true`) {
			patchField = tc.field
		}
		for _, context := range []struct {
			doc     []byte
			pointer string
		}{
			{fieldDoc(tc.field), "/components/schemas/Record/properties/value"},
			{schemas(`{"Holder":{"allOf":[{"properties":{"value":` + tc.field + `}},{"properties":{"value":{"type":"string"}}}]}}`), "/components/schemas/Holder/allOf/0/properties/value"},
			{[]byte(`{"openapi":"3.1.0","paths":{"/patch":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"value":` + patchField + `}}}}},"responses":{"204":{"description":"OK"}}}}}}`), "/paths/~1patch/patch/requestBody/content/application~1json/schema/properties/value"},
		} {
			for _, checked := range []bool{false, true} {
				r := request(context.doc, "Dto")
				r.Options.Validation = checked
				if checked && strings.Contains(context.pointer, "/allOf/") {
					// Existing NoDrop rejects the original enforcing enum before
					// a discarded annotation can acquire a scratch type/default.
					nestedNativeEnumDefaultFailure(t, r, "E_SCHEMA_UNSUPPORTED", context.pointer+"/items/items/enum")
					continue
				}
				if tc.code == "E_MODEL_LIMIT" {
					got, err := core.GenerateWithDiagnostics(r)
					ds, ok := err.(core.Diagnostics)
					if !ok || ds[0].Code != tc.code || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, context.pointer+"/default") || !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal("suppressed annotation lost budget", err)
					}
					continue
				}
				code := tc.code
				if !checked && strings.Contains(tc.field, "multipleOf") {
					code = "E_SCHEMA_UNSUPPORTED"
				}
				nestedNativeEnumDefaultFailure(t, r, code, context.pointer+tc.tail)
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsNullVocabularyAndNumericGrammarCaps(t *testing.T) {
	for _, tc := range []struct{ leaf, tail string }{
		{`{"type":"integer","nullable":true,"enum":[null,1,null]}`, "/enum/2"},
		{`{"type":["integer","null"],"enum":[null]}`, "/enum"},
		{`{"type":"number","enum":[1e-513]}`, "/enum/0"},
		{`{"type":"number","enum":[1.` + strings.Repeat("0", 256) + `]}`, "/enum/0"},
	} {
		r := request(fieldDoc(nestedDefaultChain(2, tc.leaf, `[]`)), "Dto")
		r.Options.Validation = true
		nestedNativeEnumDefaultFailure(t, r, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/items/items"+tc.tail)
	}
	for _, leaf := range []string{
		`{"type":["integer","null"],"enum":[null,1,2]}`,
		`{"type":"boolean","nullable":true,"enum":[true,null]}`,
	} {
		r := request(fieldDoc(nestedDefaultChain(2, leaf, `[[null]]`)), "Dto")
		r.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("checked nullable native vocabulary lost", err)
		}
		r.Options.Validation = false
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("null vocabulary granted without validation", err)
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsRetainedCountsBoundsAndFactors(t *testing.T) {
	for _, tc := range []struct{ field, code, tail string }{
		{`{"type":"array","minItems":1,"items":{"type":"array","items":{"type":"integer","enum":[1]}} ,"default":[]}`, "E_SCHEMA_INVALID", "/default"},
		{`{"type":"array","items":{"type":"array","minItems":1,"items":{"type":"integer","enum":[1]}},"default":[[]]}`, "E_SCHEMA_INVALID", "/default/0"},
		{`{"additionalProperties":{"additionalProperties":{"type":"boolean","enum":[true]},"maxProperties":0},"default":{"a/~":{"z":true}}}`, "E_SCHEMA_INVALID", "/default/a~1~0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2}`, `[[1]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","format":"int64","enum":[2,3,4],"multipleOf":2}`, `[[3]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1,3],"multipleOf":2}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum"},
		{nestedDefaultChain(2, `{"type":["integer","null"],"enum":[null,1],"minimum":2}`, `[]`), "E_SCHEMA_INVALID", "/items/items/enum"},
		{nestedDefaultChain(2, `{"type":"number","format":"float","enum":[1.000000001,2],"exclusiveMinimum":1}`, `[[1.000000001]]`), "E_SCHEMA_INVALID", "/default/0/0"},
		{nestedDefaultChain(2, `{"type":"number","enum":[1,2],"multipleOf":0.5}`, `[]`), "E_SCHEMA_UNSUPPORTED", "/items/items/multipleOf"},
		{nestedDefaultChain(2, `{"type":"boolean","enum":[true],"minimum":0}`, `[]`), "E_SCHEMA_UNSUPPORTED", "/items/items/minimum"},
		{nestedDefaultChain(2, `{"type":"integer","enum":[1],"minLength":1}`, `[]`), "E_SCHEMA_UNSUPPORTED", "/items/items/minLength"},
		{`{"type":"array","uniqueItems":true,"items":{"type":"array","items":{"type":"boolean","enum":[true]}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/uniqueItems"},
	} {
		r := request(fieldDoc(tc.field), "Dto")
		r.Options.Validation = true
		nestedNativeEnumDefaultFailure(t, r, tc.code, "/components/schemas/Record/properties/value"+tc.tail)
		r.Options.Validation = false
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("constrained schema admitted without opt-in", tc, err)
		}
	}
	field := `{"type":"array","minItems":1,"maxItems":2,"items":{"additionalProperties":{"type":["integer","null"],"format":"int64","enum":[null,1,2,3,4],"minimum":2,"exclusiveMaximum":5,"multipleOf":2},"minProperties":1,"maxProperties":3},"default":[{"nil":null,"ok":4}]}`
	r := nestedNativeEnumDefaultRequest(field, true, core.DataClass)
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"does not belong to enum.", "value >= 2L", "value < 5L", "value % 2L == 0L", ".size >= 1", ".size <= 3"} {
		if !strings.Contains(source(t, got, "HolderDto"), marker) {
			t.Fatal("lost recursive retained check metadata", marker)
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsExcludedRolesAndNoDrop(t *testing.T) {
	field := nestedDefaultChain(2, `{"type":"integer","enum":[1,2]}`, `[[1]]`)
	pure := strings.Replace(field, `,"default":[[1]]`, "", 1)
	for name, input := range map[string]string{
		"child":        `{"Holder":{"properties":{"value":{"type":"array","items":` + field + `}}}}`,
		"mapChild":     `{"Holder":{"properties":{"value":{"additionalProperties":` + field + `}}}}`,
		"wrapper":      `{"Holder":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		"namedLeaf":    `{"Code":{"type":"integer","enum":[1,2]},"Holder":{"properties":{"value":` + nestedDefaultChain(2, `{"$ref":"#/components/schemas/Code"}`, `[]`) + `}}}`,
		"internalLeaf": `{"Catalog":{"properties":{"code":{"type":"integer","enum":[1,2]}}},"Holder":{"properties":{"value":` + nestedDefaultChain(2, `{"$ref":"#/components/schemas/Catalog/properties/code"}`, `[]`) + `}}}`,
		"refWrapper":   `{"Tree":` + pure + `,"Holder":{"properties":{"value":{"allOf":[{"$ref":"#/components/schemas/Tree"}],"default":[[1]]}}}}`,
		"union":        `{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"value":` + field + `}},{"properties":{"tag":{"const":"two"}}}]}}`,
		"hierarchy":    `{"Base":{"discriminator":{"propertyName":"kind"},"properties":{"kind":{"type":"string"},"value":` + field + `},"required":["kind"]},"Child":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"x":{"type":"string"}}}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			for _, checked := range []bool{false, true} {
				r := request(schemas(input), "Dto")
				r.Options.Validation = checked
				if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("independent role leaked", checked, err)
				}
			}
		})
	}
	// The direct nested role never owns $ref fields; the referenced collection
	// field role separately admits this same shape with its own source grammar.
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"Tree":`+pure+`,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Tree","default":[[1]]}}}}`), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("referenced role graduation", checked, err)
		}
	}
	for _, tc := range []struct{ leaf, tail string }{
		{`{"type":"integer","enum":[1],"default":1}`, "/default"},
		{`{"type":"integer","enum":[1],"const":1}`, "/const"},
		{`{"type":"integer","enum":[1],"x-native-type":"org.example.Code"}`, "/x-native-type"},
		{`{"type":"integer","enum":[1],"x-kotlin-value-class":false}`, "/x-kotlin-value-class"},
		{`{"allOf":[{"type":"integer","enum":[1]}]}`, "/allOf"},
		{`{"oneOf":[{"type":"integer","enum":[1]},{"type":"null"}]}`, "/oneOf"},
		{`{"enum":["one"],"minLength":1}`, "/minLength"},
		{`{"enum":["one"],"pattern":"^o"}`, "/pattern"},
	} {
		for _, checked := range []bool{false, true} {
			r := request(fieldDoc(nestedDefaultChain(2, tc.leaf, `[]`)), "Dto")
			r.Options.Validation = checked
			nestedNativeEnumDefaultFailure(t, r, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/items/items"+tc.tail)
		}
	}
	// Valid discarded annotations cannot drop enforcing vocabulary metadata.
	for _, checked := range []bool{false, true} {
		r := request(schemas(`{"Holder":{"allOf":[{"properties":{"value":`+field+`}},{"properties":{"value":{"type":"string"}}}]}}`), "Dto")
		r.Options.Validation = checked
		if checked {
			nestedNativeEnumDefaultFailure(t, r, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Holder/allOf/0/properties/value/items/items/enum")
		} else {
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range got.Warnings {
				if w.Code == "W_ENUM_NATIVE_APPROXIMATION" {
					t.Fatal("scratch warning escaped", w)
				}
			}
		}
	}
}
