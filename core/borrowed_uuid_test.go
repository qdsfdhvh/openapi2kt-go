package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedUUIDNativeMapping(t *testing.T) {
	for _, fixture := range []string{"root-borrowed-builtin", "schema-internal-scalar-pointers-qualified"} {
		input, err := os.ReadFile("../tests/fixtures/http/" + fixture + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var tree any
		if err := json.Unmarshal(input, &tree); err != nil {
			t.Fatal(err)
		}
		control, err := json.Marshal(tree)
		if err != nil {
			t.Fatal(err)
		}
		// These fixtures contain no annotation payload objects: visit only schema
		// maps and preserve identical ordering in both serialized controls.
		var uuidStrings func(any)
		uuidStrings = func(value any) {
			switch v := value.(type) {
			case map[string]any:
				kind := v["type"] == "string"
				if types, ok := v["type"].([]any); ok {
					kind = reflect.DeepEqual(types, []any{"string", "null"})
				}
				_, pinned := v["const"]
				if _, formatted := v["format"]; kind && !formatted && !pinned {
					v["format"] = "uuid"
				}
				for _, child := range v {
					uuidStrings(child)
				}
			case []any:
				for _, child := range v {
					uuidStrings(child)
				}
			}
		}
		uuidStrings(tree)
		uuid, err := json.Marshal(tree)
		if err != nil {
			t.Fatal(err)
		}
		for _, checked := range []bool{false, true} {
			// The qualified fixture owns bounds, which remain validation-only.
			if !checked && fixture == "schema-internal-scalar-pointers-qualified" {
				continue
			}
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(control, "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				baseline, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				r.Document = uuid
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got.Warnings, baseline.Warnings) {
					t.Fatal("borrowed UUID mapping failed", fixture, checked, style, err)
				}
				found := false
				for _, file := range got.Artifacts.Files {
					found = found || strings.Contains(file.Content, "kotlin.uuid.Uuid")
				}
				if !found {
					t.Fatal("borrowed UUID lost native backing")
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("borrowed UUID generation drift", err)
				}
			}
		}
	}
}

func TestBorrowedUUIDFieldOwnDefault(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, extra := range []string{"", `,"nullable":true`, `,"nullable":false`} {
			for _, target := range []string{"Source/properties/id~1value~0", "Bridge", "Hops/properties/value"} {
				for _, value := range []string{`"550e8400-e29b-41d4-a716-446655440000"`, `null`} {
					body := `{"Source":{"properties":{"id/value~":{"type":"string","format":"uuid","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/id~1value~0"},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/` + target + `"` + extra + `,"default":` + value + `}}}}`
					r := request(schemas(body), "Dto")
					r.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(r)
					if extra == `,"nullable":false` && value == "null" {
						ds, ok := err.(core.Diagnostics)
						if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/v/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
							t.Fatal("UUID field default ignored nearest false", err)
						}
						continue
					}
					literal := value
					if value != "null" {
						literal = "`openapi-kt default uuid`(" + value + ")"
					}
					if err != nil || !strings.Contains(source(t, got, "HolderDto"), "public val v: kotlin.uuid.Uuid? = ") || !strings.Contains(source(t, got, "HolderDto"), literal) {
						t.Fatal("borrowed UUID own-default role lost literal", target, extra, value, err)
					}
				}
			}
		}
	}
}

func TestBorrowedUUIDDepthAndSourceGuards(t *testing.T) {
	for _, count := range []int{45, 46} {
		props := []string{`"id":{"type":"string","format":"uuid","nullable":true}`}
		for i := 0; i < count; i++ {
			target := "Source/properties/id"
			if i > 0 {
				target = fmt.Sprintf("Source/properties/h%d", i-1)
			}
			props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		input := schemas(`{"Source":{"properties":{` + strings.Join(props, ",") + `}},"Bridge":{"$ref":"#/components/schemas/Source/properties/h` + fmt.Sprint(count-1) + `"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Bridge","nullable":false}}}}`)
		for _, checked := range []bool{false, true} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if count == 45 {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("borrowed UUID cached depth was reset", count, err)
			}
		}
	}
	for _, tc := range []struct{ extra, code, pointer string }{
		{`,"minLength":-1`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/id/minLength"},
		{`,"default":"hidden"`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/id/default"},
		{`,"enum":["a"]`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`,"const":"a"`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`,"x-native-type":"org.example.Id"`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
		{`,"x-kotlin-value-class":true`, "E_REF_UNSUPPORTED", "/components/schemas/Bridge/$ref"},
	} {
		r := request(schemas(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/id"},"Source":{"properties":{"id":{"type":"string","format":"uuid"`+tc.extra+`}}}}`), "Dto")
		r.Options.Validation = true
		if tc.extra == `,"enum":["a"]` {
			metadataStringEnumGraduation(t, r)
			continue
		}
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("borrowed UUID hid invalid or excluded source", tc, err)
		}
	}
}
