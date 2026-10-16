package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const internalAllOfDefaultTarget = `{"allOf":[{"properties":{"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer","default":7}}}]}`

func internalAllOfDefaultDocument(target, field string) []byte {
	return schemas(`{"Holder":{"properties":{"v":` + field + `}},"Owner":{"properties":{"node":` + target + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Hops":{"properties":{"via":{"$ref":"#/components/schemas/Alias","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/via"}}`)
}

func TestInternalAllOfFieldDefaults(t *testing.T) {
	for _, reference := range []string{"Owner/properties/node", "Alias", "Bridge"} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				field := `{"$ref":"#/components/schemas/` + reference + `","default":{"n":2,"label":"initial"}}`
				r := request(internalAllOfDefaultDocument(referencedAllOfTarget, field), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(reference, checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "public val v: org.example.models.models.OwnerDto.Node? = `openapi-kt model default org example models models OwnerDto Node`(") || !strings.Contains(text, `{\"label\":\"initial\",\"n\":2}`) {
					t.Fatal("lost actual internal DTO default", text)
				}
				r.Document = internalAllOfDefaultDocument(referencedAllOfTarget, `{"$ref":"#/components/schemas/`+reference+`"}`)
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil || source(t, baseline, "OwnerDto") != source(t, got, "OwnerDto") || !reflect.DeepEqual(baseline.Warnings, got.Warnings) {
					t.Fatal("default changed lexical owner bytes/warnings", err)
				}
			}
		}
	}
}

func TestInternalAllOfDefaultLexicalScopes(t *testing.T) {
	for _, tc := range []struct{ owner, path, actual, value string }{
		{`{"properties":{"nodes":{"type":"array","items":` + internalAllOfDefaultTarget + `}}}`, "Owner/properties/nodes/items", "OwnerDto.NodesInner", `{"label":"x"}`},
		{`{"properties":{"nodes":{"additionalProperties":` + internalAllOfDefaultTarget + `}}}`, "Owner/properties/nodes/additionalProperties", "OwnerDto.NodesValue", `{"label":"x"}`},
		{`{"type":"array","items":` + internalAllOfDefaultTarget + `}`, "Owner/items", "OwnerDtoInner", `{"label":"x"}`},
		{`{"additionalProperties":` + internalAllOfDefaultTarget + `}`, "Owner/additionalProperties", "OwnerDtoValue", `{"label":"x"}`},
		{`{"properties":{"wrapper":{"properties":{"node":` + internalAllOfDefaultTarget + `}}}}`, "Owner/properties/wrapper/properties/node", "OwnerDto.Wrapper.Node", `{"label":"x"}`},
		{`{"oneOf":[{"allOf":[{"properties":{"kind":{"const":"all"},"label":{"type":"string"}},"required":["label"]},{"properties":{"n":{"type":"integer","default":7}}}]},{"properties":{"kind":{"const":"other"},"label":{"type":"integer"}},"required":["label"]}]}`, "Owner/oneOf/0", "OwnerDto.All", `{"kind":"all","label":"x"}`},
		{`{"properties":{"n/~":` + internalAllOfDefaultTarget + `}}`, "Owner/properties/n~1~0", "", `{"label":"escaped"}`},
	} {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"Owner":` + tc.owner + `,"Alias":{"$ref":"#/components/schemas/` + tc.path + `"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":` + tc.value + `}}}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(tc.path, checked, err)
			}
			if tc.actual != "" && !strings.Contains(source(t, got, "HolderDto"), "public val v: org.example.models.models."+tc.actual+"? =") {
				t.Fatal("lost actual lexical type", tc.path, source(t, got, "HolderDto"))
			}
		}
	}
}

func TestInternalAllOfDefaultNullability(t *testing.T) {
	for _, intent := range []string{"", `,"nullable":true`, `,"nullable":false`} {
		// Member-derived nullability is a retained internal Model property.
		target := strings.TrimSuffix(internalAllOfDefaultTarget, "}") + intent + `}`
		target = strings.Replace(target, `"allOf":[{`, `"allOf":[{"nullable":true,`, 1)
		for _, reference := range []string{"Owner/properties/node", "Alias", "Bridge"} {
			for _, consumer := range []string{"", `,"nullable":true`, `,"nullable":false`} {
				for _, checked := range []bool{false, true} {
					accepted := intent != `,"nullable":false` || reference == "Bridge"
					if consumer != "" {
						accepted = consumer == `,"nullable":true`
					}
					input := internalAllOfDefaultDocument(target, `{"$ref":"#/components/schemas/`+reference+`"`+consumer+`,"default":null}`)
					if !accepted {
						carrierDefaultFailure(t, input, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
						continue
					}
					r := request(input, "Dto")
					r.Options.Validation = checked
					if _, err := core.GenerateWithDiagnostics(r); err != nil {
						t.Fatal("effective nullable internal view", reference, intent, consumer, err)
					}
				}
			}
		}
	}
	carrierDefaultFailure(t, internalAllOfDefaultDocument(internalAllOfDefaultTarget, `{"$ref":"#/components/schemas/Owner/properties/node","default":null}`), false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default")
}

func TestInternalAllOfDefaultPresenceAndExclusions(t *testing.T) {
	field := `{"$ref":"#/components/schemas/Alias","nullable":true,"default":{"label":"x"}}`
	for _, checked := range []bool{false, true} {
		input := internalAllOfDefaultDocument(internalAllOfDefaultTarget, field)
		r := request([]byte(strings.Replace(string(input), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1)), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") {
			t.Fatal("required internal default acquired initializer", err)
		}
		requestDoc := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{"Owner":{"properties":{"node":` + internalAllOfDefaultTarget + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"}}}}`
		r.Document = []byte(requestDoc)
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("Patch internal default", err)
		}
		found := false
		for _, file := range got.Artifacts.Files {
			found = found || strings.Contains(file.Content, ".Patch<org.example.models.models.OwnerDto.Node>")
		}
		if !found {
			t.Fatal("lost internal Patch type")
		}
		discarded := schemas(`{"Owner":{"properties":{"node":` + internalAllOfDefaultTarget + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
		r.Document = discarded
		got, err = core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "HolderDto"), "openapi-kt model default") {
			t.Fatal("discarded internal default leaked", err)
		}
	}
	for _, target := range []string{
		`{"allOf":[{"properties":{"label":{"type":"string"}}},{"properties":{"label":{"type":"string"}}}]}`,
		`{"allOf":[{"properties":{"label":{"type":"string"}},"additionalProperties":true},{"properties":{"n":{"type":"integer"}}}]}`,
		`{"allOf":[{},{}]}`,
	} {
		r := request(internalAllOfDefaultDocument(target, field), "Dto")
		if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("excluded source shape authorized internal default", target, err)
		}
	}
	// Hierarchy-owned nested constructor Models retain a separate default role.
	hierarchy := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind","mapping":{"first":"First","second":"Second"}}},"First":{"properties":{"kind":{"const":"first"},"node":` + internalAllOfDefaultTarget + `}},"Second":{"properties":{"kind":{"const":"second"},"id":{"type":"integer"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/First/properties/node","default":{"label":"x"}}}}}`)
	baseline := []byte(strings.Replace(string(hierarchy), `,"default":{"label":"x"}`, "", 1))
	if _, err := core.GenerateWithDiagnostics(request(baseline, "Dto")); err != nil {
		t.Fatal("hierarchy-owned no-default reference must already work", err)
	}
	carrierDefaultFailure(t, hierarchy, false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/First/properties/node/allOf")
}

func TestInternalAllOfDefaultDiagnosticsAndSuppression(t *testing.T) {
	for _, tc := range []struct{ value, code, tail string }{
		{`{"label":false}`, "E_SCHEMA_INVALID", "/label"},
		{`{}`, "E_SCHEMA_INVALID", ""},
		{`{"z/~":1,"label":false}`, "E_SCHEMA_INVALID", "/z~1~0"},
		{`{"label":false,"z/~":1}`, "E_SCHEMA_INVALID", "/label"},
		{`{"label":"` + strings.Repeat("x", 32767) + `"}`, "E_MODEL_LIMIT", "/label"},
	} {
		for _, checked := range []bool{false, true} {
			field := `{"$ref":"#/components/schemas/Alias","nullable":true,"default":` + tc.value + `}`
			input := internalAllOfDefaultDocument(internalAllOfDefaultTarget, field)
			carrierDefaultFailure(t, input, checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
			required := []byte(strings.Replace(string(input), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1))
			carrierDefaultFailure(t, required, checked, tc.code, "/components/schemas/Holder/properties/v/default"+tc.tail)
			requestDoc := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":{"schemas":{"Owner":{"properties":{"node":` + internalAllOfDefaultTarget + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"}}}}`
			carrierDefaultFailure(t, []byte(requestDoc), checked, tc.code, "/paths/~1p/post/requestBody/content/application~1json/schema/properties/v/default"+tc.tail)
			discarded := schemas(`{"Owner":{"properties":{"node":` + internalAllOfDefaultTarget + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Holder":{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}}`)
			carrierDefaultFailure(t, discarded, checked, tc.code, "/components/schemas/Holder/allOf/0/properties/v/default"+tc.tail)
		}
	}
	// A target field's own default is valid independently but not inherited by refs.
	annotated := strings.TrimSuffix(internalAllOfDefaultTarget, "}") + `,"default":{"label":"target"}}`
	carrierDefaultFailure(t, internalAllOfDefaultDocument(annotated, `{"$ref":"#/components/schemas/Alias","default":{"label":"outer"}}`), false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Owner/properties/node/default")
}
