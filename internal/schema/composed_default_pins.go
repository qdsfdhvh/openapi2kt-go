package schema

import (
	"encoding/json"
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Match the existing emitted content/presence selector without applying defaults
// or changing numeric-token equality. Source and canonical literals use one rule.
func composedDefaultVariant(value *document.Value, union *ObjectUnion) int {
	for _, rule := range union.TagCombinations {
		matched := true
		for _, match := range rule.Matches {
			content, ok := unionTagContent(get(value, match.Wire))
			if !ok || !slices.Contains(match.Values, content) {
				matched = false
				break
			}
		}
		if matched {
			return rule.Index
		}
	}
	for _, rule := range union.Presence {
		if has(value, rule.Value) {
			return rule.Index
		}
	}
	return -1
}

// Mapped Float/Double formatting can change the lexical pins used by the
// existing runtime selector. The bounded factory literal must still decode to
// the constructor selected by the original annotation. Preserve only necessary
// root numeric pin tokens; never copy schemas or mutate original value nodes.
func preserveComposedDefaultVariant(value *document.Value, encoded, at string, union *ObjectUnion, selected int) (string, *diagnostic.Diagnostic) {
	decode := func(text string) (*document.Value, *diagnostic.Diagnostic) {
		// The original value already passed its 16,384-node/32KiB budget.
		// Mapped numeric formatting may temporarily grow its literal; the final
		// emitted literal is checked after any necessary pin restoration.
		parsed, err := document.ParseObject([]byte(text), document.MaxDocumentBytes)
		if err != nil {
			return nil, invalid("A composed default must encode to valid JSON.", at)
		}
		return parsed, nil
	}
	limit := func(text string) (string, *diagnostic.Diagnostic) {
		if len(text) > maxNestedDefaultBytes {
			return "", diagnostic.At("E_MODEL_LIMIT", "Canonical composed default exceeds the supported 32 KiB JSON byte limit.", at)
		}
		return text, nil
	}
	canonical, err := decode(encoded)
	if err != nil {
		return "", err
	}
	if composedDefaultVariant(canonical, union) == selected {
		return limit(encoded)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(encoded), &fields) != nil || fields == nil {
		return "", invalid("A composed default must encode to a JSON object.", at)
	}
	for _, rule := range union.TagCombinations {
		for _, match := range rule.Matches {
			original := get(value, match.Wire)
			if original.Kind() == document.Number {
				literal, marshalErr := original.MarshalJSON()
				if marshalErr != nil {
					return "", invalid("A composed numeric pin must encode to valid JSON.", at)
				}
				fields[match.Wire] = literal
			}
		}
	}
	restored, marshalErr := json.Marshal(fields)
	if marshalErr != nil {
		return "", invalid("A composed default must encode to valid JSON.", at)
	}
	checked, err := decode(string(restored))
	if err != nil {
		return "", err
	}
	if composedDefaultVariant(checked, union) != selected {
		return "", unsupported(at)
	}
	return limit(string(restored))
}
