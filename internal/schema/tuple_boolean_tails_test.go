package schema

import "testing"

func TestTupleBooleanTailRetainsOriginalSources(t *testing.T) {
	for _, tail := range []string{"false", "true"} {
		root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[{"type":"string"},{"type":"string"}],"items":`+tail+`}}`)
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.booleantails"}, components: map[string]*component{}, emissionHints: EmissionHints{}}
		if _, err := a.analyze(root); err != nil {
			t.Fatal(err)
		}
		c := a.components["Rows"]
		decision, err := a.arrayItems(c.Schema, c.Pointer, 0)
		prefix := get(c.Schema, "prefixItems").Items()
		if err != nil || len(decision.Prefix) != 2 || decision.Prefix[0].Schema != prefix[0] || decision.Prefix[1].Schema != prefix[1] {
			t.Fatal("Boolean tail copied or discarded original prefix sources", err)
		}
		if tail == "false" && (decision.Selected != prefix[0] || decision.Pointer != c.Pointer+"/prefixItems/0" || decision.Projection != nil) {
			t.Fatal("closed tail invented a selected source")
		}
		if tail == "true" && (decision.Selected != nil || decision.Projection == nil || decision.Projection.Kind != JSON || decision.Pointer != c.Pointer+"/prefixItems") {
			t.Fatal("unrestricted tail narrowed JSON kinds")
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("Boolean tail projection rewrote the original tree")
		}
	}
}
