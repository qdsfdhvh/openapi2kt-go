package cli_test

import (
	"path/filepath"
	"strings"
	"testing"
)

const pointerPresenceNativeInput = `{"openapi":"3.1.0","info":{"title":"Pointer Followup","version":"1"},"paths":{"/submit":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["requiredNull","requiredFalse"],"properties":{"inherited":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"memberTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}]},"memberFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}]},"annotationTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false},{"nullable":true}]},"annotationFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true},{"nullable":false}]},"outerTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}],"nullable":true},"outerFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}],"nullable":false},"requiredNull":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"requiredFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}]},"named":{"allOf":[{"$ref":"#/components/schemas/Named","nullable":true}]},"nested":{"type":"object","properties":{"inherited":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"memberTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}]}}},"patchList":{"type":"array","minItems":1,"uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true},"patchMap":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true}}}}}},"responses":{"204":{"description":"Done"}}}}},"components":{"schemas":{"Named":{"type":"string"},"Source":{"type":"object","properties":{"nullable":{"type":"string","nullable":true,"minLength":2,"maxLength":3,"pattern":"^[a-z]+$"},"plain":{"type":"string","minLength":2,"maxLength":3,"pattern":"^[a-z]+$"},"number":{"type":["integer","null"],"format":"int64","minimum":0,"exclusiveMinimum":0,"maximum":8,"exclusiveMaximum":8,"multipleOf":2}}},"Leaves":{"type":"object","required":["list","map","numbers","nullableList","nullableMap","nonNullList","nonNullMap"],"properties":{"list":{"type":"array","minItems":1,"uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/nullable"}},"map":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable"}},"numbers":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Source/properties/number"}},"nullableList":{"type":"array","minItems":1,"uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true},"nullableMap":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true},"nonNullList":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}},"nonNullMap":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}}}},"Shared":{"type":"object","properties":{"inherited":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"memberTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}]}}}}}}
`

func TestSchemaInternalScalarPointerPresenceNativeBorrowedWrappers(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, []byte(pointerPresenceNativeInput))
	actual := p.generate(t, "--validation", "--model-name-suffix", "Dto")
	t.Logf("actual inner native CLI exit=%d stdout=%s stderr=%s; desired presence assertions follow", actual.Code, actual.Stdout, actual.Stderr)
	result(t, actual, 0)
	text := string(read(t, filepath.Join(p.Output, "org/example/catalog/api/models/SubmitInputDto.kt")))
	if !strings.Contains(text, "inherited: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent") {
		t.Errorf("missing desired presence shape %q", "inherited: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent")
	}
	if !strings.Contains(text, "memberTrue: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent") {
		t.Errorf("missing desired presence shape %q", "memberTrue: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent")
	}
	if !strings.Contains(text, "annotationTrue: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent") {
		t.Errorf("missing desired presence shape %q", "annotationTrue: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent")
	}
	if !strings.Contains(text, "outerTrue: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent") {
		t.Errorf("missing desired presence shape %q", "outerTrue: org.example.catalog.models.Patch<kotlin.String> = org.example.catalog.models.Patch.Absent")
	}
	if !strings.Contains(text, "memberFalse: kotlin.String? = null") {
		t.Errorf("missing desired presence shape %q", "memberFalse: kotlin.String? = null")
	}
	if !strings.Contains(text, "annotationFalse: kotlin.String? = null") {
		t.Errorf("missing desired presence shape %q", "annotationFalse: kotlin.String? = null")
	}
	if !strings.Contains(text, "outerFalse: kotlin.String? = null") {
		t.Errorf("missing desired presence shape %q", "outerFalse: kotlin.String? = null")
	}
	if !strings.Contains(text, "requiredNull: kotlin.String?,") {
		t.Errorf("missing desired presence shape %q", "requiredNull: kotlin.String?,")
	}
	if !strings.Contains(text, "requiredFalse: kotlin.String,") {
		t.Errorf("missing desired presence shape %q", "requiredFalse: kotlin.String,")
	}
	if !strings.Contains(text, "named: kotlin.String? = null") {
		t.Errorf("missing desired presence shape %q", "named: kotlin.String? = null")
	}
	if strings.Contains(text, "requiredNull: kotlin.String? =") || strings.Contains(text, "requiredFalse: kotlin.String =") {
		t.Error("required field gained default")
	}
	shared := string(read(t, filepath.Join(p.Output, "org/example/catalog/models/SharedDto.kt")))
	if strings.Contains(shared, "Patch<") {
		t.Error("shared context gained request Patch")
	}
}
