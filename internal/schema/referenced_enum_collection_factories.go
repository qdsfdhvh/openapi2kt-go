package schema

import (
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type referencedEnumCollectionSource struct {
	Schema  *document.Value
	Pointer string
}

// Initial annotation marking reads only original schema paths. Actual typed
// reference/declaration admission still belongs to the shared planner below.
func (a *analyzer) referencedEnumCollectionSource(v *document.Value, pointer string) *referencedEnumCollectionSource {
	seen := map[*document.Value]bool{}
	edges, references := 0, 0
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if v == nil || v.Kind() != document.Object || seen[v] {
			return nil
		}
		seen[v] = true
		if has(v, "$ref") {
			ref := text(get(v, "$ref"))
			p, source, ok := a.componentSchemaAtReference(ref)
			if !ok {
				p, source, ok = a.internalEnumFactoryLookup(ref)
			}
			if !ok {
				return nil
			}
			v, pointer = source, p
			if edges > 0 {
				references++
			}
			continue
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return nil
		}
		if has(v, "enum") {
			if edges == 0 || references == 0 {
				return nil
			}
			if (kind == "" || kind == "string") && lexicalStringEnumSchema(v) || kind == "integer" || kind == "number" || kind == "boolean" {
				return &referencedEnumCollectionSource{v, pointer}
			}
			return nil
		}
		if kind == "array" && !has(v, "prefixItems") {
			v, pointer = get(v, "items"), naming.Pointer(pointer, "items")
		} else if dictionarySchema(v) {
			v, pointer = get(v, "additionalProperties"), naming.Pointer(pointer, "additionalProperties")
		} else {
			return nil
		}
		edges++
	}
	return nil
}

func (a *analyzer) independentReferencedEnumCollectionDefault(v *document.Value, pointer string) bool {
	return a.componentDefaultSchemas[v] && a.componentRoot(v, pointer) && a.referencedEnumCollectionSource(v, pointer) != nil
}

func (a *analyzer) referencedEnumCollectionFactoryNode(c *component) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	source := a.referencedEnumCollectionSource(c.Schema, c.Pointer)
	if source == nil {
		return nil, schemaReferenceUnsupported(c.Pointer)
	}
	root, err := a.lookupSchemaReference("#"+c.Pointer, c.Pointer)
	if err != nil || root.Catalog == nil || root.Schema != c.Schema || root.Pointer != c.Pointer || !namedCollectionCandidate(root.Catalog) {
		return nil, schemaReferenceUnsupported(c.Pointer)
	}
	target, err := a.lookupSchemaReference("#"+source.Pointer, c.Pointer)
	if err != nil {
		return nil, err
	}
	if !a.admitBorrowedNominalLeaf(target, c.Pointer) && a.uncheckedReferencedEnumFactoryLeaf(target) == nil {
		return nil, schemaReferenceUnsupported(c.Pointer)
	}
	if err := a.captureTupleNativeSource(c.Schema, c.Pointer, c.Schema, c.Pointer); err != nil {
		return nil, err
	}
	if err := a.captureTupleNativeSource(source.Schema, source.Pointer, source.Schema, source.Pointer); err != nil {
		return nil, err
	}
	a.namedCollectionUses[root.Catalog] = c.Pointer
	return a.referencedDefaultSchemaRole(c.Schema, c.Pointer, a.componentType(root.Catalog), 0, false, referencedCollectionOwnDefault)
}

type referencedEnumCollectionFactory struct {
	Alias                  *ValueAlias
	Owner, Canonical       *component
	Schema, Value          *document.Value
	Name, Pointer, Encoded string
	Height, Nullable       int
	TypeProof              tupleCollectionUse
	Native                 *nativeEnumOrigin
	Leaf                   referencedEnumCollectionSource
}

// Ordinary named native enums already have scalar exports without validation.
// Borrowed/internal native admission intentionally keeps its validation gate;
// a factory on this existing named collection needs the actual scalar export
// instead, which is retained and checked below.
func (a *analyzer) uncheckedReferencedEnumFactoryLeaf(target *resolvedSchema) *component {
	c := target.Catalog
	if a.options.Validation || c == nil || a.components[c.Wire] != c || c.Schema != target.Schema || c.Pointer != target.Pointer || c.Reference != "" || c.ValueClass || c.Validation != nil || !originalEnumWrapperDefaultSource(c.Schema) {
		return nil
	}
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable || kind != "integer" && kind != "number" && kind != "boolean" || has(c.Schema, "const") || has(c.Schema, "x-native-type") || has(c.Schema, "x-kotlin-value-class") || hasPrimitiveBounds(c.Schema) {
		return nil
	}
	scalar, err := scalarMapping(kind, c.Schema, c.Pointer)
	if err != nil || c.Scalar != scalar {
		return nil
	}
	return c
}

func (a *analyzer) recordReferencedEnumCollectionFactory(alias *ValueAlias, c *component, encoded string, declarations []Declaration) *diagnostic.Diagnostic {
	source := a.referencedEnumCollectionSource(c.Schema, c.Pointer)
	if source == nil {
		return nil
	}
	canonical := c
	if c.Reference != "" {
		canonical = c.BindingTarget
	}
	if canonical == nil || !namedCollectionCandidate(canonical) || !a.componentRoot(c.Schema, c.Pointer) || !a.componentDefaultSchemas[c.Schema] {
		return schemaReferenceUnsupported(c.Pointer)
	}
	factory := &referencedEnumCollectionFactory{Alias: alias, Owner: c, Canonical: canonical, Schema: c.Schema, Value: get(c.Schema, "default"), Name: c.Name, Pointer: c.Pointer, Encoded: encoded, Height: c.AliasHeight, Nullable: falseNullableIntent(c), Leaf: *source}
	target, err := a.lookupSchemaReference("#"+source.Pointer, c.Pointer)
	if err != nil {
		return err
	}
	if native := a.uncheckedReferencedEnumFactoryLeaf(target); native != nil {
		bytes, _ := native.Schema.MarshalJSON()
		origin := &nativeEnumOrigin{Owner: native, Schema: native.Schema, Pointer: native.Pointer, Bytes: string(bytes), Scalar: native.Scalar, Vocabulary: get(native.Schema, "enum"), Members: append([]*document.Value{}, get(native.Schema, "enum").Items()...), SchemaMembers: append([]document.Member{}, native.Schema.Members()...), Actual: componentDeclaration(declarations, native.Name)}
		if !nativeEnumDeclaration(origin.Actual, origin) {
			return schemaReferenceUnsupported(c.Pointer)
		}
		factory.Native = origin
	}
	a.referencedEnumFactories = append(a.referencedEnumFactories, factory)
	return nil
}

func (a *analyzer) finalizeReferencedEnumCollectionFactories() {
	for _, factory := range a.referencedEnumFactories {
		factory.TypeProof = tupleCollectionUse{Source: factory.Schema, Pointer: factory.Pointer, Snapshots: dtoOriginalGetterTypes(&factory.Alias.Type), Final: true}
	}
}

func (a *analyzer) validateReferencedEnumCollectionFactories(ds []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	names := map[string]int{}
	owners := map[*component]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	for _, factory := range a.referencedEnumFactories {
		c, alias := factory.Owner, factory.Alias
		if actual[alias] != 1 || a.components[c.Wire] != c || c.Schema != factory.Schema || get(c.Schema, "default") != factory.Value || c.Name != factory.Name || c.Pointer != factory.Pointer || alias.ClassName != factory.Name || alias.ComponentDefault != factory.Encoded || c.AliasHeight != factory.Height || falseNullableIntent(c) != factory.Nullable || c != factory.Canonical && c.BindingTarget != factory.Canonical || !tupleCollectionUseMatches(&alias.Type, factory.TypeProof) {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		source := a.referencedEnumCollectionSource(c.Schema, c.Pointer)
		if source == nil || *source != factory.Leaf {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		target, err := a.lookupSchemaReference("#"+source.Pointer, c.Pointer)
		if err != nil {
			return err
		}
		native := a.uncheckedReferencedEnumFactoryLeaf(target)
		if (native == nil) != (factory.Native == nil) || native != nil && factory.Native.Owner != native {
			return schemaReferenceUnsupported(factory.Pointer)
		}
		if origin := factory.Native; origin != nil {
			native := origin.Owner
			if a.uncheckedReferencedEnumFactoryLeaf(&resolvedSchema{Catalog: native, Schema: origin.Schema, Pointer: origin.Pointer}) != native || native.Scalar != origin.Scalar || actual[origin.Actual] != 1 || names[native.Name] != 1 || !nativeEnumDeclaration(origin.Actual, origin) || get(native.Schema, "enum") != origin.Vocabulary || !slices.Equal(origin.Members, origin.Vocabulary.Items()) || !slices.Equal(origin.SchemaMembers, native.Schema.Members()) {
				return schemaReferenceUnsupported(factory.Pointer)
			}
			bytes, _ := native.Schema.MarshalJSON()
			if string(bytes) != origin.Bytes {
				return schemaReferenceUnsupported(factory.Pointer)
			}
		}
		owners[c]++
	}
	for _, wire := range sortedComponentNames(a.components) {
		c := a.components[wire]
		if a.independentReferencedEnumCollectionDefault(c.Schema, c.Pointer) && owners[c] != 1 {
			return schemaReferenceUnsupported(c.Pointer)
		}
	}
	return nil
}
