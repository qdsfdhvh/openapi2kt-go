package core_test

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"openapi-kt/core"
)

func stringScalarDefaultDoc(leaf, value, override string) []byte {
	return schemas(`{"Status":` + leaf + `,"Holder":{"properties":{"a/~":{"$ref":"#/components/schemas/Status","default":` + value + override + `}}}}`)
}

func TestReferencedStringEnumScalarDefaultsOwnAnnotation(t *testing.T) {
	for _, leaf := range []string{`{"type":"string","enum":["on","off"]}`, `{"enum":["on","off"]}`} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				doc := stringScalarDefaultDoc(leaf, `"on"`, "")
				r := request(bytes.Replace(doc, []byte(`,"default":"on"`), nil, 1), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal("no-default control", err)
				}
				r.Document = doc
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal("own default", checked, style, err)
				}
				text := source(t, got, "HolderDto")
				if !strings.Contains(text, "val a: org.example.models.models.StatusDto?") && !strings.Contains(text, "org.example.models.models.StatusDto? =") {
					t.Fatal("actual enum type", text)
				}
				if !strings.Contains(text, "`openapi-kt default enum scalar ") || !strings.Contains(text, `"\"on\""`) || strings.Contains(text, "HolderDto.org.example") {
					t.Fatal("actual helper binding", text)
				}
				if source(t, got, "StatusDto") != source(t, baseline, "StatusDto") || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("enum declaration/codec drift")
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("determinism", err)
				}
			}
		}
	}
}

func TestReferencedStringEnumScalarDefaultsAliasesAndSharedHelpers(t *testing.T) {
	doc := schemas(`{"Source":{"properties":{"code":{"type":"string","enum":["on","off"]},"hop":{"$ref":"#/components/schemas/Source/properties/code","nullable":true},"narrow":{"$ref":"#/components/schemas/Source/properties/hop","nullable":false}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/narrow"},"Alias":{"$ref":"#/components/schemas/Bridge"},"Holder":{"properties":{"first":{"$ref":"#/components/schemas/Alias","default":"on"},"second":{"$ref":"#/components/schemas/Alias","default":"off"},"nested":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":"on"}}},"kotlin":{"type":"string"},"serializer":{"type":"string"}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(doc, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, got, "HolderDto")
		if !strings.Contains(text, "org.example.models.models.SourceDto.Code? =") || strings.Count(text, "\nprivate fun `openapi-kt default enum scalar ") != 1 {
			t.Fatal("actual lexical owner/shared helper", text)
		}
		if !strings.Contains(source(t, got, "AliasDto"), "typealias AliasDto = org.example.models.models.SourceDto.Code") {
			t.Fatal("alias ABI")
		}
	}
}

func TestReferencedStringEnumScalarDefaultsValuesAndExcludedRoles(t *testing.T) {
	at := "/components/schemas/Holder/properties/a~1~0/default"
	for _, checked := range []bool{false, true} {
		// Verbatim former builtin scalar guard, now a separately admitted role.
		graduation := request(scalarRefDoc(`{"type":"string","nullable":true,"enum":["a"]}`, `null`, `,"nullable":true`), "Dto")
		graduation.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(graduation); err != nil {
			t.Fatal("former String enum guard", err)
		}
		for _, value := range []string{`"unknown"`, `1`, `true`, `null`, `[]`, `{}`} {
			nativeEnumDefaultFailure(t, stringScalarDefaultDoc(`{"type":"string","enum":["on","off"]}`, value, ""), checked, "E_SCHEMA_INVALID", at)
		}
		for _, leaf := range []string{`{"type":"string","enum":["on"]}`, `{"type":["string","null"],"enum":["on"]}`} {
			r := request(stringScalarDefaultDoc(leaf, `null`, `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("effective nullable", err)
			}
		}
		// Intrinsic nullability can be explicitly narrowed; optional presence is independent.
		nativeEnumDefaultFailure(t, stringScalarDefaultDoc(`{"type":["string","null"],"enum":["on"]}`, `null`, `,"nullable":false`), checked, "E_SCHEMA_INVALID", at)
		for _, leaf := range []string{
			`{"type":"string","enum":["on"],"format":"uuid"}`,
			`{"type":"string","enum":["on"],"minLength":1}`,
			`{"type":"string","enum":["on"],"pattern":"on"}`,
			`{"type":"string","enum":["on"],"default":"on"}`,
			`{"type":"string","enum":["on"],"const":"on"}`,
			`{"type":"string","enum":["on"],"x-native-type":"kotlin.String"}`,
			`{"type":"string","enum":["on"],"x-kotlin-value-class":true}`,
		} {
			r := request(stringScalarDefaultDoc(leaf, `null`, `,"nullable":true`), "Dto")
			r.Options.Validation = checked
			if checked && (leaf == `{"type":"string","enum":["on"],"minLength":1}` || leaf == `{"type":"string","enum":["on"],"pattern":"on"}`) {
				checkedStringEnumRootGraduation(t, r)
				continue
			}
			if leaf == `{"type":"string","enum":["on"],"default":"on"}` {
				independentEnumFactoryGraduation(t, r)
				continue
			}
			if metadataStringEnumFixture(leaf) {
				metadataStringEnumGraduation(t, r)
				continue
			}
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("ungranted terminal", leaf, checked, err)
			}
		}
		for _, body := range []string{
			`{"Status":{"type":"string","enum":["on"]},"Holder":{"properties":{"v":{"allOf":[{"$ref":"#/components/schemas/Status"}],"default":"on"}}}}`,
			`{"Status":{"type":"string","enum":["on"]},"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"v":{"$ref":"#/components/schemas/Status","default":"on"}}},{"properties":{"tag":{"const":"b"}}}]}}`,
		} {
			r := request(schemas(body), "Dto")
			r.Options.Validation = checked
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("ungranted role", body, err)
			}
		}
	}
}

func TestReferencedStringEnumScalarDefaultsPresenceAndIdentity(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, role := range []string{"required", "patch", "discarded"} {
			field := `{"$ref":"#/components/schemas/Status","nullable":true,"default":"on"}`
			holder := `{"required":["v"],"properties":{"v":` + field + `}}`
			if role == "patch" {
				holder = `{"properties":{"v":` + field + `}}`
			}
			if role == "discarded" {
				holder = `{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`
			}
			doc := schemas(`{"Status":{"type":"string","enum":["on","off"]},"Holder":` + holder + `}`)
			if role == "patch" {
				doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":"string","enum":["on","off"]}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
			}
			r := request(doc, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(role, checked, err)
			}
			r.Document = bytes.Replace(doc, []byte(`"default":"on"`), []byte(`"default":"off"`), 1)
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("suppressed default identity", role, checked, err)
			}
			r.Document = bytes.Replace(doc, []byte(`"default":"on"`), []byte(`"default":"unknown"`), 1)
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("suppression skipped vocabulary check", role, err)
			}
		}
	}
	for _, different := range []bool{false, true} {
		other := "on"
		if different {
			other = "off"
		}
		r := request(schemas(`{"Status":{"type":"string","enum":["on","off"]},"Alpha":{"properties":{"v":{"$ref":"#/components/schemas/Status","default":"on"}}},"Zeta":{"properties":{"v":{"$ref":"#/components/schemas/Status","default":"`+other+`"}}}}`), "Dto")
		r.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(source(t, got, "ZetaDto"), "typealias ZetaDto =") == different {
			t.Fatal("default merge identity", different)
		}
	}
}

func TestReferencedStringEnumScalarDefaultsMergedLexicalOwner(t *testing.T) {
	input := schemas(`{"AlphaOwner":{"properties":{"code":{"type":"string","enum":["on","off"]}}},"ZetaOwner":{"properties":{"code":{"type":"string","enum":["on","off"]}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/ZetaOwner/properties/code","default":"on"}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(input, "Dto")
		r.Options.Validation, r.Options.MergeIdenticalModels = checked, true
		report, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, report, "ZetaOwnerDto"), "typealias ZetaOwnerDto = org.example.models.models.AlphaOwnerDto") {
			t.Fatal("owner did not merge")
		}
		text := source(t, report, "HolderDto")
		if !strings.Contains(text, "kotlinx.serialization.serializer<org.example.models.models.AlphaOwnerDto.Code>()") || strings.Contains(text, "ZetaOwnerDto") {
			t.Fatal("helper retained replaced lexical owner", text)
		}
	}
}

func TestReferencedStringEnumScalarDefaultsIndependentLimits(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			members, fields := []string{}, []string{}
			for i := 0; i < count; i++ {
				members = append(members, `"v`+strconv.Itoa(i)+`"`)
				fields = append(fields, `"v`+strconv.Itoa(i)+`":{"$ref":"#/components/schemas/Status","default":"on"}`)
			}
			for index, doc := range [][]byte{stringScalarDefaultDoc(`{"type":"string","enum":[`+strings.Join(members, ",")+`]}`, `"v0"`, ""), schemas(`{"Status":{"type":"string","enum":["on"]},"Holder":{"properties":{` + strings.Join(fields, ",") + `}}}`)} {
				r := request(doc, "Dto")
				r.Options.Validation = checked
				report, err := core.GenerateWithDiagnostics(r)
				if count == 240 && err != nil {
					t.Fatal("boundary", err)
				}
				limitCode := "E_MODEL_LIMIT"
				if index == 0 {
					limitCode = "E_ENUM_LIMIT"
				}
				if count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != limitCode || !reflect.DeepEqual(report, core.GenerationReport{})) {
					t.Fatal("limit widened", err)
				}
			}
		}
		for _, size := range []int{32766, 32767} {
			value := `"` + strings.Repeat("a", size) + `"`
			r := request(stringScalarDefaultDoc(`{"type":"string","enum":[`+value+`]}`, value, ""), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			if size == 32766 && err != nil || size == 32767 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
				t.Fatal("annotation budget", size, err)
			}
		}
	}
}
