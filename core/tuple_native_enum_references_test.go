package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleNativeEnumReferenceMinimal(t *testing.T) {
	for _, checked := range []bool{false, true} {
		rq := request(schemas(`{"Code":{"type":"integer","enum":[1,2]},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Code"},{"$ref":"#/components/schemas/Code"}]}}`), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if !checked {
			if err == nil {
				t.Fatal("unchecked native tuple reference unexpectedly admitted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "ProbeDto"), "List<kotlin.Int>") {
			t.Fatal("lost mapped native enum type", source(t, got, "ProbeDto"))
		}
	}
}

func TestTupleNativeEnumReferenceProfiles(t *testing.T) {
	for _, leaf := range []struct{ name, schema, want string }{
		{"int", `{"type":"integer","format":"int32","enum":[1,2]}`, "kotlin.Int"},
		{"long", `{"type":"integer","format":"int64","enum":[9007199254740993,9007199254740996]}`, "kotlin.Long"},
		{"float", `{"type":"number","format":"float","enum":[0.1,0.10000000149011612,-0.0]}`, "kotlin.Float"},
		{"double", `{"type":"number","format":"double","enum":[0.1,-0.0,2.5]}`, "kotlin.Double"},
		{"boolean", `{"type":"boolean","enum":[true,false]}`, "kotlin.Boolean"},
		{"unformatted-int", `{"type":"integer","enum":[1,2]}`, "kotlin.Int"},
		{"unformatted-double", `{"type":"number","enum":[1,2.5]}`, "kotlin.Double"},
	} {
		for _, tc := range []struct{ name, prefix, tail, want string }{
			{"same", `[{"$ref":"#/components/schemas/Code"},{"$ref":"#/components/schemas/Code"}]`, "", leaf.want},
			{"alias", `[{"$ref":"#/components/schemas/Alias","nullable":false},{"$ref":"#/components/schemas/Code"}]`, "", leaf.want},
			{"nullable", `[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Code","nullable":true}]`, "", leaf.want + "?"},
			{"different", `[{"$ref":"#/components/schemas/Code"},{"$ref":"#/components/schemas/Other"}]`, "", "kotlinx.serialization.json.JsonElement"},
			{"mixed", `[{"$ref":"#/components/schemas/Code"},{"type":"integer"}]`, "", "kotlinx.serialization.json.JsonElement"},
			{"tail", `[{"$ref":"#/components/schemas/Code"},{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Alias","nullable":false}`, leaf.want},
			{"scalar-tail", `[{"$ref":"#/components/schemas/Code"}]`, `,"items":{"type":"string"}`, "kotlin.String"},
			{"internal", `[{"$ref":"#/components/schemas/Source/properties/code"},{"$ref":"#/components/schemas/Bridge"}]`, "", leaf.want},
		} {
			t.Run(leaf.name+"/"+tc.name, func(t *testing.T) {
				input := schemas(`{"Code":` + leaf.schema + `,"Other":` + leaf.schema + `,"Alias":{"$ref":"#/components/schemas/Code","nullable":true},"Source":{"properties":{"code":` + leaf.schema + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/code"},"Probe":{"type":"array","prefixItems":` + tc.prefix + tc.tail + `}}`)
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					rq := request(input, "Dto")
					rq.Options.Validation = true
					rq.Options.ModelStyle = style
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(source(t, got, "ProbeDto"), "List<"+tc.want+">") {
						t.Fatal(source(t, got, "ProbeDto"))
					}
					seen := false
					for _, w := range got.Warnings {
						if w.Code == "W_TUPLE_APPROXIMATION" && *w.Pointer == "/components/schemas/Probe/prefixItems" {
							seen = true
						}
					}
					if !seen {
						t.Fatal("lost approximation", got.Warnings)
					}
					again, e := core.GenerateWithDiagnostics(rq)
					if e != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("nondeterministic", e)
					}
				}
			})
		}
	}
}

func TestTupleNativeEnumSourceCaptureCumulativeAndReuse(t *testing.T) {
	values := make([]string, 240)
	for i := range values {
		values[i] = fmt.Sprint(i)
	}
	for _, shared := range []bool{true, false} {
		entries := []string{}
		for i := 0; i < 64; i++ {
			target := 0
			if !shared {
				target = i
			}
			if !shared || i == 0 {
				entries = append(entries, fmt.Sprintf(`"Code%02d":{"type":"integer","enum":[%s]}`, target, strings.Join(values, ",")))
			}
			entries = append(entries, fmt.Sprintf(`"Probe%02d":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Code%02d"}],"items":{"type":"string"}}`, i, target))
		}
		rq := request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if shared {
			if err != nil {
				t.Fatal("repeated original vocabulary charged again", err)
			}
		} else {
			if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("independent original snapshots reset cumulative work", err)
			}
		}
	}
}

func TestTupleNativeEnumOriginalNoDropGuards(t *testing.T) {
	for _, tc := range []struct{ leaf, member, pointer string }{
		{`{"type":"integer","enum":[1,2]}`, `{"$ref":"#/components/schemas/Code","nullable":"bad"}`, "/components/schemas/Probe/prefixItems/0/nullable"},
		{`{"type":"integer","enum":[1,2]}`, `{"$ref":"#/components/schemas/Code","default":1}`, "/components/schemas/Probe/prefixItems/0/default"},
		{`{"type":"integer","enum":[1,2],"minimum":1}`, `{"$ref":"#/components/schemas/Code"}`, "/components/schemas/Code/minimum"},
		{`{"type":"integer","enum":[1,2],"default":9}`, `{"$ref":"#/components/schemas/Code"}`, "/components/schemas/Code/default"},
		{`{"type":"integer","enum":[1,"bad"]}`, `{"$ref":"#/components/schemas/Code"}`, "/components/schemas/Code/enum/1"},
	} {
		for _, tail := range []string{"", `,"items":{"type":"string"}`} {
			rq := request(schemas(`{"Code":`+tc.leaf+`,"Probe":{"type":"array","prefixItems":[`+tc.member+`]`+tail+`}}`), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("discarded source hidden", tc, tail, err)
			}
		}
	}
}

func TestTupleNativeEnumIndependentFactoryAndOwnDefaultRoles(t *testing.T) {
	rq := request(schemas(`{"Code":{"type":"integer","format":"int32","enum":[1,2],"default":1},"Alias":{"$ref":"#/components/schemas/Code","default":2},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}]},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Probe"}},"required":["rows"]}}`), "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "CodeDto"), "CodeDto default") || !strings.Contains(source(t, got, "AliasDto"), "AliasDto default") {
		t.Fatal("lost independent original factories")
	}
	if strings.Contains(source(t, got, "ProbeDto"), "default`") || !strings.Contains(source(t, got, "HolderDto"), "rows: "+rq.Options.PackageName+".models.ProbeDto,") {
		t.Fatal("tuple inherited target default", source(t, got, "HolderDto"))
	}
	for _, schema := range []string{
		`{"Code":{"type":"integer","enum":[1,2]},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Code"}],"default":[1]}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(schema), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if checked {
				if e != nil {
					t.Fatal("former selected native enum reference default refusal remains", e)
				}
				continue
			}
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("tuple broadened unrelated own/default role", checked, e)
			}
		}
	}
}
