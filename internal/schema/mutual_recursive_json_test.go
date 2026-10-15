package schema

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func jsonGraphAnalyzer(t *testing.T, schemas string) *analyzer {
	t.Helper()
	root, err := document.ParseObject([]byte(`{"openapi":"3.1.0","components":{"schemas":{`+schemas+`}}}`), document.MaxDocumentBytes)
	if err != nil {
		t.Fatal(err)
	}
	return &analyzer{root: root}
}

func jsonGraphCover(array, object string) string {
	return `{"oneOf":[{"type":"null"},{"type":"string"},{"type":"boolean"},{"type":"number"},{"type":"array","items":{"$ref":"#/components/schemas/` + array + `"}},{"type":"object","additionalProperties":{"$ref":"#/components/schemas/` + object + `"}}]}`
}

func TestMutualJSONGraphSCCHeightsAndOriginalNodes(t *testing.T) {
	// Dense cycles require a graph algorithm: revisiting the same descendants
	// as trees exhausts work exponentially. Heights must not depend on entry.
	var entries []string
	for i := 0; i < 20; i++ {
		entries = append(entries, fmt.Sprintf(`"R%d":%s`, i, jsonGraphCover(fmt.Sprint("R", (i+1)%20), fmt.Sprint("R", (i+2)%20))))
	}
	a := jsonGraphAnalyzer(t, strings.Join(entries, ","))
	schemas := get(get(a.root, "components"), "schemas")
	for i := 0; i < 20; i++ {
		name := fmt.Sprint("R", i)
		v := get(schemas, name)
		h, err := a.recursiveJSONRoot(v, "/components/schemas/"+name)
		if err != nil || h != 40 || v != get(schemas, name) {
			t.Fatal(i, h, err)
		}
	}
	// A cyclic pair with an outgoing single-root cover retains both heights.
	a = jsonGraphAnalyzer(t, `"A":`+jsonGraphCover("B", "C")+`,"B":`+jsonGraphCover("A", "C")+`,"C":`+jsonGraphCover("C", "C"))
	schemas = get(get(a.root, "components"), "schemas")
	for name, want := range map[string]int{"A": 6, "B": 6, "C": 2} {
		h, err := a.recursiveJSONRoot(get(schemas, name), "/components/schemas/"+name)
		if err != nil || h != want {
			t.Fatal(name, h, err)
		}
	}
}

func TestRecursiveJSONTransparentGraphWeights(t *testing.T) {
	a := jsonGraphAnalyzer(t, `"A":`+jsonGraphCover("ToB", "B")+`,"ToB":{"$ref":"#/components/schemas/B"},"B":`+jsonGraphCover("ToA", "A")+`,"ToA":{"$ref":"#/components/schemas/A"}`)
	schemas := get(get(a.root, "components"), "schemas")
	for _, name := range []string{"A", "B"} {
		h, err := a.recursiveJSONRoot(get(schemas, name), "/components/schemas/"+name)
		if err != nil || h != 6 {
			t.Fatal(name, h, err)
		}
	}
	a = jsonGraphAnalyzer(t, `"A":`+jsonGraphCover("Loop", "A")+`,"Loop":{"$ref":"#/components/schemas/Loop"}`)
	h, err := a.recursiveJSONRoot(get(get(get(a.root, "components"), "schemas"), "A"), "/components/schemas/A")
	if err != nil || h != 0 || len(a.recursiveJSONHeights) != 0 {
		t.Fatal("ref-only loop or failed partial cache", h, err)
	}
}

func TestMutualJSONGraphDepthBoundary(t *testing.T) {
	for _, n := range []int{24, 25} {
		var entries []string
		for i := 0; i < n; i++ {
			next := fmt.Sprint("R", (i+1)%n)
			entries = append(entries, fmt.Sprintf(`"R%d":%s`, i, jsonGraphCover(next, next)))
		}
		a := jsonGraphAnalyzer(t, strings.Join(entries, ","))
		v := get(get(get(a.root, "components"), "schemas"), "R0")
		h, err := a.recursiveJSONRoot(v, "/components/schemas/R0")
		if n == 24 {
			if err != nil || h != 48 {
				t.Fatal(h, err)
			}
		} else if err == nil || err.Code != "E_SCHEMA_DEPTH" {
			t.Fatal(h, err)
		}
	}
}
