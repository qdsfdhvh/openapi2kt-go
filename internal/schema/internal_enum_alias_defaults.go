package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type internalEnumFactoryRegistry struct {
	Factories map[*ValueAlias]*internalEnumFactory
	Order     []*internalEnumFactory
	Terminals map[*document.Value]bool
}

type internalEnumFactory struct {
	Alias                          *ValueAlias
	Owner                          *component
	Schema, Terminal, Value        *document.Value
	Pointer, Target, Name, Encoded string
	Height                         int
	Nullable                       int
	Binding                        *resolvedSchema
	BindingValue                   resolvedSchema
	TypeProof                      tupleCollectionUse
}

// Candidacy runs before the real catalog exists. Reuse the schema-path lookup
// with one original owner carrier; it grants no typed or declaration authority.
// The actual resolver/retained enum is checked again before a factory can emit.
func (a *analyzer) internalEnumFactoryLookup(reference string) (string, *document.Value, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(reference, prefix) {
		return "", nil, false
	}
	raw := strings.TrimPrefix(reference, prefix)
	slash := strings.IndexByte(raw, '/')
	if slash < 1 {
		return "", nil, false
	}
	wire, err := naming.DecodePointerSegment(raw[:slash], "")
	if err != nil {
		return "", nil, false
	}
	v := get(a.componentSchemas, wire)
	if v == nil {
		return "", nil, false
	}
	owner := &component{Wire: wire, Pointer: naming.Pointer("/components/schemas", wire), Schema: v}
	lookup := analyzer{components: map[string]*component{wire: owner}}
	target, failure := lookup.lookupSchemaReference(reference, "")
	if failure != nil || target.Catalog != nil || target.SourceCatalog != owner {
		return "", nil, false
	}
	return target.Pointer, target.Schema, true
}

func (a *analyzer) initInternalEnumFactories() {
	if a.internalEnumFactories == nil {
		a.internalEnumFactories = &internalEnumFactoryRegistry{Factories: map[*ValueAlias]*internalEnumFactory{}, Terminals: map[*document.Value]bool{}}
	}
}

func (a *analyzer) independentEnumComponentDefault(v *document.Value, pointer string) bool {
	if a.componentDefaultSchemas[v] && a.componentRoot(v, pointer) && originalEnumWrapperDefaultSource(v) {
		return true
	}
	if a.componentDefaultSchemas[v] && a.checkedStringEnumRoot(v, pointer) {
		return true
	}
	if a.internalEnumFactories != nil && a.internalEnumFactories.Terminals[v] {
		_, original, ok := a.internalEnumFactoryLookup("#" + pointer)
		return ok && original == v
	}
	if a.metadataEnumComponentDefault(v, pointer) {
		return true
	}
	if !a.componentDefaultSchemas[v] || !a.componentRoot(v, pointer) || !has(v, "$ref") {
		return false
	}
	target := a.referenceWrapperDefaultTarget(&component{Schema: v, Pointer: pointer})
	return target != nil && (target.Internal && a.enumReferenceCandidate(target.Schema) || !target.Internal && originalEnumWrapperDefaultSource(target.Schema) && a.componentRoot(target.Schema, target.Pointer))
}

func (a *analyzer) admitInternalEnumFactory(c *component, source *wrapperDefaultSource) *diagnostic.Diagnostic {
	if a.components[c.Wire] != c || !a.componentRoot(c.Schema, c.Pointer) || !a.componentDefaultSchemas[c.Schema] || !source.Internal {
		return schemaReferenceUnsupported(c.Pointer)
	}
	if a.internalEnumCollectionDefaultSource(source.Schema, source.Pointer) {
		return a.admitInternalEnumCollectionFactory(c, source)
	}
	if !a.enumReferenceCandidate(source.Schema) {
		return schemaReferenceUnsupported(c.Pointer)
	}
	target, err := a.resolveSchema(text(get(c.Schema, "$ref")), c.Pointer)
	if err != nil {
		return err
	}
	if target.Schema != source.Schema || target.Pointer != source.Pointer || !a.admitBorrowedNominalLeaf(target, c.Pointer) {
		return schemaReferenceUnsupported(c.Pointer)
	}
	a.initInternalEnumFactories()
	if has(source.Schema, "default") {
		a.internalEnumFactories.Terminals[source.Schema] = true
	}
	return a.captureTupleNativeSource(c.Schema, c.Pointer, source.Schema, source.Pointer)
}

func (a *analyzer) recordInternalEnumFactory(alias *ValueAlias, c *component, encoded string) *diagnostic.Diagnostic {
	source := a.referenceWrapperDefaultTarget(c)
	if source == nil || !source.Internal {
		return nil
	}
	if err := a.admitInternalEnumFactory(c, source); err != nil {
		return err
	}
	a.initInternalEnumFactories()
	if c.InternalModelTarget == nil || c.InternalModelTarget.Schema != source.Schema || c.InternalModelTarget.Pointer != source.Pointer {
		return schemaReferenceUnsupported(c.Pointer)
	}
	if err := a.tupleNativeSources.charge(1, 0, c.Pointer); err != nil {
		return err
	}
	proof := &internalEnumFactory{Alias: alias, Owner: c, Schema: c.Schema, Terminal: source.Schema, Value: get(c.Schema, "default"), Pointer: c.Pointer, Target: source.Pointer, Name: c.Name, Encoded: encoded, Height: c.AliasHeight, Nullable: falseNullableIntent(c), Binding: c.InternalModelTarget, BindingValue: *c.InternalModelTarget}
	a.internalEnumFactories.Factories[alias] = proof
	a.internalEnumFactories.Order = append(a.internalEnumFactories.Order, proof)
	return nil
}

func (a *analyzer) finalizeInternalEnumFactories() *diagnostic.Diagnostic {
	if a.internalEnumFactories == nil {
		return nil
	}
	for _, factory := range a.internalEnumFactories.Order {
		expected := factory.BindingValue
		if expected.Deferred != nil && expected.Deferred.Name != "" {
			expected.ModelName = expected.Deferred.Name
			expected.Deferred = nil
		}
		if factory.Owner.InternalModelTarget != factory.Binding || *factory.Binding != expected {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		factory.BindingValue = expected
		factory.TypeProof = tupleCollectionUse{Source: factory.Schema, Pointer: factory.Pointer, Snapshots: dtoOriginalGetterTypes(&factory.Alias.Type), Final: true}
	}
	return nil
}

func (a *analyzer) validateInternalEnumFactories(ds []Declaration) *diagnostic.Diagnostic {
	if a.internalEnumFactories == nil {
		return nil
	}
	counts := map[*ValueAlias]int{}
	terminals := map[*document.Value]bool{}
	for _, d := range ds {
		if alias, ok := d.(*ValueAlias); ok {
			counts[alias]++
		}
	}
	for _, factory := range a.internalEnumFactories.Order {
		c, alias := factory.Owner, factory.Alias
		if a.internalEnumFactories.Factories[alias] != factory || counts[alias] != 1 || a.components[c.Wire] != c || c.Schema != factory.Schema || c.Pointer != factory.Pointer || c.Name != factory.Name || alias.ClassName != factory.Name || get(c.Schema, "default") != factory.Value || alias.ComponentDefault != factory.Encoded || c.AliasHeight != factory.Height || falseNullableIntent(c) != factory.Nullable || c.InternalModelTarget != factory.Binding || *factory.Binding != factory.BindingValue || !tupleCollectionUseMatches(&alias.Type, factory.TypeProof) {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		source := a.referenceWrapperDefaultTarget(c)
		if source == nil || !source.Internal || source.Schema != factory.Terminal || source.Pointer != factory.Target || falsePropertySignature(Property{Type: defaultBacking(alias.Type)}) != falsePropertySignature(Property{Type: defaultBacking(a.componentType(c))}) {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		terminals[factory.Terminal] = true
	}
	for terminal := range a.internalEnumFactories.Terminals {
		if !terminals[terminal] {
			return schemaReferenceUnsupported("/components/schemas")
		}
	}
	return nil
}
