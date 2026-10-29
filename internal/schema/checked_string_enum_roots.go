package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

func (a *analyzer) checkedStringEnumRoot(v *document.Value, pointer string) bool {
	return a.checkedEnumReferenceCandidate(v) && a.componentRoot(v, pointer)
}

// Original named backing slots are allocated before forward consumers. Only
// those known zero-value slots may acquire a shape before the existing linked
// source checkpoint. This root's recorded deferred name may also resolve;
// populated nodes otherwise retain identity and values.
func (a *analyzer) finalizeCheckedEnumRootGetters() *diagnostic.Diagnostic {
	backings := map[*TypeRef]bool{}
	for _, origin := range a.namedCollectionOrigins {
		backings[&origin.Owner.AliasType] = true
	}
	for _, origin := range a.borrowedCollections {
		backings[&origin.Type] = true
	}
	for _, name := range sortedComponentNames(a.components) {
		c := a.components[name]
		source := a.checkedEnumSources[c.Schema]
		if source == nil || source.Root != c {
			continue
		}
		for i := range source.Getters {
			getter := &source.Getters[i]
			if !getter.Provisional {
				continue
			}
			if getter.Model == nil || getter.Index < 0 || getter.Index >= len(getter.Model.Properties) || len(getter.Types) == 0 {
				return schemaReferenceUnsupported(c.Pointer)
			}
			property := &getter.Model.Properties[getter.Index]
			if property.Name != getter.Name || property.WireName != getter.Wire || property.Validation != getter.Validation || namedCollectionChecks(TypeRef{Validation: property.Validation}) != getter.CheckKey || getter.Types[0].Node != &property.Type {
				return schemaReferenceUnsupported(c.Pointer)
			}
			for _, snapshot := range getter.Types {
				if snapshot.Value == (TypeRef{}) && backings[snapshot.Node] {
					continue
				}
				expected := snapshot.Value
				if binding := expected.deferredNominal; binding != nil && binding.Schema == c.Schema && binding.Owner == c && binding.Pointer == c.Pointer && binding.Name == a.options.PackageName+".models."+c.Name {
					expected.Name = binding.Name
					expected.deferredNominal = nil
				}
				value := *snapshot.Node
				value.SchemaSteps = 0
				if value != expected || namedCollectionChecks(value) != snapshot.Checks {
					return schemaReferenceUnsupported(c.Pointer)
				}
			}
			getter.Types = dtoOriginalGetterTypes(&property.Type)
			getter.Provisional = false
		}
		source.GettersFinal = true
	}
	return nil
}

type checkedEnumRootFactory struct {
	Alias                  *ValueAlias
	Owner, Canonical       *component
	Schema, Value          *document.Value
	Name, Pointer, Encoded string
	Height, Nullable       int
	TypeProof              tupleCollectionUse
}

func (a *analyzer) recordCheckedEnumRootFactory(alias *ValueAlias, c *component, encoded string) *diagnostic.Diagnostic {
	target := a.referenceWrapperDefaultTarget(c)
	if target == nil || target.Internal || !a.checkedStringEnumRoot(target.Schema, target.Pointer) {
		return nil
	}
	source := a.checkedEnumSources[target.Schema]
	if source == nil || source.Root == nil || c.BindingTarget != source.Root || !a.componentRoot(c.Schema, c.Pointer) || !a.componentDefaultSchemas[c.Schema] {
		return schemaReferenceUnsupported(c.Pointer)
	}
	source.RootFactories = append(source.RootFactories, &checkedEnumRootFactory{Alias: alias, Owner: c, Canonical: source.Root, Schema: c.Schema, Value: get(c.Schema, "default"), Name: c.Name, Pointer: c.Pointer, Encoded: encoded, Height: c.AliasHeight, Nullable: falseNullableIntent(c), TypeProof: tupleCollectionUse{Source: c.Schema, Pointer: c.Pointer, Snapshots: dtoOriginalGetterTypes(&alias.Type), Final: true}})
	return nil
}

// Root vocabularies keep their open codecs. Actual typed uses borrow the
// original component check object, whose retained declaration is proved here
// even when this component currently has no consuming field.
func (a *analyzer) validateCheckedStringEnumRoots(ds []Declaration) *diagnostic.Diagnostic {
	actual := map[*StringEnum]int{}
	aliases := map[*ValueAlias]int{}
	models := map[*Model]bool{}
	var visit func(Declaration)
	visit = func(d Declaration) {
		switch x := d.(type) {
		case *Model:
			models[x] = true
			for _, child := range x.Nested {
				visit(child)
			}
		case *ObjectUnion:
			for _, child := range x.Variants {
				visit(child)
			}
		case *PrimitiveUnion:
			if x.Object != nil {
				visit(x.Object)
			}
		case *Hierarchy:
			if x.Leaf != nil {
				visit(x.Leaf)
			}
		}
	}
	for _, d := range ds {
		visit(d)
		if enum, ok := d.(*StringEnum); ok {
			actual[enum]++
		}
		if alias, ok := d.(*ValueAlias); ok {
			aliases[alias]++
		}
	}
	for _, name := range sortedComponentNames(a.components) {
		c := a.components[name]
		if !a.checkedStringEnumRoot(c.Schema, c.Pointer) {
			continue
		}
		source := a.checkedEnumSources[c.Schema]
		bytes, _ := c.Schema.MarshalJSON()
		if source == nil || source.Root != c || source.Schema != c.Schema || source.Pointer != c.Pointer || source.Bytes != string(bytes) || source.Failure != nil || source.Checks == nil || c.Validation != source.Checks || a.componentType(c).WithoutRootNull().Validation != source.Checks || a.validationSources[source.Checks] != c.Pointer || namedCollectionChecks(TypeRef{Validation: source.Checks}) != source.CheckKey {
			return schemaReferenceUnsupported(c.Pointer)
		}
		matches := 0
		for enum, count := range actual {
			origin := a.enumOrigins[enum]
			if origin.Schema != c.Schema || origin.Pointer != c.Pointer {
				continue
			}
			if count != 1 || !source.Declarations[enum] || enum.ClassName != c.Name || origin.Nullable != c.NullableEnum {
				return schemaReferenceUnsupported(c.Pointer)
			}
			expected, err := buildStringEnumKeysFor(c.Name, c.Schema, c.Pointer, false, a)
			if err != nil {
				return err
			}
			if len(enum.Members) != len(expected.Members) {
				return schemaReferenceUnsupported(c.Pointer)
			}
			for i, member := range enum.Members {
				if member != expected.Members[i] {
					return schemaReferenceUnsupported(c.Pointer)
				}
			}
			encoded := ""
			if value := get(c.Schema, "default"); value != nil {
				data, err := value.MarshalJSON()
				if err != nil {
					return schemaReferenceUnsupported(c.Pointer)
				}
				encoded = string(data)
			}
			if enum.ComponentDefault != encoded {
				return schemaReferenceUnsupported(c.Pointer)
			}
			matches++
		}
		if matches != 1 {
			return schemaReferenceUnsupported(c.Pointer)
		}
		for _, getter := range source.Getters {
			if !models[getter.Model] {
				continue
			}
			retainsChecks := getter.Validation == source.Checks
			for _, ty := range getter.Types {
				retainsChecks = retainsChecks || ty.Value.Validation == source.Checks
			}
			if getter.Provisional || !retainsChecks || !checkedEnumGetterMatches(getter) {
				return schemaReferenceUnsupported(c.Pointer)
			}
		}
		factories := map[*component]int{}
		for _, factory := range source.RootFactories {
			owner, alias := factory.Owner, factory.Alias
			if aliases[alias] != 1 || a.components[owner.Wire] != owner || owner.BindingTarget != factory.Canonical || factory.Canonical != c || owner.Schema != factory.Schema || get(owner.Schema, "default") != factory.Value || owner.Name != factory.Name || owner.Pointer != factory.Pointer || alias.ClassName != factory.Name || alias.ComponentDefault != factory.Encoded || owner.AliasHeight != factory.Height || falseNullableIntent(owner) != factory.Nullable || !tupleCollectionUseMatches(&alias.Type, factory.TypeProof) || falsePropertySignature(Property{Type: defaultBacking(alias.Type)}) != falsePropertySignature(Property{Type: defaultBacking(a.componentType(owner))}) {
				return schemaReferenceUnsupported(factory.Pointer)
			}
			factories[owner]++
		}
		for _, wire := range sortedComponentNames(a.components) {
			owner := a.components[wire]
			if owner.BindingTarget == c && has(owner.Schema, "default") && factories[owner] != 1 {
				return schemaReferenceUnsupported(owner.Pointer)
			}
		}
	}
	return nil
}
