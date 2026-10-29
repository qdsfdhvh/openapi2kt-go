package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Candidacy reads original paths before catalog planning. The actual resolver,
// collection declaration and scalar backing still decide whether they are
// admissible; this probe never manufactures a typed reference.
func (a *analyzer) referencedScalarCollectionSource(v *document.Value, pointer string) bool {
	admitted, _ := a.referencedScalarCollectionPath(v, pointer)
	return admitted
}

func (a *analyzer) referencedScalarCollectionPath(v *document.Value, pointer string) (bool, bool) {
	seen := map[*document.Value]bool{}
	edges := 0
	physicalReference := false
	internalCollectionReference := false
	intermediateCollection := false
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || seen[v] {
			return false, false
		}
		seen[v] = true
		if has(v, "$ref") {
			if edges > 0 && a.collectionReferenceChildParent(v, pointer) != nil {
				physicalReference = true
			}
			ref := text(get(v, "$ref"))
			p, source, ok := a.componentSchemaAtReference(ref)
			if !ok {
				p, source, ok = a.internalEnumFactoryLookup(ref)
				if edges > 0 && ok {
					internalCollectionReference = true
				}
			}
			if !ok {
				return false, false
			}
			v, pointer = source, p
			continue
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return false, false
		}
		edge := ""
		if kind == "array" && !has(v, "prefixItems") {
			edge = "items"
		} else if dictionarySchema(v) {
			edge = "additionalProperties"
		}
		if edge == "" {
			return edges > 0 && physicalReference && !has(v, "enum") && collectionReferenceChildDefaultSource(v), intermediateCollection
		}
		// The newly admitted inter-container edges use actual named collection
		// declarations. Internal collection paths retain their existing roles.
		if internalCollectionReference {
			return false, false
		}
		intermediateCollection = intermediateCollection || physicalReference
		edges++
		physicalReference = false
		internalCollectionReference = false
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return false, false
}

func (a *analyzer) independentScalarCollectionComponentDefault(v *document.Value, pointer string) bool {
	return a.tupleSourceOwnDefault(v, pointer) && a.referencedScalarCollectionSource(v, pointer)
}

// Only an actual inline container ending at an independently admitted scalar
// factory needs this extra field/object role. Inter-container references keep
// their existing planner and source boundaries.
func (a *analyzer) inlineScalarFactoryReference(v *document.Value, pointer string) bool {
	if !a.referencedScalarCollectionSource(v, pointer) {
		return false
	}
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if has(v, "$ref") {
			return a.collectionReferenceChildParent(v, pointer) != nil && a.scalarReferenceHasIndependentFactory(v, pointer)
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return false
		}
		edge := ""
		if kind == "array" && !has(v, "prefixItems") {
			edge = "items"
		} else if dictionarySchema(v) {
			edge = "additionalProperties"
		}
		if edge == "" {
			return false
		}
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return false
}

func (a *analyzer) referencedScalarCollectionFactoryNode(c *component) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if !a.referencedScalarCollectionSource(c.Schema, c.Pointer) {
		return nil, schemaReferenceUnsupported(c.Pointer)
	}
	root, err := a.lookupSchemaReference("#"+c.Pointer, c.Pointer)
	if err != nil || root.Catalog == nil || root.Schema != c.Schema || root.Pointer != c.Pointer || !namedCollectionCandidate(root.Catalog) {
		return nil, schemaReferenceUnsupported(c.Pointer)
	}
	a.namedCollectionUses[root.Catalog] = c.Pointer
	validation := *a
	validation.scalarCollectionFactoryPlanning = true
	return validation.referencedDefaultSchemaRole(c.Schema, c.Pointer, a.componentType(root.Catalog), 0, false, referencedCollectionOwnDefault)
}

type referencedScalarCollectionFactory struct {
	Alias                  *ValueAlias
	Owner, Canonical       *component
	Schema, Value          *document.Value
	Name, Pointer, Encoded string
	ValueBytes             string
	Height, Nullable       int
	TypeProof              tupleCollectionUse
	Collections            []*scalarFactoryCollectionBinding
}

// Keep the physical source chain distinct from the equivalent rendered type.
// Nullable and cached heights are frozen after forward aliases have linked.
type scalarFactoryCollectionBinding struct {
	Owner                        *component
	Schema                       *document.Value
	Pointer, Name, Reference     string
	Target                       *component
	Nullable, Height, TreeHeight int
	Declaration                  *ValueAlias
	TypeProof                    tupleCollectionUse
}

func (a *analyzer) captureScalarFactoryCollections(c *component) ([]*scalarFactoryCollectionBinding, *diagnostic.Diagnostic) {
	var result, pending []*scalarFactoryCollectionBinding
	seen := map[*document.Value]bool{}
	owners := map[*component]bool{}
	v, pointer := c.Schema, c.Pointer
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || seen[v] {
			return nil, schemaReferenceUnsupported(c.Pointer)
		}
		seen[v] = true
		if err := a.captureTupleNativePath(v, pointer, c.Pointer); err != nil {
			return nil, err
		}
		owner := a.componentsByOriginalSchema(v)
		if has(v, "$ref") {
			if owner != nil {
				pending = append(pending, &scalarFactoryCollectionBinding{Owner: owner, Schema: v, Pointer: pointer, Name: owner.Name, Reference: owner.Reference})
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
			return result, nil
		}
		if err := a.captureTupleNativeSource(v, pointer, v, pointer); err != nil {
			return nil, err
		}
		if owner != nil {
			pending = append(pending, &scalarFactoryCollectionBinding{Owner: owner, Schema: v, Pointer: pointer, Name: owner.Name, Reference: owner.Reference})
		}
		for _, binding := range pending {
			if owners[binding.Owner] {
				continue
			}
			if err := a.tupleNativeSources.charge(1, 0, c.Pointer); err != nil {
				return nil, err
			}
			owners[binding.Owner] = true
			result = append(result, binding)
			if namedCollectionCandidate(binding.Owner) {
				if old, ok := a.namedCollectionUses[binding.Owner]; !ok || c.Pointer < old {
					a.namedCollectionUses[binding.Owner] = c.Pointer
				}
			}
		}
		pending = nil
		v, pointer = get(v, edge), naming.Pointer(pointer, edge)
	}
	return nil, schemaReferenceUnsupported(c.Pointer)
}

func (a *analyzer) recordReferencedScalarCollectionFactory(alias *ValueAlias, c *component, encoded string) *diagnostic.Diagnostic {
	if !a.referencedScalarCollectionSource(c.Schema, c.Pointer) {
		return nil
	}
	canonical := c
	if c.Reference != "" {
		canonical = c.BindingTarget
	}
	if canonical == nil || !namedCollectionCandidate(canonical) || !a.componentRoot(c.Schema, c.Pointer) || !a.componentDefaultSchemas[c.Schema] {
		return schemaReferenceUnsupported(c.Pointer)
	}
	if err := a.captureTupleNativeSource(c.Schema, c.Pointer, c.Schema, c.Pointer); err != nil {
		return err
	}
	var collections []*scalarFactoryCollectionBinding
	// Previously admitted inline trees already have root and scalar-leaf
	// authority. Capture the extra intermediate declarations only for the
	// new inter-container role, preserving their existing source budget.
	if _, intermediate := a.referencedScalarCollectionPath(c.Schema, c.Pointer); intermediate {
		var err *diagnostic.Diagnostic
		collections, err = a.captureScalarFactoryCollections(c)
		if err != nil {
			return err
		}
	}
	value := get(c.Schema, "default")
	if err := a.captureTupleNativePath(value, naming.Pointer(c.Pointer, "default"), c.Pointer); err != nil {
		return err
	}
	raw, _ := value.MarshalJSON()
	if err := a.tupleNativeSources.charge(1, len(raw), c.Pointer); err != nil {
		return err
	}
	a.referencedScalarFactories = append(a.referencedScalarFactories, &referencedScalarCollectionFactory{
		Alias: alias, Owner: c, Canonical: canonical, Schema: c.Schema, Value: value,
		Name: c.Name, Pointer: c.Pointer, Encoded: encoded, ValueBytes: string(raw),
		Height: c.AliasHeight, Nullable: falseNullableIntent(c), Collections: collections,
	})
	return nil
}

func (a *analyzer) freezeScalarFactoryCollections(bindings []*scalarFactoryCollectionBinding, declarations []Declaration, pointer string) *diagnostic.Diagnostic {
	for _, binding := range bindings {
		c := binding.Owner
		if c == nil {
			return schemaReferenceUnsupported(pointer)
		}
		alias, ok := componentDeclaration(declarations, binding.Name).(*ValueAlias)
		if !ok || a.components[c.Wire] != c || c.Schema != binding.Schema || c.Pointer != binding.Pointer || c.Name != binding.Name || c.Reference != binding.Reference {
			return schemaReferenceUnsupported(pointer)
		}
		binding.Target, binding.Nullable = c.BindingTarget, falseNullableIntent(c)
		binding.Height, binding.TreeHeight, binding.Declaration = c.AliasHeight, c.CollectionHeight, alias
		binding.TypeProof = tupleCollectionUse{Source: binding.Schema, Pointer: binding.Pointer, Snapshots: dtoOriginalGetterTypes(&alias.Type), Final: true}
	}
	return nil
}

func (a *analyzer) validateScalarFactoryCollections(bindings []*scalarFactoryCollectionBinding, actual map[Declaration]int, names map[string]int, pointer string) *diagnostic.Diagnostic {
	for _, binding := range bindings {
		c, alias := binding.Owner, binding.Declaration
		if c == nil || alias == nil || a.components[c.Wire] != c || c.Schema != binding.Schema ||
			c.Pointer != binding.Pointer || c.Name != binding.Name || c.Reference != binding.Reference ||
			c.BindingTarget != binding.Target || falseNullableIntent(c) != binding.Nullable ||
			c.AliasHeight != binding.Height || c.CollectionHeight != binding.TreeHeight ||
			actual[alias] != 1 || names[binding.Name] != 1 || alias.ClassName != binding.Name ||
			!tupleCollectionUseMatches(&alias.Type, binding.TypeProof) {
			return schemaReferenceUnsupported(pointer)
		}
	}
	return nil
}

func (a *analyzer) finalizeReferencedScalarCollectionFactories(declarations []Declaration) *diagnostic.Diagnostic {
	for _, factory := range a.referencedScalarFactories {
		factory.TypeProof = tupleCollectionUse{Source: factory.Schema, Pointer: factory.Pointer, Snapshots: dtoOriginalGetterTypes(&factory.Alias.Type), Final: true}
		if err := a.freezeScalarFactoryCollections(factory.Collections, declarations, factory.Pointer); err != nil {
			return err
		}
	}
	return nil
}

func (a *analyzer) validateReferencedScalarCollectionFactories(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, declaration := range declarations {
		actual[declaration]++
		names[declaration.Name()]++
	}
	for _, factory := range a.referencedScalarFactories {
		c, alias := factory.Owner, factory.Alias
		if c == nil || alias == nil || actual[alias] != 1 || names[factory.Name] != 1 ||
			a.components[c.Wire] != c || c.Schema != factory.Schema || get(c.Schema, "default") != factory.Value ||
			c.Name != factory.Name || c.Pointer != factory.Pointer || alias.ClassName != factory.Name ||
			alias.ComponentDefault != factory.Encoded || c.AliasHeight != factory.Height ||
			falseNullableIntent(c) != factory.Nullable || c != factory.Canonical && c.BindingTarget != factory.Canonical ||
			!a.independentScalarCollectionComponentDefault(c.Schema, c.Pointer) ||
			!tupleCollectionUseMatches(&alias.Type, factory.TypeProof) {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		raw, _ := factory.Value.MarshalJSON()
		if string(raw) != factory.ValueBytes {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		if err := a.validateScalarFactoryCollections(factory.Collections, actual, names, factory.Pointer); err != nil {
			return err
		}
	}
	return nil
}
