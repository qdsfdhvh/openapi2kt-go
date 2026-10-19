package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func (a *analyzer) fieldConstant(v *document.Value, name, wire, pointer string, depth int, ordinary bool) (*Constant, *diagnostic.Diagnostic) {
	// The subsequent value planner owns depth and the single source warning.
	// A literal true cannot be a const declaration or acquire a DTO identity.
	if a.booleanTrueValue(v) {
		return nil, nil
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	if has(v, "const") {
		return constantProfile(name, wire, v, pointer, depth, ordinary)
	}
	if ordinary && has(v, "$ref") && has(v, "default") {
		return nil, nil // The field-default planner owns this narrowly scoped role.
	}
	var reference *referenceView
	if target, ok := v.Get("$ref"); ok {
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return nil, err
		}
		if _, _, err := schemaType(v, pointer); err != nil {
			return nil, err
		}
		ref, ok := target.Text()
		if !ok {
			return nil, invalid("$ref must be a string.", pointer)
		}
		reference = &referenceView{Reference: ref, Pointer: pointer}
	} else {
		var err *diagnostic.Diagnostic
		reference, err = referenceWrapper(v, pointer)
		if err != nil {
			return nil, err
		}
	}
	if reference == nil {
		return nil, nil
	}
	if depth >= MaxSchemaDepth {
		return nil, diagnostic.At("E_SCHEMA_DEPTH", "Schema nesting exceeds the supported limit.", reference.Pointer)
	}
	component, err := a.resolveSchema(reference.Reference, reference.Pointer)
	if err != nil {
		return nil, err
	}
	if has(component.Schema, "const") {
		// Const body projection exits before referenceType. Keep source alias
		// work here too, rather than silently resetting it at the terminal.
		steps := max(1, component.Steps)
		if err := depthCheck(depth+steps, reference.Pointer); err != nil {
			return nil, err
		}
		return constantProfile(name, wire, component.Schema, component.Pointer, depth+steps, ordinary)
	}
	return nil, nil
}

func (a *analyzer) buildModel(name string, v *document.Value, pointer string, depth int) (*Model, *diagnostic.Diagnostic) {
	context := *a
	context.aliasPlanning = false // Nominal interiors have their own schema roots.
	if context.usage == RequestVariantUsage {
		context.usage = RequestObjectUsage
	}
	model, err := context.buildModelPlan(name, v, pointer, depth, nil, false, nil, true)
	a.warnings = context.warnings
	return model, err
}

func (a *analyzer) buildModelContext(name string, v *document.Value, pointer string, depth int, common []Property, singleEnumBody bool) (*Model, *diagnostic.Diagnostic) {
	return a.buildModelPlan(name, v, pointer, depth, common, singleEnumBody, nil, false)
}

func (a *analyzer) buildTypedModel(name string, v *document.Value, pointer string, typed map[string]typedLeafField) (*Model, *diagnostic.Diagnostic) {
	return a.buildModelPlan(name, v, pointer, 0, nil, false, typed, false)
}

func (a *analyzer) buildModelPlan(name string, v *document.Value, pointer string, depth int, common []Property, singleEnumBody bool, typed map[string]typedLeafField, ordinary bool) (*Model, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	shape, err := a.modelShape(v, pointer, depth, ordinary)
	if err != nil {
		return nil, err
	}
	return a.buildModelFromShape(name, v, pointer, shape, common, singleEnumBody, typed, ordinary)
}

// Borrowed composition alternatives use the same fields, const, validation and
// presence rules as ordinary models without fabricating expanded schema nodes.
func (a *analyzer) buildModelFromShape(name string, v *document.Value, pointer string, shape *shape, common []Property, singleEnumBody bool, typed map[string]typedLeafField, ordinary bool) (*Model, *diagnostic.Diagnostic) {
	if ordinary && shape.Nullable != nil && *shape.Nullable && a.componentRoot(v, pointer) && !directNullableDTOOwnCounts(v, pointer, shape) {
		if err := rejectNullableComponentCounts(v, pointer, shape); err != nil {
			return nil, err
		}
	}
	declared := map[string]bool{}
	for _, f := range shape.Fields {
		declared[f.Wire] = true
	}
	for _, wire := range shape.RequiredOrder {
		if !declared[wire] {
			return nil, invalid("required must contain unique declared property names.", pointer)
		}
	}
	aggregates, preserved, err := a.ordinaryPrimitiveAggregates(v, shape, ordinary)
	if err != nil {
		return nil, err
	}
	// Validate discarded nodes without allowing their temporary declarations or
	// warnings to leak into the final model. Later overrides cannot hide errors.
	for _, f := range shape.Discarded {
		if preserved[f.Pointer] {
			continue // Complete borrowed group validated and retained in the winner.
		}
		if err := a.rejectValidationProjection(f.Schema, f.Pointer); err != nil {
			return nil, err
		}
		if err := object(f.Schema, f.Pointer); err != nil {
			return nil, err
		}
		if has(f.Schema, "const") {
			if _, err := constantProfile("validation", f.Wire, f.Schema, f.Pointer, f.Depth, ordinary); err != nil {
				return nil, err
			}
		} else if has(f.Schema, "default") {
			if hasPrimitiveBounds(f.Schema) {
				for _, key := range primitiveBoundKeys {
					if has(f.Schema, key) {
						return nil, unsupported(naming.Pointer(f.Pointer, key))
					}
				}
			}
			validation := *a
			validation.warnings = nil
			if has(f.Schema, "enum") {
				if _, err := validation.enumDefaultType(f, "Validation", nil, nil); err != nil {
					return nil, err
				}
			} else {
				ty, _, err := validation.modelDefaultType(f, ordinary, "Validation", nil, nil)
				if err != nil {
					return nil, err
				}
				if a.deferredFieldDefault(f.Schema) {
					*a.referencedDefaults = append(*a.referencedDefaults, referencedFieldDefault{Field: f, Type: ty})
				}
			}
		} else {
			validation := *a
			validation.warnings = nil
			var scratch []Declaration
			if _, err := validation.propertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth); err != nil {
				return nil, err
			}
		}
	}
	// Shape discovery is also used by union/hierarchy planning. Validate extras
	// here once per built model, not on every exploratory shape traversal.
	for _, extra := range shape.Additional {
		if err := a.validateAdditionalProperties(extra.Schema, extra.Pointer, extra.Depth); err != nil {
			return nil, err
		}
	}
	for _, location := range shape.Compositions {
		a.warn("W_ALL_OF_APPROXIMATION", "Object allOf members are merged with later property overrides, not exact intersection validation.", location)
	}
	if len(shape.Fields) == 0 && common == nil && len(typed) == 0 {
		a.warn("W_JSON_ELEMENT_APPROXIMATION", "An empty object schema uses JsonElement and permits non-object values.", pointer)
	}
	model := &Model{ClassName: name, Properties: make([]Property, 0, len(shape.Fields))}
	if shape.Nullable != nil {
		model.Nullable = *shape.Nullable
	}
	usedProperties := map[string]bool{}
	usedNested := map[string]bool{strings.ToLower(name): true, "companion": true, "serializer": true}
	slots := 0
	// Generated mapping bodies occupy real serialized slots and normalized
	// names, but never become source fields or satisfy source required entries.
	var generated []Constant
	for _, wire := range typedFieldKeys(typed) {
		if declared[wire] {
			continue
		}
		mapped := typed[wire]
		if mapped.Body == nil {
			return nil, unsupported(mapped.Pointer)
		}
		property, err := naming.PropertyName(wire, mapped.Pointer)
		if err != nil {
			return nil, err
		}
		if usedProperties[strings.ToLower(property)] {
			return nil, diagnostic.At("E_NAME_COLLISION", "Property names collide after normalization.", mapped.Pointer)
		}
		usedProperties[strings.ToLower(property)] = true
		generated = append(generated, Constant{Name: property, WireName: wire, Type: TypeRef{Kind: Named, Name: mapped.Type}, Initializer: mapped.Type + ".Serializer.fromSerialName(" + naming.Literal(*mapped.Body) + ")"})
		slots++
		if slots > MaxMembers {
			return nil, diagnostic.At("E_MODEL_LIMIT", "Model exceeds the supported serialization constructor limit.", pointer)
		}
	}
	for _, f := range shape.Fields {
		property, err := naming.PropertyName(f.Wire, f.Pointer)
		if err != nil {
			return nil, err
		}
		if usedProperties[strings.ToLower(property)] {
			return nil, diagnostic.At("E_NAME_COLLISION", "Property names collide after normalization.", f.Pointer)
		}
		usedProperties[strings.ToLower(property)] = true
		value, err := a.fieldConstant(f.Schema, property, f.Wire, f.Pointer, f.Depth, ordinary)
		if err != nil {
			return nil, err
		}
		if value != nil {
			if structuredConstant(*value) && has(f.Schema, "const") {
				a.warn("W_CONST_STRUCTURED_APPROXIMATION", "A structured const field supplies its typed JSON default but does not enforce literal membership when decoding.", naming.Pointer(f.Pointer, "const"))
			}
			if mapped, ok := typed[f.Wire]; ok {
				if mapped.Body == nil {
					return nil, unsupported(f.Pointer)
				}
				value.Type = TypeRef{Kind: Named, Name: mapped.Type}
				value.Initializer = mapped.Type + ".Serializer.fromSerialName(" + naming.Literal(*mapped.Body) + ")"
			}
			slots += value.Type.Slots()
			if slots > MaxMembers {
				return nil, diagnostic.At("E_MODEL_LIMIT", "Model exceeds the supported JVM serialization constructor size; split it into smaller objects.", pointer)
			}
			model.Constants = append(model.Constants, *value)
			continue
		}
		if mapped, ok := typed[f.Wire]; ok {
			var scratch []Declaration
			if _, err := a.propertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth); err != nil {
				return nil, err
			}
			if mapped.Body != nil {
				model.Constants = append(model.Constants, Constant{Name: property, WireName: f.Wire, Type: TypeRef{Kind: Named, Name: mapped.Type}, Initializer: mapped.Type + ".Serializer.fromSerialName(" + naming.Literal(*mapped.Body) + ")"})
			} else {
				model.Properties = append(model.Properties, Property{Name: property, WireName: f.Wire, Type: TypeRef{Kind: Named, Name: mapped.Type}})
			}
			slots++
			if slots > MaxMembers {
				return nil, diagnostic.At("E_MODEL_LIMIT", "Model exceeds the supported serialization constructor limit.", pointer)
			}
			continue
		}
		nestedName, err := naming.ClassName(f.Wire, f.Pointer)
		if err != nil {
			return nil, err
		}
		shared := commonProperty(common, f.Wire)
		var validationNested []Declaration
		nested, used := &model.Nested, usedNested
		if shared != nil {
			nested, used = &validationNested, map[string]bool{}
		}
		var ty TypeRef
		var validation *PrimitiveValidation
		kind, _, kindErr := schemaType(f.Schema, f.Pointer)
		if kindErr != nil {
			return nil, kindErr
		}
		checkedNativeEnum := a.options.Validation && has(f.Schema, "enum") && (kind == "integer" || kind == "number" || kind == "boolean")
		if checkedNativeEnum {
			ty, validation, err = a.nativeEnumFieldType(f, has(f.Schema, "default"))
		} else if has(f.Schema, "default") && has(f.Schema, "enum") {
			ty, err = a.enumDefaultType(f, nestedName, nested, used)
		} else if has(f.Schema, "default") {
			ty, validation, err = a.modelDefaultType(f, ordinary, nestedName, nested, used)
		} else if ordinary && directStringEnumCollectionBounds(f.Schema) {
			ty, err = a.stringEnumCollectionField(f, nestedName, nested, used)
		} else {
			ty, validation, err = a.modelPropertyType(f.Schema, nestedName, f.Pointer, nested, used, f.Depth)
		}
		if err != nil {
			return nil, err
		}
		annotationType := ty
		if aggregate, ok := aggregates[f.Wire]; ok {
			validation = aggregate
		}
		serializer := ""
		if !has(f.Schema, "default") {
			serializer, err = a.fieldSerializer(f.Schema, f.Pointer, f.Depth)
			if err != nil {
				return nil, err
			}
		}
		if serializer != "" {
			ty = withFieldSerializer(ty, serializer)
		}
		explicitNullable := ty.Kind == Nullable
		if a.usage != SharedUsage {
			explicitNullable, err = a.requestNullable(f.Schema, f.Pointer)
			if err != nil {
				return nil, err
			}
			if explicitNullable {
				ty = ty.Null()
			}
		}
		if !shape.Required[f.Wire] {
			ty = ty.Null()
		}
		defaultNull := ty.Kind == Nullable
		if has(f.Schema, "default") && shape.Required[f.Wire] {
			defaultNull = false
		}
		if a.usage == RequestObjectUsage && readOnlySchema(f.Schema) {
			// A read-only field is produced by the server, so a request must not
			// send it unless the caller explicitly asks to. Patch.Absent keeps that
			// decision at the call site instead of silently including the field.
			// The response model is untouched: it keeps the declared requiredness.
			element := ty.WithoutRootNull()
			ty = TypeRef{Kind: PatchValue, Name: a.patchPackage() + ".Patch", Element: &element}
			defaultNull = false
		} else if a.usage != SharedUsage {
			if shape.Required[f.Wire] {
				defaultNull = false
			} else if explicitNullable && a.usage == RequestObjectUsage {
				element := ty.WithoutRootNull()
				ty = TypeRef{Kind: PatchValue, Name: a.patchPackage() + ".Patch", Element: &element}
				defaultNull = false
			}
		}
		if shared != nil {
			actual, _, err := a.actualField(&f)
			if err != nil {
				return nil, err
			}
			values := get(actual, "enum")
			if singleEnumBody && values.Kind() == document.Array && len(values.Items()) == 1 {
				literal, ok := values.Items()[0].Text()
				if !ok {
					return nil, unsupported(f.Pointer)
				}
				model.Constants = append(model.Constants, Constant{Name: property, WireName: f.Wire, Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}, Initializer: naming.Literal(literal)})
				slots++
				if slots > MaxMembers {
					return nil, diagnostic.At("E_MODEL_LIMIT", "Variant exceeds the supported serialization constructor limit.", pointer)
				}
				continue
			}
			if bare(shared.Type).Kind == Map || bare(shared.Type).Kind == JSONObject {
				// Public dictionary identity joins nullable presence across branches;
				// retain this variant's original element/count validation plan.
				if shared.Type.Kind == Nullable {
					ty = ty.Null()
				}
			} else {
				ty = shared.Type
			}
			if serializer != "" {
				ty = withFieldSerializer(ty, serializer)
			}
		}
		slots += ty.Slots()
		if slots > MaxMembers {
			return nil, diagnostic.At("E_MODEL_LIMIT", "Model exceeds the supported JVM constructor size; split it into smaller objects.", pointer)
		}
		initializer, err := a.modelDefault(f, ty, shape.Required[f.Wire])
		if err != nil {
			return nil, err
		}
		if has(f.Schema, "default") && a.deferredFieldDefault(f.Schema) {
			*a.referencedDefaults = append(*a.referencedDefaults, referencedFieldDefault{Field: f, Type: annotationType, Model: model, Index: len(model.Properties)})
		}
		model.Properties = append(model.Properties, Property{Name: property, WireName: f.Wire, Type: ty, DefaultNull: defaultNull, Default: initializer, Validation: validation})
	}
	if len(model.Properties) == 0 {
		model.Properties = nil // Preserve the original empty-plan representation.
	}
	model.Constants = append(model.Constants, generated...)
	model.PropertyCount, err = a.ordinaryObjectCounts(v, pointer, shape, ordinary, len(model.Properties) > 0)
	if err != nil {
		return nil, err
	}
	if a.modelOrigins != nil {
		a.modelOrigins[model] = v
	}
	if a.modelPointers != nil {
		a.modelPointers[model] = pointer
	}
	return model, nil
}
