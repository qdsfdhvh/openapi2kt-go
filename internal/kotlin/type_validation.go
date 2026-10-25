package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func typeHasValidation(t schema.TypeRef) bool {
	if t.Validation != nil || t.Collection != nil {
		return true
	}
	return t.Element != nil && typeHasValidation(*t.Element)
}

// Each receiver lives in a local lambda to avoid collisions among sibling
// checks and schema properties. Scalar alias/inline element leaves carry checks;
// named DTOs/value classes already enforce their own constructor init.
func emitTypeChecks(t schema.TypeRef, receiver, label string, indent int, out *strings.Builder) {
	if !typeHasValidation(t) {
		return
	}
	pad := strings.Repeat("    ", indent)
	if t.Collection != nil {
		fmt.Fprintf(out, "%s%s.let { value ->\n", pad, receiver)
		emitCollectionChecks(t.Collection, "value", label, indent+1, out)
		if t.Collection.Unique {
			if t.Collection.UniqueDTO != nil {
				fmt.Fprintf(out, "%s    val keys = value.map { item -> %s }\n", pad, dtoUniqueKey(*t.Element, t.Collection.UniqueDTO, "item"))
				fmt.Fprintf(out, "%s    require(keys.toSet().size == keys.size) { \"Field value contains duplicate entries for uniqueItems.\" }\n", pad)
			} else if structuralUniqueElement(*t.Element) {
				fmt.Fprintf(out, "%s    %s(value.map { item -> %s })\n", pad, structuralUniqueHelperName, structuralUniqueKey(*t.Element, "item", 0))
			} else if mapUniqueElement(*t.Element) {
				fmt.Fprintf(out, "%s    %s(value)\n", pad, mapUniqueHelperName)
			} else if floatingUniqueElement(*t.Element) {
				fmt.Fprintf(out, "%s    %s(value)\n", pad, floatingUniqueHelperName)
			} else {
				value := "value"
				leaf := t.Element.WithoutRootNull()
				if leaf.Kind == schema.Named && leaf.Element == nil {
					// The analyzer admits nominal uniqueness only for an original
					// String enum source. Compare payloads so a known member and an
					// Unrecognized instance with the same wire token collide.
					member := "it.serialName"
					if t.Element.Kind == schema.Nullable {
						member = "it?.serialName"
					}
					value += ".map { " + member + " }"
				}
				emitUniqueChecks(value, indent+1, out)
			}
		}
		if t.Collection.UniqueJSON {
			fmt.Fprintf(out, "%s    `openapi-kt unique JSON items`(value)\n", pad)
		}
		if t.Element != nil && typeHasValidation(*t.Element) {
			child := "value"
			if t.Kind == schema.Map {
				child += ".values"
			}
			fmt.Fprintf(out, "%s    %s.forEach { value ->\n", pad, child)
			emitTypeChecks(*t.Element, "value", label, indent+2, out)
			fmt.Fprintf(out, "%s    }\n", pad)
		}
		fmt.Fprintf(out, "%s}\n", pad)
		return
	}
	switch t.Kind {
	case schema.Named:
		if t.Validation != nil {
			// Direct checked open-enum leaves retain their nominal type and codec.
			// Validate the actual payload, including Unrecognized, at this field.
			fmt.Fprintf(out, "%s%s.serialName.let { value ->\n", pad, receiver)
			emitPrimitiveChecks(t.Validation, "kotlin.String", "value", label, indent+1, out)
			fmt.Fprintf(out, "%s}\n", pad)
			return
		}
		target := *t.Element
		if t.ArrayAliasView == schema.NonNullArrayAlias {
			target = target.WithoutRootNull()
		}
		emitTypeChecks(target, receiver, label, indent, out)
		return
	case schema.Nullable:
		fmt.Fprintf(out, "%s%s?.let { value ->\n", pad, receiver)
		emitTypeChecks(*t.Element, "value", label, indent+1, out)
	case schema.PatchValue:
		fmt.Fprintf(out, "%s%s.valueOrNull?.let { value ->\n", pad, receiver)
		emitTypeChecks(*t.Element, "value", label, indent+1, out)
	case schema.List, schema.Map:
		if t.Kind == schema.Map {
			receiver += ".values"
		}
		fmt.Fprintf(out, "%s%s.forEach { value ->\n", pad, receiver)
		emitTypeChecks(*t.Element, "value", label, indent+1, out)
	default:
		fmt.Fprintf(out, "%s%s.let { value ->\n", pad, receiver)
		emitPrimitiveChecks(t.Validation, t.Name, "value", label, indent+1, out)
	}
	fmt.Fprintf(out, "%s}\n", pad)
}

func emitCollectionChecks(v *schema.CollectionValidation, value, label string, indent int, out *strings.Builder) {
	pad := strings.Repeat("    ", indent)
	if v.Minimum != nil {
		fmt.Fprintf(out, "%srequire(%s.size >= %d) { %s }\n", pad, value, *v.Minimum, naming.Literal(label+" has fewer entries than "+v.MinimumKey+"."))
	}
	if v.Maximum != nil {
		fmt.Fprintf(out, "%srequire(%s.size <= %d) { %s }\n", pad, value, *v.Maximum, naming.Literal(label+" has more entries than "+v.MaximumKey+"."))
	}
}

// A private, non-inline file helper keeps 240-field constructors below the JVM
// method bytecode limit. Its escaped name cannot be produced by schema naming:
// generated identifiers contain ASCII letters/digits/underscores, never spaces.
const uniqueHelperName = "`openapi-kt uniqueItems`"
const mapUniqueHelperName = "`openapi-kt unique map items`"
const floatingUniqueHelperName = "`openapi-kt unique floating items`"

func mapUniqueElement(t schema.TypeRef) bool {
	for t.Kind == schema.Nullable || t.Kind == schema.Named && t.Element != nil {
		t = *t.Element
	}
	if t.Kind != schema.Map || t.Element == nil {
		return false
	}
	leaf := *t.Element
	for leaf.Kind == schema.Nullable || leaf.Kind == schema.Named && leaf.Element != nil {
		leaf = *leaf.Element
	}
	return leaf.Kind == schema.Scalar && leaf.Name != "kotlin.Float" && leaf.Name != "kotlin.Double"
}

func floatingUniqueElement(t schema.TypeRef) bool {
	t = t.WithoutRootNull()
	return t.Kind == schema.Scalar && (t.Name == "kotlin.Float" || t.Name == "kotlin.Double")
}

func emitUniqueChecks(value string, indent int, out *strings.Builder) {
	fmt.Fprintf(out, "%s%s(%s)\n", strings.Repeat("    ", indent), uniqueHelperName, value)
}

func emitUniqueHelper(out *strings.Builder, floating, maps bool) {
	fmt.Fprintf(out, "\nprivate fun <T : kotlin.Comparable<T>> %s(input: kotlin.collections.List<T?>) {\n", uniqueHelperName)
	out.WriteString("    require(input.count { it == null } <= 1) { \"Field value contains duplicate entries for uniqueItems.\" }\n")
	out.WriteString("    val sorted = input.filterNotNull().sorted()\n")
	out.WriteString("    for (index in 1 until sorted.size) {\n")
	out.WriteString("        require(sorted[index - 1] != sorted[index]) { \"Field value contains duplicate entries for uniqueItems.\" }\n")
	out.WriteString("    }\n}\n")
	if maps {
		fmt.Fprintf(out, "\nprivate fun <T> %s(input: kotlin.collections.List<kotlin.collections.Map<kotlin.String, T>?>) {\n", mapUniqueHelperName)
		out.WriteString("    require(input.toSet().size == input.size) { \"Field value contains duplicate entries for uniqueItems.\" }\n}\n")
	}
	if floating {
		fmt.Fprintf(out, "\nprivate fun <T : kotlin.Number> %s(input: kotlin.collections.List<T?>) {\n", floatingUniqueHelperName)
		out.WriteString("    require(input.count { it == null } <= 1) { \"Field value contains duplicate entries for uniqueItems.\" }\n")
		// Widening Float preserves its already mapped value exactly. Concrete
		// Double equality below treats signed zeros equally; generic/boxed
		// Comparable equality deliberately remains unchanged for other leaves.
		out.WriteString("    val values = input.filterNotNull().map { it.toDouble() }\n")
		out.WriteString("    require(values.all { it.isFinite() }) { \"Field value contains non-finite entries for uniqueItems.\" }\n")
		out.WriteString("    val sorted = values.sorted()\n")
		out.WriteString("    for (index in 1 until sorted.size) {\n")
		out.WriteString("        require(sorted[index - 1] != sorted[index]) { \"Field value contains duplicate entries for uniqueItems.\" }\n")
		out.WriteString("    }\n}\n")
	}
}

func typeHasUnique(t schema.TypeRef) bool {
	return typeUsesUnique(t, nil)
}

func typeUsesUnique(t schema.TypeRef, accept func(schema.TypeRef) bool) bool {
	return t.Collection != nil && t.Collection.Unique && (accept == nil || t.Element != nil && accept(*t.Element)) ||
		t.Element != nil && typeUsesUnique(*t.Element, accept)
}

// Only executable consumers need a helper: standalone typealiases deliberately
// do not enforce checks. Inspect semantic IR rather than source text/literals.
func declarationHasUnique(d schema.Declaration) bool {
	return declarationUsesUnique(d, nil)
}

func declarationUsesUnique(d schema.Declaration, accept func(schema.TypeRef) bool) bool {
	switch d := d.(type) {
	case *schema.Model:
		for _, p := range d.Properties {
			if typeUsesUnique(p.Type, accept) {
				return true
			}
		}
		for _, nested := range d.Nested {
			if declarationUsesUnique(nested, accept) {
				return true
			}
		}
	case *schema.ObjectUnion:
		for _, variant := range d.Variants {
			if declarationUsesUnique(variant, accept) {
				return true
			}
		}
	case *schema.PrimitiveUnion:
		return d.Object != nil && declarationUsesUnique(d.Object, accept)
	case *schema.Hierarchy:
		return d.Leaf != nil && declarationUsesUnique(d.Leaf, accept)
	case *schema.SSEEvents:
		for _, variant := range d.Variants {
			if variant.Data != nil && typeUsesUnique(*variant.Data, accept) {
				return true
			}
		}
	}
	return false
}

func groupHasUnique(group schema.APIGroup) bool {
	return groupUsesUnique(group, nil)
}

func groupUsesUnique(group schema.APIGroup, accept func(schema.TypeRef) bool) bool {
	for _, op := range group.Operations {
		for _, p := range op.Parameters {
			if typeUsesUnique(p.Type, accept) {
				return true
			}
		}
		if op.Body != nil && typeUsesUnique(op.Body.Type, accept) || typeUsesUnique(op.Response, accept) {
			return true
		}
	}
	return false
}

// Wide checked operations must not inline every container/leaf loop into one
// JVM method. Keep smaller historical operations unchanged; a wide operation
// shares only byte-identical checks with the same rendered input type. Helpers
// are private and non-inline, and retain the original preflight order/labels.
type operationCheckPlan struct {
	Names        map[int]string
	Declarations string
}

func planOperationInputChecks(op schema.Operation, ordinal int) operationCheckPlan {
	types := make([]schema.TypeRef, 0, len(op.Parameters)+1)
	checked := 0
	for _, p := range op.Parameters {
		types = append(types, p.Type)
		if typeHasValidation(p.Type) {
			checked++
		}
	}
	if op.Body != nil {
		types = append(types, op.Body.Type)
		if typeHasValidation(op.Body.Type) {
			checked++
		}
	}
	plan := operationCheckPlan{Names: map[int]string{}}
	if checked < 64 {
		return plan
	}
	seen := map[string]string{}
	var declarations strings.Builder
	for index, ty := range types {
		if !typeHasValidation(ty) {
			continue
		}
		var checks strings.Builder
		emitTypeChecks(ty, "input", "Field value", 2, &checks)
		key := RenderType(ty) + "\n" + checks.String()
		name := seen[key]
		if name == "" {
			name = fmt.Sprintf("`openapi-kt validate input %d %d`", ordinal, len(seen)+1)
			seen[key] = name
			fmt.Fprintf(&declarations, "    private fun %s(input: %s) {\n%s    }\n\n", name, RenderType(ty), checks.String())
		}
		plan.Names[index] = name
	}
	plan.Declarations = declarations.String()
	return plan
}

func emitOperationInputChecks(op schema.Operation, plan operationCheckPlan, indent int, out *strings.Builder) {
	emit := func(index int, ty schema.TypeRef, receiver string) {
		if name := plan.Names[index]; name != "" {
			fmt.Fprintf(out, "%s%s(%s)\n", strings.Repeat("    ", indent), name, receiver)
		} else {
			emitTypeChecks(ty, receiver, "Field value", indent, out)
		}
	}
	for index, p := range op.Parameters {
		emit(index, p.Type, naming.CodeName(p.Name))
	}
	if op.Body != nil {
		emit(len(op.Parameters), op.Body.Type, "input")
	}
}
