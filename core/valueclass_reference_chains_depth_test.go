package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestValueClassInternalChainSourceAndNoDrop(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			for _, typed := range []bool{false, true} {
				limit := 47
				if alias {
					limit--
				}
				if typed {
					limit -= 2
				}
				for _, n := range []int{limit, limit + 1} {
					fields := []string{}
					for i := range n {
						target := "Value"
						if alias {
							target = "NamedAlias"
						}
						if i > 0 {
							target = fmt.Sprint("Owner/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					input := `"Value":{"type":"integer","format":"int64","x-kotlin-value-class":true,"nullable":true,"minimum":1},"NamedAlias":{"$ref":"#/components/schemas/Value"},"Owner":{"properties":{` + strings.Join(fields, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
					if typed {
						input += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}`
					}
					req := request(schemas(`{`+input+`}`), "Dto")
					req.Options.Validation = checked
					r, e := core.GenerateWithDiagnostics(req)
					if n == limit {
						if e != nil {
							t.Fatal(checked, alias, typed, n, e)
						}
					} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
						t.Fatal("source cache reset", checked, alias, typed, n, e)
					}
				}
			}
		}
	}
	for _, tc := range []struct{ terminal, code, pointer string }{
		{`{"type":"integer","format":"int64","x-kotlin-value-class":false}`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`{"type":"string","format":"date-time"}`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`{"type":"integer","x-native-type":"bigint"}`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`{"type":"integer","format":"date","x-kotlin-value-class":true}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Value/format"},
		{`{"type":"integer","x-kotlin-value-class":true,"minimum":2,"maximum":1}`, "E_SCHEMA_INVALID", "/components/schemas/Value/maximum"},
	} {
		input := schemas(`{"Value":` + tc.terminal + `,"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"}}`)
		if tc.terminal == `{"type":"string","format":"date-time"}` {
			// Preserve the original input as an explicit temporal-chain positive.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "BridgeDto"), "public typealias BridgeDto = kotlin.time.Instant\n") {
				t.Fatal("temporal source did not retain its scalar backing")
			}
			continue
		}
		if tc.terminal == `{"type":"integer","x-native-type":"bigint"}` {
			// Preserve the original magic source input; only its independently
			// supported chain role graduates, retaining field-only string codec.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "BridgeDto"), "public typealias BridgeDto = kotlin.Long\n") || !strings.Contains(source(t, r, "OwnerDto"), "LongAsStringSerializer::class") {
				t.Fatal("bigint chain backing/field codec lost")
			}
			continue
		}
		aliasFailure(t, input, tc.code, tc.pointer)
	}
	aliasFailure(t, schemas(`{"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Bridge"}}}}`), "E_REF_CYCLE", "/components/schemas/Bridge/$ref")
	aliasFailure(t, schemas(`{"Value":{"type":"integer","x-kotlin-value-class":true},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Value","minimum":1}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/hop"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Owner/properties/hop/minimum")
}

func TestValueClassInternalChainsThroughExistingRootBridge(t *testing.T) {
	input := schemas(`{"Value":{"type":"integer","format":"int64","x-kotlin-value-class":true,"nullable":true},"Owner":{"properties":{"first":{"$ref":"#/components/schemas/Value"},"second":{"$ref":"#/components/schemas/Bridge"}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/first","nullable":false},"Again":{"$ref":"#/components/schemas/Owner/properties/second"}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "AgainDto"), "= org.example.models.models.ValueDto\n") {
		t.Fatal("canonical false root bridge lost")
	}
}
