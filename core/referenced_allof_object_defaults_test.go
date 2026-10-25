package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const referencedAllOfTarget = `{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer"},"child":{"properties":{"flag":{"type":"boolean"}},"default":{"flag":true}}}}]}`

func referencedAllOfDocument(target, field string) []byte {
	return schemas(`{"Holder":{"properties":{"v":` + field + `}},"Value":` + target + `,"Alias":{"$ref":"#/components/schemas/Value"}}`)
}

func TestReferencedAllOfObjectFieldDefaults(t *testing.T) {
	for _, reference := range []string{"Value", "Alias"} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				field := `{"$ref":"#/components/schemas/` + reference + `","default":{"n":2,"label":"initial"}}`
				r := request(referencedAllOfDocument(referencedAllOfTarget, field), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(reference, checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "public val v: org.example.models.models.ValueDto? = `openapi-kt model default org example models models ValueDto`(") || !strings.Contains(text, `{\"label\":\"initial\",\"n\":2}`) {
					t.Fatal("canonical actual named DTO default", text)
				}
				// Adding a consumer annotation does not alter the retained target.
				r.Document = referencedAllOfDocument(referencedAllOfTarget, `{"$ref":"#/components/schemas/`+reference+`"}`)
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || source(t, baseline, "ValueDto") != source(t, got, "ValueDto") || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("default changed target bytes or warnings", err)
				}
			}
		}
	}
	// Graduate the original ordinary allOf target guard, including root own fields.
	input := schemas(`{"Inner":{"allOf":[{"type":"object","properties":{"a":{"type":"string"}}}],"properties":{"b":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"b":"x"}}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
		t.Fatal("original allOf object default guard did not graduate", err)
	}
}

func TestReferencedAllOfDefaultWarningOwnership(t *testing.T) {
	target := strings.Replace(referencedAllOfTarget, `"n":{"type":"integer"}`, `"n":{"type":"integer","enum":[1,2]}`, 1)
	for _, checked := range []bool{false, true} {
		r := request(referencedAllOfDocument(target, `{"$ref":"#/components/schemas/Value","default":{"label":"x","n":2}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		r.Document = referencedAllOfDocument(target, `{"$ref":"#/components/schemas/Value"}`)
		baseline, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
			t.Fatalf("default changed source warnings (validation=%v): got %v baseline %v: %v", checked, got.Warnings, baseline.Warnings, err)
		}
	}
}

func TestReferencedAllOfDefaultThroughInternalBridge(t *testing.T) {
	definitions := `"Value":` + referencedAllOfTarget + `,"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Alias":{"$ref":"#/components/schemas/Hops/properties/hop"},`
	for _, reference := range []string{"Alias", "Hops/properties/hop"} {
		for _, checked := range []bool{false, true} {
			field := `{"$ref":"#/components/schemas/` + reference + `","default":{"label":"bridge"}}`
			for _, required := range []bool{false, true} {
				requirement := ""
				if required {
					requirement = `,"required":["v"]`
				}
				input := schemas(`{` + definitions + `"Holder":{"properties":{"v":` + field + `}` + requirement + `}}`)
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(reference, required, checked, err)
				}
				if strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") == required {
					t.Fatal("bridge default/required presence changed")
				}
				bad := []byte(strings.Replace(string(input), `"label":"bridge"`, `"label":false`, 1))
				carrierDefaultFailure(t, bad, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/label")
			}
			discarded := schemas(`{` + definitions + `"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			r := request(discarded, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("discarded internal bridge", err)
			}
			carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"label":"bridge"`, `"label":false`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/label")
		}
	}
	// An internal hop's own nullable intent precedes the terminal, while an
	// explicit consumer false still narrows it before optional presence.
	nullableBridge := schemas(`{` + strings.Replace(definitions, `"hop":{"$ref":"#/components/schemas/Value"}`, `"hop":{"$ref":"#/components/schemas/Value","nullable":true}`, 1) + `"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":null}}}}`)
	if _, err := core.GenerateWithDiagnostics(request(nullableBridge, "Dto")); err != nil {
		t.Fatal("internal nullable intent lost", err)
	}
	carrierDefaultFailure(t, []byte(strings.Replace(string(nullableBridge), `"default":null`, `"nullable":false,"default":null`, 1)), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	// An intermediate ordinary field may have its own valid default, but a
	// reference chain consumer cannot inherit or silently ignore that annotation.
	annotatedHop := schemas(`{` + strings.Replace(definitions, `"hop":{"$ref":"#/components/schemas/Value"}`, `"hop":{"$ref":"#/components/schemas/Value","default":{"label":"hop"}}`, 1) + `"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":{"label":"outer"}}}}}`)
	carrierDefaultFailure(t, annotatedHop, false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/hop/default")
	// The original internal-owned allOf default guard now graduates while
	// retaining the no-default reference's lexical constructor and serializer.
	owned := schemas(`{"Owner":{"properties":{"node":` + referencedAllOfTarget + `}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/node","default":{"label":"x"}}}}}`)
	baseline := []byte(strings.Replace(string(owned), `,"default":{"label":"x"}`, "", 1))
	if _, err := core.GenerateWithDiagnostics(request(baseline, "Dto")); err != nil {
		t.Fatal("internal-owned no-default reference must already work", err)
	}
	got, err := core.GenerateWithDiagnostics(request(owned, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "HolderDto"), "org.example.models.models.OwnerDto.Node? =") {
		t.Fatal("internal allOf default guard did not graduate", err)
	}
}

func TestReferencedAllOfDefaultNullability(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, targetNullable := range []bool{false, true} {
			target := referencedAllOfTarget
			if targetNullable {
				target = strings.TrimSuffix(target, "}") + `,"nullable":true}`
			}
			for _, tc := range []struct {
				consumer string
				accepted bool
			}{
				{"", targetNullable},
				{`,"nullable":true`, true},
				{`,"nullable":false`, false},
			} {
				input := referencedAllOfDocument(target, `{"$ref":"#/components/schemas/Value"`+tc.consumer+`,"default":null}`)
				if !tc.accepted {
					carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
					continue
				}
				r := request(input, "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || !strings.Contains(source(t, got, "HolderDto"), "ValueDto? = null") {
					t.Fatal("effective nullable referenced allOf default", err)
				}
			}
		}
		// Nearest alias nullability, followed by explicit consumer narrowing.
		input := schemas(`{"Value":` + referencedAllOfTarget + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":null}}}}`)
		r := request(input, "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("nullable alias", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(input), `"default":null`, `"nullable":false,"default":null`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
	}
}

func TestReferencedAllOfDefaultMemberDiagnostics(t *testing.T) {
	for _, tc := range []struct{ value, tail string }{
		{`{"label":false}`, "/label"},
		{`{"label":"x","n":false}`, "/n"},
		{`{"label":"x","z/~":1}`, "/z~1~0"},
		{`{"label":"x","child":{"flag":"wrong"}}`, "/child/flag"},
		{`{}`, ""},
		{`[]`, ""},
	} {
		for _, checked := range []bool{false, true} {
			input := referencedAllOfDocument(referencedAllOfTarget, `{"$ref":"#/components/schemas/Alias","default":`+tc.value+`}`)
			carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default"+tc.tail)
		}
	}
	bounded := strings.Replace(referencedAllOfTarget, `"n":{"type":"integer"}`, `"n":{"type":"integer","minimum":1}`, 1)
	carrierDefaultFailure(t, referencedAllOfDocument(bounded, `{"$ref":"#/components/schemas/Value","default":{"label":"x","n":0}}`), true, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/n")
	large := `{"$ref":"#/components/schemas/Value","default":{"label":"` + strings.Repeat("x", 32767) + `"}}`
	carrierDefaultFailure(t, referencedAllOfDocument(referencedAllOfTarget, large), false, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/v/default/label")
}

func TestReferencedAllOfDefaultExclusions(t *testing.T) {
	for _, target := range []string{
		`{"allOf":[{"properties":{"label":{"type":"string"}}},{"properties":{"label":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"label":{"type":"string"}},"additionalProperties":true},{"properties":{"n":{"type":"integer"}}}]}`,
		`{"allOf":[{"properties":{"label":{"const":"x"}}}]}`,
		`{"allOf":[{},{}]}`,
	} {
		for _, checked := range []bool{false, true} {
			r := request(referencedAllOfDocument(target, `{"$ref":"#/components/schemas/Value","default":{"label":"x"}}`), "Dto")
			r.Options.Validation = checked
			if checked && target == allOfOverrideDuplicateLabel {
				allOfDefaultOverridePositive(t, r)
				continue
			}
			report, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("excluded allOf target default leaked", target, err)
			}
		}
	}
	// Definitions may own their root default as an opt-in factory; the
	// consuming role still never inherits it.
	withDefault := strings.TrimSuffix(referencedAllOfTarget, "}") + `,"default":{"label":"target"}}`
	if got, err := core.GenerateWithDiagnostics(request(referencedAllOfDocument(withDefault, `{"$ref":"#/components/schemas/Value"}`), "Dto")); err != nil {
		t.Fatal(err)
	} else if text := source(t, got, "ValueDto"); !strings.Contains(text, "public fun default(): ValueDto =") {
		t.Fatal("allOf component factory missing", text)
	}
	intermediate := schemas(`{"Value":` + referencedAllOfTarget + `,"Alias":{"$ref":"#/components/schemas/Value","default":{"label":"alias"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":{"label":"x"}}}}}`)
	carrierDefaultFailure(t, intermediate, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Alias/default")
	// Preserve the original rejecting collection case as a graduation regression:
	// a consuming referenced collection now owns its outer default annotation.
	collection := schemas(`{"Value":` + referencedAllOfTarget + `,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Rows","default":[{"label":"x"}]}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(collection, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt default tree Alias") {
			t.Fatal("referenced allOf collection default did not graduate", err)
		}
	}
	hierarchy := schemas(`{"Base":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"leaf":"#/components/schemas/Leaf"}}},"Leaf":{"allOf":[{"properties":{"label":{"type":"string"}}}]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Leaf","default":{"kind":"leaf","label":"x"}}}}}`)
	baseline := []byte(strings.Replace(string(hierarchy), `,"default":{"kind":"leaf","label":"x"}`, "", 1))
	if _, err := core.GenerateWithDiagnostics(request(baseline, "Dto")); err != nil {
		t.Fatal("hierarchy exclusion must retain its no-default reference", err)
	}
	carrierDefaultFailure(t, hierarchy, true, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Leaf/allOf")
}

func TestReferencedAllOfDefaultBeforePresenceAndOverride(t *testing.T) {
	for _, checked := range []bool{false, true} {
		field := `{"$ref":"#/components/schemas/Value","nullable":true,"default":{"label":"x"}}`
		input := referencedAllOfDocument(referencedAllOfTarget, field)
		required := []byte(strings.Replace(string(input), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1))
		r := request(required, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), " = ") {
			t.Fatal("required reference acquired initializer", err)
		}
		requestDoc := string(carrierRequestDocument(field, false))
		requestDoc = strings.TrimSuffix(requestDoc, "}") + `,"components":{"schemas":{"Value":` + referencedAllOfTarget + `}}}`
		r.Document = []byte(requestDoc)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("Patch allOf reference", err)
		}
		found := false
		for _, file := range got.Artifacts.Files {
			found = found || strings.Contains(file.Content, ".Patch<org.example.models.models.ValueDto>")
		}
		if !found {
			t.Fatal("nullable request reference lost Patch")
		}
		carrierDefaultFailure(t, []byte(strings.Replace(requestDoc, `"default":{"label":"x"}`, `"default":{"label":false}`, 1)), checked, "E_SCHEMA_INVALID", "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default/label")
		discarded := schemas(`{"Value":` + referencedAllOfTarget + `,"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		r.Document = discarded
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") {
			t.Fatal("discarded reference leaked initializer", err)
		}
		carrierDefaultFailure(t, []byte(strings.Replace(string(discarded), `"default":{"label":"x"}`, `"default":{"label":false}`, 1)), checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/allOf/0/properties/v/default/label")
	}
}
