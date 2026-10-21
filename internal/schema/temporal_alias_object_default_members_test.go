package schema

import (
	"strings"
	"testing"
)

func TestTemporalAliasObjectDefaultMemberAuthority(t *testing.T) {
	for _, format := range []string{"date", "date-time"} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Day":{"type":["string","null"],"format":"` + format + `"},"Alias":{"$ref":"#/components/schemas/Day","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Holder":{"properties":{"v":{"properties":{"member":{"$ref":"#/components/schemas/Hops/properties/value"}}}}}}}}`)
		a, root := referencedCollectionTestAnalyzer(t, input)
		member := get(get(get(get(get(referencedSchemas(root), "Holder"), "properties"), "v"), "properties"), "member")
		target := get(referencedSchemas(root), "Day")
		before, _ := root.MarshalJSON()
		work := 0
		a.objectDefaultWork = &work
		node, err := a.objectDefaultPropertyNode(member, "/member", 1, true)
		if err != nil {
			t.Fatal(err)
		}
		if node.schema != target || node.pointer != "/components/schemas/Day" || node.ty.Kind != Scalar || !strings.Contains(node.ty.Name, "LocalDate") && !strings.Contains(node.ty.Name, "Instant") || work != 3 {
			t.Fatal("lost original temporal source, false or hop work", node, work)
		}
		a.objectDefaultWork = nil
		alias := a.componentsByOriginalSchema(get(referencedSchemas(root), "Alias"))
		saved := alias.Pointer
		alias.Pointer = "/wrong"
		if _, err = a.objectDefaultPropertyNode(member, "/member", 1, true); err == nil {
			t.Fatal("mismatched temporal hop admitted")
		}
		alias.Pointer = saved
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("rewrote source")
		}
	}
}
