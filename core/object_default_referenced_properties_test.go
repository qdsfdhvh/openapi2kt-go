package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestObjectDefaultReferencedProperties(t *testing.T) {
	// A direct inline object default may reference plain object and scalar/enum
	// properties; the outer DTO serializer still owns the nested codecs.
	direct := schemas(`{"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"Mode":{"type":"string","enum":["a","b"]},"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Leaf"},"mode":{"$ref":"#/components/schemas/Mode"}},"default":{"leaf":{"z":"1"},"mode":"a"}}}}}`)
	got := report(t, direct, "Dto")
	text := source(t, got, "HolderDto")
	for _, marker := range []string{
		"public val leaf: org.example.models.models.LeafDto? = null,",
		"public val mode: org.example.models.models.ModeDto? = null,",
		"`openapi-kt model default org example models models HolderDto V`(",
		`{\"leaf\":{\"z\":\"1\"},\"mode\":\"a\"}`,
	} {
		if !strings.Contains(text, marker) {
			t.Fatal("lost direct referenced property", marker, text)
		}
	}
	// The referenced object contract's target walk reuses the same property rule.
	referenced := schemas(`{"Leaf":{"type":"object","properties":{"z":{"type":"string"}}},"Inner":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Leaf"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Inner","default":{"leaf":{"z":"1"}}}}}}`)
	got = report(t, referenced, "Dto")
	if !strings.Contains(source(t, got, "HolderDto"), `{\"leaf\":{\"z\":\"1\"}}`) {
		t.Fatal(source(t, got, "HolderDto"))
	}
}

func TestObjectDefaultReferencedPropertiesGuards(t *testing.T) {
	// Missing required members inside a referenced object property keep the
	// consuming member Pointer.
	missing := schemas(`{"Leaf":{"type":"object","required":["z"],"properties":{"z":{"type":"string"}}},"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Leaf"}},"default":{"leaf":{}}}}}}`)
	r := request(missing, "Dto")
	r.Options.Validation = true
	report, err := core.GenerateWithDiagnostics(r)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/v/default/leaf" || !reflect.DeepEqual(report, core.GenerationReport{}) {
		t.Fatalf("missing required referenced property: %v", err)
	}
	// Reference chains stay with their separate roles.
	for _, input := range []string{
		`{"Leaf":{"type":"object","properties":{"z":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Leaf"},"Holder":{"properties":{"v":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Alias"}},"default":{"leaf":{"z":"1"}}}}}}`,
	} {
		r := request(schemas(input), "Dto")
		r.Options.Validation = true
		if err := func() (err error) { _, err = core.GenerateWithDiagnostics(r); return }(); err == nil {
			t.Fatalf("excluded referenced property role leaked: %s", input)
		}
	}
}
