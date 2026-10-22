package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableComposedAndInternalObjectReferenceMembers(t *testing.T) {
	for _, terminal := range []string{
		`{"nullable":true,"allOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]},{"properties":{"stamp":{"const":"shared"}}}]}`,
		`{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}`,
	} {
		for _, owner := range []string{"named", "direct", "composed", "union-nested", "union-variant"} {
			// An intrinsically nullable inline allOf union alternative is not
			// an admitted original owner; reference borrowing cannot enable it.
			if owner == "union-variant" && strings.Contains(terminal, `"allOf"`) {
				continue
			}
			base := `"Target":` + terminal
			ref := "#/components/schemas/Target"
			if owner == "direct" {
				base = `"Owner":{"properties":{"target":` + terminal + `}}`
				ref = "#/components/schemas/Owner/properties/target"
			}
			if owner == "composed" {
				base = `"Owner":{"allOf":[{"properties":{"target":` + terminal + `}},{"properties":{"extra":{"type":"string"}}}]}`
				ref = "#/components/schemas/Owner/allOf/0/properties/target"
			}
			if owner == "union-nested" {
				base = `"Owner":{"oneOf":[{"properties":{"kind":{"const":"owner-a"},"target":` + terminal + `}},{"properties":{"kind":{"const":"owner-b"},"note":{"type":"string"}}}]}`
				ref = "#/components/schemas/Owner/oneOf/0/properties/target"
			}
			if owner == "union-variant" {
				base = `"Owner":{"oneOf":[` + terminal + `,{"properties":{"kind":{"const":"b"},"note":{"type":"string"}}}]}`
				ref = "#/components/schemas/Owner/oneOf/0"
			}
			for _, annotation := range []string{"", `,"nullable":false`, `,"nullable":true`} {
				for _, branchRef := range []string{ref, "#/components/schemas/Alias"} {
					input := schemas(`{` + base + `,"Alias":{"$ref":"` + ref + `"},"Choice":{"oneOf":[{"$ref":"` + branchRef + `"` + annotation + `},{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]}}`)
					rq := request(input, "Dto")
					r, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(terminal, owner, annotation, err, *err.(core.Diagnostics)[0].Pointer)
					}
					text := source(t, r, "ChoiceDto")
					if strings.Contains(text, "NullChoiceDto") != (annotation != `,"nullable":false`) || !strings.Contains(text, "id: kotlin.String") {
						t.Fatal("nullable member shape or intent lost", text)
					}
					again, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(r, again) {
						t.Fatal("nondeterministic member borrowing", err)
					}
				}
			}
		}
	}
}

func TestNullableAllOfMemberOriginalNarrowedAliasInput(t *testing.T) {
	input := schemas(`{"Value":{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]}]},"False":{"$ref":"#/components/schemas/Value","nullable":false},"Other":{"oneOf":[{"$ref":"#/components/schemas/False"},{"properties":{"ok":{"type":"boolean"}}}]}}`)
	r := report(t, input, "Dto")
	text := source(t, r, "OtherDto")
	if !strings.Contains(text, "id: kotlin.String") || strings.Contains(text, "NullOtherDto") {
		t.Fatal("original false alias lost constructor or narrowed intent", text)
	}
}

func TestNullableInternalUnionMemberForwardAliasIntent(t *testing.T) {
	for _, terminal := range []string{
		`{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}`,
		`{"nullable":true,"allOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}]}`,
	} {
		for _, union := range []string{"oneOf", "anyOf"} {
			for _, intent := range []string{"", `,"nullable":false`, `,"nullable":true`} {
				input := schemas(`{"Box":{"properties":{"value":{"` + union + `":[{"$ref":"#/components/schemas/Alias"` + intent + `},{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}}}]}}},"Export":{"$ref":"#/components/schemas/Box/properties/value"},"ExportFalse":{"$ref":"#/components/schemas/Export","nullable":false},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Owner":{"properties":{"node":` + terminal + `}}}`)
				r := report(t, input, "Dto")
				alias := source(t, r, "ExportDto")
				wantNullable := intent != `,"nullable":false`
				if strings.Contains(alias, "BoxDto.Value?") != wantNullable || !strings.Contains(alias, "BoxDto.Value") || strings.Contains(source(t, r, "ExportFalseDto"), "BoxDto.Value?") {
					t.Fatal("forward member alias cached incorrect nullable intent", terminal, union, intent, alias)
				}
			}
		}
	}
}

func TestNullableComposedReferenceMemberSourceBudgets(t *testing.T) {
	for _, n := range []int{44, 45} {
		parts := []string{`"Owner":{"properties":{"target":{"nullable":true,"allOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]}]}}}`}
		for i := range n {
			ref := "Owner/properties/target"
			if i != 0 {
				ref = fmt.Sprintf("A%02d", i-1)
			}
			parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, ref))
		}
		parts = append(parts, fmt.Sprintf(`"Choice":{"oneOf":[{"$ref":"#/components/schemas/A%02d","nullable":false},{"properties":{"kind":{"const":"b"},"other":{"type":"boolean"}}}]}`, n-1))
		r, err := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(parts, ",")+"}"), "Dto"))
		if n == 44 && err != nil || n == 45 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0) {
			t.Fatal("erased aliases or narrowed shape lost source depth", n, err)
		}
	}
}

func TestNullableComposedReferenceMemberExcludedRoles(t *testing.T) {
	for _, source := range []string{
		`{"nullable":true,"allOf":[{"properties":{"kind":{"const":"a"}}}]}`,
		`{"nullable":true,"allOf":[{"properties":{"id":{"type":"string","format":"unsupported"}}}]}`,
		`{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}},"minProperties":0}]}`,
		`{"nullable":true,"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}}},{"properties":{"kind":{"const":"b"}}}]}]}`,
	} {
		rq := request(schemas(`{"Target":`+source+`,"Choice":{"oneOf":[{"$ref":"#/components/schemas/Target","nullable":true},{"properties":{"ok":{"type":"boolean"}}}]}}`), "Dto")
		rq.Options.Validation = true
		if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("unsupported source acquired ordinary member shape", source, r, err)
		}
	}
	for _, owner := range []string{
		`{"allOf":[{"properties":{"node":{"nullable":true,"properties":{"id":{"type":"string"}}}}},{"properties":{"node":{"type":"string"}}}]}`,
		`{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"},"node":{"nullable":true,"properties":{"id":{"type":"string"}}}}},{"properties":{"kind":{"const":"b"}}}]},{"oneOf":[{"properties":{"side":{"const":"x"}}},{"properties":{"side":{"const":"y"}}}]}]}`,
	} {
		ref := "Owner/allOf/0/properties/node"
		if strings.Contains(owner, `"oneOf"`) {
			ref = "Owner/allOf/0/oneOf/0/properties/node"
		}
		input := schemas(`{"Owner":` + owner + `,"Choice":{"oneOf":[{"$ref":"#/components/schemas/` + ref + `","nullable":true},{"properties":{"ok":{"type":"boolean"}}}]}}`)
		if r, err := core.GenerateWithDiagnostics(request(input, "Dto")); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("discarded or ambiguous source acquired variant authority", r, err)
		}
	}
}
