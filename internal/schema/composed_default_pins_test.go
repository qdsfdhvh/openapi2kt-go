package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestComposedDefaultKeepsSelectedVariantAfterNumericFormatting(t *testing.T) {
	rule := func(values []string, index int) ObjectTagRule {
		return ObjectTagRule{Matches: []ObjectTagMatch{{Wire: "n", Values: values}}, Index: index}
	}
	for _, tc := range []struct {
		name, original, canonical string
		rules                     []ObjectTagRule
		presence                  []DispatchRule
		changed                   bool
	}{
		{"float32 rounding", `{"n":16777217}`, `{"n":1.6777216e+07}`, []ObjectTagRule{rule([]string{"16777217", "16777219"}, 0)}, nil, true},
		{"float64 rounding", `{"n":9007199254740993}`, `{"n":9.007199254740992e+15}`, []ObjectTagRule{rule([]string{"9007199254740993", "9007199254740995"}, 0)}, nil, true},
		{"integral float token", `{"n":1.0}`, `{"n":1}`, []ObjectTagRule{rule([]string{"1.0"}, 0)}, nil, true},
		{"exponent token", `{"n":1e-9}`, `{"n":1e-09}`, []ObjectTagRule{rule([]string{"1e-9"}, 0)}, nil, true},
		{"different known variant", `{"n":1.0}`, `{"n":1}`, []ObjectTagRule{rule([]string{"1.0"}, 0), rule([]string{"1"}, 1)}, nil, true},
		{"presence becomes another tag", `{"n":1e0,"a":"present"}`, `{"a":"present","n":1}`, []ObjectTagRule{rule([]string{"1.0"}, 0), rule([]string{"1"}, 1)}, []DispatchRule{{Value: "a", Index: 0}}, true},
		{"same variant keeps canonical bytes", `{"n":1.0}`, `{"n":1}`, []ObjectTagRule{rule([]string{"1", "1.0"}, 0)}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original, err := document.ParseObject([]byte(tc.original), document.MaxDocumentBytes)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := original.MarshalJSON()
			union := &ObjectUnion{TagCombinations: tc.rules, Presence: tc.presence}
			if composedDefaultVariant(original, union) != 0 {
				t.Fatal("bad original selection control")
			}
			fixed, failure := preserveComposedDefaultVariant(original, tc.canonical, "/default", union, 0)
			if failure != nil || (fixed != tc.canonical) != tc.changed {
				t.Fatal("pin restoration or stable literal changed", fixed, failure)
			}
			parsed, err := document.ParseObject([]byte(fixed), document.MaxDocumentBytes)
			if err != nil || composedDefaultVariant(parsed, union) != 0 {
				t.Fatal("encoded default changed selected constructor", fixed, err)
			}
			again, failure := preserveComposedDefaultVariant(original, tc.canonical, "/default", union, 0)
			if failure != nil || fixed != again {
				t.Fatal("pin restoration nondeterminism", failure)
			}
			after, _ := original.MarshalJSON()
			if string(before) != string(after) {
				t.Fatal("pin restoration mutated original value nodes")
			}
		})
	}
}

func TestComposedDefaultPinRestorationFinalByteBudget(t *testing.T) {
	union := &ObjectUnion{TagCombinations: []ObjectTagRule{{Matches: []ObjectTagMatch{{Wire: "n", Values: []string{"16777217"}}}, Index: 0}}}
	header := `{"n":16777217,"padding":""}`
	source := strings.Replace(header, `"padding":""`, `"padding":"`+strings.Repeat("x", maxNestedDefaultBytes-len(header))+`"`, 1)
	original, err := document.ParseObject([]byte(source), document.MaxDocumentBytes)
	if err != nil {
		t.Fatal(err)
	}
	work, bytes := 0, 0
	if err := nestedDefaultBudget(original, "/default", 0, &work, &bytes); err != nil || bytes != maxNestedDefaultBytes {
		t.Fatal("exact original budget control failed", bytes, err)
	}
	canonical := strings.Replace(source, "16777217", "1.6777216e+07", 1)
	fixed, failure := preserveComposedDefaultVariant(original, canonical, "/default", union, 0)
	if failure != nil || len(fixed) != maxNestedDefaultBytes {
		t.Fatal("bounded final pin literal rejected because of intermediate formatting", len(fixed), failure)
	}

	source = `{"kind":"a","values":[` + strings.TrimSuffix(strings.Repeat("1e5,", 8000), ",") + `]}`
	original, err = document.ParseObject([]byte(source), document.MaxDocumentBytes)
	if err != nil {
		t.Fatal(err)
	}
	work, bytes = 0, 0
	if err := nestedDefaultBudget(original, "/default", 0, &work, &bytes); err != nil {
		t.Fatal("original compact numeric array must fit", err)
	}
	union = &ObjectUnion{TagCombinations: []ObjectTagRule{{Matches: []ObjectTagMatch{{Wire: "kind", Values: []string{"a"}}}, Index: 0}}}
	canonical = strings.ReplaceAll(source, "1e5", "100000")
	if _, err := preserveComposedDefaultVariant(original, canonical, "/default", union, 0); err == nil || err.Code != "E_MODEL_LIMIT" || err.Pointer == nil || *err.Pointer != "/default" {
		t.Fatal("canonical expansion escaped final 32KiB literal budget", err)
	}
}
