package core_test

import (
	"os"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHierarchyNullMembersKeepPhysicalClassesAndTypedIntent(t *testing.T) {
	b, e := os.ReadFile("../tests/fixtures/http/hierarchy-null-members.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, validation := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct {
			name, canonical string
			nullable        bool
		}{{"ERoot", "Root", true}, {"EGroup", "Group", true}, {"ParentBridge", "Root", true}, {"ParentFalse", "Root", false}, {"LeafBridge", "First", true}, {"LeafFalse", "First", false}} {
			want := "public typealias " + tc.name + "Dto = org.example.models.models." + tc.canonical + "Dto"
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.name+"Dto"), want) {
				t.Fatal(tc)
			}
		}
		if !strings.Contains(source(t, r, "RootDto"), "public sealed class RootDto {") || !strings.Contains(source(t, r, "GroupDto"), ": org.example.models.models.RootDto() {") {
			t.Fatal("nullable physical inheritance")
		}
		holder := source(t, r, "IntrinsicHolderDto")
		if !strings.Contains(holder, "public val root: org.example.models.models.RootDto? = null,") || !strings.Contains(holder, "public val group: org.example.models.models.GroupDto? = null,") || !strings.Contains(holder, "public val leaf: org.example.models.models.FirstDto? = null,") {
			t.Fatal(holder)
		}
		for _, file := range r.Artifacts.Files {
			if strings.Contains(file.Content, "data object NullRootDto") {
				t.Fatal("hierarchy null carrier invented")
			}
		}
	}
}
