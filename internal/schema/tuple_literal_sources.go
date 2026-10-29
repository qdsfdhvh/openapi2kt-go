package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type tupleLiteralSource struct {
	Original, Terminal *document.Value
	Pointer, Target    string
	Steps              int
	Nullable           bool
	Bindings           []tupleLiteralBinding
	Scalar             string
	StructuredKey      string
}

type tupleLiteralBinding struct {
	Owner                       *component
	Name, Scalar                string
	BindingTarget, ScalarTarget *component
	AliasHeight                 int
	NullableIntent              int
}

type tupleLiteralUse struct {
	Source    *tupleLiteralSource
	Snapshots []namedCollectionSourceType
}

type tupleLiteralRegistry struct {
	Sources        map[*document.Value]*tupleLiteralSource
	Order          []*tupleLiteralSource
	Uses           map[*TypeRef]tupleLiteralUse
	Selections     []tupleInlineAllOfSelection
	StructuredKeys map[*document.Value]string
}

func (a *analyzer) initTupleLiterals() {
	if a.tupleLiterals == nil {
		a.tupleLiterals = &tupleLiteralRegistry{Sources: map[*document.Value]*tupleLiteralSource{}, Uses: map[*TypeRef]tupleLiteralUse{}, StructuredKeys: map[*document.Value]string{}}
	}
}

func tupleNullCandidate(v *document.Value) bool {
	kind, _, err := schemaType(v, "")
	return err == nil && kind == "null" || !has(v, "type") && has(v, "const") && get(v, "const").Kind() == document.Null
}

func tupleScalarConstCandidate(v *document.Value) bool {
	return has(v, "const") && !structuredLiteral(v) && !tupleNullCandidate(v)
}

// Original scalar domains authorize their actual codecs. Matching Kotlin type
// names cannot bypass the original vocabulary, const or reference grammar.
func (a *analyzer) tupleLiteralSourceType(original *document.Value, source string, terminal *document.Value, pointer string, start, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	_, nullable, err := schemaType(terminal, pointer)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	scalar := "kotlinx.serialization.json.JsonNull"
	scalarConst := tupleScalarConstCandidate(terminal) || structuredLiteral(terminal)
	keys := []string{"type", "nullable", "enum", "const"}
	if scalarConst {
		keys = []string{"type", "nullable", "const", "format", "x-native-type"}
	}
	keys = a.tupleSourceKeys(terminal, pointer, keys)
	if err := checkKeys(terminal, keys, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if has(terminal, "const") {
		var value *Constant
		var err *diagnostic.Diagnostic
		if structuredLiteral(terminal) {
			// Source validation needs the original kind, not a temporary const
			// initializer. Actual selected element planning still owns its codec.
			var kind string
			var literal *document.Value
			kind, literal, err = constantDeclaration(terminal, pointer, depth)
			if err == nil {
				var name string
				name, _, err = structuredConstantShape(terminal, literal, kind, pointer)
				value = &Constant{Type: TypeRef{Kind: Scalar, Name: name}}
			}
		} else {
			value, err = constant("Literal", "literal", terminal, pointer, depth)
		}
		if err != nil {
			return tupleScalarSignature{}, err
		}
		scalar = value.Type.Name
	} else if has(terminal, "enum") {
		if _, err := nullOnlyEnumProfile(terminal, pointer, a.tupleSourceOwnDefault(terminal, pointer)); err != nil {
			return tupleScalarSignature{}, err
		}
	} else if _, err := scalarMapping("null", terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if original != terminal {
		target, err := a.resolveSchema(text(get(original, "$ref")), source)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		ty, err := a.referenceType(target, nullableOverride(original), start, source)
		if err != nil {
			return tupleScalarSignature{}, err
		}
		backing := defaultBacking(ty)
		bare := backing.WithoutRootNull()
		if bare.Kind != Scalar || bare.Name != scalar || bare.Validation != nil || bare.Serializer != "" {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
		nullable = backing.Kind == Nullable
	}
	if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	a.initTupleLiterals()
	r := a.tupleLiterals
	if old := r.Sources[original]; old != nil {
		if old.Terminal != terminal || old.Pointer != source || old.Target != pointer || old.Nullable != nullable || old.Steps != depth-start || old.Scalar != scalar {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	} else {
		if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
			return tupleScalarSignature{}, err
		}
		entry := &tupleLiteralSource{Original: original, Terminal: terminal, Pointer: source, Target: pointer, Steps: depth - start, Nullable: nullable, Scalar: scalar}
		for node, at := original, source; ; {
			if owner := a.componentsByOriginalSchema(node); owner != nil {
				if err := a.tupleNativeSources.charge(1, 0, source); err != nil {
					return tupleScalarSignature{}, err
				}
				entry.Bindings = append(entry.Bindings, tupleLiteralBinding{owner, owner.Name, owner.Scalar, owner.BindingTarget, owner.ScalarTarget, owner.AliasHeight, falseNullableIntent(owner)})
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
	// Null profiles share an exact singleton codec. Other const signatures keep
	// original terminal identity, including sources discarded by projection.
	if scalarConst {
		return tupleScalarSignature{Kind: "const", Nullable: nullable, ConstSource: terminal}, nil
	}
	return tupleScalarSignature{Kind: "null", Nullable: nullable}, nil
}

func (a *analyzer) tupleLiteralWork(element TypeRef, decision *arrayItemsDecision) TypeRef {
	if !a.aliasPlanning || a.tupleLiterals == nil || decision == nil {
		return element
	}
	for _, prefix := range decision.Prefix {
		if source := a.tupleLiterals.Sources[prefix.Schema]; source != nil {
			element.SchemaSteps = max(element.SchemaSteps, source.Steps)
		}
	}
	return element
}

func (a *analyzer) selectTupleLiteral(array *document.Value, pointer string, decision *arrayItemsDecision) {
	if a.tupleLiterals == nil || decision.Projection != nil || a.tupleLiterals.Sources[decision.Selected] == nil {
		return
	}
	for _, old := range a.tupleLiterals.Selections {
		if old.Array == array {
			return
		}
	}
	a.tupleLiterals.Selections = append(a.tupleLiterals.Selections, tupleInlineAllOfSelection{array, decision.Selected, pointer})
}

func (a *analyzer) recordTupleLiteralUse(t *TypeRef, source *tupleLiteralSource) {
	a.tupleLiterals.Uses[t] = tupleLiteralUse{source, dtoOriginalGetterTypes(t)}
}

// Only an original positional source grants this const codec view. In
// particular, a native const must use valueConstant before the ordinary native
// schema branch, which intentionally does not accept const annotations.
func (a *analyzer) tupleLiteralElementType(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if a.tupleLiterals == nil || a.tupleLiterals.Sources[v] == nil {
		return nil, nil
	}
	source := a.tupleLiterals.Sources[v]
	if !tupleScalarConstCandidate(source.Terminal) && !structuredLiteral(source.Terminal) {
		return nil, nil
	}
	if source.Pointer != pointer {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	var ty TypeRef
	if has(v, "$ref") {
		target, err := a.resolveSchema(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, err
		}
		ty, err = a.referenceType(target, nullableOverride(v), depth, pointer)
		if err != nil {
			return nil, err
		}
	} else {
		value, err := valueConstant("Literal", "literal", v, pointer, depth)
		if err != nil {
			return nil, err
		}
		ty = value.Type
		if source.Nullable && ty.Kind != Nullable {
			ty = ty.Null()
		}
		a.warnValueConst(v, pointer)
	}
	backing := defaultBacking(ty)
	if backing.WithoutRootNull().Kind != Scalar || backing.WithoutRootNull().Name != source.Scalar || (backing.Kind == Nullable) != source.Nullable {
		return nil, schemaReferenceUnsupported(pointer)
	}
	return &ty, nil
}

func (a *analyzer) tupleLiteralDefaultNode(v *document.Value, pointer string, actual TypeRef, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if !tupleNullCandidate(v) && !tupleScalarConstCandidate(v) && !structuredLiteral(v) {
		return nil, schemaReferenceUnsupported(pointer)
	}
	if _, err := a.tupleLiteralSourceType(v, pointer, v, pointer, depth, depth); err != nil {
		return nil, err
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, ty: defaultBacking(actual), nullOnly: tupleNullCandidate(v)}
	if !node.nullOnly {
		node.literalSource = a.tupleLiterals.Sources[v]
		if structuredLiteral(v) {
			key, ok := a.tupleLiterals.StructuredKeys[v]
			if !ok {
				var err *diagnostic.Diagnostic
				key, err = rawJSONDefaultKey(get(v, "const"), naming.Pointer(pointer, "const"))
				if err != nil {
					return nil, err
				}
				if err := a.tupleNativeSources.charge(1, len(key), pointer); err != nil {
					return nil, err
				}
				a.tupleLiterals.StructuredKeys[v] = key
			}
			node.literalSource.StructuredKey = key
		}
	}
	return node, nil
}

// Match the original JSON domain before mapped floating precision or native
// wire conversion. Canonical values still decode through the actual codec.
func (source *tupleLiteralSource) defaultValue(value *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	pin := get(source.Terminal, "const")
	if structuredLiteral(source.Terminal) {
		key, err := rawJSONDefaultKey(value, pointer)
		if err != nil {
			return "", err
		}
		if key != source.StructuredKey {
			return "", invalid("Tuple default member must equal its complete original structured const.", pointer)
		}
		encoded, _ := value.MarshalJSON()
		return string(encoded), nil
	}
	kind := "number"
	if pin.Kind() == document.String {
		kind = "string"
	} else if pin.Kind() == document.Boolean {
		kind = "boolean"
	}
	if !primitivePinEqual(pin, value, kind, pointer) {
		return "", invalid("Tuple default member must equal its original const literal.", pointer)
	}
	if source.Scalar == "kotlin.Long" && text(get(source.Terminal, "x-native-type")) == "bigint" && pin.Kind() == document.String {
		literal, err := strconv.ParseInt(text(value), 10, 64)
		if err != nil {
			return "", invalid("Const default must fit its mapped Long codec.", pointer)
		}
		return strconv.FormatInt(literal, 10), nil
	}
	if _, err := scalarLiteral(source.Scalar, value, pointer); err != nil {
		return "", err
	}
	if source.Scalar == "kotlin.time.Instant" && !collectionDefaultInstantPattern.MatchString(text(value)) {
		return "", invalid("Timestamp const default must fit the Kotlin Instant format and precision.", pointer)
	}
	if source.Scalar == "kotlin.Float" || source.Scalar == "kotlin.Double" {
		number, _ := floatLiteral(value)
		bits := 64
		if source.Scalar == "kotlin.Float" {
			number, bits = float64(float32(number)), 32
		}
		return strconv.FormatFloat(number, 'g', -1, bits), nil
	}
	encoded, _ := value.MarshalJSON()
	return string(encoded), nil
}

func (a *analyzer) tupleLiteralFieldDefault(v *document.Value, pointer, name string, decision *arrayItemsDecision, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	element, err := a.collectionElementType(decision.Selected, name+"Inner", decision.Pointer, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	child, err := a.tupleLiteralDefaultNode(decision.Selected, decision.Pointer, element, depth+1)
	if err != nil {
		return TypeRef{}, "", err
	}
	ty, err := a.checkedArrayItems(element, v, pointer, decision)
	if err != nil {
		return TypeRef{}, "", err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, "", err
	}
	if nullable {
		ty = ty.Null()
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: child}
	value, at := get(v, "default"), naming.Pointer(pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return TypeRef{}, "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return TypeRef{}, "", err
	}
	if value.Kind() == document.Null {
		return ty, "null", nil
	}
	initializer, err := referencedDefaultInitializer(node, ty, value, encoded, at)
	return ty, initializer, err
}
