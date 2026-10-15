package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumCollectionDefaultsActualDedupAndMerge(t *testing.T) {
	shape := `{"properties":{"codes":{"type":"array","items":{"enum":["one","two"]},"default":["one"]},"labels":{"additionalProperties":{"enum":["yes","no"]},"default":{"a":"yes"}}}}`
	changed := strings.Replace(shape, `"default":["one"]`, `"default":["two"]`, 1)
	operation := func(id, schema string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + schema + `}}}}}}`
	}
	for _, checked := range []bool{false, true} {
		for _, mode := range []string{"dedup", "merge"} {
			doc := `{"openapi":"3.1.0","paths":{"/a":` + operation("alpha", shape) + `,"/b":` + operation("beta", shape) + `,"/c":` + operation("gamma", changed) + `}}`
			if mode == "merge" {
				doc = `{"openapi":"3.1.0","components":{"schemas":{"Alpha":` + shape + `,"Zeta":` + shape + `,"Different":` + changed + `,"Use":{"properties":{"list":{"$ref":"#/components/schemas/Zeta/properties/codes/items"},"map":{"$ref":"#/components/schemas/Zeta/properties/labels/additionalProperties"}}}}},"paths":{"/a":` + operation("one", shape) + `}}`
			}
			r := request([]byte(doc), "Dto")
			r.Options.Validation = checked
			r.Options.DeduplicateModels = mode == "dedup"
			r.Options.MergeIdenticalModels = mode == "merge"
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(mode, err)
			}
			canonical := "api/models/AlphaResponseDto"
			alias := "BetaResponseDto"
			distinct := "GammaResponseDto"
			if mode == "merge" {
				canonical = "models/AlphaDto"
				alias = "ZetaDto"
				distinct = "DifferentDto"
			}
			if !strings.Contains(source(t, got, alias), "public typealias") || strings.Contains(source(t, got, distinct), "public typealias") {
				t.Fatal("Optimization lost default identity", mode)
			}
			text := source(t, got, canonical)
			owner := "org.example.models." + strings.ReplaceAll(canonical, "/", ".")
			for _, leaf := range []string{"CodesInner", "LabelsValue"} {
				if !strings.Contains(text, owner+"."+leaf) {
					t.Fatal("Helper lost canonical owner", mode, leaf, text)
				}
			}
			if mode == "merge" {
				uses := source(t, got, "UseDto")
				if !strings.Contains(uses, owner+".CodesInner") || !strings.Contains(uses, owner+".LabelsValue") || !strings.Contains(source(t, got, "OneResponseDto"), "public typealias") {
					t.Fatal("Merged original enum leaf reference lost", uses)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Optimization nondeterministic", mode, err)
			}
		}
	}
}

func TestStringEnumCollectionDefaultsMapUsesExistingValueClassifier(t *testing.T) {
	for _, name := range []string{"companion", "serializer", "kotlin", "kotlinx"} {
		for _, checked := range []bool{false, true} {
			r := request(schemas(`{"Holder":{"properties":{"`+name+`":{"additionalProperties":{"enum":["one"]},"default":{"a":"one"}}}}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "HolderDto"), "Value>") {
				t.Fatal("Map default changed existing lexical name", name, err)
			}
		}
	}
}
