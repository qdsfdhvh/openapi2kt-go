package schema

import (
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// These annotations validate independently; they do not initialize a List or
// Map. Limit this role to the original direct field of an ordinary named DTO.
type inlineBuiltinChildDefault struct {
	Schema, Parent, Owner, Value *document.Value
	Pointer, SchemaBytes         string
	Scalar                       string
	Nullable                     bool
	Members                      []document.Member
	Uses                         []*inlineBuiltinChildUse
	FinalUseCount                int
}

type inlineBuiltinChildUse struct {
	Actual   *TypeRef
	Proof    tupleCollectionUse
	Retained bool
}

func (a *analyzer) inlineBuiltinChildParent(v *document.Value, pointer string) (*document.Value, *document.Value) {
	parent := a.collectionReferenceChildParent(v, pointer)
	parts := strings.Split(pointer, "/")
	if parent == nil || len(parts) != 7 || parts[1] != "components" || parts[2] != "schemas" || parts[4] != "properties" {
		return nil, nil
	}
	ownerName, err := naming.DecodePointerSegment(parts[3], pointer)
	if err != nil {
		return nil, nil
	}
	fieldName, err := naming.DecodePointerSegment(parts[5], pointer)
	if err != nil {
		return nil, nil
	}
	owner := get(get(get(a.root, "components"), "schemas"), ownerName)
	if owner == nil || get(get(owner, "properties"), fieldName) != parent || has(owner, "default") && !a.componentDefaultSchemas[owner] ||
		has(owner, "$ref") || has(owner, "allOf") || has(owner, "oneOf") || has(owner, "anyOf") ||
		a.hierarchySources[owner] {
		return nil, nil
	}
	return parent, owner
}

func (a *analyzer) inlineBuiltinCollectionChildDefault(v *document.Value, pointer string, depth int) (bool, *diagnostic.Diagnostic) {
	if v == nil || !has(v, "default") || has(v, "$ref") || has(v, "enum") || has(v, "const") ||
		has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") || has(v, "x-native-type") || has(v, "x-kotlin-value-class") {
		return false, nil
	}
	parent, owner := a.inlineBuiltinChildParent(v, pointer)
	if parent == nil {
		return false, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return false, err
	}
	if old := a.inlineBuiltinChildDefaults[v]; old != nil {
		if !a.validatedInlineBuiltinChildDefault(v, pointer) {
			return false, schemaReferenceUnsupported(pointer)
		}
		return true, nil
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return false, err
	}
	if !slices.Contains([]string{"string", "integer", "number", "boolean"}, kind) {
		return false, nil
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return false, err
	}
	if !slices.Contains([]string{
		"kotlin.String", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double", "kotlin.Boolean",
		"kotlin.uuid.Uuid", "kotlinx.datetime.LocalDate", "kotlin.time.Instant",
	}, scalar) {
		return false, nil
	}
	value := get(v, "default")
	at := naming.Pointer(pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return false, err
	}
	// Reuse the ordinary scalar grammar and its opt-in bound checks without
	// borrowing its field initializer.
	validation := *a
	validation.warnings = nil
	if _, _, err := validation.modelDefaultType(field{Schema: v, Pointer: pointer, Depth: depth}, true, "Validation", nil, nil); err != nil {
		return false, err
	}
	// Collection defaults use the intersection of Go and Kotlin Instant
	// grammars, which is narrower than Go's RFC3339 parser alone.
	if scalar == "kotlin.time.Instant" && value.Kind() != document.Null && !collectionDefaultInstantPattern.MatchString(text(value)) {
		return false, invalid("Timestamp default must fit the Kotlin Instant format and precision.", at)
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
	if a.inlineBuiltinChildDefaults == nil {
		a.inlineBuiltinChildDefaults = map[*document.Value]*inlineBuiltinChildDefault{}
	}
	if a.inlineBuiltinChildDefaultOrder == nil {
		a.inlineBuiltinChildDefaultOrder = &[]*inlineBuiltinChildDefault{}
	}
	source := &inlineBuiltinChildDefault{
		Schema: v, Parent: parent, Owner: owner, Value: value, Pointer: pointer,
		SchemaBytes: string(raw), Scalar: scalar, Nullable: nullable,
		Members: append([]document.Member(nil), v.Members()...),
	}
	a.inlineBuiltinChildDefaults[v] = source
	*a.inlineBuiltinChildDefaultOrder = append(*a.inlineBuiltinChildDefaultOrder, source)
	return true, nil
}

func (a *analyzer) validatedInlineBuiltinChildDefault(v *document.Value, pointer string) bool {
	source := a.inlineBuiltinChildDefaults[v]
	parent, owner := a.inlineBuiltinChildParent(v, pointer)
	if source == nil || source.Schema != v || source.Pointer != pointer || source.Parent != parent ||
		source.Owner != owner || source.Value != get(v, "default") {
		return false
	}
	raw, _ := v.MarshalJSON()
	kind, nullable, err := schemaType(v, pointer)
	if err != nil || nullable != source.Nullable {
		return false
	}
	scalar, err := scalarMapping(kind, v, pointer)
	return err == nil && scalar == source.Scalar && string(raw) == source.SchemaBytes &&
		slices.Equal(v.Members(), source.Members)
}

func (a *analyzer) bindInlineBuiltinChildDefault(v *document.Value, pointer string, actual *TypeRef) *diagnostic.Diagnostic {
	source := a.inlineBuiltinChildDefaults[v]
	if source == nil {
		return nil
	}
	if !a.validatedInlineBuiltinChildDefault(v, pointer) || actual == nil {
		return schemaReferenceUnsupported(pointer)
	}
	if err := a.tupleNativeSources.charge(1, 0, pointer); err != nil {
		return err
	}
	source.Uses = append(source.Uses, &inlineBuiltinChildUse{Actual: actual})
	return nil
}

func (a *analyzer) finalizeInlineBuiltinChildDefaults(declarations []Declaration, client *Client) *diagnostic.Diagnostic {
	if a.inlineBuiltinChildDefaultOrder == nil {
		return nil
	}
	retained := collectionChildDefaultRetainedTypes(declarations, client)
	for _, source := range *a.inlineBuiltinChildDefaultOrder {
		if source == nil || !a.validatedInlineBuiltinChildDefault(source.Schema, source.Pointer) ||
			source.FinalUseCount > len(source.Uses) {
			return schemaReferenceUnsupported("")
		}
		var checks *PrimitiveValidation
		var err *diagnostic.Diagnostic
		if a.options.Validation {
			checks, err = a.primitiveValidation(source.Schema, source.Scalar, source.Pointer)
			if err != nil {
				return err
			}
		}
		for _, use := range source.Uses[source.FinalUseCount:] {
			if use == nil || use.Actual == nil {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if retained[use.Actual] {
				actual := use.Actual.WithoutRootNull()
				if (use.Actual.Kind == Nullable) != source.Nullable || actual.Kind != Scalar || actual.Name != source.Scalar ||
					namedCollectionChecks(actual) != namedCollectionChecks(TypeRef{Validation: checks}) ||
					actual.Validation != nil && a.validationSources[actual.Validation] != source.Pointer {
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

func (a *analyzer) validateInlineBuiltinChildDefaults(declarations []Declaration, client *Client, pending bool) *diagnostic.Diagnostic {
	if a.inlineBuiltinChildDefaultOrder == nil {
		return nil
	}
	retained := collectionChildDefaultRetainedTypes(declarations, client)
	if len(*a.inlineBuiltinChildDefaultOrder) != len(a.inlineBuiltinChildDefaults) {
		return schemaReferenceUnsupported("")
	}
	for _, source := range *a.inlineBuiltinChildDefaultOrder {
		if source == nil || !a.validatedInlineBuiltinChildDefault(source.Schema, source.Pointer) ||
			source.FinalUseCount > len(source.Uses) || !pending && source.FinalUseCount != len(source.Uses) {
			return schemaReferenceUnsupported("")
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
			}
		}
	}
	return nil
}
