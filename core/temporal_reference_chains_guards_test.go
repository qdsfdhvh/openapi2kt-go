package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTemporalReferenceChainsSourceCacheAndSlotLimits(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, namedAlias := range []bool{false, true} {
			for _, typed := range []bool{false, true} {
				limit := 47
				if namedAlias {
					limit--
				}
				if typed {
					limit -= 2
				}
				for _, n := range []int{limit, limit + 1} {
					fields := []string{}
					for i := range n {
						target := "Time"
						if namedAlias {
							target = "TimeAlias"
						}
						if i > 0 {
							target = fmt.Sprint("Hops/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					body := `"Time":{"type":"string","format":"date-time","nullable":true},"TimeAlias":{"$ref":"#/components/schemas/Time"},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
					if typed {
						body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}`
					}
					r := request(schemas(`{`+body+`}`), "Dto")
					r.Options.Validation = checked
					got, e := core.GenerateWithDiagnostics(r)
					if n == limit {
						if e != nil {
							t.Fatal(checked, namedAlias, typed, n, e)
						}
					} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
						t.Fatal("source height reset", checked, namedAlias, typed, n, e)
					}
				}
			}
		}
		for _, nullable := range []bool{false, true} {
			for _, n := range []int{240, 241} {
				fields, required := []string{}, []string{}
				for i := range n {
					fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":%v}`, i, nullable))
					required = append(required, fmt.Sprintf(`"p%d"`, i))
				}
				body := `"Time":{"type":"string","format":"date-time","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Time"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Wide":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
				r := request(schemas(`{`+body+`}`), "Dto")
				r.Options.Validation = checked
				got, e := core.GenerateWithDiagnostics(r)
				if n == 240 {
					if e != nil {
						t.Fatal(nullable, n, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
					t.Fatal("scalar layout lost", nullable, n, e)
				}
			}
		}
	}
}

func TestTemporalReferenceChainsKeepOtherScalarAndShapePolicies(t *testing.T) {
	for _, tc := range []struct{ terminal, code, pointer string }{
		{`{"type":"integer","format":"int64"}`, "", ""},                // Original input graduates to the independently verified builtin profile.
		{`{"type":"integer","x-native-type":"bigint"}`, "", ""},        // Original named bigint input graduates with Long backing; other source roles below stay excluded.
		{`{"type":"string","format":"binary"}`, "bytes-graduated", ""}, // Preserve original byte/binary input; native ByteArray source-role increment.
		{`{"type":"string","x-native-type":"org.example.Custom"}`, "caller-custom-graduated", ""},
		{`{"type":"string","x-native-type":"kotlin.time.Instant"}`, "caller-instant-graduated", ""},
		{`{"type":"object"}`, "raw-graduated", ""}, // Original raw root input graduates through its actual empty Model role.
		{`{"type":"string","format":"date-time","const":"2024-02-29T12:34:56Z"}`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`{"type":"string","format":"unsupported"}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Time/format"},
	} {
		input := schemas(`{"Time":` + tc.terminal + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Time"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`)
		if tc.code == "raw-graduated" {
			got := report(t, input, "Dto")
			if !strings.Contains(source(t, got, "BridgeDto"), "= kotlinx.serialization.json.JsonElement\n") {
				t.Fatal("raw source backing lost")
			}
		} else if strings.HasPrefix(tc.code, "caller-") {
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			scalar := "org.example.Custom"
			if tc.code == "caller-instant-graduated" {
				scalar = "kotlin.time.Instant"
			}
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "= "+scalar+"\n") {
				t.Fatal("original caller-native source backing lost", err)
			}
		} else if tc.code == "bytes-graduated" {
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "public typealias BridgeDto = kotlin.ByteArray\n") || strings.Contains(source(t, got, "HopsDto"), "Serializable(with") {
				t.Fatal("original binary input lost native backing", err)
			}
		} else if tc.code == "" {
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "public typealias BridgeDto = kotlin.Long\n") {
				t.Fatal("original builtin chain did not retain native backing", err)
			}
		} else {
			aliasFailure(t, input, tc.code, tc.pointer)
		}
	}
	aliasFailure(t, schemas(`{"Time":{"type":"string","format":"date-time"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Time","minLength":1}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Hops/properties/value/minLength")
	aliasFailure(t, schemas(`{"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"}}}}`), "E_REF_CYCLE", "/components/schemas/Bridge/$ref")
	base := `"Time":{"type":"string","format":"date-time"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Time"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}`
	aliasFailure(t, schemas(`{`+base+`,"Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},{"properties":{"id":{"type":"string"}}}]}}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/allOf/0")
	aliasFailure(t, schemas(`{`+base+`,"Unused":{"properties":{"invalid":{"type":"string","format":"unsupported"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Unused/properties/invalid/format")
}

func TestTemporalReferenceChainsHTTPScalarLayout(t *testing.T) {
	for _, n := range []int{240, 241} {
		parameters := []string{}
		for i := range n {
			parameters = append(parameters, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/Bridge","nullable":false}}`, i))
		}
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Time":{"type":"string","format":"date-time"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Time"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}},"paths":{"/time":{"get":{"parameters":[` + strings.Join(parameters, ",") + `],"responses":{"204":{}}}}}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if n == 240 {
			if e != nil || strings.Contains(source(t, r, "DefaultApi"), "rawValue") {
				t.Fatal(n, e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, e)
		}
	}
}
