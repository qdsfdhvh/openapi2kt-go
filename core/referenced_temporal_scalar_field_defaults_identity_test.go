package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func temporalDefaultOperations(value string) []byte {
	doc := nestedBuiltinDefaultOperations(`{"$ref":"#/components/schemas/Stamp","default":"` + value + `"}`)
	return []byte(strings.TrimSuffix(string(doc), "}") + `,"components":{"schemas":{"Stamp":{"type":"string","format":"date-time","nullable":true}}}}`)
}

func TestReferencedTemporalScalarFieldDefaultsOptimizationAndLexemeLock(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, mode := range []string{"dedup", "merge", "combined"} {
				r := request(temporalDefaultOperations("2024-02-29T12:34:56+02:30"), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				r.Options.DeduplicateModels = mode != "merge"
				r.Options.MergeIdenticalModels = mode != "dedup"
				r.Options.HoistSharedShapes = mode == "combined"
				r.Options.LockHoistedNames = mode == "combined"
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				alias := "BetaResponseDto"
				if mode == "combined" {
					alias = "AlphaResponseDto"
				}
				if !strings.Contains(source(t, got, alias), "typealias "+alias) {
					t.Fatal("reuse", mode)
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				same, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, same) {
					t.Fatal("lock replay", err)
				}
				// Provider codecs normalize time zones, but default identity preserves
				// the original lexeme. Equivalent instants do not collapse source defaults.
				r.Document = temporalDefaultOperations("2024-02-29T10:04:56Z")
				changed, err := core.GenerateWithDiagnostics(r)
				if mode == "combined" {
					if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
						t.Fatal("lexeme drift", err)
					}
				} else if err != nil || reflect.DeepEqual(got, changed) {
					t.Fatal("lexeme identity", err)
				}
				r.PreviousNameLock = nil
				doc := string(temporalDefaultOperations("2024-02-29T12:34:56+02:30"))
				at := strings.Index(doc, `"/b":`)
				r.Document = []byte(doc[:at] + strings.Replace(doc[at:], "2024-02-29T12:34:56+02:30", "2024-02-29T10:04:56Z", 1))
				separate, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"AlphaResponseDto", "BetaResponseDto"} {
					if strings.Contains(source(t, separate, name), "typealias "+name) {
						t.Fatal("different defaults collapsed", mode)
					}
				}
			}
		}
	}
	for _, role := range []string{"required", "patch"} {
		holder := `{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Stamp","default":"2024-02-29T00:00:00Z"}}}`
		doc := schemas(`{"Stamp":{"type":"string","format":"date-time","nullable":true},"Holder":` + holder + `}`)
		if role == "patch" {
			holder = strings.Replace(holder, `"required":["v"],`, "", 1)
			doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Stamp":{"type":"string","format":"date-time","nullable":true}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
		}
		r := request(doc, "Dto")
		first, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		r.Document = []byte(strings.ReplaceAll(string(doc), "2024-02-29T00:00:00Z", "2024-03-01T00:00:00Z"))
		second, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("suppressed annotation identity", role, err)
		}
	}
}

func TestReferencedTemporalScalarFieldDefaultsReferenceAndInheritanceGuards(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, own := range []string{"", `,"default":null`} {
			// A `$ref` wrapper now owns a factory on its own alias, but a field that
			// references it must still not inherit the value: the wrapper case is
			// asserted positively below instead of being rejected wholesale.
			for _, body := range []string{} {
				r := request(schemas(body), "Dto")
				r.Options.Validation = checked
				if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("inherited default authorized", body, err)
				}
			}
			// Graduated: the wrapper alias owns the factory while the referencing
			// field keeps its neutral optional presence. A field that also carries
			// its own annotation is a different, still-rejected role.
			if own != "" {
				continue
			}
			wrapper := request(schemas(`{"Day":{"type":"string","format":"date"},"Alias":{"$ref":"#/components/schemas/Day","default":"2024-02-29"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias"}}}}`), "Dto")
			wrapper.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(wrapper)
			if err != nil {
				t.Fatal("wrapper factory lost", err)
			}
			if text := source(t, report, "AliasDto"); !strings.Contains(text, "public fun `AliasDto default`(): AliasDto =") {
				t.Fatal("wrapper did not own the factory", text)
			}
			if text := source(t, report, "HolderDto"); !strings.Contains(text, "public val v: kotlinx.datetime.LocalDate? = null,") {
				t.Fatal("wrapper default was inherited by a field", text)
			}
		}
		for _, ref := range []string{"#/components/schemas/Missing", "#/components/schemas/Day~2", "#/components/schemas/Day/example", "https://example.test/date.json"} {
			r := request(schemas(`{"Day":{"type":"string","format":"date","example":{"type":"string"}},"Holder":{"properties":{"v":{"$ref":"`+ref+`","default":null}}}}`), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(ref, err)
			}
		}
		for _, body := range []string{
			`{"Day":{"type":"string","format":"date"},"Hops":{"properties":{"a":{"$ref":"#/components/schemas/Hops/properties/b"},"b":{"$ref":"#/components/schemas/Hops/properties/a"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/a","default":null}}}}`,
			`{"Day":{"type":"string","format":"date"},"Alias":{"$ref":"#/components/schemas/Alias"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias","default":null}}}}`,
			`{"Day":{"type":"string","format":true},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Day","default":null}}}}`,
			`{"Day":{"type":"string","format":"date"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Day","default":"2024-02-29"}}},"Unused":{"properties":{"bad":{"type":"string","format":"date-time","minLength":0}}}}`,
			`{"Day":{"type":"string","format":"date"},"Variant":{"properties":{"v":{"$ref":"#/components/schemas/Day","default":"2024-02-29"}}},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Variant"},{"properties":{"other":{"type":"integer"}}}]}}`,
		} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("source/role guard", body, err)
			}
		}
	}
}
