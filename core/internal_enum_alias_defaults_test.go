package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInternalEnumAliasFactoryExistingPathRoles(t *testing.T) {
	for _, tc := range []struct{ owner, path string }{
		{`{"type":"array","items":{"type":"string","enum":["on","off"]}}`, "Owner/items"},
		{`{"additionalProperties":{"type":"string","enum":["on","off"]}}`, "Owner/additionalProperties"},
		{`{"allOf":[{"properties":{"state":{"type":"string","enum":["on","off"]}}},{"properties":{"other":{"type":"integer"}}}]}`, "Owner/allOf/0/properties/state"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Owner":`+tc.owner+`,"Alias":{"$ref":"#/components/schemas/`+tc.path+`","default":"off"}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(tc.path, checked, err)
			}
			if !strings.Contains(source(t, got, "AliasDto"), "public fun `AliasDto default`()") {
				t.Fatal("existing internal declaration lost factory", tc.path)
			}
		}
	}
}

func TestInternalEnumAliasFactoryInvalidAnnotations(t *testing.T) {
	for _, tc := range []struct{ leaf, value, nullable, pointer string }{
		{`{"enum":["on","off"]}`, `"future"`, "", "/components/schemas/Alias/default"},
		{`{"enum":["on","off"]}`, `null`, "", "/components/schemas/Alias/default"},
		{`{"type":"string","format":"date","enum":["on","off"]}`, `7`, "", "/components/schemas/Alias/default"},
		{`{"enum":[null,"on"]}`, `null`, `,"nullable":false`, "/components/schemas/Alias/default"},
		{`{"enum":["on","off"],"default":"future"}`, `"on"`, "", "/components/schemas/Owner/properties/state/default"},
		{`{"enum":["on","on"]}`, `"on"`, "", "/components/schemas/Owner/properties/state/enum/1"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Owner":{"properties":{"state":`+tc.leaf+`}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":`+tc.value+tc.nullable+`}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("internal factory lost strict original annotation", tc, checked, err)
			}
		}
	}
}

func TestInternalEnumAliasFactoryRejectedSources(t *testing.T) {
	for _, body := range []string{
		`{"Owner":{"properties":{"state":{"enum":["on"],"format":"unsupported"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":"on"}}`,
		`{"Owner":{"properties":{"state":{"enum":["on"],"const":"on"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":"on"}}`,
		`{"Owner":{"properties":{"state":{"type":"string","enum":["on"],"minLength":1}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":"on"}}`,
		`{"Owner":{"properties":{"state":{"enum":["on"],"x-kotlin-value-class":true}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":"on"}}`,
		`{"Owner":{"properties":{"state":{"$ref":"#/components/schemas/Alias"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":"on"}}`,
		`{"Owner":{"properties":{"state":{"type":"integer","enum":[1,2]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":1}}`,
		`{"Owner":{"type":"array","items":{"type":"string","enum":["on"],"default":"on"}},"Alias":{"$ref":"#/components/schemas/Owner/items","default":"on"}}`,
		`{"Owner":{"type":"object","default":{"state":{"enum":["on"]}}},"Alias":{"$ref":"#/components/schemas/Owner/default/state","default":"on"}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			if checked && body == `{"Owner":{"properties":{"state":{"type":"string","enum":["on"],"minLength":1}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state","default":"on"}}` {
				if _, err := core.GenerateWithDiagnostics(rq); err != nil {
					t.Fatal("original checked internal enum factory did not graduate", err)
				}
				continue
			}
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("factory granted unsupported source authority", body, checked, err)
			}
		}
	}
}

func TestInternalEnumAliasFactoryDepth(t *testing.T) {
	for _, hops := range []int{3, 50} {
		parts := []string{`"Owner":{"properties":{"state":{"enum":["on","off"]}}}`}
		for i := 0; i < hops; i++ {
			target := fmt.Sprintf("Alias%d", i+1)
			if i == hops-1 {
				target = "Owner/properties/state"
			}
			parts = append(parts, fmt.Sprintf(`"Alias%d":{"$ref":"#/components/schemas/%s","default":"on"}`, i, target))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if hops == 3 && err != nil || hops == 50 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("internal default factory erased alias depth", hops, err)
		}
	}
}

func TestInternalEnumAliasOwnFactories(t *testing.T) {
	for _, leaf := range []string{`{"enum":["on","off"]}`, `{"enum":[null,"on","null"]}`, `{"type":"string","enum":["on","off"],"default":"on"}`, `{"type":"string","format":"date","enum":["on","off"]}`, `{"type":"string","x-native-type":"org.example.provider.Amount","enum":["on","off"]}`} {
		for _, owner := range []struct{ body, path string }{
			{`{"properties":{"state":` + leaf + `}}`, "Owner/properties/state"},
			{`{"properties":{"nested":{"properties":{"state":` + leaf + `}}}}`, "Owner/properties/nested/properties/state"},
		} {
			for _, checked := range []bool{false, true} {
				body := `{"Owner":` + owner.body + `,"Alias":{"$ref":"#/components/schemas/` + owner.path + `","default":"on"},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Maybe","nullable":false,"default":"on"},"Unannotated":{"$ref":"#/components/schemas/Alias"}}`
				rq := request(schemas(body), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(leaf, owner.path, checked, err, *err.(core.Diagnostics)[0].Pointer)
				}
				for _, name := range []string{"Alias", "Maybe", "Narrow"} {
					if !strings.Contains(source(t, got, name+"Dto"), "public fun `"+name+"Dto default`") {
						t.Fatal("missing independent internal enum factory", name)
					}
				}
				if strings.Contains(source(t, got, "UnannotatedDto"), " default`") {
					t.Fatal("inherited target factory")
				}
				if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("internal enum factory nondeterminism", err)
				}
			}
		}
	}
}

func TestInternalEnumAliasFactoriesInConsumers(t *testing.T) {
	for _, tail := range []string{"", `,"format":"date"`, `,"x-native-type":"org.example.provider.Amount"`} {
		for _, checked := range []bool{false, true} {
			body := `{"Owner":{"properties":{"state":{"type":"string","enum":["on","off"]` + tail + `}}},"Hops":{"properties":{"state":{"$ref":"#/components/schemas/Owner/properties/state","nullable":true}}},"Alias":{"$ref":"#/components/schemas/Hops/properties/state","default":"off"},"Holder":{"properties":{"state":{"$ref":"#/components/schemas/Alias","default":"on"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":["on",null]}}},"Root":{"properties":{"state":{"$ref":"#/components/schemas/Alias"}},"default":{"state":"on"}}}`
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal(tail, checked, err, *err.(core.Diagnostics)[0].Pointer)
			}
			rq.Document = []byte(strings.Replace(string(rq.Document), `"enum":["on","off"]`, `"enum":["on","off"],"default":"on"`, 1))
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("independent source field and consumer annotations", tail, checked, err)
			}
		}
	}
}
