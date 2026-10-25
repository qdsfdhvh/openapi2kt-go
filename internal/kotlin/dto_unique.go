package kotlin

import (
	"strings"

	"openapi-kt/internal/schema"
)

func typeHasDTOUnique(t schema.TypeRef) bool {
	return t.Collection != nil && t.Collection.UniqueDTO != nil || t.Element != nil && typeHasDTOUnique(*t.Element)
}

// Getter keys preserve mapped primitive equality in both generated class
// styles. The source planner proves the exact original constructor fields.
func dtoUniqueKey(t schema.TypeRef, key *schema.DTOUniqueKey, receiver string) string {
	value := receiver
	if t.Kind == schema.Nullable {
		value = "dto"
	}
	fields := make([]string, 0, len(key.Fields))
	for _, field := range key.Fields {
		getter := value + "." + field.Name
		scalar := field.Type.WithoutRootNull()
		if scalar.Kind == schema.Scalar && scalar.Name != "kotlin.Float" && scalar.Name != "kotlin.Double" {
			fields = append(fields, getter)
		} else {
			fields = append(fields, structuralUniqueKey(field.Type, getter, 0))
		}
	}
	expression := "kotlin.collections.listOf<kotlin.Any?>(" + strings.Join(fields, ", ") + ")"
	if t.Kind == schema.Nullable {
		return receiver + "?.let { dto -> " + expression + " }"
	}
	return expression
}
