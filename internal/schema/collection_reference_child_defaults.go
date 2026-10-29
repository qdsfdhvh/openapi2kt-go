package schema

import (
	"slices"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type collectionReferenceChildDefault struct {
	Schema, Parent, Reference, Value, Target *document.Value
	TargetCatalog, TargetScalar              *component
	Pointer, TargetPointer, ValueBytes       string
	Encoded                                  string
	Bindings                                 []tupleLiteralBinding
	ScalarSources                            []collectionChildScalarSource
	Collections                              []*scalarFactoryCollectionBinding
	Collection, CollectionValueReady         bool
	Factories, CollectionFactories           []collectionChildScalarFactory
	Uses                                     []*collectionReferenceChildDefaultUse
	FinalUseCount, FinalFactoryCount         int
	FinalCollectionFactoryCount              int
	InlineScalarCollection                   bool
	InlineEnumCollection                     bool
	ReferencedEnumCollection                 bool
	InlineEnumLeaf                           *document.Value
	InlineEnumPointer                        string
	InlineEnumPeer                           *StringEnum
	InlineEnumPeerOrigin                     nominalOrigin
	InlineEnumPeerSnapshot                   Declaration
	ReferencedEnumNative                     *nativeEnumOrigin
	ReferencedEnumNativeFactory              string
	ReferencedEnumNativeScalar               *ScalarAlias
	ReferencedEnumNativeType                 tupleCollectionUse
}

type collectionChildScalarSource struct {
	Owner              *component
	Schema             *document.Value
	Pointer, Reference string
	Validation         *PrimitiveValidation
}

type collectionChildScalarFactory struct {
	Owner       *component
	Declaration Declaration
	Value       *document.Value
	ValueBytes  string
	Encoded     string
}

type collectionReferenceChildDefaultUse struct {
	Type        TypeRef
	Actual      *TypeRef
	Depth       int
	FinalDepth  int
	TypeProof   tupleCollectionUse
	ActualProof tupleCollectionUse
	Retained    bool
}

// Resolve only original physical paths; an equal copied node or a tuple
// projection must not acquire a collection-child annotation role.
func (a *analyzer) collectionReferenceChildParent(v *document.Value, pointer string) *document.Value {
	at := strings.LastIndex(pointer, "/")
	if a.root == nil || at < 0 {
		return nil
	}
	edge := pointer[at+1:]
	if edge != "items" && edge != "additionalProperties" {
		return nil
	}
	parent := a.root
	for _, raw := range strings.Split(strings.TrimPrefix(pointer[:at], "/"), "/") {
		key, err := naming.DecodePointerSegment(raw, pointer)
		if err != nil || parent == nil {
			return nil
		}
		switch parent.Kind() {
		case document.Object:
			parent = get(parent, key)
		case document.Array:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(parent.Items()) {
				return nil
			}
			parent = parent.Items()[index]
		default:
			return nil
		}
	}
	if parent == nil || get(parent, edge) != v {
		return nil
	}
	if edge == "additionalProperties" {
		if dictionarySchema(parent) {
			return parent
		}
		return nil
	}
	kind, _, err := schemaType(parent, pointer[:at])
	if err == nil && kind == "array" && !has(parent, "prefixItems") {
		return parent
	}
	return nil
}

// Admit only original enum or builtin scalar sources. The reference planner
// still decides whether the actual source path and linked type are supported;
// referencedDefaultValue then verifies the complete source grammar and value.
func collectionReferenceChildDefaultSource(v *document.Value) bool {
	if originalEnumWrapperDefaultSource(v) {
		return true
	}
	if v == nil || has(v, "$ref") || has(v, "enum") || has(v, "const") ||
		has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") ||
		has(v, "x-native-type") || has(v, "x-kotlin-value-class") {
		return false
	}
	kind, _, err := schemaType(v, "")
	if err != nil || kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
		return false
	}
	scalar, err := scalarMapping(kind, v, "")
	if err != nil {
		return false
	}
	switch scalar {
	case "kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double",
		"kotlin.uuid.Uuid", "kotlinx.datetime.LocalDate", "kotlin.time.Instant", "kotlin.ByteArray":
		return true
	default:
		return false
	}
}

// A borrowed collection value may pass through an independently admitted
// scalar component factory. Its annotation stays with the original component,
// never with the collection element or the outer literal.
func (a *analyzer) independentScalarComponentDefault(v *document.Value, pointer string) bool {
	if !a.tupleSourceOwnDefault(v, pointer) || !has(v, "default") {
		return false
	}
	if !has(v, "$ref") {
		return !has(v, "enum") && collectionReferenceChildDefaultSource(v)
	}
	target := a.referenceWrapperDefaultTarget(&component{Schema: v, Pointer: pointer})
	return target != nil && !target.Internal && !has(target.Schema, "enum") &&
		a.componentRoot(target.Schema, target.Pointer) && collectionReferenceChildDefaultSource(target.Schema)
}

func (a *analyzer) scalarReferenceHasIndependentFactory(v *document.Value, pointer string) bool {
	seen := map[*document.Value]bool{}
	found := false
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || seen[v] {
			return false
		}
		seen[v] = true
		found = found || a.independentScalarComponentDefault(v, pointer)
		if !has(v, "$ref") {
			return found && !has(v, "enum") && collectionReferenceChildDefaultSource(v)
		}
		var err *diagnostic.Diagnostic
		v, pointer, err = a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return false
		}
	}
	return false
}

// A child can be reachable both through the catalog backing and the emitted
// alias. Keep the original named collection declaration in the final proof so
// replacing its element cannot be hidden by another retained consumer.
func (a *analyzer) bindScalarReferenceCollectionOwner(pointer string) {
	for _, owner := range a.components {
		if !namedCollectionCandidate(owner) || !strings.HasPrefix(pointer, owner.Pointer+"/") {
			continue
		}
		if a.namedCollectionUses == nil {
			a.namedCollectionUses = map[*component]string{}
		}
		if previous, ok := a.namedCollectionUses[owner]; !ok || pointer < previous {
			a.namedCollectionUses[owner] = pointer
		}
	}
}

func collectionReferenceChildDefaultTarget(target *resolvedSchema) bool {
	if target == nil {
		return false
	}
	if collectionReferenceChildDefaultSource(target.Schema) {
		return true
	}
	// Legacy builtin aliases retain their own source schema in lookup while
	// ScalarTarget binds the already admitted canonical builtin declaration.
	catalog := target.Catalog
	return catalog != nil && target.Schema == catalog.Schema && target.Pointer == catalog.Pointer &&
		catalog.ScalarTarget != nil && catalog.Reference != "" &&
		catalog.Scalar == catalog.ScalarTarget.Scalar && catalog.Validation == catalog.ScalarTarget.Validation &&
		builtinPrimitiveBinding(catalog.ScalarTarget) &&
		collectionReferenceChildDefaultSource(catalog.ScalarTarget.Schema)
}

func collectionReferenceChildScalarTarget(target *resolvedSchema) *component {
	if target == nil || target.Catalog == nil {
		return nil
	}
	return target.Catalog.ScalarTarget
}

// Inline scalar collection children borrow only finite original named
// collection roots and aliases. The collection tree itself contains no refs;
// its actual linked collection declaration still decides admission.
func (a *analyzer) inlineScalarCollectionChildSource(v *document.Value, pointer string) bool {
	leaf, _ := a.inlineCollectionChildTerminal(v, pointer, false)
	return leaf != nil
}

func (a *analyzer) inlineEnumCollectionChildSource(v *document.Value, pointer string) bool {
	leaf, _ := a.inlineCollectionChildTerminal(v, pointer, true)
	return leaf != nil
}

// Each reference after a collection edge must begin at an original physical
// child and resolve through named components. A referenced collection may
// continue the path; the final referenced component must be an enum.
func (a *analyzer) referencedEnumCollectionChildTerminal(v *document.Value, pointer string) (*document.Value, string) {
	seen := map[*document.Value]bool{}
	edges := 0
	physicalReference := false
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || seen[v] {
			return nil, ""
		}
		seen[v] = true
		if has(v, "$ref") {
			if edges > 0 && !physicalReference {
				if a.collectionReferenceChildParent(v, pointer) == nil {
					return nil, ""
				}
				physicalReference = true
			}
			p, target, ok := a.componentSchemaAtReference(text(get(v, "$ref")))
			if !ok {
				return nil, ""
			}
			v, pointer = target, p
			continue
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return nil, ""
		}
		edge := ""
		if kind == "array" && !has(v, "prefixItems") {
			edge = "items"
		} else if dictionarySchema(v) {
			edge = "additionalProperties"
		}
		if edge == "" {
			if physicalReference && originalEnumWrapperDefaultSource(v) {
				return v, pointer
			}
			return nil, ""
		}
		edges++
		physicalReference = false
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return nil, ""
}

func (a *analyzer) referencedEnumCollectionChildSource(v *document.Value, pointer string) bool {
	leaf, _ := a.referencedEnumCollectionChildTerminal(v, pointer)
	return leaf != nil
}

func (a *analyzer) inlineCollectionChildTerminal(v *document.Value, pointer string, enum bool) (*document.Value, string) {
	seen := map[*document.Value]bool{}
	edges := 0
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || seen[v] {
			return nil, ""
		}
		seen[v] = true
		if has(v, "$ref") {
			if edges != 0 {
				return nil, ""
			}
			p, target, ok := a.componentSchemaAtReference(text(get(v, "$ref")))
			if !ok {
				return nil, ""
			}
			v, pointer = target, p
			continue
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return nil, ""
		}
		edge := ""
		if kind == "array" && !has(v, "prefixItems") {
			edge = "items"
		} else if dictionarySchema(v) {
			edge = "additionalProperties"
		}
		if edge == "" {
			if edges > 0 && (enum && originalEnumWrapperDefaultSource(v) ||
				!enum && !has(v, "enum") && collectionReferenceChildDefaultSource(v)) {
				return v, pointer
			}
			return nil, ""
		}
		edges++
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return nil, ""
}

func (a *analyzer) independentInlineScalarCollectionChildTargetDefault(v *document.Value, pointer string) bool {
	return a.collectionChildInlineScalar && a.tupleSourceOwnDefault(v, pointer) && a.inlineScalarCollectionChildSource(v, pointer)
}

func (a *analyzer) independentInlineEnumCollectionChildTargetDefault(v *document.Value, pointer string) bool {
	return a.collectionChildInlineEnum && a.tupleSourceOwnDefault(v, pointer) && a.inlineEnumCollectionChildSource(v, pointer)
}

// Collection-valued child annotations borrow only an already admitted named
// finite List/Map with a supported scalar leaf. The actual linked collection
// declaration, not its equivalent Kotlin spelling, grants this role.
func (a *analyzer) collectionReferenceChildCollectionTarget(target *resolvedSchema) bool {
	if target == nil || target.Catalog == nil {
		return false
	}
	c := target.CanonicalCollection
	if c == nil {
		c = target.Catalog
		if c.BindingTarget != nil {
			c = c.BindingTarget
		}
	}
	return namedCollectionCandidate(c) && target.Schema == c.Schema && target.Pointer == c.Pointer &&
		(a.referencedScalarCollectionSource(c.Schema, c.Pointer) || a.inlineScalarCollectionChildSource(c.Schema, c.Pointer) || a.inlineEnumCollectionChildSource(c.Schema, c.Pointer) || a.referencedEnumCollectionChildSource(c.Schema, c.Pointer))
}

func (a *analyzer) collectionReferenceChildDefaultValue(v *document.Value, pointer string, depth int, ty TypeRef, target *resolvedSchema) (*nestedDefaultNode, string, *diagnostic.Diagnostic) {
	f := field{Schema: v, Pointer: pointer, Depth: depth}
	if a.collectionReferenceChildCollectionTarget(target) {
		validation := *a
		validation.collectionChildInlineScalar = a.inlineScalarCollectionChildSource(target.Schema, target.Pointer)
		validation.collectionChildInlineEnum = a.inlineEnumCollectionChildSource(target.Schema, target.Pointer)
		return validation.referencedDefaultValueRole(f, ty, referencedCollectionOwnDefault)
	}
	if originalEnumWrapperDefaultSource(target.Schema) {
		return a.referencedDefaultValue(f, ty)
	}
	return a.referencedDefaultValueRole(f, ty, referencedScalarChildOwnDefault)
}

func (a *analyzer) collectionChildScalarBindings(original *document.Value, source string, terminal *document.Value, pointer string) ([]tupleLiteralBinding, *diagnostic.Diagnostic) {
	var bindings []tupleLiteralBinding
	node, at := original, source
	for steps := 0; steps <= MaxSchemaDepth; steps++ {
		if owner := a.componentsByOriginalSchema(node); owner != nil {
			if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
				return nil, err
			}
			bindings = append(bindings, tupleLiteralBinding{owner, owner.Name, owner.Scalar, owner.BindingTarget, owner.ScalarTarget, owner.AliasHeight, falseNullableIntent(owner)})
		}
		if node == terminal && at == pointer {
			return bindings, nil
		}
		if !has(node, "$ref") {
			return nil, schemaReferenceUnsupported(source)
		}
		var err *diagnostic.Diagnostic
		node, at, err = a.defaultSource(text(get(node, "$ref")), at)
		if err != nil {
			return nil, err
		}
	}
	return nil, schemaReferenceUnsupported(source)
}

// A collection-valued child must also retain the scalar aliases at the end of
// its target's physical collection path. Matching Kotlin element types alone
// cannot distinguish two different same-type scalar component bindings.
func (a *analyzer) collectionChildCollectionScalarBindings(original *document.Value, source string) ([]tupleLiteralBinding, *diagnostic.Diagnostic) {
	v, pointer := original, source
	var leaf *document.Value
	var leafPointer string
	seen := map[*document.Value]bool{}
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || seen[v] {
			return nil, schemaReferenceUnsupported(source)
		}
		seen[v] = true
		if has(v, "$ref") {
			if a.collectionReferenceChildParent(v, pointer) != nil {
				leaf, leafPointer = v, pointer
			}
			var err *diagnostic.Diagnostic
			v, pointer, err = a.defaultSource(text(get(v, "$ref")), pointer)
			if err != nil {
				return nil, err
			}
			continue
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		edge := ""
		if kind == "array" && !has(v, "prefixItems") {
			edge = "items"
		} else if dictionarySchema(v) {
			edge = "additionalProperties"
		}
		if edge == "" {
			if has(v, "enum") && leaf == nil && a.inlineEnumCollectionChildSource(original, source) {
				return nil, nil
			}
			if has(v, "enum") && leaf != nil && a.referencedEnumCollectionChildSource(original, source) {
				return a.collectionChildScalarBindings(leaf, leafPointer, v, pointer)
			}
			if !collectionReferenceChildDefaultSource(v) || has(v, "enum") {
				return nil, schemaReferenceUnsupported(source)
			}
			if leaf == nil {
				if a.inlineScalarCollectionChildSource(original, source) {
					return nil, nil
				}
				return nil, schemaReferenceUnsupported(source)
			}
			return a.collectionChildScalarBindings(leaf, leafPointer, v, pointer)
		}
		leaf, leafPointer = nil, ""
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return nil, schemaReferenceUnsupported(source)
}

func (a *analyzer) collectionReferenceChildDefault(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	return a.collectionReferenceChildSource(v, pointer, depth, false)
}

// The outer-default walk also records an unannotated physical reference when
// it borrows independently owned scalar factories. This proof must not depend
// on whether the collection child happens to have its own annotation.
func (a *analyzer) collectionReferenceChildSource(v *document.Value, pointer string, depth int, outer bool) (*TypeRef, *diagnostic.Diagnostic) {
	if !has(v, "$ref") || !has(v, "default") && !outer || a.collectionReferenceChildParent(v, pointer) == nil {
		return nil, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := checkKeys(v, []string{"$ref", "nullable", "default"}, pointer); err != nil {
		return nil, err
	}
	if _, _, err := schemaType(v, pointer); err != nil {
		return nil, err
	}
	reference, ok := get(v, "$ref").Text()
	if !ok {
		return nil, invalid("$ref must be a string.", pointer)
	}
	target, err := a.resolveSchema(reference, pointer)
	if err != nil {
		return nil, err
	}
	// Scalar and enum children and the bounded named-collection path above
	// acquire this role. Other collection and source kinds keep their rules.
	collection := a.collectionReferenceChildCollectionTarget(target)
	if !collection && !collectionReferenceChildDefaultTarget(target) {
		return nil, nil
	}
	if collection && !has(v, "default") {
		return nil, nil
	}
	if !has(v, "default") && (!outer || !a.scalarCollectionFactoryPlanning && !a.scalarReferenceHasIndependentFactory(v, pointer)) {
		return nil, nil
	}
	ty, err := a.referenceType(target, nullableOverride(v), depth, pointer)
	if err != nil {
		return nil, err
	}
	encoded := ""
	if has(v, "default") && !collection {
		_, encoded, err = a.collectionReferenceChildDefaultValue(v, pointer, depth, ty, target)
		if err != nil {
			return nil, err
		}
	}
	if a.collectionChildDefaults == nil {
		a.collectionChildDefaults = map[*document.Value]*collectionReferenceChildDefault{}
	}
	if a.collectionChildDefaultOrder == nil {
		a.collectionChildDefaultOrder = &[]*collectionReferenceChildDefault{}
	}
	source := a.collectionChildDefaults[v]
	if source == nil {
		if err := a.captureTupleNativeSource(v, pointer, target.Schema, target.Pointer); err != nil {
			return nil, err
		}
		scalarTarget := collectionReferenceChildScalarTarget(target)
		terminal, terminalPointer := target.Schema, target.Pointer
		if scalarTarget != nil && target.Schema == target.Catalog.Schema && has(target.Schema, "$ref") {
			// Legacy aliases resolve to their own schema while their linked type
			// comes from a canonical scalar. Retain the whole physical path as
			// well as that component identity.
			if err := a.captureTupleNativeSource(v, pointer, scalarTarget.Schema, scalarTarget.Pointer); err != nil {
				return nil, err
			}
			terminal, terminalPointer = scalarTarget.Schema, scalarTarget.Pointer
		}
		var bindings []tupleLiteralBinding
		var scalarSources []collectionChildScalarSource
		var collections []*scalarFactoryCollectionBinding
		if collection {
			collections, err = a.captureScalarFactoryCollections(&component{Schema: v, Pointer: pointer})
			if err != nil {
				return nil, err
			}
			bindings, err = a.collectionChildCollectionScalarBindings(v, pointer)
			if err != nil {
				return nil, err
			}
			for _, binding := range bindings {
				owner := binding.Owner
				scalarSources = append(scalarSources, collectionChildScalarSource{
					Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer,
					Reference: owner.Reference, Validation: owner.Validation,
				})
			}
		} else if !originalEnumWrapperDefaultSource(terminal) {
			bindings, err = a.collectionChildScalarBindings(v, pointer, terminal, terminalPointer)
			if err != nil {
				return nil, err
			}
		}
		value := get(v, "default")
		var raw []byte
		if value != nil {
			if err := a.captureTupleNativePath(value, naming.Pointer(pointer, "default"), pointer); err != nil {
				return nil, err
			}
			raw, _ = value.MarshalJSON()
			if err := a.tupleNativeSources.charge(0, len(raw), pointer); err != nil {
				return nil, err
			}
		}
		source = &collectionReferenceChildDefault{
			Schema: v, Parent: a.collectionReferenceChildParent(v, pointer),
			Reference: get(v, "$ref"), Value: value, Pointer: pointer,
			Target: target.Schema, TargetPointer: target.Pointer,
			TargetCatalog: target.Catalog, TargetScalar: scalarTarget, ValueBytes: string(raw), Encoded: encoded, Bindings: bindings,
			ScalarSources: scalarSources, Collections: collections, Collection: collection,
			InlineScalarCollection:   collection && a.inlineScalarCollectionChildSource(v, pointer),
			InlineEnumCollection:     collection && a.inlineEnumCollectionChildSource(v, pointer),
			ReferencedEnumCollection: collection && a.referencedEnumCollectionChildSource(v, pointer),
		}
		a.collectionChildDefaults[v] = source
		*a.collectionChildDefaultOrder = append(*a.collectionChildDefaultOrder, source)
	} else if !a.validatedCollectionReferenceChildDefault(v, pointer) ||
		source.Target != target.Schema || source.TargetPointer != target.Pointer ||
		source.TargetCatalog != target.Catalog || source.TargetScalar != collectionReferenceChildScalarTarget(target) ||
		source.Collection != collection || !collection && source.Encoded != encoded {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if err := a.tupleNativeSources.charge(1, 0, pointer); err != nil {
		return nil, err
	}
	source.Uses = append(source.Uses, &collectionReferenceChildDefaultUse{Type: ty, Depth: depth})
	return &ty, nil
}

func (a *analyzer) validatedCollectionReferenceChildDefault(v *document.Value, pointer string) bool {
	record, ok := a.collectionChildDefaults[v]
	return ok && record != nil && record.Schema == v && record.Pointer == pointer &&
		record.Parent == a.collectionReferenceChildParent(v, pointer) &&
		record.Reference == get(v, "$ref") && record.Value == get(v, "default")
}

// Compare the linked type and its actual validation objects without requiring
// separately allocated nullable wrappers to have the same address.
func collectionChildDefaultTypesMatch(left, right TypeRef) bool {
	if dtoUniqueCollectionReferenceTypeSignature(left) != dtoUniqueCollectionReferenceTypeSignature(right) {
		return false
	}
	for depth := 0; ; depth++ {
		if depth > MaxSchemaDepth || left.Validation != right.Validation || left.Collection != right.Collection {
			return false
		}
		if left.Element == nil || right.Element == nil {
			return left.Element == nil && right.Element == nil
		}
		left, right = *left.Element, *right.Element
	}
}

// The element address belongs to the real List/Map planner. Keep it separate
// from the preflight type copy so a later replacement or changed checks cannot
// silently reuse the annotation's source permission.
func (a *analyzer) bindCollectionReferenceChildDefault(v *document.Value, pointer string, actual *TypeRef) *diagnostic.Diagnostic {
	source := a.collectionChildDefaults[v]
	if source == nil {
		return nil
	}
	if !a.validatedCollectionReferenceChildDefault(v, pointer) || actual == nil {
		return schemaReferenceUnsupported(pointer)
	}
	for i := len(source.Uses) - 1; i >= 0; i-- {
		use := source.Uses[i]
		if use.Actual != nil {
			continue
		}
		if !collectionChildDefaultTypesMatch(use.Type, *actual) {
			return schemaReferenceUnsupported(pointer)
		}
		use.Actual = actual
		return nil
	}
	return schemaReferenceUnsupported(pointer)
}

func (a *analyzer) collectionChildDefaultTarget(source *collectionReferenceChildDefault, use *collectionReferenceChildDefaultUse) *diagnostic.Diagnostic {
	if source == nil {
		return schemaReferenceUnsupported("")
	}
	if use == nil || !a.validatedCollectionReferenceChildDefault(source.Schema, source.Pointer) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	if source.Value != nil {
		raw, _ := source.Value.MarshalJSON()
		if string(raw) != source.ValueBytes {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	for _, binding := range source.Bindings {
		owner := binding.Owner
		if owner == nil || a.components[owner.Wire] != owner || owner.Name != binding.Name || owner.Scalar != binding.Scalar ||
			owner.BindingTarget != binding.BindingTarget || owner.ScalarTarget != binding.ScalarTarget ||
			owner.AliasHeight != binding.AliasHeight || falseNullableIntent(owner) != binding.NullableIntent {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	if source.Collection && len(source.ScalarSources) != len(source.Bindings) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	for i, binding := range source.ScalarSources {
		owner := binding.Owner
		if i >= len(source.Bindings) || source.Bindings[i].Owner != owner ||
			owner == nil || a.components[owner.Wire] != owner || owner.Schema != binding.Schema ||
			owner.Pointer != binding.Pointer || owner.Reference != binding.Reference || owner.Validation != binding.Validation {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	target, err := a.resolveSchema(text(source.Reference), source.Pointer)
	if err != nil {
		return err
	}
	if target.Schema != source.Target || target.Pointer != source.TargetPointer || target.Catalog != source.TargetCatalog ||
		collectionReferenceChildScalarTarget(target) != source.TargetScalar ||
		source.Collection != a.collectionReferenceChildCollectionTarget(target) ||
		source.InlineScalarCollection != (source.Collection && a.inlineScalarCollectionChildSource(source.Schema, source.Pointer)) ||
		source.InlineEnumCollection != (source.Collection && a.inlineEnumCollectionChildSource(source.Schema, source.Pointer)) ||
		source.ReferencedEnumCollection != (source.Collection && a.referencedEnumCollectionChildSource(source.Schema, source.Pointer)) ||
		!source.Collection && !collectionReferenceChildDefaultTarget(target) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	ty, err := a.referenceType(target, nullableOverride(source.Schema), use.Depth, source.Pointer)
	if err != nil {
		return err
	}
	if !collectionChildDefaultTypesMatch(use.Type, ty) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	if source.Value != nil {
		validation := *a
		validation.collectionChildRevalidation = source.Collection
		_, encoded, err := validation.collectionReferenceChildDefaultValue(source.Schema, source.Pointer, use.Depth, use.Type, target)
		if err != nil {
			return err
		}
		if encoded != source.Encoded {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	return nil
}

func collectionChildScalarFactoryValue(d Declaration) string {
	switch factory := d.(type) {
	case *StringEnum:
		return factory.ComponentDefault
	case *ScalarAlias:
		return factory.ComponentDefault
	case *ValueAlias:
		return factory.ComponentDefault
	default:
		return ""
	}
}

func (a *analyzer) collectionChildEnumTerminal(source *collectionReferenceChildDefault) (*document.Value, string) {
	if source.InlineEnumCollection && !source.ReferencedEnumCollection {
		return a.inlineCollectionChildTerminal(source.Schema, source.Pointer, true)
	}
	if source.ReferencedEnumCollection && !source.InlineEnumCollection {
		return a.referencedEnumCollectionChildTerminal(source.Schema, source.Pointer)
	}
	return nil, ""
}

// Freeze the actual enum export and source, rather than trusting a collection
// element with an equivalent rendered type.
func (a *analyzer) freezeEnumCollectionChild(source *collectionReferenceChildDefault, declarations []Declaration) *diagnostic.Diagnostic {
	leaf, pointer := a.collectionChildEnumTerminal(source)
	if leaf == nil {
		return schemaReferenceUnsupported(source.Pointer)
	}
	source.InlineEnumLeaf, source.InlineEnumPointer = leaf, pointer
	kind, _, err := schemaType(leaf, pointer)
	if err != nil {
		return err
	}
	if kind != "" && kind != "string" {
		if source.ReferencedEnumCollection {
			owner := a.componentsByOriginalSchema(leaf)
			if owner == nil || a.components[owner.Wire] != owner || owner.Schema != leaf || owner.Pointer != pointer {
				return schemaReferenceUnsupported(source.Pointer)
			}
			vocabulary := get(leaf, "enum")
			raw, _ := leaf.MarshalJSON()
			origin := &nativeEnumOrigin{
				Owner: owner, Schema: leaf, Pointer: pointer, Bytes: string(raw),
				Scalar: owner.Scalar, Validation: owner.Validation,
				Checks:     namedCollectionChecks(TypeRef{Validation: owner.Validation}),
				Vocabulary: vocabulary, Members: append([]*document.Value{}, vocabulary.Items()...),
				SchemaMembers: append([]document.Member{}, leaf.Members()...),
				Actual:        componentDeclaration(declarations, owner.Name),
			}
			if !nativeEnumDeclaration(origin.Actual, origin) {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if err := a.tupleNativeSources.charge(1+len(origin.Members), len(raw), source.Pointer); err != nil {
				return err
			}
			source.ReferencedEnumNative = origin
			source.ReferencedEnumNativeFactory = collectionChildScalarFactoryValue(origin.Actual)
			switch actual := origin.Actual.(type) {
			case *ScalarAlias:
				snapshot := *actual
				source.ReferencedEnumNativeScalar = &snapshot
			case *ValueAlias:
				source.ReferencedEnumNativeType = tupleCollectionUse{
					Source: leaf, Pointer: pointer, Snapshots: dtoOriginalGetterTypes(&actual.Type), Final: true,
				}
			default:
				return schemaReferenceUnsupported(source.Pointer)
			}
		}
		return nil
	}
	for _, declaration := range declarations {
		peer, ok := declaration.(*StringEnum)
		if !ok {
			continue
		}
		origin := a.enumOrigins[peer]
		if origin.Schema != leaf || origin.Pointer != pointer {
			continue
		}
		if source.InlineEnumPeer != nil {
			return schemaReferenceUnsupported(source.Pointer)
		}
		bytes := len(peer.ClassName)
		for _, member := range peer.Members {
			bytes += len(member.Name) + len(member.Wire)
		}
		if err := a.tupleNativeSources.charge(1+len(peer.Members), bytes, source.Pointer); err != nil {
			return err
		}
		source.InlineEnumPeer, source.InlineEnumPeerOrigin = peer, origin
		source.InlineEnumPeerSnapshot = tupleUnionDeclarationSnapshot(peer)
	}
	if source.InlineEnumPeer == nil || source.InlineEnumPeerSnapshot == nil {
		return schemaReferenceUnsupported(source.Pointer)
	}
	return nil
}

func (a *analyzer) validateEnumCollectionChild(source *collectionReferenceChildDefault, actual map[Declaration]int, names map[string]int) *diagnostic.Diagnostic {
	if !source.InlineEnumCollection && !source.ReferencedEnumCollection {
		if source.InlineEnumLeaf != nil || source.InlineEnumPointer != "" || source.InlineEnumPeer != nil ||
			source.InlineEnumPeerSnapshot != nil || source.InlineEnumPeerOrigin != (nominalOrigin{}) ||
			source.ReferencedEnumNative != nil || source.ReferencedEnumNativeFactory != "" ||
			source.ReferencedEnumNativeScalar != nil || source.ReferencedEnumNativeType.Final {
			return schemaReferenceUnsupported(source.Pointer)
		}
		return nil
	}
	leaf, pointer := a.collectionChildEnumTerminal(source)
	if leaf == nil || leaf != source.InlineEnumLeaf || pointer != source.InlineEnumPointer {
		return schemaReferenceUnsupported(source.Pointer)
	}
	kind, _, err := schemaType(leaf, pointer)
	if err != nil {
		return err
	}
	if kind != "" && kind != "string" {
		if source.InlineEnumPeer != nil || source.InlineEnumPeerSnapshot != nil || source.InlineEnumPeerOrigin != (nominalOrigin{}) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		origin := source.ReferencedEnumNative
		if source.InlineEnumCollection {
			if origin != nil || source.ReferencedEnumNativeFactory != "" || source.ReferencedEnumNativeScalar != nil || source.ReferencedEnumNativeType.Final {
				return schemaReferenceUnsupported(source.Pointer)
			}
			return nil
		}
		if origin == nil || origin.Owner == nil || a.components[origin.Owner.Wire] != origin.Owner ||
			origin.Owner.Schema != leaf || origin.Owner.Pointer != pointer || origin.Schema != leaf ||
			origin.Pointer != pointer || origin.Scalar != origin.Owner.Scalar || origin.Validation != origin.Owner.Validation ||
			namedCollectionChecks(TypeRef{Validation: origin.Owner.Validation}) != origin.Checks ||
			actual[origin.Actual] != 1 || names[origin.Owner.Name] != 1 || !nativeEnumDeclaration(origin.Actual, origin) ||
			collectionChildScalarFactoryValue(origin.Actual) != source.ReferencedEnumNativeFactory ||
			get(leaf, "enum") != origin.Vocabulary || !slices.Equal(origin.Members, origin.Vocabulary.Items()) ||
			!slices.Equal(origin.SchemaMembers, leaf.Members()) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		raw, _ := leaf.MarshalJSON()
		if string(raw) != origin.Bytes {
			return schemaReferenceUnsupported(source.Pointer)
		}
		switch exported := origin.Actual.(type) {
		case *ScalarAlias:
			if source.ReferencedEnumNativeScalar == nil || *source.ReferencedEnumNativeScalar != *exported ||
				source.ReferencedEnumNativeType.Final {
				return schemaReferenceUnsupported(source.Pointer)
			}
		case *ValueAlias:
			proof := source.ReferencedEnumNativeType
			if source.ReferencedEnumNativeScalar != nil || proof.Source != leaf || proof.Pointer != pointer ||
				!tupleCollectionUseMatches(&exported.Type, proof) {
				return schemaReferenceUnsupported(source.Pointer)
			}
		default:
			return schemaReferenceUnsupported(source.Pointer)
		}
		return nil
	}
	if source.ReferencedEnumNative != nil || source.ReferencedEnumNativeFactory != "" ||
		source.ReferencedEnumNativeScalar != nil || source.ReferencedEnumNativeType.Final {
		return schemaReferenceUnsupported(source.Pointer)
	}
	peer := source.InlineEnumPeer
	if peer == nil || actual[peer] != 1 || names[peer.Name()] != 1 ||
		a.enumOrigins[peer] != source.InlineEnumPeerOrigin ||
		source.InlineEnumPeerOrigin.Schema != leaf || source.InlineEnumPeerOrigin.Pointer != pointer ||
		!tupleUnionDeclarationMatches(peer, source.InlineEnumPeerSnapshot) {
		return schemaReferenceUnsupported(source.Pointer)
	}
	return nil
}

// Operation planning can add further uses after component linking, so freeze
// every original use only after both component and operation planning finish.
func (a *analyzer) finalizeCollectionReferenceChildDefaults(declarations []Declaration, client *Client) *diagnostic.Diagnostic {
	if a.collectionChildDefaultOrder == nil {
		return nil
	}
	a.collectionChildDefaultClient = client
	// Forward named collection aliases may not have backing types when their
	// physical child is first seen. Validate each collection-valued annotation
	// after linking and before freezing any uses it may discover.
	for i := 0; i < len(*a.collectionChildDefaultOrder); i++ {
		source := (*a.collectionChildDefaultOrder)[i]
		if source == nil || !source.Collection || source.CollectionValueReady {
			continue
		}
		if len(source.Uses) == 0 || source.Value == nil {
			return schemaReferenceUnsupported(source.Pointer)
		}
		target, err := a.resolveSchema(text(source.Reference), source.Pointer)
		if err != nil {
			return err
		}
		_, source.Encoded, err = a.collectionReferenceChildDefaultValue(source.Schema, source.Pointer, source.Uses[0].Depth, source.Uses[0].Type, target)
		if err != nil {
			return err
		}
		source.CollectionValueReady = true
	}
	retained := collectionChildDefaultRetainedTypes(declarations, client)
	for _, source := range *a.collectionChildDefaultOrder {
		if source == nil {
			return schemaReferenceUnsupported("")
		}
		if len(source.Uses) == 0 || source.FinalUseCount > len(source.Uses) || a.collectionChildDefaults[source.Schema] != source {
			return schemaReferenceUnsupported(source.Pointer)
		}
		if source.FinalUseCount == 0 {
			if err := a.freezeScalarFactoryCollections(source.Collections, declarations, source.Pointer); err != nil {
				return err
			}
			if source.InlineEnumCollection || source.ReferencedEnumCollection {
				if err := a.freezeEnumCollectionChild(source, declarations); err != nil {
					return err
				}
			}
			if source.InlineScalarCollection || source.InlineEnumCollection || source.ReferencedEnumCollection {
				for _, binding := range source.Collections {
					owner := binding.Owner
					if owner == nil || !has(owner.Schema, "default") {
						continue
					}
					declaration := binding.Declaration
					encoded := collectionChildScalarFactoryValue(declaration)
					if !a.tupleSourceOwnDefault(owner.Schema, owner.Pointer) || encoded == "" {
						return schemaReferenceUnsupported(source.Pointer)
					}
					value := get(owner.Schema, "default")
					if err := a.captureTupleNativePath(value, naming.Pointer(owner.Pointer, "default"), source.Pointer); err != nil {
						return err
					}
					raw, _ := value.MarshalJSON()
					if err := a.tupleNativeSources.charge(1, len(raw), source.Pointer); err != nil {
						return err
					}
					source.CollectionFactories = append(source.CollectionFactories, collectionChildScalarFactory{
						Owner: owner, Declaration: declaration, Value: value, ValueBytes: string(raw), Encoded: encoded,
					})
				}
				source.FinalCollectionFactoryCount = len(source.CollectionFactories)
			}
			for _, binding := range source.Bindings {
				owner := binding.Owner
				if owner == nil || !has(owner.Schema, "default") {
					continue
				}
				declaration := componentDeclaration(declarations, owner.Name)
				encoded := collectionChildScalarFactoryValue(declaration)
				if !a.tupleSourceOwnDefault(owner.Schema, owner.Pointer) || encoded == "" {
					return schemaReferenceUnsupported(source.Pointer)
				}
				value := get(owner.Schema, "default")
				if err := a.captureTupleNativePath(value, naming.Pointer(owner.Pointer, "default"), source.Pointer); err != nil {
					return err
				}
				raw, _ := value.MarshalJSON()
				if err := a.tupleNativeSources.charge(1, len(raw), source.Pointer); err != nil {
					return err
				}
				source.Factories = append(source.Factories, collectionChildScalarFactory{
					Owner: owner, Declaration: declaration, Value: value, ValueBytes: string(raw), Encoded: encoded,
				})
			}
			source.FinalFactoryCount = len(source.Factories)
		} else if source.FinalFactoryCount != len(source.Factories) || source.FinalCollectionFactoryCount != len(source.CollectionFactories) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, use := range source.Uses[source.FinalUseCount:] {
			if use.Actual == nil || !collectionChildDefaultTypesMatch(use.Type, *use.Actual) {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if err := a.collectionChildDefaultTarget(source, use); err != nil {
				return err
			}
			use.FinalDepth = use.Depth
			use.TypeProof = tupleCollectionUse{Source: source.Schema, Pointer: source.Pointer, Snapshots: dtoOriginalGetterTypes(&use.Type), Final: true}
			use.ActualProof = tupleCollectionUse{Source: source.Schema, Pointer: source.Pointer, Snapshots: dtoOriginalGetterTypes(use.Actual), Final: true}
			use.Retained = retained[use.Actual]
		}
		source.FinalUseCount = len(source.Uses)
	}
	return nil
}

func (a *analyzer) validateCollectionReferenceChildDefaults(declarations []Declaration) *diagnostic.Diagnostic {
	return a.validateFrozenCollectionReferenceChildDefaults(declarations, false)
}

func (a *analyzer) validateFrozenCollectionReferenceChildDefaults(declarations []Declaration, allowPending bool) *diagnostic.Diagnostic {
	if a.collectionChildDefaultOrder == nil {
		return nil
	}
	retained := collectionChildDefaultRetainedTypes(declarations, a.collectionChildDefaultClient)
	actual, names := map[Declaration]int{}, map[string]int{}
	for _, declaration := range declarations {
		actual[declaration]++
		names[declaration.Name()]++
	}
	for _, source := range *a.collectionChildDefaultOrder {
		if source == nil {
			return schemaReferenceUnsupported("")
		}
		if a.collectionChildDefaults[source.Schema] != source || source.FinalUseCount > len(source.Uses) ||
			!allowPending && (source.FinalUseCount == 0 || source.FinalUseCount != len(source.Uses)) ||
			source.FinalFactoryCount != len(source.Factories) ||
			source.FinalCollectionFactoryCount != len(source.CollectionFactories) ||
			source.Collection && !source.CollectionValueReady && source.FinalUseCount != 0 {
			return schemaReferenceUnsupported(source.Pointer)
		}
		if source.FinalUseCount == 0 {
			continue
		}
		if err := a.validateScalarFactoryCollections(source.Collections, actual, names, source.Pointer); err != nil {
			return err
		}
		if err := a.validateEnumCollectionChild(source, actual, names); err != nil {
			return err
		}
		collectionFactoryIndex := 0
		if source.InlineScalarCollection || source.InlineEnumCollection || source.ReferencedEnumCollection {
			for _, binding := range source.Collections {
				owner := binding.Owner
				if owner == nil {
					return schemaReferenceUnsupported(source.Pointer)
				}
				if !has(owner.Schema, "default") {
					continue
				}
				if collectionFactoryIndex >= len(source.CollectionFactories) {
					return schemaReferenceUnsupported(source.Pointer)
				}
				factory := source.CollectionFactories[collectionFactoryIndex]
				value := get(owner.Schema, "default")
				raw, _ := value.MarshalJSON()
				if factory.Owner != owner || factory.Declaration != binding.Declaration || factory.Value != value ||
					string(raw) != factory.ValueBytes || !a.tupleSourceOwnDefault(owner.Schema, owner.Pointer) ||
					collectionChildScalarFactoryValue(factory.Declaration) != factory.Encoded {
					return schemaReferenceUnsupported(source.Pointer)
				}
				collectionFactoryIndex++
			}
		}
		if collectionFactoryIndex != len(source.CollectionFactories) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		factoryIndex := 0
		for _, binding := range source.Bindings {
			if binding.Owner == nil {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if !has(binding.Owner.Schema, "default") {
				continue
			}
			if factoryIndex >= len(source.Factories) || source.Factories[factoryIndex].Owner != binding.Owner {
				return schemaReferenceUnsupported(source.Pointer)
			}
			factoryIndex++
		}
		if factoryIndex != len(source.Factories) {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, factory := range source.Factories {
			owner := factory.Owner
			if owner == nil || a.components[owner.Wire] != owner || !a.tupleSourceOwnDefault(owner.Schema, owner.Pointer) ||
				get(owner.Schema, "default") != factory.Value || componentDeclaration(declarations, owner.Name) != factory.Declaration ||
				collectionChildScalarFactoryValue(factory.Declaration) != factory.Encoded {
				return schemaReferenceUnsupported(source.Pointer)
			}
			count := 0
			for _, declaration := range declarations {
				if declaration.Name() == owner.Name {
					count++
				}
			}
			if count != 1 {
				return schemaReferenceUnsupported(source.Pointer)
			}
			raw, _ := factory.Value.MarshalJSON()
			if string(raw) != factory.ValueBytes {
				return schemaReferenceUnsupported(source.Pointer)
			}
		}
		for _, use := range source.Uses[:source.FinalUseCount] {
			if use == nil || use.Actual == nil || use.Retained && !retained[use.Actual] || use.Depth != use.FinalDepth || use.TypeProof.Source != source.Schema || use.TypeProof.Pointer != source.Pointer || !tupleCollectionUseMatches(&use.Type, use.TypeProof) ||
				use.ActualProof.Source != source.Schema || use.ActualProof.Pointer != source.Pointer || !tupleCollectionUseMatches(use.Actual, use.ActualProof) || !collectionChildDefaultTypesMatch(use.Type, *use.Actual) {
				return schemaReferenceUnsupported(source.Pointer)
			}
			if err := a.collectionChildDefaultTarget(source, use); err != nil {
				return err
			}
		}
	}
	return nil
}

// Scratch validation may see the same original schema, but only addresses in
// returned declarations and operations are retained for output.
func collectionChildDefaultRetainedTypes(declarations []Declaration, client *Client) map[*TypeRef]bool {
	retained := map[*TypeRef]bool{}
	var typeRef func(*TypeRef)
	typeRef = func(ty *TypeRef) {
		if ty == nil || retained[ty] {
			return
		}
		retained[ty] = true
		typeRef(ty.Element)
	}
	var declaration func(Declaration)
	declaration = func(d Declaration) {
		switch d := d.(type) {
		case *Model:
			for i := range d.Properties {
				typeRef(&d.Properties[i].Type)
			}
			for i := range d.Constants {
				typeRef(&d.Constants[i].Type)
			}
			for _, nested := range d.Nested {
				declaration(nested)
			}
		case *ValueAlias:
			typeRef(&d.Type)
		case *ObjectUnion:
			for i := range d.Common {
				typeRef(&d.Common[i].Type)
			}
			for _, variant := range d.Variants {
				declaration(variant)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				declaration(d.Object)
			}
		case *Hierarchy:
			for i := range d.Fields {
				typeRef(&d.Fields[i].Type)
			}
			if d.Leaf != nil {
				declaration(d.Leaf)
			}
		case *SSEEvents:
			for i := range d.Variants {
				typeRef(d.Variants[i].Data)
			}
		}
	}
	for _, d := range declarations {
		declaration(d)
	}
	if client == nil {
		return retained
	}
	for _, d := range client.Models {
		declaration(d)
	}
	for gi := range client.Groups {
		for oi := range client.Groups[gi].Operations {
			op := &client.Groups[gi].Operations[oi]
			typeRef(&op.Response)
			for pi := range op.Parameters {
				typeRef(&op.Parameters[pi].Type)
			}
			if op.Body != nil {
				typeRef(&op.Body.Type)
			}
			for ei := range op.Errors {
				typeRef(op.Errors[ei].Type)
			}
			if op.SSE != nil {
				declaration(op.SSE)
			}
			for _, d := range op.Models {
				declaration(d)
			}
		}
	}
	return retained
}
