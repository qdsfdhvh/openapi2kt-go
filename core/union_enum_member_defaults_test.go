package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const unionEnumMemberBranches = `[{"type":"object","required":["kind"],"properties":{"kind":{"const":"a"},"mode":{"type":"string","enum":["on","off"],"default":"on"},"number":{"type":"integer","format":"int32","enum":[1,2],"default":1}}},{"type":"object","required":["kind"],"properties":{"kind":{"const":"b"},"other":{"type":"string"}}}]`

func TestOrdinaryObjectUnionEnumMemberDefaultControl(t *testing.T) {
	branches := unionEnumMemberBranches
	for _, keyword := range []string{"oneOf", "anyOf"} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, own := range []bool{false, true} {
					annotation := ""
					if own {
						annotation = `,"default":{"kind":"a"}`
					}
					input := schemas(`{"Holder":{"properties":{"value":{"` + keyword + `":` + branches + annotation + `}}}}`)
					rq := request(input, "Dto")
					rq.Options.Validation, rq.Options.ModelStyle = checked, style
					first, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal("selected enum member own default lost", keyword, checked, style, own, err)
					}
					again, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(first, again) {
						t.Fatal("selected member default nondeterminism", err)
					}
				}
			}
		}
	}
}

func TestComposedObjectUnionEnumMemberOwnDefaults(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, pair := range []struct{ enumOwn, outerOwn bool }{{false, true}, {true, false}, {true, true}} {
				branches := unionEnumMemberBranches
				if !pair.enumOwn {
					branches = strings.ReplaceAll(strings.ReplaceAll(branches, `,"default":"on"`, ""), `,"default":1`, "")
				}
				annotation := ""
				if pair.outerOwn {
					annotation = `,"default":{"version":"v1","kind":"a","mode":"on","number":1}`
				}
				input := schemas(`{"U":{"allOf":[{"required":["version"],"properties":{"version":{"const":"v1"}}},{"oneOf":` + branches + `}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/U"` + annotation + `}}}}`)
				rq := request(input, "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = checked, style
				first, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("composed selected enum member own default lost", checked, style, pair, err)
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(first, again) {
					t.Fatal("composed member default nondeterminism", err)
				}
			}
		}
	}
}

func TestComposedExistingNumericPinDefaultKeepsLiteralSelection(t *testing.T) {
	// This schema was already accepted without a member-own annotation. Its
	// Float formatting previously turned the known default into Unknown at JVM
	// initialization; preserve the pin only inside the decoding literal.
	for _, own := range []string{"", `,"default":16777217`} {
		input := schemas(`{"U":{"allOf":[{"required":["version"],"properties":{"version":{"const":"v1"}}},{"oneOf":[{"required":["kind"],"properties":{"kind":{"const":"a"},"floating":{"type":"number","format":"float","enum":[16777217,16777219]` + own + `}}},{"required":["kind"],"properties":{"kind":{"const":"b"},"other":{"type":"string"}}}]}]},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/U","default":{"version":"v1","kind":"a","floating":16777217}}}}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !strings.Contains(source(t, got, "HolderDto"), `\"floating\":16777217`) {
				t.Fatal("default lost the original selected numeric pin", checked, own, err)
			}
		}
	}
}
