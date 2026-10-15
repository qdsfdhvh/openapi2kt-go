package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRootBorrowedBuiltinOriginalLeafValidation(t *testing.T) {
	for _, tc := range []struct{ leaf, code, pointer string }{
		{`{"type":"string","default":"x"}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/default"},
		{`{"type":"string","minLength":-1}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/leaf/minLength"},
		{`{"type":"string","pattern":"(?=a)"}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/pattern"},
		{`{"type":"number","multipleOf":0}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/leaf/multipleOf"},
	} {
		// ABridge sorts before its lexical source and has no consumer. It must
		// validate the original leaf rather than hide it behind root rejection.
		input := schemas(`{"ABridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"Source":{"properties":{"leaf":` + tc.leaf + `}}}`)
		r := request(input, "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != tc.code || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer || len(got.Artifacts.Files) != 0 {
			t.Fatal("original leaf grammar masked", tc, err)
		}
	}
}

func TestRootBorrowedBuiltinCachedSourceDepthAndNativeSlots(t *testing.T) {
	for _, count := range []int{46, 47} {
		props := []string{`"leaf":{"type":["integer","null"],"format":"int64"}`}
		for i := 0; i < count; i++ {
			target := "Source/properties/leaf"
			if i > 0 {
				target = fmt.Sprintf("Source/properties/h%d", i-1)
			}
			props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		input := schemas(`{"Source":{"properties":{` + strings.Join(props, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/h` + fmt.Sprint(count-1) + `","nullable":false},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}}`)
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if count == 46 {
				// count hops + root incoming + consumer incoming + field depth =49.
				if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("borrowed cached source height reset", count, checked, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
				t.Fatal(count, checked, err)
			}
		}
	}
	for _, nullable := range []bool{false, true} {
		limit := 120
		if nullable {
			limit = 240
		}
		for _, count := range []int{limit, limit + 1} {
			props, required := []string{}, []string{}
			for i := 0; i < count; i++ {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%t}`, i, nullable))
				required = append(required, fmt.Sprintf(`"p%d"`, i))
			}
			input := schemas(`{"Source":{"properties":{"leaf":{"type":"integer","format":"int64"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if count == limit {
				if err != nil {
					t.Fatal(nullable, count, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
				t.Fatal("borrowed root became nominal one-slot type", nullable, count, err)
			}
		}
	}
}
