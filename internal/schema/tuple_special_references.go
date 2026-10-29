package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

type tupleSpecialSource struct {
	Original, Terminal    *document.Value
	Pointer, Target, Kind string
	Root                  *component
	Name, Signature       string
	Steps                 int
	Nullable              bool
	Bindings              []tupleLiteralBinding
}

type tupleSpecialRegistry struct {
	Sources    map[*document.Value]*tupleSpecialSource
	Order      []*tupleSpecialSource
	Uses       map[*TypeRef]tupleCollectionUse
	Selections []tupleInlineAllOfSelection
}

func (a *analyzer) initTupleSpecials() {
	if a.tupleSpecials == nil {
		a.tupleSpecials = &tupleSpecialRegistry{Sources: map[*document.Value]*tupleSpecialSource{}, Uses: map[*TypeRef]tupleCollectionUse{}}
	}
}

func (a *analyzer) tupleSpecialKind(v *document.Value) string {
	c := a.componentsByOriginalSchema(v)
	if c == nil {
		return ""
	}
	switch {
	case metadataStringEnumCandidate(v):
		return "enum"
	case c.ValueClass:
		return "valueclass"
	case bigintBinding(c):
		return "bigint"
	case bytesBinding(c):
		return "bytes"
	case callerNativeBinding(c):
		return "native"
	case text(get(v, "x-native-type")) == "date" && temporalBinding(c):
		return "temporal"
	case !c.BooleanRoot && !has(v, "allOf") && a.rawTrueCandidate(c):
		return "raw"
	}
	return ""
}

// These roles borrow actual existing root declarations and their codecs. A
// provider classifier or physical value class is never reconstructed from a
// Kotlin spelling, and bigint remains a numeric collection element.
func (a *analyzer) tupleSpecialType(original *document.Value, source string, terminal *document.Value, pointer string, depth int, kind string) (TypeRef, *component, *diagnostic.Diagnostic) {
	if kind == "enum" {
		if _, err := buildStringEnumKeysFor("TupleMember", terminal, pointer, false, a); err != nil {
			return TypeRef{}, nil, err
		}
		// Resolve named aliases before capturing their original source heights.
		// Internal lexical targets use the shared nominal ownership authority.
		if declared, err := a.resolveDeclared(text(get(original, "$ref")), source); err == nil && declared.Reference != "" && a.borrowedEnumAliasVisit != nil {
			if err := a.borrowedEnumAliasVisit(declared, depth); err != nil {
				return TypeRef{}, nil, err
			}
		}
	}
	target, err := a.resolveSchema(text(get(original, "$ref")), source)
	if err != nil {
		return TypeRef{}, nil, err
	}
	if target.Schema != terminal || target.Pointer != pointer || kind == "" || a.tupleSpecialKind(terminal) != kind {
		return TypeRef{}, nil, schemaReferenceUnsupported(source)
	}
	var admitted bool
	var root *component
	switch kind {
	case "enum":
		root = a.componentsByOriginalSchema(terminal)
		admitted = root != nil && root.Scalar == "" && !root.ValueClass && root.Reference == "" && root.InternalModelTarget == nil && root.BindingTarget == nil
	case "valueclass":
		admitted = a.admitBorrowedValueClass(target, source)
		root = target.PhysicalValueClass
	case "bigint":
		admitted = a.admitBorrowedBigint(target, source)
		root = target.CanonicalBigint
	case "bytes":
		admitted = a.admitBorrowedBytes(target, source)
		root = target.CanonicalBytes
	case "native":
		admitted = a.admitBorrowedCallerNative(target, source)
		root = target.CanonicalCallerNative
	case "temporal":
		admitted = a.admitBorrowedTemporal(target, source)
		root = target.CanonicalTemporal
	case "raw":
		admitted = a.admitBorrowedRawTrue(target, source)
		root = target.CanonicalJSON
	}
	if !admitted || root == nil || root.Schema != terminal || root.Pointer != pointer || a.components[root.Wire] != root {
		return TypeRef{}, nil, schemaReferenceUnsupported(source)
	}
	ty, err := a.referenceType(target, nullableOverride(original), depth, source)
	if err != nil {
		return TypeRef{}, nil, err
	}
	ty = defaultBacking(ty)
	bare := ty.WithoutRootNull()
	valid := bare.Kind == Scalar && bare.Name == root.Scalar && bare.Element == nil && bare.Serializer == "" && bare.Validation == nil
	if kind == "valueclass" {
		valid = bare.Kind == InlineValue && bare.Name == a.options.PackageName+".models."+root.Name && bare.Element != nil && bare.Element.Kind == Scalar && bare.Element.Name == root.Scalar && bare.Serializer == ""
	} else if kind == "raw" {
		valid = tupleCoverBacking(ty)
	} else if kind == "enum" {
		valid = bare.Kind == Named && bare.Name == a.options.PackageName+".models."+root.Name && bare.Element == nil && bare.Serializer == ""
	}
	if !valid {
		return TypeRef{}, nil, schemaReferenceUnsupported(source)
	}
	return ty, root, nil
}

func (a *analyzer) tupleSpecialReference(original *document.Value, source string, terminal *document.Value, pointer string, depth, steps int, kind string) (tupleScalarSignature, *diagnostic.Diagnostic) {
	ty, root, err := a.tupleSpecialType(original, source, terminal, pointer, depth, kind)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	a.initTupleSpecials()
	r := a.tupleSpecials
	signature := falsePropertySignature(Property{Type: ty})
	if old := r.Sources[original]; old != nil {
		if old.Terminal != terminal || old.Pointer != source || old.Target != pointer || old.Kind != kind || old.Root != root || old.Name != root.Name || old.Signature != signature || old.Steps != steps {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	} else {
		if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
			return tupleScalarSignature{}, err
		}
		entry := &tupleSpecialSource{Original: original, Terminal: terminal, Pointer: source, Target: pointer, Kind: kind, Root: root, Name: root.Name, Signature: signature, Steps: steps, Nullable: ty.Kind == Nullable}
		for node, at := original, source; ; {
			if c := a.componentsByOriginalSchema(node); c != nil {
				if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
					return tupleScalarSignature{}, err
				}
				entry.Bindings = append(entry.Bindings, tupleLiteralBinding{c, c.Name, c.Scalar, c.BindingTarget, c.ScalarTarget, c.AliasHeight, falseNullableIntent(c)})
			}
			if node == terminal {
				break
			}
			node, at, err = a.defaultSource(text(get(node, "$ref")), at)
			if err != nil {
				return tupleScalarSignature{}, err
			}
		}
		r.Sources[original] = entry
		r.Order = append(r.Order, entry)
	}
	return tupleScalarSignature{Kind: "tuple_special_" + kind, Nullable: ty.Kind == Nullable, ModelSource: terminal}, nil
}

func (a *analyzer) tupleSpecialElement(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if a.tupleSpecials == nil || a.tupleSpecials.Sources[v] == nil {
		return nil, nil
	}
	source := a.tupleSpecials.Sources[v]
	if source.Pointer != pointer {
		return nil, schemaReferenceUnsupported(pointer)
	}
	ty, root, err := a.tupleSpecialType(v, pointer, source.Terminal, source.Target, depth, source.Kind)
	if err != nil {
		return nil, err
	}
	if root != source.Root || root.Name != source.Name || falsePropertySignature(Property{Type: ty}) != source.Signature || (ty.Kind == Nullable) != source.Nullable {
		return nil, schemaReferenceUnsupported(pointer)
	}
	return &ty, nil
}

func (a *analyzer) tupleSpecialWork(element TypeRef, decision *arrayItemsDecision) TypeRef {
	if !a.aliasPlanning || a.tupleSpecials == nil || decision == nil {
		return element
	}
	for _, prefix := range decision.Prefix {
		if source := a.tupleSpecials.Sources[prefix.Schema]; source != nil {
			element.SchemaSteps = max(element.SchemaSteps, source.Steps)
		}
	}
	return element
}

func (a *analyzer) recordTupleSpecialUse(t *TypeRef, source *document.Value) {
	a.tupleSpecials.Uses[t] = tupleCollectionUse{Source: source, Pointer: a.tupleSpecials.Sources[source].Pointer, Snapshots: dtoOriginalGetterTypes(t), Final: true}
}

func (a *analyzer) selectTupleSpecial(array *document.Value, pointer string, decision *arrayItemsDecision) {
	if a.tupleSpecials == nil || decision.Projection != nil || a.tupleSpecials.Sources[decision.Selected] == nil {
		return
	}
	for _, old := range a.tupleSpecials.Selections {
		if old.Array == array {
			return
		}
	}
	a.tupleSpecials.Selections = append(a.tupleSpecials.Selections, tupleInlineAllOfSelection{array, decision.Selected, pointer})
}

func (a *analyzer) tupleSpecialDefaultElement(v *document.Value, pointer string, actual TypeRef) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	source := a.tupleSpecials.Sources[v]
	if source.Pointer != pointer || falsePropertySignature(Property{Type: defaultBacking(actual)}) != source.Signature {
		return nil, schemaReferenceUnsupported(pointer)
	}
	ty := defaultBacking(actual)
	if source.Kind == "enum" {
		return &nestedDefaultNode{schema: source.Terminal, pointer: source.Target, ty: ty, stringEnum: true}, nil
	}
	if source.Kind == "valueclass" {
		// The actual wrapper serializer owns emission. Validate the literal
		// against its original raw scalar/checks, with the actual nullable view.
		ty = TypeRef{Kind: Scalar, Name: source.Root.Scalar, Validation: source.Root.Validation}
		if source.Nullable {
			ty = ty.Null()
		}
	}
	return &nestedDefaultNode{schema: source.Terminal, pointer: source.Target, ty: ty, rawJSON: source.Kind == "raw"}, nil
}

func (a *analyzer) tupleSpecialDeclaration(d Declaration, source *tupleSpecialSource) bool {
	switch source.Kind {
	case "enum":
		vocabulary, ok := d.(*StringEnum)
		if !ok || a.tupleObjects == nil {
			return false
		}
		origin, known := a.enumOrigins[vocabulary]
		proof, captured := a.tupleObjects.UnionDeclarations[d]
		return known && captured && origin == proof.Origin && origin.Schema == source.Terminal && origin.Pointer == source.Target && proof.Source == source.Terminal && proof.Pointer == source.Target && tupleUnionDeclarationMatches(d, proof.Snapshot)
	case "valueclass":
		wrapper, ok := d.(*PrimitiveValueClass)
		if !ok {
			return false
		}
		origin, known := a.valueClassOrigins[wrapper]
		return known && origin.Schema == source.Terminal && origin.Pointer == source.Target
	case "bigint":
		origin, known := a.bigintOrigins[d]
		return known && origin.Owner == source.Root && bigintDeclaration(d, origin)
	case "bytes":
		origin, known := a.bytesOrigins[d]
		return known && origin.Owner == source.Root && bytesDeclaration(d, origin)
	case "native":
		origin, known := a.callerNativeOrigins[d]
		return known && origin.Owner == source.Root && callerNativeDeclaration(d, origin)
	case "temporal":
		origin, known := a.temporalOrigins[d]
		return known && origin.Owner == source.Root && temporalDeclaration(d, origin)
	case "raw":
		origin, known := a.rawTrueOrigins[d]
		return known && origin.Owner == source.Root && a.rawTrueDeclaration(d, origin)
	}
	return false
}

func (a *analyzer) validateTupleSpecials(plan *Plan, used map[*document.Value]int) *diagnostic.Diagnostic {
	r := a.tupleSpecials
	if r == nil || len(r.Order) == 0 {
		return nil
	}
	// The shared borrowed-leaf checks retain original source, codec and
	// physical declaration identities, including erased-only prefixes.
	for _, validate := range []func([]Declaration) *diagnostic.Diagnostic{a.validateBorrowedValueClassLeaves, a.validateBorrowedBigintLeaves, a.validateBorrowedBytesLeaves, a.validateBorrowedCallerNativeLeaves, a.validateBorrowedTemporalLeaves, a.validateBorrowedRawTrueLeaves} {
		if err := validate(plan.Models); err != nil {
			return err
		}
	}
	checked := map[*component]bool{}
	for _, source := range r.Order {
		if r.Sources[source.Original] != source || a.components[source.Root.Wire] != source.Root || source.Root.Name != source.Name {
			return schemaReferenceUnsupported(source.Pointer)
		}
		steps, node, at := 0, source.Original, source.Pointer
		for has(node, "$ref") && steps <= MaxSchemaDepth {
			var err *diagnostic.Diagnostic
			node, at, err = a.defaultSource(text(get(node, "$ref")), at)
			if err != nil {
				return err
			}
			steps++
		}
		if node != source.Terminal || at != source.Target || steps != source.Steps {
			return schemaReferenceUnsupported(source.Pointer)
		}
		for _, binding := range source.Bindings {
			c := binding.Owner
			if a.components[c.Wire] != c || c.Name != binding.Name || c.Scalar != binding.Scalar || c.BindingTarget != binding.BindingTarget || c.ScalarTarget != binding.ScalarTarget || c.AliasHeight != binding.AliasHeight || falseNullableIntent(c) != binding.NullableIntent {
				return schemaReferenceUnsupported(source.Pointer)
			}
		}
		if _, err := a.tupleSpecialElement(source.Original, source.Pointer, 0); err != nil {
			return err
		}
		if !checked[source.Root] {
			count := 0
			for _, d := range plan.Models {
				if d.Name() == source.Name {
					if !a.tupleSpecialDeclaration(d, source) {
						return schemaReferenceUnsupported(source.Pointer)
					}
					count++
				}
			}
			if count != 1 {
				return schemaReferenceUnsupported(source.Pointer)
			}
			checked[source.Root] = true
		}
	}
	for _, selection := range r.Selections {
		if r.Sources[selection.Source] == nil || used[selection.Source] == 0 {
			return schemaReferenceUnsupported(selection.Pointer)
		}
	}
	return nil
}
