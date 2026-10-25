package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/schema"
)

const structuralUniqueHelperName = "`openapi-kt unique structural items`"

func structuralUniqueElement(t schema.TypeRef) bool {
	for t.Kind == schema.Nullable || t.Kind == schema.Named && t.Element != nil {
		t = *t.Element
	}
	return (t.Kind == schema.List || t.Kind == schema.Map) && !mapUniqueElement(t)
}

// Normalize through typed operations, without reflection or serialization.
// The analyzer admits only finite List/Map trees with mapped scalar/enum leaves.
func structuralUniqueKey(t schema.TypeRef, receiver string, depth int) string {
	name := fmt.Sprintf("item%d", depth)
	if t.Kind == schema.Named && t.Element != nil {
		target := *t.Element
		if t.ArrayAliasView == schema.NonNullArrayAlias {
			target = target.WithoutRootNull()
		}
		return structuralUniqueKey(target, receiver, depth+1)
	}
	if t.Kind == schema.Nullable {
		return receiver + "?.let { " + name + " -> " + structuralUniqueKey(*t.Element, name, depth+1) + " }"
	}
	switch t.Kind {
	case schema.List:
		return receiver + ".map { " + name + " -> " + structuralUniqueKey(*t.Element, name, depth+1) + " }"
	case schema.Map:
		return receiver + ".mapValues { (_, " + name + ") -> " + structuralUniqueKey(*t.Element, name, depth+1) + " }"
	case schema.Named:
		return receiver + ".serialName"
	case schema.Scalar:
		if t.Name == "kotlin.Float" || t.Name == "kotlin.Double" {
			zero := "0.0"
			if t.Name == "kotlin.Float" {
				zero += "f"
			}
			return receiver + ".let { " + name + " -> require(" + name + ".isFinite()) { \"Field value contains non-finite entries for uniqueItems.\" }; if (" + name + " == " + zero + ") " + zero + " else " + name + " }"
		}
	}
	return receiver
}

func emitStructuralUniqueHelper(out *strings.Builder) {
	fmt.Fprintf(out, "\nprivate fun <T> %s(input: kotlin.collections.List<T>) {\n", structuralUniqueHelperName)
	out.WriteString("    require(input.toSet().size == input.size) { \"Field value contains duplicate entries for uniqueItems.\" }\n}\n")
}
