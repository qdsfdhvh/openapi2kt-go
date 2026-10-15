package schema

import "openapi-kt/internal/document"

// Source eligibility is independent of public nullable alias spelling. Malformed
// format values in a potential string temporal still receive scalar grammar
// diagnostics in the normal catalog pass; this predicate validates no payloads.
func nullableTemporalCandidate(v *document.Value, kind string) bool {
	if has(v, "enum") || has(v, "const") || has(v, "x-kotlin-value-class") {
		return false
	}
	if has(v, "x-native-type") {
		return text(get(v, "x-native-type")) == "date" && (kind == "native" || kind == "string" || kind == "integer" || kind == "number" || kind == "boolean")
	}
	format := get(v, "format")
	return kind == "string" && has(v, "format") && (text(format) == "date" || text(format) == "date-time" || format.Kind() != document.String)
}

// Temporal root intent is limited to the existing date/date-time/date-native
// profiles, not arbitrary caller classifiers that happen to have these names.
func temporalBinding(c *component) bool {
	_, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable && !c.NullableTemporal || c.ValueClass || has(c.Schema, "enum") || has(c.Schema, "const") || c.Validation != nil {
		return false
	}
	if has(c.Schema, "x-native-type") {
		return text(get(c.Schema, "x-native-type")) == "date" && c.Scalar == "kotlin.time.Instant"
	}
	return c.Scalar == "kotlin.time.Instant" && text(get(c.Schema, "format")) == "date-time" ||
		c.Scalar == "kotlinx.datetime.LocalDate" && text(get(c.Schema, "format")) == "date"
}
