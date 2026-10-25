package schema

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type dtoUniqueReferenceTree struct {
	Schema *document.Value
	Bytes  string
	Nodes  []borrowedAllOfNode
}
type dtoUniqueReferenceOwner struct {
	OriginalType                           TypeRef
	Owner                                  *component
	Schema                                 *document.Value
	Wire, Name, Pointer, Reference, Scalar string
	Validation                             *PrimitiveValidation
	BindingTarget, ScalarTarget            *component
	InternalTarget                         *resolvedSchema
	Height                                 int
	HasNullable, Nullable                  bool
	Type                                   string
	Actual                                 Declaration
}
type dtoUniqueScalarReference struct {
	CollectionType TypeRef
	Borrowed       []*borrowedCollectionSource
	Collection     bool
	EnumSchema     *document.Value
	EnumPointer    string
	EnumActual     *StringEnum
	Trees          []*dtoUniqueReferenceTree
	Path           []borrowedAllOfNode
	Owners         []*dtoUniqueReferenceOwner
	Type           string
	Initializer    dtoUniqueContainerInitializer
}

func dtoUniquePhysicalScalarTypeSignature(t TypeRef) string {
	t.SchemaSteps = 0
	if t.Element != nil {
		copy := *t.Element
		copy.SchemaSteps = 0
		t.Element = &copy
	}
	encoded, _ := json.Marshal(t)
	return string(encoded)
}
func dtoUniqueScalarTypeSignature(t TypeRef) string {
	// Deferred enum names are bound by the original-enum/physical-name barrier.
	if t.Kind == Named && t.Element == nil {
		t.Name = dtoUniqueEnumKind
	}
	if t.Kind == Nullable && t.Element != nil {
		copy := *t.Element
		if copy.Kind == Named && copy.Element == nil {
			copy.Name = dtoUniqueEnumKind
		}
		t.Element = &copy
	}
	return dtoUniquePhysicalScalarTypeSignature(t)
}

func (b *dtoUniqueCaptureBudget) referenceNode(v *document.Value, p string) (borrowedAllOfNode, *diagnostic.Diagnostic) {
	if old, ok := b.ReferenceSnapshots[v]; ok {
		return old, nil
	}
	node := borrowedAllOfNode{v, append([]document.Member{}, v.Members()...), append([]*document.Value{}, v.Items()...)}
	if b.ReferenceNodes == nil {
		b.ReferenceNodes = map[*document.Value]bool{}
	}
	if !b.ReferenceNodes[v] {
		if err := b.charge(1+len(node.Members)+len(node.Items), 0, p); err != nil {
			return borrowedAllOfNode{}, err
		}
		b.ReferenceNodes[v] = true
	}
	if b.ReferenceSnapshots == nil {
		b.ReferenceSnapshots = map[*document.Value]borrowedAllOfNode{}
	}
	b.ReferenceSnapshots[v] = node
	return node, nil
}

func (b *dtoUniqueCaptureBudget) referenceTree(v *document.Value, p string) (*dtoUniqueReferenceTree, *diagnostic.Diagnostic) {
	if b.ReferenceTrees == nil {
		b.ReferenceTrees = map[*document.Value]*dtoUniqueReferenceTree{}
	}
	if old := b.ReferenceTrees[v]; old != nil {
		return old, nil
	}
	raw, _ := v.MarshalJSON()
	if err := b.charge(0, len(raw), p); err != nil {
		return nil, err
	}
	tree := &dtoUniqueReferenceTree{Schema: v, Bytes: string(raw)}
	seen := map[*document.Value]bool{}
	var walk func(*document.Value, int) *diagnostic.Diagnostic
	walk = func(v *document.Value, depth int) *diagnostic.Diagnostic {
		if err := depthCheck(depth, p); err != nil {
			return err
		}
		if seen[v] {
			return nil
		}
		seen[v] = true
		node, err := b.referenceNode(v, p)
		if err != nil {
			return err
		}
		tree.Nodes = append(tree.Nodes, node)
		for _, m := range v.Members() {
			if !slices.Contains(metadata, m.Name) {
				if err := walk(m.Value, depth+1); err != nil {
					return err
				}
			}
		}
		for _, item := range v.Items() {
			if err := walk(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(v, 0); err != nil {
		return nil, err
	}
	b.ReferenceTrees[v] = tree
	return tree, nil
}

// Keep the physical path through the original document as well as catalog
// bindings. A cached lookup cannot authorize a same-byte replacement ancestor.
func (a *analyzer) dtoUniqueReferencePath(reference, source string) (*document.Value, *component, []borrowedAllOfNode, *diagnostic.Diagnostic) {
	if !strings.HasPrefix(reference, "#/components/schemas/") {
		return nil, nil, nil, schemaReferenceUnsupported(source)
	}
	raw := strings.Split(strings.TrimPrefix(reference, "#/"), "/")
	keys := make([]string, len(raw))
	for i, key := range raw {
		decoded, err := naming.DecodePointerSegment(key, source)
		if err != nil {
			return nil, nil, nil, err
		}
		keys[i] = decoded
	}
	if len(keys) < 3 {
		return nil, nil, nil, schemaReferenceUnsupported(source)
	}
	owner := a.components[keys[2]]
	if owner == nil {
		return nil, nil, nil, schemaReferenceUnsupported(source)
	}
	v := a.root
	path := []borrowedAllOfNode{}
	for _, key := range keys {
		if v == nil {
			return nil, nil, nil, schemaReferenceUnsupported(source)
		}
		if err := a.dtoUniqueCaptures.charge(1, 0, source); err != nil {
			return nil, nil, nil, err
		}
		node, err := a.dtoUniqueCaptures.referenceNode(v, source)
		if err != nil {
			return nil, nil, nil, err
		}
		path = append(path, node)
		if v.Kind() == document.Array {
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(v.Items()) {
				return nil, nil, nil, schemaReferenceUnsupported(source)
			}
			v = v.Items()[i]
		} else {
			v = get(v, key)
		}
	}
	if v == nil {
		return nil, nil, nil, schemaReferenceUnsupported(source)
	}
	return v, owner, path, nil
}

func (a *analyzer) dtoUniqueScalarReference(f field, annotation, actual TypeRef, required bool, source string) (*dtoUniqueScalarReference, *diagnostic.Diagnostic) {
	collection := a.dtoUniqueBuiltinCollectionReferenceSource(f.Schema, f.Pointer)
	if !uniqueScalarType(annotation) && !uniqueStringEnumType(annotation) && !collection {
		return nil, unsupported(source)
	}
	proof := &dtoUniqueScalarReference{Collection: collection, Type: dtoUniqueScalarTypeSignature(actual), Initializer: dtoUniqueContainerInitializer{DefaultNull: actual.Kind == Nullable}}
	if collection {
		proof.CollectionType = actual
		mapped := annotation
		if actual.Kind == Nullable {
			mapped = mapped.Null()
		}
		proof.Type = dtoUniqueCollectionReferenceTypeSignature(dtoUniqueFieldType(mapped))
	}
	if has(f.Schema, "default") {
		node, encoded, err := a.referencedDefaultValue(f, annotation)
		if err != nil {
			return nil, err
		}
		if required {
			proof.Initializer.DefaultNull = false
		} else {
			proof.Initializer.Literal, err = referencedDefaultInitializer(node, actual, get(f.Schema, "default"), encoded, naming.Pointer(f.Pointer, "default"))
			if err != nil {
				return nil, err
			}
		}
	}
	v, p := f.Schema, f.Pointer
	seen := map[*document.Value]bool{}
	owners := map[*component]bool{}
	for depth := 0; ; depth++ {
		if err := a.dtoUniqueCaptures.charge(1, 0, source); err != nil {
			return nil, err
		}
		if err := depthCheck(depth, source); err != nil {
			return nil, err
		}
		if seen[v] {
			return nil, schemaReferenceUnsupported(source)
		}
		seen[v] = true
		tree, err := a.dtoUniqueCaptures.referenceTree(v, source)
		if err != nil {
			return nil, err
		}
		proof.Trees = append(proof.Trees, tree)
		if !has(v, "$ref") {
			// Follow only original container edges leading to a reference.
			// Old inline terminal trees keep their original capture costs.
			if collection && dtoUniqueCollectionTreeHasReference(v, 0) {
				edge := "items"
				if dictionarySchema(v) {
					edge = "additionalProperties"
				}
				v, p = get(v, edge), naming.Pointer(p, edge)
				continue
			}
			break
		}
		reference := text(get(v, "$ref"))
		target, owner, path, err := a.dtoUniqueReferencePath(reference, source)
		if err != nil {
			return nil, err
		}
		original, at, err := a.defaultSource(reference, p)
		if err != nil {
			return nil, err
		}
		if original != target {
			return nil, schemaReferenceUnsupported(source)
		}
		proof.Path = append(proof.Path, path...)
		if collection {
			if origin := a.borrowedCollections[target]; origin != nil {
				proof.Borrowed = append(proof.Borrowed, origin)
			}
		}

		if !owners[owner] {
			owners[owner] = true
			state := &dtoUniqueReferenceOwner{Owner: owner, Schema: owner.Schema, Wire: owner.Wire, Name: owner.Name, Pointer: owner.Pointer, Reference: owner.Reference, Scalar: owner.Scalar, Validation: owner.Validation, BindingTarget: owner.BindingTarget, ScalarTarget: owner.ScalarTarget, InternalTarget: owner.InternalModelTarget, Height: owner.AliasHeight, Type: dtoUniqueScalarTypeSignature(a.componentType(owner)), OriginalType: a.componentType(owner)}
			if owner.BindingNullable != nil {
				state.HasNullable = true
				state.Nullable = *owner.BindingNullable
			}
			if collection && namedCollectionCandidate(owner) {
				if previous, ok := a.namedCollectionUses[owner]; !ok || source < previous {
					a.namedCollectionUses[owner] = source
				}
			}
			proof.Owners = append(proof.Owners, state)
		}
		v, p = original, at
	}
	if collection && dtoUniqueStringEnumTreeSource(v) {
		for has(v, "items") || dictionarySchema(v) {
			edge := "items"
			if dictionarySchema(v) {
				edge = "additionalProperties"
			}
			v, p = get(v, edge), naming.Pointer(p, edge)
		}
		proof.EnumSchema, proof.EnumPointer = v, p
	}
	if uniqueStringEnumType(annotation) {
		if !plainStringEnumSchema(v) {
			return nil, unsupported(source)
		}
		proof.EnumSchema = v
		proof.EnumPointer = p
	}
	return proof, nil
}

func dtoUniqueReferenceNodesMatch(nodes []borrowedAllOfNode) bool {
	for _, node := range nodes {
		if !slices.Equal(node.Node.Members(), node.Members) || !slices.Equal(node.Node.Items(), node.Items) {
			return false
		}
		for _, member := range node.Members {
			if get(node.Node, member.Name) != member.Value {
				return false
			}
		}
	}
	return true
}
func (a *analyzer) dtoUniqueScalarReferenceMatches(proof *dtoUniqueScalarReference, p Property, plan *Plan) bool {
	actualType := dtoUniqueScalarTypeSignature(p.Type)
	if proof.Collection {
		actualType = dtoUniqueCollectionReferenceTypeSignature(dtoUniqueFieldType(p.Type))
		observed, original := p.Type, proof.CollectionType
		for observed.Kind == Nullable && original.Kind == Nullable {
			observed, original = *observed.Element, *original.Element
		}
		if observed.Kind == Named && observed.Element != original.Element {
			return false
		}
		if dtoUniqueCollectionReferenceTypeSignature(p.Type) != dtoUniqueCollectionReferenceTypeSignature(proof.CollectionType) {
			return false
		}
		for _, origin := range proof.Borrowed {
			if a.borrowedCollections[origin.Schema] != origin || len(origin.Types) == 0 {
				return false
			}
			for _, snapshot := range origin.Types {
				value := *snapshot.Node
				value.SchemaSteps = 0
				if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
					return false
				}
			}
		}
	}
	if actualType == "" || actualType != proof.Type || p.Validation != nil || p.Default != proof.Initializer.Literal || p.DefaultNull != proof.Initializer.DefaultNull || !dtoUniqueReferenceNodesMatch(proof.Path) {
		return false
	}
	for _, tree := range proof.Trees {
		raw, _ := tree.Schema.MarshalJSON()
		if string(raw) != tree.Bytes || !dtoUniqueReferenceNodesMatch(tree.Nodes) {
			return false
		}
	}
	if proof.EnumSchema != nil {
		enumType := p.Type.WithoutRootNull()
		if proof.Collection {
			for depth := 0; depth <= MaxSchemaDepth && enumType.Element != nil; depth++ {
				enumType = *enumType.Element
			}
		}
		if !uniqueStringEnumType(enumType) {
			return false
		}
		enumName := enumType.Name
		if proof.Collection && !strings.HasPrefix(enumName, a.options.PackageName+".") {
			// Collection aliases retain local peer/nested names in the models
			// namespace. Compare that real spelling with the original registry;
			// the registry/object/schema/member checks remain authoritative.
			enumName = a.options.PackageName + ".models." + enumName
		}
		actual := a.dtoReferenceEnums[proof.EnumSchema]
		origin, ok := a.enumOrigins[actual]
		if actual == nil || a.dtoReferenceFinalEnums[actual] != 1 || a.dtoReferenceFinalEnumSources[proof.EnumSchema] != 1 || !ok || origin.Schema != proof.EnumSchema || origin.Pointer != proof.EnumPointer || a.dtoReferenceEnumNames[actual] != enumName || a.dtoReferenceFinalEnumNames[actual] != enumName || proof.EnumActual != nil && proof.EnumActual != actual {
			return false
		}
		expected, err := buildStringEnumKeysFor(actual.ClassName, proof.EnumSchema, proof.EnumPointer, false, a)
		if err != nil || !slices.Equal(actual.Members, expected.Members) {
			return false
		}
		proof.EnumActual = actual
	}
	for _, state := range proof.Owners {
		c := state.Owner
		if a.components[state.Wire] != c || c.Schema != state.Schema || c.Wire != state.Wire || c.Name != state.Name || c.Pointer != state.Pointer || c.Reference != state.Reference || c.Scalar != state.Scalar || c.Validation != state.Validation || c.BindingTarget != state.BindingTarget || c.ScalarTarget != state.ScalarTarget || c.InternalModelTarget != state.InternalTarget || c.AliasHeight != state.Height || (c.BindingNullable != nil) != state.HasNullable || state.HasNullable && *c.BindingNullable != state.Nullable {
			return false
		}
		expectedType, currentType := state.Type, dtoUniqueScalarTypeSignature(a.componentType(c))
		if proof.Collection {
			expectedType = dtoUniqueCollectionReferenceTypeSignature(state.OriginalType)
			currentType = dtoUniqueCollectionReferenceTypeSignature(a.componentType(c))
		}
		if expectedType == "" || currentType != expectedType {
			return false
		}
		count := 0
		var actual Declaration
		for _, d := range plan.Models {
			if d.Name() == state.Name {
				count++
				actual = d
			}
		}
		if count != 1 || a.dtoReferenceDeclarations[c] != actual || state.Actual != nil && state.Actual != actual {
			return false
		}
		switch d := actual.(type) {
		case *ScalarAlias:
			if c.Scalar == "" || d.Scalar != c.Scalar {
				return false
			}
		case *ValueAlias:
			signature := dtoUniquePhysicalScalarTypeSignature(d.Type)
			if proof.Collection && dtoUniqueBuiltinCollectionReferenceType(d.Type, 0) {
				signature = dtoUniqueCollectionReferenceTypeSignature(d.Type)
			}
			if original, ok := a.dtoReferenceDeclarationTypes[c]; ok && signature != original {
				return false
			}
			if !proof.Collection && (c.Reference != "" || c.Scalar != "") {
				if dtoUniqueScalarTypeSignature(d.Type) != state.Type {
					return false
				}
			}
		case *StringEnum:
			origin, ok := a.enumOrigins[d]
			if !ok || origin.Schema != state.Schema || origin.Pointer != state.Pointer || a.dtoReferenceEnums[state.Schema] != d {
				return false
			}
		case *Model:
			if a.modelOrigins[d] != state.Schema || a.modelPointers[d] != state.Pointer {
				return false
			}
		default:
			return false
		}
		if c.Reference == "" && c.Scalar != "" {
			if builtinPrimitiveBinding(c) {
				origin, ok := a.builtinOrigins[actual]
				if !ok || origin.Owner != c || origin.Schema != state.Schema || origin.Pointer != state.Pointer || origin.Scalar != state.Scalar || origin.Validation != state.Validation || !builtinDeclaration(actual, origin) {
					return false
				}
			} else if nativeEnumBinding(c) {
				origin := a.nativeEnumOrigins[c]
				if origin == nil || origin.Actual != actual || origin.Owner != c || origin.Schema != state.Schema || origin.Pointer != state.Pointer || origin.Scalar != state.Scalar || origin.Validation != state.Validation || !nativeEnumDeclaration(actual, origin) || namedCollectionChecks(TypeRef{Validation: c.Validation}) != origin.Checks {
					return false
				}
			} else if temporalBinding(c) {
				origin, ok := a.temporalOrigins[actual]
				if !ok || origin.Owner != c || origin.Schema != state.Schema || origin.Pointer != state.Pointer || origin.Scalar != state.Scalar || !temporalDeclaration(actual, origin) {
					return false
				}
			} else {
				return false
			}
		}
		state.Actual = actual
	}
	return true
}

// Record original planner results before later clients/defaults and final
// authority checks. A matching name/type cannot substitute a copied export.
func (a *analyzer) recordDTOReferenceDeclarations(declarations []Declaration) {
	var visit func(Declaration, string)
	visit = func(d Declaration, scope string) {
		switch d := d.(type) {
		case *StringEnum:
			if origin, ok := a.enumOrigins[d]; ok {
				a.dtoReferenceEnums[origin.Schema] = d
				a.dtoReferenceEnumNames[d] = scope + "." + d.ClassName
			}
		case *Model:
			a.recordDTOCollectionReferenceGetters(d)
			for _, child := range d.Nested {
				visit(child, scope+"."+d.ClassName)
			}
		case *ObjectUnion:
			for _, child := range d.Variants {
				visit(child, scope+"."+d.ClassName)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object, scope)
			}
		case *Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf, scope)
			}
		}
	}
	for _, d := range declarations {
		visit(d, a.options.PackageName+".models")
	}
	byName := make(map[string]Declaration, len(declarations))
	for _, d := range declarations {
		byName[d.Name()] = d
	}
	for _, c := range a.components {
		if d := byName[c.Name]; d != nil {
			a.dtoReferenceDeclarations[c] = d
			if alias, ok := d.(*ValueAlias); ok {
				ty := alias.Type.WithoutRootNull()
				if ty.Kind == Scalar || ty.Kind == Named && ty.Element == nil {
					a.dtoReferenceDeclarationTypes[c] = dtoUniquePhysicalScalarTypeSignature(alias.Type)
				} else if dtoUniqueBuiltinCollectionReferenceType(alias.Type, 0) {
					a.dtoReferenceDeclarationTypes[c] = dtoUniqueCollectionReferenceTypeSignature(alias.Type)
				}
			}
		}
	}
}

// Only the consuming field owns default here; resolve its original target for
// enum eligibility without fabricating a default-free copy of that field.
func (a *analyzer) dtoUniqueStringEnumReferenceSource(v *document.Value, p string) bool {
	if !has(v, "$ref") || checkKeys(v, []string{"$ref", "nullable", "default"}, p) != nil {
		return false
	}
	target, err := a.resolveSchema(text(get(v, "$ref")), naming.Pointer(p, "$ref"))
	return err == nil && a.uniqueStringEnumSource(target.Schema, target.Pointer, 0)
}
