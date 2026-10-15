package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableObjectMembersOriginalValidationAndNullOverlap(t *testing.T) {
	for _, member := range []string{
		`{"nullable":true,"properties":{"kind":{"const":"a"},"id":{"type":"string","format":"unsupported"}}}`,
		`{"nullable":true,"properties":{"kind":{"const":"a"}},"required":["missing"]}`,
		`{"nullable":true,"properties":{"kind":{"const":"a"}},"unexpected":true}`,
		`{"nullable":true,"allOf":[{"properties":{"kind":{"const":"a"}}}]}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(nullableObjectMemberDocument(member), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("member nullability hid invalid source", member, got, err)
		}
	}
	input := schemas(`{"Choice":{"oneOf":[{"nullable":true,"properties":{"kind":{"const":"a"}}},{"type":["object","null"],"properties":{"kind":{"const":"b"}}}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, w := range got.Warnings {
		if w.Code == "W_ONEOF_NULL_OVERLAP" {
			count++
			if w.Pointer == nil || *w.Pointer != "/components/schemas/Choice/oneOf" {
				t.Fatal(w)
			}
		}
	}
	if count != 1 {
		t.Fatal("overlapping member null intent lacks original warning", got.Warnings)
	}
}

func TestNullableObjectMembersRootBranchesAndSlots(t *testing.T) {
	for _, width := range []int{239, 240} {
		branches := []string{}
		for i := 0; i < width; i++ {
			intent := ""
			if i == 0 {
				intent = `"nullable":true,`
			}
			branches = append(branches, fmt.Sprintf(`{%s"properties":{"kind":{"const":"k%d"}}}`, intent, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Choice":{"oneOf":[`+strings.Join(branches, ",")+`]}}`), "Dto"))
		if width == 239 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("member null bypassed root branch budget", width, got, err)
		}
	}
	for _, width := range []int{239, 240} {
		props, required := []string{`"kind":{"const":"a"}`}, []string{}
		for i := 0; i < width; i++ {
			props = append(props, fmt.Sprintf(`"p%d":{"type":"boolean"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		member := `{"nullable":true,"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
		got, err := core.GenerateWithDiagnostics(request(nullableObjectMemberDocument(member), "Dto"))
		if width == 239 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("nullable member bypassed physical slots", width, got, err)
		}
	}
}
