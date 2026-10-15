package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumNullMembersKeepActualOpenVocabularyAndNullableOwners(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, leaf := range []string{`{"type":["string","null"],"enum":[null,"ready","done"]}`, `{"type":"string","nullable":true,"enum":["ready",null,"done"]}`, `{"nullable":true,"enum":["ready","done",null]}`} {
			input := schemas(`{"State":` + leaf + `,"Alias":{"$ref":"#/components/schemas/State"},"False":{"$ref":"#/components/schemas/Alias","nullable":false},"Owner":{"properties":{"state":` + leaf + `}},"Nested":{"$ref":"#/components/schemas/Owner/properties/state"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Nested"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias"},"narrow":{"$ref":"#/components/schemas/False"},"rows":{"$ref":"#/components/schemas/Rows"},"defaulted":{"type":["string","null"],"enum":[null,"ready","done"],"default":"ready"}},"required":["value","narrow"]}}`)
			r := request(input, "Dto")
			r.Options.Validation = validation
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal(validation, leaf, e)
			}
			state := source(t, got, "StateDto")
			if strings.Count(state, "public data object") != 2 || !strings.Contains(state, "Unrecognized") || strings.Contains(state, "serialName: kotlin.String = \"null\"") {
				t.Fatal(state)
			}
			if !strings.Contains(source(t, got, "NestedDto"), "OwnerDto.State?") || !strings.Contains(source(t, got, "FalseDto"), "StateDto\n") {
				t.Fatal("Physical enum owner or nearest false view lost")
			}
			holder := source(t, got, "HolderDto")
			if !strings.Contains(holder, "public val value: org.example.models.models.StateDto?") || strings.Contains(holder, "public val narrow: org.example.models.models.StateDto?") || !strings.Contains(holder, "fromSerialName(value") {
				t.Fatal(holder)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal(e)
			}
		}
	}
}
