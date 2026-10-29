package schema

import (
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Index physical ownership using the propertyType/dictionaryType declaration
// rules: containers add Inner/Value to a local name, never a namespace. Root
// collections emit top-level peers; properties emit inside their model owner.
// References add no owners/catalog entries. Final declarations confirm every
// used classifier, and the ordinary planner still validates the whole source.
func (a *analyzer) indexInternalModels(hierarchy map[string]*Hierarchy) {
	a.internalModels = map[*document.Value]string{}
	a.internalNamedModels = map[*document.Value]string{}
	// Allocate before usage-specific analyzer copies. The hierarchy-filtered
	// second index must not discard root aliases recorded during catalog binding.
	if a.internalModelUses == nil {
		a.internalModelUses = map[string]string{}
	}
	var visit func(*document.Value, string, string, string, int, bool)
	visit = func(v *document.Value, p, scope, local string, depth int, root bool) {
		if depth > MaxSchemaDepth || v.Kind() != document.Object {
			return
		}
		kind, _, err := schemaType(v, p)
		if err != nil || kind != "object" && kind != "array" || !directInternalOwner(v) {
			return
		}
		switch {
		case kind == "array":
			// Tuple projection is a separate planner, not physical DTO ownership.
			if !has(v, "prefixItems") {
				visit(get(v, "items"), naming.Pointer(p, "items"), scope, local+"Inner", depth+1, false)
			}
			return
		case kind == "object" && dictionarySchema(v):
			visit(get(v, "additionalProperties"), naming.Pointer(p, "additionalProperties"), scope, local+"Value", depth+1, false)
			return
		}
		if !directInternalModel(v, p) {
			return
		}
		name := scope + local
		if root {
			a.internalNamedModels[v] = a.options.PackageName + ".models." + name
		} else {
			a.internalModels[v] = a.options.PackageName + ".models." + name
		}
		propertiesPointer := naming.Pointer(p, "properties")
		for _, f := range get(v, "properties").Members() {
			at := naming.Pointer(propertiesPointer, f.Name)
			nested, err := naming.ClassName(f.Name, at)
			if err == nil {
				visit(f.Value, at, name+".", nested, depth+1, false)
			}
		}
	}
	for _, c := range a.components {
		if node := hierarchy[c.Wire]; node == nil {
			visit(c.Schema, c.Pointer, "", c.Name, 0, true)
		} else if node.ClassName == c.Name {
			// Only a real named hierarchy declaration owns this classifier;
			// its nested model occurrences need a separate ownership contract.
			a.internalNamedModels[c.Schema] = a.options.PackageName + ".models." + node.ClassName
		}
	}
}

func directInternalOwner(v *document.Value) bool {
	if v == nil || v.Kind() != document.Object {
		return false
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "enum", "const", "x-native-type", "x-kotlin-value-class", "discriminator"} {
		if has(v, key) {
			return false
		}
	}
	return true
}

func directInternalModel(v *document.Value, p string) bool {
	if !directInternalOwner(v) {
		return false
	}
	kind, _, err := schemaType(v, p)
	props := get(v, "properties")
	if err != nil || kind != "object" || props.Kind() != document.Object || len(props.Members()) == 0 {
		return false
	}
	for _, f := range props.Members() {
		if !has(f.Value, "const") {
			return true
		}
	}
	return false
}

// composedModelCandidate recognizes only provisional ordinary allOf syntax.
// Final retained Model occurrences, rather than these source keys, authorize it.
func composedModelCandidate(v *document.Value) bool {
	if v == nil || v.Kind() != document.Object || !has(v, "allOf") {
		return false
	}
	for _, key := range []string{"$ref", "oneOf", "anyOf", "enum", "const", "x-native-type", "x-kotlin-value-class", "discriminator"} {
		if has(v, key) {
			return false
		}
	}
	return true
}

func enumNominalCandidate(v *document.Value) bool {
	// Direct DTO collection bounds belong to that field use, not the open
	// vocabulary codec. Borrowing this leaf would erase its enforcing slot.
	if v == nil || v.Kind() != document.Object || has(v, "$ref") || hasPrimitiveBounds(v) || !lexicalStringEnumSchema(v) {
		return false
	}
	kind, _, err := schemaType(v, "")
	return err == nil && (kind == "" || kind == "string")
}

func unionNominalCandidate(v *document.Value) bool {
	if v == nil || v.Kind() != document.Object || has(v, "discriminator") || has(v, "$ref") {
		return false
	}
	return has(v, "oneOf") || has(v, "anyOf")
}

func (a *analyzer) nominalSourceNullable(target *resolvedSchema) (bool, *diagnostic.Diagnostic) {
	if target.Nullable != nil {
		return *target.Nullable, nil
	}
	if unionNominalCandidate(target.Schema) {
		if binding := a.deferredNominals[target.Schema]; binding != nil && binding.Name != "" {
			return binding.Nullable, nil
		}
		if target.Catalog != nil {
			return false, nil
		} // Standalone unions retain Null carriers.
		view, err := a.readUnion(target.Schema, target.Pointer)
		if err != nil {
			return false, err
		}
		if err := a.deriveObjectReferenceNullability(view, 0); err != nil {
			return false, err
		}
		return view.Nullable, nil // Actual retained union is rechecked at the final barrier.
	}
	if composedModelCandidate(target.Schema) {
		// A composition product is a union, not an ordinary Model terminal.
		// Retained-declaration authorization preserves its reference diagnostic.
		if a.containsObjectCompositionUnion(target.Schema, 0, map[*document.Value]bool{}) {
			if binding := a.deferredNominals[target.Schema]; binding != nil && binding.Name != "" {
				return binding.Nullable, nil
			}
			if target.Catalog != nil {
				return false, nil // Standalone products retain their Null carrier.
			}
			return a.compositionNullable(target.Schema, target.Pointer, 0, &compositionBudget{remaining: 16384, enabled: true}, map[*document.Value]bool{})
		}
		shape, err := a.objectShape(target.Schema, target.Pointer, 0)
		if err != nil {
			return false, err
		}
		return shape.Nullable != nil && *shape.Nullable, nil
	}
	_, nullable, err := schemaType(target.Schema, target.Pointer)
	return nullable, err
}

// bindInternalModel records an existing owner without adding catalog identity.
// Both direct typed references and eligible chain terminals use the same seam.
func (a *analyzer) bindInternalModel(target *resolvedSchema, source string) bool {
	name := a.internalModels[target.Schema]
	lexicalHierarchy := target.SourceCatalog != nil && target.DirectLexical && a.hierarchySources[target.SourceCatalog.Schema]
	lexicalComposed := target.SourceCatalog != nil && (target.ComposedLexical || target.DirectLexical && (unionNominalCandidate(target.Schema) || a.enumReferenceCandidate(target.Schema)))
	if lexicalHierarchy || composedModelCandidate(target.Schema) || unionNominalCandidate(target.Schema) || a.enumReferenceCandidate(target.Schema) || name == "" {
		binding := a.deferredNominals[target.Schema]
		if (!lexicalHierarchy && !lexicalComposed && (binding == nil || binding.Owner != target.SourceCatalog || !target.DirectLexical)) || !directInternalModel(target.Schema, target.Pointer) && !composedModelCandidate(target.Schema) && !unionNominalCandidate(target.Schema) && !a.enumReferenceCandidate(target.Schema) {
			return false
		}
		if a.deferredNominals == nil || a.nominalsLinked && (binding == nil || binding.Name == "") {
			return false
		}
		if binding == nil {
			binding = &deferredNominal{Schema: target.Schema, Owner: target.SourceCatalog, Source: source, Pointer: target.Pointer}
			a.deferredNominals[target.Schema] = binding
		}
		if binding.Name != "" {
			target.ModelName = binding.Name
			target.Deferred = nil
			if _, exists := a.internalModelUses[binding.Name]; !exists {
				a.internalModelUses[binding.Name] = source
			}
		} else {
			target.Deferred = binding
		}
		return true
	}
	target.ModelName = name
	if a.internalModelUses == nil {
		a.internalModelUses = map[string]string{}
	}
	if _, exists := a.internalModelUses[name]; !exists {
		a.internalModelUses[name] = source
	}
	return true
}

func (a *analyzer) physicalDTOName(v *document.Value) string {
	if name := a.internalModels[v]; name != "" {
		return name
	}
	if b := a.deferredNominals[v]; b != nil && b.Name != "" {
		return b.Name
	}
	return a.internalNamedModels[v]
}

// Named chain terminals retain actual catalog identity and may reuse existing
// aliases to a physical ordinary DTO or named hierarchy declaration. Provisional
// graph candidates are authorized again by final declaration identity.
func (a *analyzer) bindNamedChainModel(target *resolvedSchema, source string) bool {
	if target.Catalog == nil {
		return false
	}
	canonical := target.PhysicalValueClass
	if canonical == nil && target.Catalog.ValueClass {
		canonical = target.Catalog
	}
	if canonical != nil {
		if canonical.Schema != target.Schema || !canonical.ValueClass {
			return false
		}
		binding := a.deferredNominals[target.Schema]
		if a.nominalsLinked && (binding == nil || binding.Name == "") {
			return false
		}
		if binding == nil {
			binding = &deferredNominal{Schema: target.Schema, Owner: canonical, Source: source, Pointer: target.Pointer}
			a.deferredNominals[target.Schema] = binding
		}
		target.PhysicalValueClass = canonical
		if binding.Name != "" {
			target.ModelName = binding.Name
			target.Deferred = nil
			a.internalModelUses[binding.Name] = source
		} else {
			target.Deferred = binding
		}
		return true
	}
	name := a.internalNamedModels[target.Schema]
	if target.Catalog.InternalModelTarget != nil {
		if binding := target.Catalog.InternalModelTarget.Deferred; binding != nil {
			target.Deferred = binding
			return true
		}
		name = a.physicalDTOName(target.Schema)
		if name != target.ModelName {
			return false
		}
	}
	if name == "" && (composedModelCandidate(target.Schema) || unionNominalCandidate(target.Schema) || a.enumReferenceCandidate(target.Schema)) {
		binding := a.deferredNominals[target.Schema]
		if a.nominalsLinked && (binding == nil || binding.Name == "") {
			return false
		}
		if binding == nil {
			binding = &deferredNominal{Schema: target.Schema, Owner: target.Catalog, Source: source, Pointer: target.Pointer}
			a.deferredNominals[target.Schema] = binding
		}
		if binding.Name != "" {
			target.ModelName = binding.Name
			if _, exists := a.internalModelUses[binding.Name]; !exists {
				a.internalModelUses[binding.Name] = source
			}
		} else {
			target.Deferred = binding
		}
		return true
	}
	if name == "" {
		return false
	}
	if _, exists := a.internalModelUses[name]; !exists {
		a.internalModelUses[name] = source
	}
	return true
}

// Syntax is not enough to invent a classifier. Check each used owner against
// final shared declarations, after all components and operation uses validate.
func (a *analyzer) validateInternalModels(declarations []Declaration) *diagnostic.Diagnostic {
	if err := a.validateTupleInlineDTOProof(); err != nil {
		return err
	}
	if err := a.validateFalseDTOOwnerDefaults(declarations); err != nil {
		return err
	}
	if err := a.validateLiteralFalseFields(declarations); err != nil {
		return err
	}
	if err := a.validateFalseInlineCollectionBindings(declarations); err != nil {
		return err
	}
	if err := a.validateFalseDTOFieldDefaults(declarations); err != nil {
		return err
	}
	if err := a.validateFalseCollectionBindings(declarations); err != nil {
		return err
	}
	if err := a.validateFalseFieldBindings(declarations); err != nil {
		return err
	}
	if err := a.validateFalseRootBindings(declarations); err != nil {
		return err
	}
	if err := a.validateTupleNativeSources(); err != nil {
		return err
	}
	if err := a.validateNativeEnumBindings(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedDTOLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedEnumLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateCheckedStringEnumRoots(declarations); err != nil {
		return err
	}
	if err := a.validateInternalEnumFactories(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedValueClassLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedAllOfLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedUnionLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedHierarchyLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedBuiltinLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedTemporalLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedBigintLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedBytesLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedCallerNativeLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedRawTrueLeaves(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedRecursiveCoverLeaves(declarations); err != nil {
		return err
	}
	actual := map[string]bool{}
	var visit func(Declaration, string)
	visit = func(d Declaration, parent string) {
		if h, ok := d.(*Hierarchy); ok {
			actual[parent+"."+h.Name()] = true
			if h.Leaf != nil {
				visit(h.Leaf, parent)
			}
			return
		}
		if v, ok := d.(*PrimitiveValueClass); ok {
			actual[parent+"."+v.Name()] = true
			return
		}
		if e, ok := d.(*StringEnum); ok {
			actual[parent+"."+e.Name()] = true
			return
		}
		if u, ok := d.(*ObjectUnion); ok {
			actual[parent+"."+u.Name()] = true
			for _, v := range u.Variants {
				visit(v, parent+"."+u.Name())
			}
			return
		}
		if u, ok := d.(*PrimitiveUnion); ok {
			actual[parent+"."+u.Name()] = true
			if u.Object != nil {
				visit(u.Object, parent)
			}
			return
		}
		if m, ok := d.(*Model); ok {
			name := parent + "." + m.Name()
			actual[name] = len(m.Properties) > 0
			for _, nested := range m.Nested {
				visit(nested, name)
			}
		}
	}
	for _, d := range declarations {
		visit(d, a.options.PackageName+".models")
	}
	names := make([]string, 0, len(a.internalModelUses))
	for name := range a.internalModelUses {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !actual[name] {
			return schemaReferenceUnsupported(a.internalModelUses[name])
		}
	}
	if err := a.validateBorrowedCollectionBindings(declarations); err != nil {
		return err
	}
	if err := a.validateBorrowedRootBindings(declarations); err != nil {
		return err
	}
	if err := a.validateBuiltinBindings(declarations); err != nil {
		return err
	}
	if err := a.validateNamedCollectionBindings(declarations); err != nil {
		return err
	}
	if err := a.validateReferencedEnumCollectionFactories(declarations); err != nil {
		return err
	}
	if err := a.validateRecursiveCoverBindings(declarations); err != nil {
		return err
	}
	if err := a.validateRawTrueBindings(declarations); err != nil {
		return err
	}
	if err := a.validateCallerNativeBindings(declarations); err != nil {
		return err
	}
	if err := a.validateBytesBindings(declarations); err != nil {
		return err
	}
	if err := a.validateBigintBindings(declarations); err != nil {
		return err
	}
	return a.validateTemporalBindings(declarations)
}
