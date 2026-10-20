package kotlin

import (
	"strings"

	"openapi-kt/internal/schema"
)

// Private file-level helpers stay outside constructor scopes where legitimate
// DTO fields and nested classifiers may shadow temporal package qualifiers.
func emitDefaultHelpers(declaration schema.Declaration, out *strings.Builder) {
	instant, date, uuid := false, false, false
	var visit func(schema.Declaration)
	visit = func(d schema.Declaration) {
		switch d := d.(type) {
		case *schema.Model:
			for _, c := range d.Constants {
				uuid = uuid || c.Type.Name == "kotlin.uuid.Uuid"
			}
			for _, p := range d.Properties {
				if p.Default == "" {
					continue
				}
				target := p.Type.WithoutRootNull()
				instant = instant || target.Name == "kotlin.time.Instant"
				date = date || target.Name == "kotlinx.datetime.LocalDate"
				uuid = uuid || target.Name == "kotlin.uuid.Uuid"
			}
			for _, n := range d.Nested {
				visit(n)
			}
		case *schema.ObjectUnion:
			for _, v := range d.Variants {
				visit(v)
			}
		case *schema.PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		case *schema.Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf)
			}
		}
	}
	visit(declaration)
	if uuid {
		out.WriteString("\nprivate fun `openapi-kt default uuid`(value: kotlin.String): kotlin.uuid.Uuid = kotlin.uuid.Uuid.parse(value)\n")
	}
	if instant {
		out.WriteString("\nprivate fun `openapi-kt default instant`(value: kotlin.String): kotlin.time.Instant = kotlin.time.Instant.parse(value)\n")
	}
	if date {
		out.WriteString("\nprivate fun `openapi-kt default date`(value: kotlin.String): kotlinx.datetime.LocalDate = kotlinx.datetime.LocalDate.parse(value)\n")
	}
}
