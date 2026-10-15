package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestExplicitNullLiteralsKeepCanonicalCarrierBodyAndPresence(t *testing.T) {
	input := schemas(`{"Pinned":{"type":"null","const":null},"Vocabulary":{"type":"null","enum":[null]},"Alias":{"$ref":"#/components/schemas/Vocabulary"},"Holder":{"properties":{"kotlinx":{"type":"string"},"fixed":{"type":"null","const":null},"borrowedConst":{"$ref":"#/components/schemas/Pinned"},"required":{"type":"null","enum":[null]},"optional":{"$ref":"#/components/schemas/Alias"},"rows":{"type":"array","items":{"type":"null","const":null}},"map":{"type":"object","additionalProperties":{"type":"null","enum":[null]}},"nested":{"properties":{"kotlinx":{"type":"boolean"},"fixed":{"type":"null","const":null}}}},"required":["required"]},"Only":{"properties":{"value":{"type":"null","const":null}}}}`)
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal(checked, style, e)
			}
			text := source(t, got, "HolderDto")
			for _, marker := range []string{"public val required: kotlinx.serialization.json.JsonNull,", "public val optional: kotlinx.serialization.json.JsonNull? = null", "public val fixed: kotlinx.serialization.json.JsonNull = `openapi-kt null literal`()", "public val borrowedConst: kotlinx.serialization.json.JsonNull = `openapi-kt null literal`()", "List<kotlinx.serialization.json.JsonNull>", "Map<kotlin.String, kotlinx.serialization.json.JsonNull>", "private fun `openapi-kt null literal`(): kotlinx.serialization.json.JsonNull = kotlinx.serialization.json.JsonNull"} {
				if !strings.Contains(text, marker) {
					t.Fatal(marker, text)
				}
			}
			if strings.Count(text, "private fun `openapi-kt null literal`") != 1 || len(got.Warnings) != 0 {
				t.Fatal("Null-only carrier invented approximation or repeated helper", got.Warnings, text)
			}
			for _, name := range []string{"PinnedDto", "VocabularyDto", "AliasDto"} {
				if !strings.Contains(source(t, got, name), "public typealias "+name+" = kotlinx.serialization.json.JsonNull") {
					t.Fatal("Synthetic null enum or absent alias", source(t, got, name))
				}
			}
			if !strings.Contains(source(t, got, "OnlyDto"), "public class OnlyDto") {
				t.Fatal("Const-only model lost existing body API")
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Null literal rerun changed", e)
			}
		}
	}
}

func TestExplicitNullLiteralOperationsRetainInlineUnitAndTypedPayload(t *testing.T) {
	for _, literal := range []string{`{"type":"null","const":null}`, `{"type":"null","enum":[null]}`} {
		input := []byte(`{"openapi":"3.1.0","paths":{"/value":{"post":{"operationId":"send","requestBody":{"required":true,"content":{"application/json":{"schema":` + literal + `}}},"responses":{"200":{"content":{"application/json":{"schema":` + literal + `}}}}}},"/events":{"get":{"operationId":"watch","responses":{"200":{"content":{"text/event-stream":{"schema":{"properties":{"event":{"const":"nothing"},"data":` + literal + `},"required":["data"]}}}}}}}}}`)
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal(literal, checked, e)
			}
			api := source(t, got, "DefaultApi")
			if !strings.Contains(api, "input: kotlinx.serialization.json.JsonNull,") || !strings.Contains(api, "): kotlin.Unit") || !strings.Contains(source(t, got, "WatchEventDto"), "public val data: kotlinx.serialization.json.JsonNull,") {
				t.Fatal("Null payload changed inline Unit projection", api)
			}
		}
	}
}
