package schema

import (
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// An annotation belongs to the original physical collection edge. It does
// not supply a container initializer or a new nominal/reference admission.
type inlineEnumCollectionChildDefault struct {
	Schema, Parent, Value, Vocabulary *document.Value
	Pointer, SchemaBytes, ValueBytes  string
	Uses                              []*inlineEnumCollectionChildUse
	FinalUseCount                     int
}

type inlineEnumCollectionChildUse struct {
	Actual      *TypeRef
	Proof       tupleCollectionUse
	Retained    bool
	Enum        *StringEnum
	EnumName    string
	EnumMembers []EnumMember
}

func (a *analyzer) inlineEnumCollectionChildDefault(v *document.Value, pointer string, depth int) (bool, *diagnostic.Diagnostic) {
	if v == nil || has(v, "$ref") || !has(v, "default") || !originalEnumWrapperDefaultSource(v) {
		return false, nil
	}
	parent := a.collectionReferenceChildParent(v, pointer)
	if parent == nil {
		return false, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return false, err
	}
	if old := a.inlineEnumChildDefaults[v]; old != nil {
		if old.Schema != v || old.Parent != parent || old.Pointer != pointer ||
			old.Value != get(v, "default") || old.Vocabulary != get(v, "enum") {
			return false, schemaReferenceUnsupported(pointer)
		}
		raw, _ := v.MarshalJSON()
		if old.SchemaBytes != string(raw) {
			return false, schemaReferenceUnsupported(pointer)
		}
		return true, nil
	}
	value := get(v, "default")
	at := naming.Pointer(pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return false, err
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return false, err
	}
	if a.inlineEnumChildProbe {
		return true, nil
	}
	validation := *a
	validation.warnings = nil
	switch {
	case (kind == "" || kind == "string") && hasPrimitiveBounds(v) && a.options.Validation:
		_, err = validation.checkedStringEnumLeaf(v, "Validation", pointer, nil, nil, depth, true)
	case kind == "integer" || kind == "number" || kind == "boolean":
		if a.options.Validation {
			_, err = validation.nativeEnumValueTypeOwnDefault(v, pointer, depth, true)
		} else {
			_, err = validation.enumDefaultType(field{Schema: v, Pointer: pointer, Depth: depth}, "Validation", nil, nil)
		}
	default:
		_, err = validation.enumDefaultType(field{Schema: v, Pointer: pointer, Depth: depth}, "Validation", nil, nil)
	}
	if err != nil {
		return false, err
	}
	if err := a.captureTupleNativeSource(v, pointer, v, pointer); err != nil {
		return false, err
	}
	if err := a.captureTupleNativePath(value, at, pointer); err != nil {
		return false, err
	}
	raw, _ := v.MarshalJSON()
	valueBytes, _ := value.MarshalJSON()
	if err := a.tupleNativeSources.charge(1, len(valueBytes), pointer); err != nil {
		return false, err
	}
	if a.inlineEnumChildDefaults == nil {
		a.inlineEnumChildDefaults = map[*document.Value]*inlineEnumCollectionChildDefault{}
	}
	if a.inlineEnumChildDefaultOrder == nil {
		a.inlineEnumChildDefaultOrder = &[]*inlineEnumCollectionChildDefault{}
	}
	source := &inlineEnumCollectionChildDefault{
		Schema: v, Parent: parent, Value: value, Vocabulary: get(v, "enum"),
		Pointer: pointer, SchemaBytes: string(raw), ValueBytes: string(valueBytes),
	}
	a.inlineEnumChildDefaults[v] = source
	*a.inlineEnumChildDefaultOrder = append(*a.inlineEnumChildDefaultOrder, source)
	return true, nil
}

// A root factory probe must not hide an invalid child behind the root default.
// It only grants provisional root recognition when the failure belongs to the
// physical inline enum child; the ordinary planner still validates both nodes.
func (a *analyzer) inlineEnumChildDefaultFailure(v *document.Value, pointer string, failure *diagnostic.Diagnostic) bool {
	if failure == nil || failure.Pointer == nil {
		return false
	}
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || has(v, "$ref") || has(v, "prefixItems") {
			return false
		}
		if depth > 0 && has(v, "default") && originalEnumWrapperDefaultSource(v) && a.collectionReferenceChildParent(v, pointer) != nil {
			at := naming.Pointer(pointer, "default")
			return *failure.Pointer == at || strings.HasPrefix(*failure.Pointer, at+"/")
		}
		edge := ""
		switch {
		case bareArrayDefaultSchema(v):
			edge = "items"
		case dictionarySchema(v):
			edge = "additionalProperties"
		default:
			return false
		}
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return false
}

func (a *analyzer) validatedInlineEnumCollectionChildDefault(v *document.Value, pointer string) bool {
	source := a.inlineEnumChildDefaults[v]
	if source == nil || source.Schema != v || source.Pointer != pointer ||
		source.Parent != a.collectionReferenceChildParent(v, pointer) ||
		source.Value != get(v, "default") || source.Vocabulary != get(v, "enum") {
		return false
	}
	raw, _ := v.MarshalJSON()
	value, _ := source.Value.MarshalJSON()
	return string(raw) == source.SchemaBytes && string(value) == source.ValueBytes
}

// Record each real collection element address. A later source walk can validate
// the same annotation in scratch space, but cannot replace the retained type.
func (a *analyzer) bindInlineEnumCollectionChildDefault(v *document.Value, pointer string, actual *TypeRef) *diagnostic.Diagnostic {
	source := a.inlineEnumChildDefaults[v]
	if source == nil {
		return nil
	}
	if !a.validatedInlineEnumCollectionChildDefault(v, pointer) || actual == nil {
		return schemaReferenceUnsupported(pointer)
	}
	if err := a.tupleNativeSources.charge(1, 0, pointer); err != nil {
		return err
	}
	source.Uses = append(source.Uses, &inlineEnumCollectionChildUse{Actual: actual})
	return nil
}

func (a *analyzer) finalizeInlineEnumCollectionChildDefaults(declarations []Declaration, client *Client) *diagnostic.Diagnostic {
	if a.inlineEnumChildDefaultOrder == nil {
		return nil
	}
	retained := collectionChildDefaultRetainedTypes(declarations, client)
	enums := a.inlineEnumRetainedDeclarations(declarations, client)
	for _, source := range *a.inlineEnumChildDefaultOrder {
		if source == nil || !a.validatedInlineEnumCollectionChildDefault(source.Schema, source.Pointer) {
			return schemaReferenceUnsupported("")
		}
		if source.FinalUseCount > len(source.Uses) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		kind, nullable, failure := schemaType(source.Schema, source.Pointer)
		if failure != nil {
			return failure
		}
		var expectedChecks *PrimitiveValidation
		if a.options.Validation && (kind == "integer" || kind == "number" || kind == "boolean") {
			probe := *a
			probe.warnings = nil
			probe.validationSources = map[*PrimitiveValidation]string{}
			probe.emissionHints = EmissionHints{}
			ty, err := probe.nativeEnumValueTypeOwnDefault(source.Schema, source.Pointer, 0, true)
			if err != nil {
				return err
			}
			expectedChecks = ty.WithoutRootNull().Validation
		}
		for _, use := range source.Uses[source.FinalUseCount:] {
			if use == nil || use.Actual == nil {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if retained[use.Actual] {
				actual := use.Actual.WithoutRootNull()
				if (use.Actual.Kind == Nullable) != nullable {
					return schemaReferenceUnsupported(source.Pointer)
				}
				switch kind {
				case "", "string":
					if actual.Kind == Named {
						matches := 0
						for _, enum := range enums[source.Schema] {
							origin := a.enumDiagnosticOrigins[enum]
							if origin.Pointer != source.Pointer || actual.Name != enum.ClassName && !strings.HasSuffix(actual.Name, "."+enum.ClassName) {
								continue
							}
							keys := []string{"type", "nullable", "enum", "format", "x-native-type", "default"}
							if a.options.Validation && hasPrimitiveBounds(source.Schema) {
								keys = append(keys, primitiveBoundKeys...)
							}
							expected, err := buildStringEnumAllowedKeys(enum.ClassName, source.Schema, source.Pointer, keys)
							if err != nil {
								return err
							}
							if !slices.Equal(enum.Members, expected.Members) {
								return schemaReferenceUnsupported(source.Pointer)
							}
							use.Enum = enum
							use.EnumName = enum.ClassName
							use.EnumMembers = append([]EnumMember{}, enum.Members...)
							matches++
						}
						if matches != 1 {
							return schemaReferenceUnsupported(source.Pointer)
						}
					} else if actual.Kind == Scalar {
						scalar, err := scalarMapping("string", source.Schema, source.Pointer)
						if err != nil || actual.Name != scalar {
							return schemaReferenceUnsupported(source.Pointer)
						}
					} else {
						return schemaReferenceUnsupported(source.Pointer)
					}
					if a.options.Validation && hasPrimitiveBounds(source.Schema) {
						checks := a.checkedEnumSources[source.Schema]
						if checks == nil || actual.Validation != checks.Checks || a.validationSources[actual.Validation] != source.Pointer {
							return schemaReferenceUnsupported(source.Pointer)
						}
					} else if actual.Validation != nil {
						return schemaReferenceUnsupported(source.Pointer)
					}
				case "integer", "number", "boolean":
					scalar, err := scalarMapping(kind, source.Schema, source.Pointer)
					if err != nil || actual.Kind != Scalar || actual.Name != scalar {
						return schemaReferenceUnsupported(source.Pointer)
					}
					if a.options.Validation {
						if actual.Validation == nil || a.validationSources[actual.Validation] != source.Pointer ||
							namedCollectionChecks(actual) != namedCollectionChecks(TypeRef{Validation: expectedChecks}) {
							return schemaReferenceUnsupported(source.Pointer)
						}
					} else if actual.Validation != nil {
						return schemaReferenceUnsupported(source.Pointer)
					}
				default:
					return schemaReferenceUnsupported(source.Pointer)
				}
			}
			use.Proof = tupleCollectionUse{Source: source.Schema, Pointer: source.Pointer, Snapshots: dtoOriginalGetterTypes(use.Actual), Final: true}
			use.Retained = retained[use.Actual]
		}
		source.FinalUseCount = len(source.Uses)
	}
	return nil
}

func (a *analyzer) inlineEnumRetainedDeclarations(declarations []Declaration, client *Client) map[*document.Value][]*StringEnum {
	retained := map[*document.Value][]*StringEnum{}
	seen := map[Declaration]bool{}
	var visit func(Declaration)
	visit = func(d Declaration) {
		if d == nil || seen[d] {
			return
		}
		seen[d] = true
		switch d := d.(type) {
		case *StringEnum:
			if origin, ok := a.enumDiagnosticOrigins[d]; ok {
				retained[origin.Schema] = append(retained[origin.Schema], d)
			}
		case *Model:
			for _, child := range d.Nested {
				visit(child)
			}
		case *ObjectUnion:
			for _, variant := range d.Variants {
				visit(variant)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		case *Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf)
			}
		}
	}
	for _, d := range declarations {
		visit(d)
	}
	if client != nil {
		for _, d := range client.Models {
			visit(d)
		}
		for _, group := range client.Groups {
			for _, operation := range group.Operations {
				for _, d := range operation.Models {
					visit(d)
				}
			}
		}
	}
	return retained
}

func (a *analyzer) validateInlineEnumCollectionChildDefaults(declarations []Declaration, pending bool) *diagnostic.Diagnostic {
	if a.inlineEnumChildDefaultOrder == nil {
		return nil
	}
	if len(*a.inlineEnumChildDefaultOrder) != len(a.inlineEnumChildDefaults) {
		return schemaReferenceUnsupported("")
	}
	retained := collectionChildDefaultRetainedTypes(declarations, a.collectionChildDefaultClient)
	enums := a.inlineEnumRetainedDeclarations(declarations, a.collectionChildDefaultClient)
	for _, source := range *a.inlineEnumChildDefaultOrder {
		if source == nil {
			return schemaReferenceUnsupported("")
		}
		if !a.validatedInlineEnumCollectionChildDefault(source.Schema, source.Pointer) ||
			source.FinalUseCount > len(source.Uses) || !pending && source.FinalUseCount != len(source.Uses) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, use := range source.Uses[:source.FinalUseCount] {
			if use == nil || use.Actual == nil || use.Proof.Source != source.Schema || use.Proof.Pointer != source.Pointer ||
				!tupleCollectionUseMatches(use.Actual, use.Proof) || use.Retained && !retained[use.Actual] {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if use.Retained {
				actual := use.Actual.WithoutRootNull()
				if actual.Validation != nil && a.validationSources[actual.Validation] != source.Pointer {
					return schemaReferenceUnsupported(source.Pointer)
				}
				if use.Enum != nil && (actual.Kind != Named || !slices.Contains(enums[source.Schema], use.Enum) ||
					use.Enum.ClassName != use.EnumName || !slices.Equal(use.Enum.Members, use.EnumMembers) ||
					a.enumDiagnosticOrigins[use.Enum].Schema != source.Schema || a.enumDiagnosticOrigins[use.Enum].Pointer != source.Pointer) {
					return schemaReferenceUnsupported(source.Pointer)
				}
			}
		}
	}
	return nil
}
