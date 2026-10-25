package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Preserve member/property order while making the null-source intent of an
// existing synthetic fixture explicit on every operand, rather than relying
// on the retired first-member propagation rule.
func nullableAllOfMembers(t *testing.T, schema string) string {
	t.Helper()
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(schema), &root); err != nil {
		t.Fatal(err)
	}
	var members []json.RawMessage
	if err := json.Unmarshal(root["allOf"], &members); err != nil {
		t.Fatal(err)
	}
	for i, member := range members {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(member, &fields); err != nil {
			t.Fatal(err)
		}
		if _, ok := fields["nullable"]; !ok {
			members[i] = []byte(`{"nullable":true,` + strings.TrimPrefix(string(member), "{"))
		}
	}
	encoded, err := json.Marshal(members)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Replace(schema, string(root["allOf"]), string(encoded), 1)
}

func intersectionObject(nullable bool, property string) string {
	return fmt.Sprintf(`{"nullable":%t,"properties":{%q:{"type":"string"}},"required":[%q]}`, nullable, property, property)
}

func intersectionProduct(left, right bool) string {
	return fmt.Sprintf(`{"allOf":[{"nullable":%t,"oneOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]},%s]}`, left, intersectionObject(right, "label"))
}

func TestNullableIntersectionOrdinaryAndProductNullSets(t *testing.T) {
	for _, left := range []bool{false, true} {
		for _, right := range []bool{false, true} {
			ordinary := `{"allOf":[` + intersectionObject(left, "left") + `,` + intersectionObject(right, "right") + `]}`
			input := schemas(`{"Value":` + ordinary + `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value"}},"required":["value"]}}`)
			rq := request(input, "Dto")
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("ordinary intersection", left, right, err)
			}
			want := "public val value: org.example.models.models.ValueDto,"
			if left && right {
				want = "public val value: org.example.models.models.ValueDto? = null,"
			}
			if !strings.Contains(source(t, got, "HolderDto"), want) {
				t.Fatal("ordinary nullability is not the member intersection", left, right, source(t, got, "HolderDto"))
			}
			product := intersectionProduct(left, right)
			rq.Document = schemas(`{"Value":` + product + `}`)
			got, err = core.GenerateWithDiagnostics(rq)
			if err != nil || strings.Contains(source(t, got, "ValueDto"), "NullValueDto") != (left && right) {
				t.Fatal("product carrier nullability", left, right, err)
			}
			rq.Document = fieldDoc(product)
			got, err = core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("inline product", left, right, err)
			}
			want = "public val value: Value,"
			if left && right {
				want = "public val value: Value? = null,"
			}
			if !strings.Contains(source(t, got, "RecordDto"), want) {
				t.Fatal("inline product null view", left, right, source(t, got, "RecordDto"))
			}
		}
	}
}

func TestNullableIntersectionAliasesAndPhysicalSources(t *testing.T) {
	input := schemas(`{"A":` + intersectionObject(true, "left") + `,"B":{"nullable":true,"allOf":[{"properties":{"right":{"type":"string"}},"required":["right"]}]},"False":{"$ref":"#/components/schemas/A","nullable":false},"Pair":{"allOf":[{"$ref":"#/components/schemas/A"},{"$ref":"#/components/schemas/B"}]},"Narrow":{"allOf":[{"$ref":"#/components/schemas/False"},{"$ref":"#/components/schemas/B"}]},"Alias":{"$ref":"#/components/schemas/Pair"},"Holder":{"properties":{"pair":{"$ref":"#/components/schemas/Alias"},"narrow":{"$ref":"#/components/schemas/Narrow"}},"required":["pair","narrow"]}}`)
	rq := request(input, "Dto")
	first, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, first, "AliasDto"), "= org.example.models.models.PairDto?\n") || !strings.Contains(source(t, first, "HolderDto"), "public val narrow: org.example.models.models.NarrowDto,") {
		t.Fatal("nearest alias null intent was lost")
	}
	second, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("intersection reference output drift", err)
	}
}

func TestNullableIntersectionMixedInlineMigration(t *testing.T) {
	// This was a nullable inline view under first-member propagation. The
	// non-null operand now narrows the intersection; explicit root true keeps
	// the prior widened view when a caller deliberately needs it.
	value := `{"allOf":[` + intersectionObject(true, "left") + `,` + intersectionObject(false, "right") + `]}`
	for _, explicit := range []bool{false, true} {
		shape := value
		if explicit {
			shape = `{"nullable":true,` + strings.TrimPrefix(value, "{")
		}
		got, err := core.GenerateWithDiagnostics(request(fieldDoc(shape), "Dto"))
		if err != nil || strings.Contains(source(t, got, "RecordDto"), "public val value: Value? = null,") != explicit {
			t.Fatal("mixed inline migration/root override", explicit, err)
		}
		withDefault := strings.TrimSuffix(shape, "}") + `,"default":null}`
		for _, required := range []bool{false, true} {
			input := fieldDoc(withDefault)
			if !required {
				input = []byte(strings.Replace(string(input), `,"required":["value"]`, "", 1))
			}
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if explicit {
				if err != nil {
					t.Fatal("explicit nullable root lost null default", required, err)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/properties/value/default" {
				t.Fatal("mixed null default escaped migration checks before required suppression", required, err)
			}
		}
	}
}

func TestNullableIntersectionOriginalGraduations(t *testing.T) {
	for _, body := range []string{
		`{"allOf":[{"nullable":true,"properties":{"id":{"type":"string"}}}]}`,
		`{"allOf":[{"nullable":true,"oneOf":[{"properties":{"kind":{"const":"a"}}},{"properties":{"kind":{"const":"b"}}}]}]}`,
		`{"nullable":true,"allOf":[{"oneOf":[{"nullable":true,"properties":{"id":{"type":"string"}}},{"properties":{"ok":{"type":"boolean"}}}]}]}`,
		`{"nullable":true,"allOf":[{"oneOf":[{"properties":{"id":{"type":"string"}}},{"type":"null"}]}]}`,
		`{"nullable":true,"allOf":[{"oneOf":[{"properties":{"id":{"type":"string"}}}]},{"nullable":true,"properties":{"name":{"type":"string"}}}]}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Value":`+body+`}`), "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal("original nullable intersection did not graduate", checked, body, err)
			}
		}
	}
	value := `{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]}]}`
	base := schemas(`{"Value":` + value + `,"False":{"$ref":"#/components/schemas/Value","nullable":false}}`)
	first, err := core.GenerateWithDiagnostics(request(base, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	input := schemas(`{"Value":` + value + `,"False":{"$ref":"#/components/schemas/Value","nullable":false},"Other":{"allOf":[{"$ref":"#/components/schemas/Value","nullable":false},{"properties":{"more":{"type":"string"}}}]}}`)
	second, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || source(t, first, "ValueDto") != source(t, second, "ValueDto") {
		t.Fatal("nullable operand changed its canonical constructor/codec", err)
	}
}

func TestNullableIntersectionDerivedProductBranchBudget(t *testing.T) {
	for _, width := range []int{239, 240} {
		branches := make([]string, width)
		for i := range branches {
			branches[i] = fmt.Sprintf(`{"properties":{"kind":{"const":"v%d"}}}`, i)
		}
		rq := request(schemas(`{"Value":{"allOf":[{"nullable":true,"oneOf":[`+strings.Join(branches, ",")+`]}]}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (width == 239) || err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT") {
			t.Fatal("derived Null carrier escaped branch budget", width, err)
		}
	}
}

func TestNullableIntersectionOverlappingOneOfNullsKeepWarning(t *testing.T) {
	input := schemas(`{"Value":{"allOf":[{"oneOf":[` + intersectionObject(true, "left") + `,` + intersectionObject(true, "right") + `]},{"nullable":true,"properties":{"stamp":{"const":"shared"}}}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, warning := range got.Warnings {
		if warning.Code == "W_ONEOF_NULL_OVERLAP" && warning.Pointer != nil && *warning.Pointer == "/components/schemas/Value/allOf/0/oneOf" {
			return
		}
	}
	t.Fatal("overlapping nullable object alternatives lost their approximation warning", got.Warnings)
}
