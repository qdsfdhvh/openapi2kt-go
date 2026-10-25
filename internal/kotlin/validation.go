package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func emitValueClass(d *schema.PrimitiveValueClass, out *strings.Builder) {
	fmt.Fprintf(out, "@kotlinx.serialization.Serializable\n@kotlin.jvm.JvmInline\npublic value class %s(\n    public val rawValue: %s,\n)", d.ClassName, d.Scalar)
	if d.Validation == nil {
		out.WriteString("\n")
		return
	}
	out.WriteString(" {\n    init {\n")
	emitPrimitiveChecks(d.Validation, d.Scalar, "rawValue", "rawValue", 2, out)
	out.WriteString("    }\n}\n")
}

func modelHasValidation(model *schema.Model) bool {
	for _, p := range model.Properties {
		if p.Validation != nil || typeHasValidation(p.Type) {
			return true
		}
	}
	return false
}

func emitModelValidation(model *schema.Model, indent int, hints schema.EmissionHints, out *strings.Builder) {
	if !modelHasValidation(model) {
		return
	}
	pad := strings.Repeat("    ", indent)
	helpers := primitiveHelperNames(model, hints)
	fmt.Fprintf(out, "%sinit {\n", pad)
	for i, p := range model.Properties {
		if p.Validation == nil {
			if helper := helpers[i]; helper != "" {
				fmt.Fprintf(out, "%s    this.%s(this.%s)\n", pad, helper, p.Name)
			} else {
				emitTypeChecks(p.Type, "this."+p.Name, "Field value", indent+1, out)
			}
			continue
		}
		receiver, call := "this."+p.Name, ".let"
		if p.Type.Kind == schema.Nullable {
			call = "?.let"
		}
		if p.Type.Kind == schema.PatchValue {
			receiver += ".valueOrNull"
			call = "?.let"
		}
		fmt.Fprintf(out, "%s    %s%s { value ->\n", pad, receiver, call)
		if helper := helpers[i]; helper != "" {
			fmt.Fprintf(out, "%s        this.%s(value)\n", pad, helper)
		} else {
			emitPrimitiveChecks(p.Validation, primitiveCheckScalar(p.Type), "value", "Field value", indent+2, out)
		}
		fmt.Fprintf(out, "%s    }\n", pad)
	}
	fmt.Fprintf(out, "%s}\n", pad)
	emitted := map[string]bool{}
	for i, p := range model.Properties {
		if helper := helpers[i]; helper != "" {
			if emitted[helper] {
				continue
			}
			emitted[helper] = true
			ty := p.Type
			if p.Validation != nil && (ty.Kind == schema.Nullable || ty.Kind == schema.PatchValue) {
				ty = *ty.Element
			}
			fmt.Fprintf(out, "\n%sprivate fun %s(value: %s) {\n", pad, helper, RenderType(ty))
			if p.Validation == nil {
				emitTypeChecks(p.Type, "value", "Field value", indent+1, out)
			} else {
				emitPrimitiveChecks(p.Validation, primitiveCheckScalar(p.Type), "value", "Field value", indent+1, out)
			}
			fmt.Fprintf(out, "%s}\n", pad)
		}
	}
}

// Each non-inline function contains one fixed primitive check set. No instance
// state, extra serializer field, cast, regex cache, or source-count expansion.
func primitiveHelperNames(model *schema.Model, hints schema.EmissionHints) map[int]string {
	names := map[int]string{}
	structural := map[string]string{}
	budgeted := inlineModelValidationTooLarge(model, hints)
	used := map[string]bool{"companion": true, "serializer": true, strings.ToLower(model.ClassName): true}
	reserve := func(name string) { used[strings.ToLower(strings.Trim(name, "`"))] = true }
	for _, p := range model.Properties {
		reserve(p.Name)
	}
	for _, c := range model.Constants {
		reserve(c.Name)
	}
	for _, d := range model.Nested {
		reserve(d.Name())
	}
	if model.PropertyCount != nil {
		reserve(propertyCountSerializerName(model))
	}
	for i, p := range model.Properties {
		sharedBudget := false
		if p.Validation != nil {
			if !hints[p.Validation] {
				continue
			}
		} else if !typeNeedsPrimitiveHelper(p.Type, hints) {
			if !budgeted || !typeHasInlineValidationTraversal(p.Type) {
				continue
			}
			sharedBudget = true
		}
		key := ""
		if p.Validation == nil && (typeUsesUnique(p.Type, structuralUniqueElement) || typeHasDTOUnique(p.Type)) {
			var checks strings.Builder
			emitTypeChecks(p.Type, "value", "Field value", 0, &checks)
			key = RenderType(p.Type) + "\n" + checks.String()
			if helper := structural[key]; helper != "" {
				names[i] = helper
				continue
			}
		} else if sharedBudget {
			key = "budget\n" + validationHelperKey(p)
			if helper := structural[key]; helper != "" {
				names[i] = helper
				continue
			}
		}
		name := "openapiKtValidatePrimitive"
		for suffix := 2; used[strings.ToLower(name)]; suffix++ {
			name = fmt.Sprintf("openapiKtValidatePrimitive%d", suffix)
		}
		reserve(name)
		names[i] = name
		if key != "" {
			structural[key] = name
		}
	}
	return names
}

// Bound the source expanded into one constructor's validation body. Nested
// inline collection loops make the JVM optimizer's control-flow work grow far
// beyond source width alone. Larger bodies use ordinary private typed methods;
// this changes neither input limits nor public types, checks, state or codecs.
const inlineValidationSourceLimit = 12 * 1024

func inlineModelValidationTooLarge(model *schema.Model, hints schema.EmissionHints) bool {
	var checks strings.Builder
	for _, p := range model.Properties {
		if p.Validation != nil || typeNeedsPrimitiveHelper(p.Type, hints) || !typeHasInlineValidationTraversal(p.Type) {
			continue
		}
		emitTypeChecks(p.Type, "value", "Field value", 0, &checks)
		if checks.Len() > inlineValidationSourceLimit {
			return true
		}
	}
	return false
}

func typeHasInlineValidationTraversal(ty schema.TypeRef) bool {
	if ty.Element == nil {
		return false
	}
	if ty.Kind == schema.List || ty.Kind == schema.Map {
		return typeHasValidation(*ty.Element)
	}
	return typeHasInlineValidationTraversal(*ty.Element)
}

// Only identical actual Kotlin parameter types and emitted checks share a
// method. Collection helpers retain the full original type, including nullable
// roots and Patch views. Existing scalar and shared uniqueness helpers stay put.
func validationHelperKey(p schema.Property) string {
	var checks strings.Builder
	emitTypeChecks(p.Type, "value", "Field value", 0, &checks)
	return RenderType(p.Type) + "\n" + checks.String()
}

// Only plan-hinted typed leaves demand a model helper. The full property type
// and unchanged recursive checker retain null/Patch/container semantics.
func typeNeedsPrimitiveHelper(ty schema.TypeRef, hints schema.EmissionHints) bool {
	return hints[ty.Validation] || typeUsesUnique(ty, structuralUniqueElement) || typeHasDTOUnique(ty) || ty.Element != nil && typeNeedsPrimitiveHelper(*ty.Element, hints)
}

// Messages use fixed labels, never raw values or source property names.
// Callers isolate local length/index/value variables in their check scopes.
func emitPrimitiveChecks(v *schema.PrimitiveValidation, scalar, value, label string, indent int, out *strings.Builder) {
	pad := strings.Repeat("    ", indent)
	if scalar == "kotlin.uuid.Uuid" {
		fmt.Fprintf(out, "%sval text = %s.toString()\n", pad, value)
		value = "text"
	}
	check := func(condition, message string) {
		fmt.Fprintf(out, "%srequire(%s) { %s }\n", pad, condition, naming.Literal(label+" "+message))
	}
	if v.MinLength != nil || v.MaxLength != nil {
		fmt.Fprintf(out, "%sval length = %s.indices.count { i -> i == 0 || !%s[i].isLowSurrogate() || !%s[i - 1].isHighSurrogate() }\n", pad, value, value, value)
		if v.MinLength != nil {
			check(fmt.Sprintf("length >= %d", *v.MinLength), "is shorter than minLength.")
		}
		if v.MaxLength != nil {
			check(fmt.Sprintf("length <= %d", *v.MaxLength), "is longer than maxLength.")
		}
	}
	if v.Pattern != nil {
		// A literal receiver avoids package lookup being shadowed by a DTO's
		// legitimate `kotlin` property. toRegex is a default String extension.
		check(naming.Literal(*v.Pattern)+".toRegex().containsMatchIn("+value+")", "does not match pattern.")
	}
	if v.Floating {
		check(value+".isFinite()", "must be finite.")
	}
	if len(v.Enum) != 0 {
		conditions := make([]string, len(v.Enum))
		for i, literal := range v.Enum {
			conditions[i] = value + " == " + literal
		}
		check(strings.Join(conditions, " || "), "does not belong to enum.")
	}
	if v.Minimum != nil {
		if v.Minimum.Exclusive {
			check(value+" > "+v.Minimum.Literal, "must be greater than exclusiveMinimum.")
		} else {
			check(value+" >= "+v.Minimum.Literal, "is below minimum.")
		}
	}
	if v.Maximum != nil {
		if v.Maximum.Exclusive {
			check(value+" < "+v.Maximum.Literal, "must be less than exclusiveMaximum.")
		} else {
			check(value+" <= "+v.Maximum.Literal, "is above maximum.")
		}
	}
	if v.ExclusiveMinimum != nil {
		check(value+" > "+v.ExclusiveMinimum.Literal, "must be greater than exclusiveMinimum.")
	}
	if v.ExclusiveMaximum != nil {
		check(value+" < "+v.ExclusiveMaximum.Literal, "must be less than exclusiveMaximum.")
	}
	if v.MultipleOf != nil {
		check(value+" % "+v.MultipleOf.Literal+" == "+v.MultipleOf.Zero, "does not satisfy multipleOf.")
	}
}

// Nullable and Patch checks operate on the non-null value extracted by the caller.
func primitiveCheckScalar(ty schema.TypeRef) string {
	for (ty.Kind == schema.Nullable || ty.Kind == schema.PatchValue) && ty.Element != nil {
		ty = *ty.Element
	}
	return ty.Name
}
