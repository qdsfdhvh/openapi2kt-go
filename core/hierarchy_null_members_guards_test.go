package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHierarchyNullableMemberAliasIntentAndParentFalse(t *testing.T) {
	const leaves = `"Leaf":{"nullable":true,"properties":{"kind":{"type":"string","const":"leaf"},"id":{"type":"string"}},"required":["id"]},"Other":{"properties":{"kind":{"type":"string","const":"other"},"id":{"type":"string"}},"required":["id"]}`
	for _, tc := range []struct {
		alias, branch string
		want          bool
	}{
		{`"$ref":"#/components/schemas/Leaf","nullable":true`, ``, true},
		{`"$ref":"#/components/schemas/Leaf","nullable":true`, `,"nullable":false`, false},
		{`"$ref":"#/components/schemas/Leaf","nullable":false`, ``, false},
		{`"$ref":"#/components/schemas/Leaf"`, ``, true},
	} {
		b := schemas(`{` + leaves + `,"Maybe":{` + tc.alias + `},"Root":{"nullable":false,"oneOf":[{"$ref":"#/components/schemas/Maybe"` + tc.branch + `},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Use":{"$ref":"#/components/schemas/Root"},"Narrow":{"$ref":"#/components/schemas/Root","nullable":false}}`)
		for _, validation := range []bool{false, true} {
			req := request(b, "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(tc, e)
			}
			want := "= org.example.models.models.RootDto"
			if tc.want {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, "UseDto"), want) || !strings.Contains(source(t, r, "NarrowDto"), "= org.example.models.models.RootDto\n") {
				t.Fatal(tc, source(t, r, "UseDto"))
			}
		}
	}
}

func TestHierarchyNullableMembersSingletonFalseCodec(t *testing.T) {
	for _, branch := range []string{`{"type":"null"},{"$ref":"#/components/schemas/Leaf"}`, `{"$ref":"#/components/schemas/Maybe"}`} {
		body := `{"Root":{"oneOf":[` + branch + `],"discriminator":{"propertyName":"tag"}},"Leaf":{"type":"object"},"Maybe":{"$ref":"#/components/schemas/Leaf","nullable":true},"Narrow":{"$ref":"#/components/schemas/Leaf","nullable":false}}`
		r, e := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(source(t, r, "LeafDto"), "if (element == kotlinx.serialization.json.JsonNull)") {
			t.Fatal("nullable member's physical singleton codec consumed null", source(t, r, "LeafDto"))
		}
	}
}

func TestHierarchyAliasMemberInheritsDerivedNullability(t *testing.T) {
	for _, override := range []string{"", `,"nullable":false`} {
		body := `{"Root":{"nullable":false,"oneOf":[{"type":"null"},{"$ref":"#/components/schemas/First"}],"discriminator":{"propertyName":"tag"}},"First":{"properties":{"id":{"type":"string"}},"required":["id"]},"Alias":{"$ref":"#/components/schemas/Root"` + override + `},"Parent":{"oneOf":[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"family"}},"Other":{"properties":{"id":{"type":"string"}},"required":["id"]},"Use":{"$ref":"#/components/schemas/Parent"},"Input":{"properties":{"parent":{"$ref":"#/components/schemas/Parent"}},"required":["parent"]}}`
		for _, validation := range []bool{false, true} {
			req := request(schemas(body), "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			want := "= org.example.models.models.ParentDto"
			field := "public val parent: org.example.models.models.ParentDto"
			if override == "" {
				want += "?"
				field += "? = null"
			}
			if !strings.Contains(source(t, r, "UseDto"), want+"\n") || !strings.Contains(source(t, r, "InputDto"), field+",") {
				t.Fatal(override, source(t, r, "UseDto"), source(t, r, "InputDto"))
			}
		}
	}
}

func TestHierarchyNullMembersOriginalSourceGuards(t *testing.T) {
	leaf := `"Leaf":{"properties":{"id":{"type":"string"}},"required":["id"]}`
	for _, tc := range []struct{ branch, code, pointer string }{
		{`{"type":"null","nullable":true}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/0/nullable"},
		{`{"type":"null","const":null}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/0/const"},
		{`{"$ref":"#/components/schemas/Leaf","nullable":"invalid"}`, "E_SCHEMA_INVALID", "/components/schemas/Root/oneOf/0/nullable"},
	} {
		aliasFailure(t, schemas(`{"Root":{"oneOf":[`+tc.branch+`,{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},`+leaf+`}`), tc.code, tc.pointer)
	}
	aliasFailure(t, schemas(`{"Root":{"oneOf":[{"type":"null"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf")
	b := schemas(`{"Root":{"oneOf":[{"type":"null"},{"type":"null"},{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},` + leaf + `}`)
	r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	count := 0
	for _, w := range r.Warnings {
		if w.Code == "W_ONEOF_NULL_OVERLAP" {
			count++
			if *w.Pointer != "/components/schemas/Root/oneOf" {
				t.Fatal(w)
			}
		}
	}
	if count != 1 {
		t.Fatal("null overlap warning", r.Warnings)
	}
	for _, n := range []int{240, 241} {
		branches := []string{`{"$ref":"#/components/schemas/Leaf"}`}
		for range n - 1 {
			branches = append(branches, `{"type":"null"}`)
		}
		b := schemas(`{"Root":{"oneOf":[` + strings.Join(branches, ",") + `],"discriminator":{"propertyName":"tag"}},` + leaf + `}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	// Null consumes a source edge even when it has no physical classifier.
	for _, n := range []int{47, 48} {
		nodes := []string{leaf}
		for i := range n {
			next := "Leaf"
			if i < n-1 {
				next = fmt.Sprint("D", i+1)
			}
			nodes = append(nodes, fmt.Sprintf(`"D%d":{"oneOf":[{"type":"null"},{"$ref":"#/components/schemas/%s"}],"discriminator":{"propertyName":"tag"}}`, i, next))
		}
		r, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(nodes, ",")+`}`), "Dto"))
		if n == 47 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}
