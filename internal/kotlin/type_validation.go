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
			emitUniqueChecks("value", indent+1, out)
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

func emitUniqueChecks(value string, indent int, out *strings.Builder) {
	fmt.Fprintf(out, "%s%s(%s)\n", strings.Repeat("    ", indent), uniqueHelperName, value)
}

func emitUniqueHelper(out *strings.Builder) {
	fmt.Fprintf(out, "\nprivate fun <T : kotlin.Comparable<T>> %s(input: kotlin.collections.List<T?>) {\n", uniqueHelperName)
	out.WriteString("    require(input.count { it == null } <= 1) { \"Field value contains duplicate entries for uniqueItems.\" }\n")
	out.WriteString("    val sorted = input.filterNotNull().sorted()\n")
	out.WriteString("    for (index in 1 until sorted.size) {\n")
	out.WriteString("        require(sorted[index - 1] != sorted[index]) { \"Field value contains duplicate entries for uniqueItems.\" }\n")
	out.WriteString("    }\n}\n")
}

func typeHasUnique(t schema.TypeRef) bool {
	return t.Collection != nil && t.Collection.Unique || t.Element != nil && typeHasUnique(*t.Element)
}

// Only executable consumers need a helper: standalone typealiases deliberately
// do not enforce checks. Inspect semantic IR rather than source text/literals.
func declarationHasUnique(d schema.Declaration) bool {
	switch d := d.(type) {
	case *schema.Model:
		for _, p := range d.Properties {
			if typeHasUnique(p.Type) {
				return true
			}
		}
		for _, nested := range d.Nested {
			if declarationHasUnique(nested) {
				return true
			}
		}
	case *schema.ObjectUnion:
		for _, variant := range d.Variants {
			if declarationHasUnique(variant) {
				return true
			}
		}
	case *schema.PrimitiveUnion:
		return d.Object != nil && declarationHasUnique(d.Object)
	case *schema.Hierarchy:
		return d.Leaf != nil && declarationHasUnique(d.Leaf)
	case *schema.SSEEvents:
		for _, variant := range d.Variants {
			if variant.Data != nil && typeHasUnique(*variant.Data) {
				return true
			}
		}
	}
	return false
}

func groupHasUnique(group schema.APIGroup) bool {
	for _, op := range group.Operations {
		for _, p := range op.Parameters {
			if typeHasUnique(p.Type) {
				return true
			}
		}
		if op.Body != nil && typeHasUnique(op.Body.Type) || typeHasUnique(op.Response) {
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
