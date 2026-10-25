package schema

import "testing"

func TestTupleReferenceSourceIdentityDepthAndWork(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Text":{"type":"string","nullable":true},"Alias":{"$ref":"#/components/schemas/Text","nullable":false},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias"}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	source := get(get(get(referencedSchemas(root), "Hops"), "properties"), "value")
	before, _ := root.MarshalJSON()
	work := 0
	signature, err := a.tupleScalar(source, "/components/schemas/Hops/properties/value", 1, &work)
	if err != nil || signature.Kind != "string" || signature.Nullable || work != 2 {
		t.Fatal("lost original source/nullable/hops", signature, work, err)
	}
	for _, spare := range []int{2, 1} {
		work = 16384 - spare
		_, err := a.tupleScalar(source, "/components/schemas/Hops/properties/value", 1, &work)
		if spare == 2 {
			if err != nil || work != 16384 {
				t.Fatal("exact source work rejected", work, err)
			}
		} else if err == nil || err.Code != "E_MODEL_LIMIT" {
			t.Fatal("source work reset", err)
		}
	}
	for _, depth := range []int{MaxSchemaDepth - 2, MaxSchemaDepth - 1} {
		work = 0
		_, err := a.tupleScalar(source, "/components/schemas/Hops/properties/value", depth, &work)
		if depth == MaxSchemaDepth-2 {
			if err != nil {
				t.Fatal("exact source depth rejected", err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal("source depth reset", err)
		}
	}
	saved := a.components["Alias"].Pointer
	a.components["Alias"].Pointer = "/wrong"
	work = 0
	if _, err := a.tupleScalar(source, "/components/schemas/Hops/properties/value", 1, &work); err == nil {
		t.Fatal("lost original source identity")
	}
	a.components["Alias"].Pointer = saved
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("tuple rewrote original scalar nodes")
	}
}
