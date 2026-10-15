package kotlin

import "openapi-kt/internal/schema"

// Ktor's default Int response transformer dispatches on the erased classifier
// and parses "null" as a non-null integer before ContentNegotiation can decode
// it. Nullable JSON Int roots therefore use the configured JSON codec directly.
// Non-null Int and existing raw String/ByteArray policies remain unchanged.
func nullableIntResponse(op schema.Operation) bool {
	t := op.Response
	return op.SSE == nil && t.Kind == schema.Nullable && t.Element != nil && t.Element.Kind == schema.Scalar && t.Element.Name == "kotlin.Int"
}
func groupHasNullableIntResponse(g schema.APIGroup) bool {
	for _, op := range g.Operations {
		if nullableIntResponse(op) {
			return true
		}
	}
	return false
}
func groupNeedsJSON(g schema.APIGroup) bool { return groupHasSSE(g) || groupHasNullableIntResponse(g) }
func clientNeedsJSON(c *schema.Client) bool {
	for _, g := range c.Groups {
		if groupNeedsJSON(g) {
			return true
		}
	}
	return false
}
