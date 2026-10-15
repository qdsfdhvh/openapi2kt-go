package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestRootBorrowedCollectionFullSourceDepthCheckedEdgesAndSlots(t *testing.T) {
	for _, count := range []int{44, 45} {
		props := []string{`"list":{"type":"array","items":{"type":"string"}}`}
		next := "list"
		for i := 0; i < count; i++ {
			key := fmt.Sprintf("h%d", i)
			props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
			next = key
		}
		doc := schemas(`{"Source":{"properties":{` + strings.Join(props, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/` + next + `","nullable":false},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}}`)
		for _, checked := range []bool{false, true} {
			r := request(doc, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if count == 44 {
				if err != nil {
					t.Fatal("source48 rejected", checked, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("cached sourcework reset", count, checked, err)
			}
		}
	}
	for _, count := range []int{16, 17} {
		props := []string{`"leaf":{"type":"string","minLength":2}`}
		next := "leaf"
		for i := 0; i < count; i++ {
			key := fmt.Sprintf("l%d", i)
			props = append(props, fmt.Sprintf(`%q:{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Source/properties/%s"}}`, key, next))
			next = key
		}
		doc := schemas(`{"Source":{"properties":{` + strings.Join(props, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/` + next + `"}}`)
		r := request(doc, "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if count == 16 {
			if err != nil {
				t.Fatal("checked16 rejected", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("checked17 admitted", err)
		}
	}
	for _, count := range []int{240, 241} {
		var fields, names []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge"}`, i))
			names = append(names, fmt.Sprintf(`"p%d"`, i))
		}
		doc := schemas(`{"Source":{"properties":{"list":{"type":"array","items":{"type":"integer","format":"int64"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/list"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(names, ",") + `]}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 240 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed container lost one slot", err)
		}
	}
}
func TestRootBorrowedCollectionGrammarCyclesAndExcludedNamedLeaves(t *testing.T) {
	for _, tc := range []struct{ leaf, code, pointer string }{
		{`{"type":"array","items":{"type":"string","default":"x"}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/list/items/default"},
		{`{"type":"array","minItems":-1,"items":{"type":"string"}}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/list/minItems"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/list"}}`, "E_REF_UNSUPPORTED", "/components/schemas/Source/properties/list/items"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Named"}}`, "E_REF_UNSUPPORTED", "/components/schemas/Source/properties/list/items"},
	} {
		doc := schemas(`{"ABridge":{"$ref":"#/components/schemas/Source/properties/list"},"Named":{"properties":{"id":{"type":"string"}}},"Source":{"properties":{"list":` + tc.leaf + `}}}`)
		r := request(doc, "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if strings.Contains(tc.leaf, `"#/components/schemas/Named"`) {
			// Preserve original input as the new actual ordinary DTO leaf profile.
			borrowedDTOGraduation(t, doc, "org.example.models.models.NamedDto")
			continue
		}
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || *err.(core.Diagnostics)[0].Pointer != tc.pointer || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed original grammar/role hidden", tc, err)
		}
	}
}

func TestRootBorrowedCollectionNestedErasedSourceHeight(t *testing.T) {
	for _, count := range []int{43, 44} {
		props := []string{`"leaf":{"type":"string"}`}
		next := "leaf"
		for i := 0; i < count; i++ {
			key := fmt.Sprintf("h%d", i)
			props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
			next = key
		}
		props = append(props, `"list":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/`+next+`"}}`)
		doc := schemas(`{"Source":{"properties":{` + strings.Join(props, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/list","nullable":false},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}}`)
		for _, checked := range []bool{false, true} {
			r := request(doc, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if count == 43 {
				if err != nil {
					t.Fatal("nested source48 rejected", err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("nested borrowed item refs source49 reset", checked, err)
			}
		}
	}
}
