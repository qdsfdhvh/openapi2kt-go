package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

const bigintSerializerName = "LongAsStringSerializer"

func nullableBigintCandidate(v *document.Value, kind string) bool {
	if has(v, "enum") || has(v, "const") || has(v, "x-kotlin-value-class") || text(get(v, "x-native-type")) != "bigint" {
		return false
	}
	switch kind {
	case "native", "string", "integer", "number", "boolean":
		return true
	default:
		return false
	}
}

// Bigint root intent changes only Kotlin nullability. Field-use string codecs
// remain separate from the unannotated native Long root/list/map codec.
func bigintBinding(c *component) bool {
	_, nullable, err := schemaType(c.Schema, c.Pointer)
	return err == nil && (!nullable || c.NullableBigint) && !c.ValueClass && c.Validation == nil &&
		!has(c.Schema, "enum") && !has(c.Schema, "const") &&
		text(get(c.Schema, "x-native-type")) == "bigint" && c.Scalar == "kotlin.Long"
}

// fieldSerializer is a field-use decision. A native bigint scalar in a simple
// root/array/map value remains an unannotated Long; a direct DTO field (including
// an actual referenced target) uses quoted decimal encoding.
func (a *analyzer) fieldSerializer(v *document.Value, pointer string, depth int) (string, *diagnostic.Diagnostic) {
	if a.booleanTrueValue(v) {
		return "", depthCheck(depth, pointer)
	}
	if err := depthCheck(depth, pointer); err != nil {
		return "", err
	}
	if has(v, "$ref") {
		component, err := a.resolveSchema(text(get(v, "$ref")), pointer)
		if err != nil {
			return "", err
		}
		return a.fieldSerializer(component.Schema, component.Pointer, depth+1)
	}
	wrapper, err := referenceWrapper(v, pointer)
	if err != nil {
		return "", err
	}
	if wrapper != nil {
		component, err := a.resolveSchema(wrapper.Reference, wrapper.Pointer)
		if err != nil {
			return "", err
		}
		return a.fieldSerializer(component.Schema, component.Pointer, depth+1)
	}
	// A nominal hierarchy reference is already validated by its own planner;
	// it is not a scalar use site and must not be re-read as a plain union.
	if has(v, "discriminator") {
		return "", nil
	}
	union, err := readUnion(v, pointer)
	if err != nil {
		return "", err
	}
	if union != nil && union.HasNull && len(union.Branches) == 1 {
		branch := union.Branches[0]
		return a.fieldSerializer(branch.Schema, branch.Pointer, depth+1)
	}
	// String/untyped enums are vocabulary declarations, not Long use sites.
	// Applying a bigint serializer to that nominal type would not compile.
	if has(v, "enum") {
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return "", err
		}
		if kind == "" || kind == "string" {
			return "", nil
		}
	}
	if text(get(v, "x-native-type")) == "bigint" {
		return a.runtimePackage() + "." + bigintSerializerName, nil
	}
	return "", nil
}

func withFieldSerializer(ty TypeRef, serializer string) TypeRef {
	if ty.Kind == Nullable {
		inner := withFieldSerializer(*ty.Element, serializer)
		ty.Element = &inner
	} else {
		ty.Serializer = serializer
	}
	return ty
}

func typeHasSerializer(ty TypeRef) bool {
	return ty.Serializer != "" || ty.Element != nil && typeHasSerializer(*ty.Element)
}

func declarationHasScalarSerializer(d Declaration) bool {
	switch value := d.(type) {
	case *Model:
		for _, property := range value.Properties {
			if typeHasSerializer(property.Type) {
				return true
			}
		}
		for _, constant := range value.Constants {
			if typeHasSerializer(constant.Type) {
				return true
			}
		}
		for _, nested := range value.Nested {
			if declarationHasScalarSerializer(nested) {
				return true
			}
		}
	case *ObjectUnion:
		for _, variant := range value.Variants {
			if declarationHasScalarSerializer(variant) {
				return true
			}
		}
	case *PrimitiveUnion:
		return value.Object != nil && declarationHasScalarSerializer(value.Object)
	case *Hierarchy:
		return value.Leaf != nil && declarationHasScalarSerializer(value.Leaf)
	}
	return false
}

func (a *analyzer) scalarRuntime(plan *Plan) *diagnostic.Diagnostic {
	declarations := append([]Declaration{}, plan.Models...)
	if plan.Client != nil {
		declarations = append(declarations, plan.Client.Models...)
	}
	for _, d := range declarations {
		if declarationHasScalarSerializer(d) {
			plan.BigintPackage = a.runtimePackage()
			break
		}
	}
	if plan.BigintPackage == a.options.PackageName+".models" {
		for _, component := range a.components {
			if strings.EqualFold(component.Name, bigintSerializerName) {
				return diagnostic.At("E_NAME_COLLISION", "A model conflicts with the required scalar serializer utility.", component.Pointer)
			}
		}
		// Hoisted hierarchy enums and dictionary peers also own root model files,
		// even when they are not catalog component identities.
		for _, declaration := range plan.Models {
			if strings.EqualFold(declaration.Name(), bigintSerializerName) {
				return diagnostic.New("E_NAME_COLLISION", "An emitted model conflicts with the required scalar serializer utility.")
			}
		}
	}
	return nil
}
