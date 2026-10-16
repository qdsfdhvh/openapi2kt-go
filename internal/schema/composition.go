package schema

import (
	"slices"
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type field struct {
	Wire, Pointer string
	Schema        *document.Value
	Depth         int
}

type shape struct {
	Fields, Discarded []field
	// Contributions retain source traversal order before nested winner lowering.
	// Original nodes/depths remain borrowed; Fields/Discarded preserve legacy APIs.
	Contributions []field
	CountSources  []field // Original counted objects in member-subtree then own order.
	Additional    []field
	Required      map[string]bool
	RequiredOrder []string
	Nullable      *bool
	Compositions  []string
}

type referenceView struct {
	Reference, Pointer string
	Nullable           bool
}

func referenceWrapper(v *document.Value, pointer string) (*referenceView, *diagnostic.Diagnostic) {
	return parseReferenceWrapper(v, pointer, false)
}

// allowDefault identifies an annotation-only wrapper before a field default
// can turn its source into an expanded DTO. It does not admit that wrapper.
func parseReferenceWrapper(v *document.Value, pointer string, allowDefault bool) (*referenceView, *diagnostic.Diagnostic) {
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	members := get(v, "allOf")
	if members.Kind() != document.Array {
		return nil, nil
	}
	for _, member := range v.Members() {
		if member.Name != "allOf" && member.Name != "nullable" && !(allowDefault && member.Name == "default") && !slices.Contains(metadata, member.Name) {
			return nil, nil
		}
	}
	keys := []string{"allOf", "nullable"}
	if allowDefault {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	var nullable *bool
	if has(v, "nullable") {
		_, n, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		nullable = &n
	}
	var reference *referenceView
	for i, member := range members.Items() {
		location := naming.Pointer(naming.Pointer(pointer, "allOf"), strconv.Itoa(i))
		if err := object(member, location); err != nil {
			return nil, err
		}
		if has(member, "$ref") {
			if err := checkKeys(member, []string{"$ref", "nullable"}, location); err != nil {
				return nil, err
			}
			if _, _, err := schemaType(member, location); err != nil {
				return nil, err
			}
			target, ok := get(member, "$ref").Text()
			if !ok {
				return nil, invalid("$ref must be a string.", location)
			}
			if reference != nil {
				return nil, nil
			}
			reference = &referenceView{Reference: target, Pointer: location}
		} else {
			for _, annotation := range member.Members() {
				if annotation.Name != "nullable" && !slices.Contains(metadata, annotation.Name) {
					return nil, nil
				}
			}
			if err := checkKeys(member, []string{"nullable"}, location); err != nil {
				return nil, err
			}
			if nullable == nil && has(member, "nullable") {
				_, n, err := schemaType(member, location)
				if err != nil {
					return nil, err
				}
				nullable = &n
			}
		}
	}
	if reference != nil && nullable != nil {
		reference.Nullable = *nullable
	}
	return reference, nil
}

type compositionBudget struct {
	remaining int
	enabled   bool
}

func (b *compositionBudget) spend(pointer string) *diagnostic.Diagnostic {
	if !b.enabled {
		return nil
	}
	if b.remaining == 0 {
		return diagnostic.At("E_SCHEMA_LIMIT", "Object composition expansion exceeds the supported work budget; simplify the reference graph.", pointer)
	}
	b.remaining--
	return nil
}

func (a *analyzer) objectShape(v *document.Value, pointer string, depth int) (*shape, *diagnostic.Diagnostic) {
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	return a.modelShape(v, pointer, depth, false)
}

// Only ordinary model traversal may retain whole-object count sources.
// Exploratory union/hierarchy shapes keep their existing fail-closed grammar.
func (a *analyzer) modelShape(v *document.Value, pointer string, depth int, ordinary bool) (*shape, *diagnostic.Diagnostic) {
	budget := &compositionBudget{remaining: 16384, enabled: has(v, "allOf")}
	return a.walkShape(v, pointer, depth, []string{pointer}, budget, ordinary)
}

func (a *analyzer) walkShape(v *document.Value, pointer string, depth int, stack []string, budget *compositionBudget, counts bool) (*shape, *diagnostic.Diagnostic) {
	return a.walkShapeMembers(v, pointer, depth, stack, budget, counts, false)
}

func (a *analyzer) objectOwnShape(v *document.Value, pointer string, depth int) (*shape, *diagnostic.Diagnostic) {
	return a.walkShapeMembers(v, pointer, depth, nil, &compositionBudget{}, false, true)
}

func (a *analyzer) walkShapeMembers(v *document.Value, pointer string, depth int, stack []string, budget *compositionBudget, counts, ownOnly bool) (*shape, *diagnostic.Diagnostic) {
	if err := budget.spend(pointer); err != nil {
		return nil, err
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	keys := []string{"type", "nullable", "properties", "required", "allOf", "additionalProperties"}
	if a.objectDefaultSchemas[v] {
		// A direct ordinary field's own object default is validated by its
		// dedicated planner; the shared type plan borrows the same source node
		// and must not reject its annotation.
		keys = append(keys, "default")
	}
	if counts && a.options.Validation {
		keys = append(keys, "minProperties", "maxProperties")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	if kind != "" && kind != "object" {
		return nil, unsupported(naming.Pointer(pointer, "type"))
	}
	if has(v, "additionalProperties") {
		props := get(v, "properties")
		if props.Kind() != document.Object && has(v, "properties") {
			return nil, invalid("properties must be an object.", naming.Pointer(pointer, "properties"))
		}
		if props.Kind() != document.Object || len(props.Members()) == 0 {
			return nil, unsupported(naming.Pointer(pointer, "additionalProperties"))
		}
	}
	result := &shape{Required: map[string]bool{}}
	if has(v, "additionalProperties") {
		result.Additional = append(result.Additional, field{Schema: v, Pointer: pointer, Depth: depth})
	}
	if has(v, "nullable") || nullable {
		result.Nullable = &nullable
	}
	if required, ok := v.Get("required"); ok {
		if required.Kind() != document.Array {
			return nil, invalid("required must be an array.", pointer)
		}
		for _, entry := range required.Items() {
			wire, ok := entry.Text()
			if !ok {
				return nil, invalid("required entries must be strings.", pointer)
			}
			if result.Required[wire] {
				return nil, invalid("required must contain unique declared property names.", pointer)
			}
			result.Required[wire] = true
			result.RequiredOrder = append(result.RequiredOrder, wire)
		}
	}
	positions := map[string]int{}
	if members, ok := v.Get("allOf"); ok && !ownOnly {
		location := naming.Pointer(pointer, "allOf")
		if members.Kind() != document.Array || len(members.Items()) == 0 {
			return nil, invalid("allOf must be a non-empty schema array.", location)
		}
		wrapper, err := referenceWrapper(v, pointer)
		if err != nil {
			return nil, err
		}
		if wrapper != nil {
			return nil, unsupported(location)
		}
		for i, member := range members.Items() {
			p := naming.Pointer(location, strconv.Itoa(i))
			if err := object(member, p); err != nil {
				return nil, err
			}
			var part *shape
			if target, ok := member.Get("$ref"); ok {
				if err := checkKeys(member, []string{"$ref", "nullable"}, p); err != nil {
					return nil, err
				}
				if _, _, err := schemaType(member, p); err != nil {
					return nil, err
				}
				reference, ok := target.Text()
				if !ok {
					return nil, invalid("$ref must be a string.", p)
				}
				declared, e := a.resolveDeclared(reference, p)
				if e != nil {
					return nil, e
				}
				if declared.InternalModelTarget != nil {
					return nil, schemaReferenceUnsupported(p)
				}
				if declared.NullableDTO || declared.NullableEnum || declared.NullableValueClass || declared.NullableTemporal || declared.NullableBigint || declared.NullableBytes || declared.NullableCallerNative || declared.NullableRaw || declared.NullablePrimitive || declared.NullableNativeEnum || declared.BindingTarget != nil && (declared.BindingTarget.NullableDTO || declared.BindingTarget.NullableEnum || declared.BindingTarget.NullableValueClass || declared.BindingTarget.NullableTemporal || declared.BindingTarget.NullableBigint || declared.BindingTarget.NullableBytes || declared.BindingTarget.NullableCallerNative || declared.BindingTarget.NullableRaw || declared.BindingTarget.NullablePrimitive || declared.BindingTarget.NullableNativeEnum) {
					return nil, unsupported(naming.Pointer(p, "$ref"))
				}
				component, aliasSteps := declared, 0
				if declared.BindingTarget != nil {
					if declared.BindingNullable != nil && *declared.BindingNullable && !collectionAliasSchema(declared.BindingTarget.Schema) {
						return nil, unsupported(naming.Pointer(p, "$ref"))
					}
					component, aliasSteps = declared.BindingTarget, declared.AliasHeight
				}
				if slices.Contains(stack, component.Pointer) {
					if hasObjectCounts(v) {
						return nil, rejectObjectCounts(v, pointer)
					}
					return nil, diagnostic.At("E_REF_CYCLE", "Object allOf reference expansion is cyclic.", naming.Pointer(p, "$ref"))
				}
				part, err = a.walkShape(component.Schema, component.Pointer, depth+1+aliasSteps, append(stack, component.Pointer), budget, counts)
				if err == nil {
					part.Nullable = nil // reference annotations do not describe this type
				}
			} else {
				properties := get(member, "properties")
				memberCounts := counts && (properties.Kind() == document.Object && len(properties.Members()) > 0 || has(member, "allOf"))
				part, err = a.walkShape(member, p, depth+1, stack, budget, memberCounts)
			}
			if err != nil {
				return nil, err
			}
			if result.Nullable == nil {
				result.Nullable = part.Nullable
			}
			for _, wire := range part.RequiredOrder {
				if !result.Required[wire] {
					result.Required[wire] = true
					result.RequiredOrder = append(result.RequiredOrder, wire)
				}
			}
			result.CountSources = append(result.CountSources, part.CountSources...)
			result.Contributions = append(result.Contributions, part.Contributions...)
			result.Discarded = append(result.Discarded, part.Discarded...)
			result.Additional = append(result.Additional, part.Additional...)
			result.Compositions = append(result.Compositions, part.Compositions...)
			for _, f := range part.Fields {
				if err := result.insert(positions, f, budget, pointer); err != nil {
					return nil, err
				}
			}
		}
		result.Compositions = append(result.Compositions, location)
	}
	if properties, ok := v.Get("properties"); ok {
		if properties.Kind() != document.Object {
			return nil, invalid("properties must be an object.", pointer)
		}
		// Reserve only within the existing member budget; insert still owns
		// limit checks and composition override order.
		result.Fields = slices.Grow(result.Fields, min(len(properties.Members()), MaxMembers-len(result.Fields)))
		propertiesPointer := naming.Pointer(pointer, "properties")
		for _, member := range properties.Members() {
			f := field{Wire: member.Name, Schema: member.Value, Pointer: naming.Pointer(propertiesPointer, member.Name), Depth: depth + 1}
			result.Contributions = append(result.Contributions, f)
			if err := result.insert(positions, f, budget, pointer); err != nil {
				return nil, err
			}
		}
	}
	if hasObjectCounts(v) {
		if len(result.Fields) == 0 {
			return nil, rejectObjectCounts(v, pointer)
		}
		result.CountSources = append(result.CountSources, field{Schema: v, Pointer: pointer, Depth: depth})
	}
	return result, nil
}

func (s *shape) insert(positions map[string]int, f field, budget *compositionBudget, pointer string) *diagnostic.Diagnostic {
	if err := budget.spend(f.Pointer); err != nil {
		return err
	}
	if index, ok := positions[f.Wire]; ok {
		s.Discarded = append(s.Discarded, s.Fields[index])
		s.Fields[index] = f
	} else {
		if len(s.Fields) >= MaxMembers {
			return diagnostic.At("E_MODEL_LIMIT", "Model exceeds the supported JVM constructor size; split it into smaller objects.", pointer)
		}
		positions[f.Wire] = len(s.Fields)
		s.Fields = append(s.Fields, f)
	}
	return nil
}
