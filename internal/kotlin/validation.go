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
	for i, p := range model.Properties {
		if helper := helpers[i]; helper != "" {
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
		if p.Validation != nil {
			if !hints[p.Validation] {
				continue
			}
		} else if !typeNeedsPrimitiveHelper(p.Type, hints) {
			continue
		}
		name := "openapiKtValidatePrimitive"
		for suffix := 2; used[strings.ToLower(name)]; suffix++ {
			name = fmt.Sprintf("openapiKtValidatePrimitive%d", suffix)
		}
		reserve(name)
		names[i] = name
	}
	return names
}

// Only plan-hinted typed leaves demand a model helper. The full property type
// and unchanged recursive checker retain null/Patch/container semantics.
func typeNeedsPrimitiveHelper(ty schema.TypeRef, hints schema.EmissionHints) bool {
	return hints[ty.Validation] || ty.Element != nil && typeNeedsPrimitiveHelper(*ty.Element, hints)
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
