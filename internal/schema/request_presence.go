package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// UsageContext separates field defaults from the context of nested objects.
// Root request unions deliberately retain SharedUsage, matching response defaults.
type UsageContext uint8

const (
	SharedUsage UsageContext = iota
	RequestObjectUsage
	RequestVariantUsage
)

func declarationHasPatch(d Declaration) bool {
	switch value := d.(type) {
	case *Model:
		return ModelHasPatch(value)
	case *ObjectUnion:
		for _, variant := range value.Variants {
			if ModelHasPatch(variant) {
				return true
			}
		}
	}
	return false
}

// ModelHasPatch reports request-only carriers throughout nested declaration trees.
func ModelHasPatch(model *Model) bool {
	for _, p := range model.Properties {
		if p.Type.Kind == PatchValue {
			return true
		}
	}
	for _, d := range model.Nested {
		if declarationHasPatch(d) {
			return true
		}
	}
	return false
}

// Presence follows explicit schema intent, not an alias's incidental nullable
// Kotlin spelling. A reference's own nullable override takes precedence.
func (a *analyzer) requestNullable(v *document.Value, pointer string) (bool, *diagnostic.Diagnostic) {
	// JsonNull is a JSON value, not an implicit nullable/Patch intent.
	if a.booleanTrueValue(v) {
		return false, nil
	}
	if has(v, "nullable") {
		value, _ := get(v, "nullable").Bool()
		return value, nil
	}
	// Borrowed terminal wrappers follow the same effective nullable intent as
	// their typed use. Named wrappers deliberately keep their historical policy.
	wrapper, err := referenceWrapper(v, pointer)
	if err != nil {
		return false, err
	}
	if wrapper != nil {
		target, err := a.resolveSchema(wrapper.Reference, wrapper.Pointer)
		if err != nil {
			return false, err
		}
		if target.Catalog == nil || target.CollectionBinding || target.NullableBinding || nullableArraySchema(target.Schema) {
			if override := wrapperNullableOverride(v); override != nil {
				return *override, nil
			}
			if target.Catalog == nil || target.CollectionBinding || target.NullableBinding {
				if target.ModelName != "" || target.Deferred != nil || target.Catalog != nil && target.Catalog.NullableDTO {
					return a.nominalSourceNullable(target)
				}
				return target.effectiveNullable()
			}
			return true, nil // Intrinsically nullable array, absent wrapper override.
		}
		return false, nil // Preserve named scalar/nominal annotation-only wrapper policy.
	}
	if has(v, "$ref") {
		component, err := a.resolveSchema(text(get(v, "$ref")), pointer)
		if err != nil {
			return false, err
		}
		if component.Catalog == nil || component.CollectionBinding || component.NullableBinding {
			if component.ModelName != "" || component.Deferred != nil || component.Catalog != nil && component.Catalog.NullableDTO {
				return a.nominalSourceNullable(component)
			}
			return component.effectiveNullable()
		}
		v, pointer = component.Schema, component.Pointer
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return false, err
	}
	if has(v, "allOf") {
		return a.compositionNullable(v, pointer, 0, &compositionBudget{remaining: 16384, enabled: true}, map[*document.Value]bool{})
	}
	union, err := a.readUnion(v, pointer)
	if err != nil {
		return false, err
	}
	if union != nil {
		if err := a.deriveObjectReferenceNullability(union, 0); err != nil {
			return false, err
		}
	}
	return nullable || union != nil && union.Nullable, nil
}

// The runtime package is a single planning decision. Multi-spec configuration
// selects a shared package without changing field or emitter rules.
func (a *analyzer) patchPackage() string { return a.runtimePackage() }

func (a *analyzer) requestRuntime(plan *Plan) *diagnostic.Diagnostic {
	if plan.Client == nil {
		return nil
	}
	for _, d := range plan.Client.Models {
		if declarationHasPatch(d) {
			plan.PatchPackage = a.patchPackage()
		}
	}
	if plan.PatchPackage == "" || plan.PatchPackage != a.options.PackageName+".models" {
		return nil
	}
	for _, d := range plan.Models {
		if strings.EqualFold(d.Name(), "Patch") || strings.EqualFold(d.Name(), "PatchSerializer") {
			pointer := "/components/schemas"
			for _, c := range a.components {
				if c.Name == d.Name() {
					pointer = c.Pointer
					break
				}
			}
			return diagnostic.At("E_NAME_COLLISION", "A component conflicts with the required Patch runtime.", pointer)
		}
	}
	return nil
}
