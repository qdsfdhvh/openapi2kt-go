package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func selfMergeObject(name, minimum, value string) string {
	ref := `{"$ref":"#/components/schemas/` + name + `"}`
	return `{"properties":{"value":{"type":"integer","minimum":` + minimum + `,"default":` + value + `},"next":` + ref + `,"children":{"type":"array","items":` + ref + `},"byName":{"type":"object","additionalProperties":` + ref + `},"meta":{"properties":{"parent":` + ref + `}}}}`
}

func TestMergeOwnedSelfReferencesPreserveShapeChecksAndNominalBoundaries(t *testing.T) {
	input := schemas(`{"A":` + selfMergeObject("A", "2", "2") + `,"Z":` + selfMergeObject("Z", "2", "2") + `,"DifferentBound":` + selfMergeObject("DifferentBound", "1", "2") + `,"DifferentDefault":` + selfMergeObject("DifferentDefault", "2", "3") + `,"MutualA":{"properties":{"next":{"$ref":"#/components/schemas/MutualB"}}},"MutualB":{"properties":{"next":{"$ref":"#/components/schemas/MutualA"}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		r := request(input, "Dto")
		r.Options.Validation = true
		r.Options.ModelStyle = style
		unmerged, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, unmerged, "ZDto"), "typealias") {
			t.Fatal("Default-off changed recursive nominal identity", err)
		}
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "ZDto"), "public typealias ZDto = org.example.models.models.ADto") {
			t.Fatal("Equivalent owned self graphs did not merge", err)
		}
		for _, name := range []string{"DifferentBoundDto", "DifferentDefaultDto", "MutualADto", "MutualBDto"} {
			if strings.Contains(source(t, got, name), "typealias "+name) {
				t.Fatal("Unequal or mutual graph collapsed", name)
			}
		}
		if !reflect.DeepEqual(unmerged.Warnings, got.Warnings) || !strings.Contains(source(t, got, "ADto"), "value >= 2") {
			t.Fatal("Recursive merge changed warnings/checks")
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("Recursive merging is nondeterministic", err)
		}
		for _, excluded := range []string{"ADto", "ZDto"} {
			r.Options.ExcludeMergeModels = []string{"org.example.models.models." + excluded}
			separate, err := core.GenerateWithDiagnostics(r)
			if err != nil || strings.Contains(source(t, separate, "ADto"), "typealias") || strings.Contains(source(t, separate, "ZDto"), "typealias") {
				t.Fatal("Excluded recursive owner was absorbed in either direction", excluded, err)
			}
		}
	}
}

func TestMergeOwnedSelfReferencesDoNotNormalizeExternalNominalEdges(t *testing.T) {
	for _, pair := range []string{
		`"A":{"properties":{"next":{"$ref":"#/components/schemas/A"}}},"Z":{"properties":{"next":{"$ref":"#/components/schemas/A"}}}`,
		`"A":{"properties":{"next":{"$ref":"#/components/schemas/A"}}},"Z":{"properties":{"next":{"$ref":"#/components/schemas/Z"}}},"Native":{"properties":{"value":{"type":"string","x-native-type":"org.example.models.models.ADto"}}}`,
		`"A":{"properties":{"next":{"$ref":"#/components/schemas/B"}}},"B":{"properties":{"next":{"$ref":"#/components/schemas/A"}}},"Z":{"properties":{"next":{"$ref":"#/components/schemas/Z"}}}`,
	} {
		r := request(schemas(`{`+pair+`}`), "Dto")
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || strings.Contains(source(t, got, "ADto"), "typealias") || strings.Contains(source(t, got, "ZDto"), "typealias") {
			t.Fatal("A reference to another owner acquired self identity", pair, err)
		}
	}
}
