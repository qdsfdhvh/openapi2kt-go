package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// resolvedSchema distinguishes a real catalog binding from an original borrowed
// terminal. Internal nodes never acquire component identity or new declarations;
// an eligible DTO target may bind an existing physical nested classifier.
type resolvedSchema struct {
	Catalog *component
	Schema  *document.Value
	Pointer string
	// Borrowed reference chains retain terminal source and cumulative use-site work.
	Steps                 int
	Nullable              *bool
	CollectionBinding     bool
	ModelName             string                    // Existing fully qualified internal DTO owner, if any.
	PhysicalValueClass    *component                // Existing canonical wrapper; preserves InlineValue layout.
	CanonicalTemporal     *component                // Original transparent temporal scalar mapping, not a nominal owner.
	CanonicalNativeEnum   *component                // Actual named checked native enum source and scalar export.
	CanonicalBuiltin      *component                // Original ordinary builtin source and validation, not a nominal owner.
	CanonicalCollection   *component                // Original named linked container source, distinct from borrowed tree.
	CanonicalCover        *component                // Complete original recursive JSON cover with graph suffix work.
	CanonicalJSON         *component                // Original named raw/true source; final declaration permission is separate.
	CanonicalCallerNative *component                // Original caller-native string/profile; actual provider is caller-owned.
	CanonicalBytes        *component                // Original byte/binary format, separate from caller ByteArray.
	CanonicalBigint       *component                // Original bigint marker; field-only codec stays a use-site decision.
	BorrowedScalar        *borrowedScalarSource     // Original internal scalar leaf; no canonical component.
	BorrowedCollection    *borrowedCollectionSource // Original admitted internal finite collection graph.
	// Eligible nominal root bindings retain explicit inherited nullable intent.
	NullableBinding bool
	SourceCatalog   *component // Original lexical owner, never a fabricated target.
	DirectLexical   bool
	ComposedLexical bool
	Deferred        *deferredNominal
}

func schemaReferenceUnsupported(pointer string) *diagnostic.Diagnostic {
	// Retain the historical self-target reference diagnostic verbatim.
	return diagnostic.At("E_REF_UNSUPPORTED", "Only local named component references are supported.", pointer)
}

// resolveSchema is the shared source/type consumer seam. Nominal planners keep
// resolve, which admits only actual catalog references. Borrowed collections are
// admitted as a finite tree before any generic typed planner can create names.
func (a *analyzer) resolveSchema(reference, source string) (*resolvedSchema, *diagnostic.Diagnostic) {
	target, err := a.lookupSchemaReference(reference, source)
	if err != nil || target.Catalog != nil {
		return target, err
	}
	if a.bindInternalModel(target, source) {
		return target, nil
	}
	if has(target.Schema, "$ref") {
		return a.resolveInternalReferenceChain(target, source)
	}
	if err := a.admitInternalSchema(target.Schema, target.Pointer, source); err != nil {
		return nil, err
	}
	return target, nil
}

// Lookup preserves original schema role and node identity without admission.
// Recursive admission separately bounds the transparent target graph.
func (a *analyzer) lookupSchemaReference(reference, source string) (*resolvedSchema, *diagnostic.Diagnostic) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(reference, prefix) || strings.Contains(reference, "%") || !strings.Contains(reference[len(prefix):], "/") {
		c, err := a.resolveDeclared(reference, source)
		if err != nil {
			return nil, err
		}
		if target := c.InternalModelTarget; target != nil {
			// Keep the actual alias catalog entry and its complete source work.
			// Typed chains may bind its eligible physical DTO; borrowed builtin
			// container admission must not disguise it as a scalar terminal.
			return &resolvedSchema{Catalog: c, Schema: target.Schema, Pointer: target.Pointer, Steps: 1 + c.AliasHeight, Nullable: c.BindingNullable, ModelName: target.ModelName, NullableBinding: true, Deferred: target.Deferred, PhysicalValueClass: target.PhysicalValueClass, CanonicalTemporal: target.CanonicalTemporal, CanonicalBuiltin: target.CanonicalBuiltin, CanonicalNativeEnum: target.CanonicalNativeEnum, CanonicalBigint: target.CanonicalBigint, CanonicalBytes: target.CanonicalBytes, CanonicalCallerNative: target.CanonicalCallerNative, CanonicalJSON: target.CanonicalJSON, CanonicalCover: target.CanonicalCover, CanonicalCollection: target.CanonicalCollection, BorrowedScalar: target.BorrowedScalar, BorrowedCollection: target.BorrowedCollection}, nil
		}
		if c.BindingTarget != nil {
			target := c.BindingTarget
			return &resolvedSchema{Catalog: target, Schema: target.Schema, Pointer: target.Pointer, Steps: 1 + c.AliasHeight, Nullable: c.BindingNullable, CollectionBinding: collectionAliasSchema(target.Schema), NullableBinding: c.BindingNullable != nil && !collectionAliasSchema(target.Schema)}, nil
		}
		var hierarchyNullable *bool
		if c.NullableHierarchy {
			flag := true
			hierarchyNullable = &flag
		}
		return &resolvedSchema{Catalog: c, Schema: c.Schema, Pointer: c.Pointer, Nullable: hierarchyNullable, NullableBinding: c.NullableDTO || c.NullableEnum || c.NullableValueClass || c.NullableTemporal || c.NullableBigint || c.NullableBytes || c.NullableCallerNative || c.NullableRaw || c.NullablePrimitive || c.NullableNativeEnum || c.NullableHierarchy || c.NullableStructuredConst}, nil
	}
	raw := strings.Split(reference[len(prefix):], "/")
	keys := make([]string, len(raw))
	for i, s := range raw {
		key, err := naming.DecodePointerSegment(s, source)
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}
	c := a.components[keys[0]]
	if c == nil {
		return nil, diagnostic.At("E_REF_MISSING", "Referenced component does not exist.", source)
	}
	v, p := c.Schema, c.Pointer
	directLexical := directInternalOwner(v)
	crossesComposition := has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf")
	composedLexical := !has(v, "$ref") && !has(v, "enum") && !has(v, "const") && !has(v, "x-native-type") && !has(v, "x-kotlin-value-class") && !has(v, "discriminator")
	missing := func() (*resolvedSchema, *diagnostic.Diagnostic) {
		return nil, diagnostic.At("E_REF_MISSING", "Schema reference target does not exist.", source)
	}
	for i := 1; i < len(keys); {
		if v.Kind() != document.Object {
			return nil, schemaReferenceUnsupported(source)
		}
		directLexical = directLexical && directInternalOwner(v)
		crossesComposition = crossesComposition || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf")
		composedLexical = composedLexical && !has(v, "$ref") && !has(v, "enum") && !has(v, "const") && !has(v, "x-native-type") && !has(v, "x-kotlin-value-class") && !has(v, "discriminator")
		edge := keys[i]
		child, exists := v.Get(edge)
		if !exists {
			return missing()
		}
		kind, _, err := schemaType(v, p)
		if err != nil {
			return nil, err
		}
		switch edge {
		case "properties":
			if kind != "object" || child.Kind() != document.Object || i+1 >= len(keys) {
				return nil, schemaReferenceUnsupported(source)
			}
			p = naming.Pointer(naming.Pointer(p, edge), keys[i+1])
			v = get(child, keys[i+1])
			i += 2
		case "items":
			composedLexical = composedLexical && !has(v, "prefixItems")
			directLexical = directLexical && !has(v, "prefixItems")
			if kind != "array" {
				return nil, schemaReferenceUnsupported(source)
			}
			p = naming.Pointer(p, edge)
			v = child
			i++
		case "additionalProperties":
			composedLexical = composedLexical && dictionarySchema(v)
			directLexical = directLexical && dictionarySchema(v)
			if kind != "object" || child.Kind() != document.Object {
				return nil, schemaReferenceUnsupported(source)
			}
			p = naming.Pointer(p, edge)
			v = child
			i++
		case "allOf", "oneOf", "anyOf":
			crossesComposition = true
			directLexical = false
			if child.Kind() != document.Array || i+1 >= len(keys) {
				return nil, schemaReferenceUnsupported(source)
			}
			key := keys[i+1]
			valid := key != "" && (len(key) == 1 || key[0] != '0')
			for _, c := range []byte(key) {
				valid = valid && c >= '0' && c <= '9'
			}
			if !valid {
				return nil, invalid("Expected a canonical JSON Pointer array index.", source)
			}
			index, e := strconv.Atoi(key)
			if e != nil || index >= len(child.Items()) {
				return missing()
			}
			p = naming.Pointer(naming.Pointer(p, edge), key)
			v = child.Items()[index]
			i += 2
		default:
			return nil, schemaReferenceUnsupported(source)
		}
		if v == nil {
			return missing()
		}
	}
	return &resolvedSchema{Schema: v, Pointer: p, SourceCatalog: c, DirectLexical: directLexical, ComposedLexical: composedLexical && (crossesComposition || composedModelCandidate(v))}, nil
}

// Nominal shape planners need the same alias-hop accounting as typed uses.
// Ordinary direct references retain their historical planner checks/locations.
func (a *analyzer) resolveNominal(reference, source string, depth int) (*component, int, *diagnostic.Diagnostic) {
	declared, err := a.resolveDeclared(reference, source)
	if err != nil {
		return nil, 0, err
	}
	if declared.InternalModelTarget != nil {
		return nil, 0, schemaReferenceUnsupported(source)
	}
	if declared.NullableDTO || declared.NullableEnum || declared.NullableValueClass || declared.NullableTemporal || declared.NullableBigint || declared.NullableBytes || declared.NullableCallerNative || declared.NullableRaw || declared.NullablePrimitive || declared.NullableNativeEnum || declared.BindingTarget != nil && (declared.BindingTarget.NullableDTO || declared.BindingTarget.NullableEnum || declared.BindingTarget.NullableValueClass || declared.BindingTarget.NullableTemporal || declared.BindingTarget.NullableBigint || declared.BindingTarget.NullableBytes || declared.BindingTarget.NullableCallerNative || declared.BindingTarget.NullableRaw || declared.BindingTarget.NullablePrimitive || declared.BindingTarget.NullableNativeEnum) {
		return nil, 0, unsupported(source)
	}
	if declared.BindingTarget != nil {
		if declared.BindingNullable != nil && *declared.BindingNullable && !collectionAliasSchema(declared.BindingTarget.Schema) {
			return nil, 0, unsupported(source)
		}
		steps := 1 + declared.AliasHeight
		if err := depthCheck(depth+steps, source); err != nil {
			return nil, 0, err
		}
		return declared.BindingTarget, steps, nil
	}
	return declared, 1, nil
}

// Checked native enums reuse the direct typed scalar parser. Other terminals
// retain the historical borrowed builtin grammar and diagnostics.
func (a *analyzer) terminalInternalScalar(v *document.Value, p, source string) *diagnostic.Diagnostic {
	if a.options.Validation && has(v, "enum") {
		kind, _, err := schemaType(v, p)
		if err != nil {
			return err
		}
		if kind == "integer" || kind == "number" || kind == "boolean" {
			_, err := a.nativeEnumValueType(v, p, 0)
			return err
		}
	}
	return terminalInternalScalar(v, p, source)
}

func terminalInternalScalar(v *document.Value, p, source string) *diagnostic.Diagnostic {
	if v.Kind() != document.Object {
		return schemaReferenceUnsupported(source)
	}
	for _, key := range []string{"$ref", "enum", "const", "x-native-type", "x-kotlin-value-class", "allOf", "oneOf", "anyOf"} {
		if has(v, key) {
			return schemaReferenceUnsupported(source)
		}
	}
	kind, _, err := schemaType(v, p)
	if err != nil {
		return err
	}
	format := ""
	if value, ok := v.Get("format"); ok {
		var valid bool
		format, valid = value.Text()
		if !valid {
			return invalid("format must be a string.", p)
		}
	}
	legal := kind == "string" && format == "" || kind == "boolean" && format == "" || kind == "integer" && (format == "" || format == "int32" || format == "int64") || kind == "number" && (format == "" || format == "float" || format == "double")
	if !legal {
		return schemaReferenceUnsupported(source)
	}
	return nil
}

// referenceType adds one use-site schema step, never the physical owner path.
// Named references preserve the old alias depth and nullable behavior exactly.
func (a *analyzer) referenceType(target *resolvedSchema, override *bool, depth int, source string) (TypeRef, *diagnostic.Diagnostic) {
	if target.Catalog != nil {
		if target.CanonicalCover != nil {
			if err := depthCheck(depth+max(1, target.Steps)+a.recursiveJSONHeights[target.CanonicalCover.Schema], source); err != nil {
				return TypeRef{}, err
			}
		}
		if target.Steps > 1 {
			if err := depthCheck(depth+target.Steps, source); err != nil {
				return TypeRef{}, err
			}
		}
		if target.BorrowedCollection != nil {
			a.aliasUses[collectionAliasUse{target.Catalog, depth + max(0, target.Steps-1-target.Catalog.AliasHeight), source}] = struct{}{}
		} else if target.CanonicalCollection != nil {
			// Forward backing height is unavailable to early/late typed uses;
			// retain all incoming hops and recheck the canonical suffix later.
			a.aliasUses[collectionAliasUse{target.CanonicalCollection, depth + max(0, target.Steps-1), source}] = struct{}{}
		} else if collectionAliasSchema(target.Catalog.Schema) || target.Catalog.RecursiveJSON {
			// Forward alias plans are not complete while models are analyzed.
			// Preserve the original use location/depth and check cached height
			// after every alias and operation has been linked.
			if a.aliasUses == nil {
				a.aliasUses = map[collectionAliasUse]struct{}{}
			}
			a.aliasUses[collectionAliasUse{target.Catalog, depth + max(0, target.Steps-1), source}] = struct{}{}
		}
		if err := scalarAliasDepth(target.Catalog, depth, source); err != nil {
			return TypeRef{}, err
		}
		ty := a.componentType(target.Catalog)
		if target.CanonicalTemporal != nil {
			ty = a.componentType(target.CanonicalTemporal)
		} else if target.CanonicalNativeEnum != nil {
			ty = a.componentType(target.CanonicalNativeEnum)
		} else if target.CanonicalBuiltin != nil {
			ty = a.componentType(target.CanonicalBuiltin)
		} else if target.CanonicalBigint != nil {
			ty = a.componentType(target.CanonicalBigint)
		} else if target.CanonicalBytes != nil {
			ty = a.componentType(target.CanonicalBytes)
		} else if target.CanonicalCallerNative != nil {
			ty = a.componentType(target.CanonicalCallerNative)
		} else if target.CanonicalJSON != nil {
			ty = a.rawTrueType(target.CanonicalJSON)
		} else if target.CanonicalCover != nil {
			ty = TypeRef{Kind: JSON}
		} else if target.CanonicalCollection != nil {
			ty = a.namedCollectionType(target.CanonicalCollection)
		} else if target.BorrowedCollection != nil {
			ty = a.componentType(target.Catalog)
		}
		if target.CollectionBinding {
			if target.CanonicalCollection == nil {
				ty = bindingCollectionView(ty)
			}
			if override == nil {
				override = target.Nullable
			}
		}
		if target.NullableBinding && override == nil {
			override = target.Nullable
		}
		if override != nil {
			if *override {
				ty = ty.Null()
			} else if target.CollectionBinding || target.NullableBinding || nullableArraySchema(target.Schema) {
				ty = ty.WithoutRootNull()
			}
		}
		// Attach erased work after root narrowing, which may remove an outer
		// Nullable wrapper; otherwise explicit false could discard its hops.
		if a.aliasPlanning && target.CanonicalCover != nil {
			ty.SchemaSteps = max(1+target.Catalog.AliasHeight, target.Steps) + a.recursiveJSONHeights[target.CanonicalCover.Schema]
		} else if a.aliasPlanning && (bare(ty).Kind == Scalar || target.CanonicalJSON != nil) {
			ty.SchemaSteps = max(1+target.Catalog.AliasHeight, target.Steps)
		} else if a.aliasPlanning && target.Steps > 1 {
			ty.SchemaSteps = target.Steps
			if target.CollectionBinding || target.Catalog.RecursiveJSON {
				ty.SchemaSteps--
			}
		}
		return ty, nil
	}
	steps := max(1, target.Steps)
	if err := depthCheck(depth+steps, source); err != nil {
		return TypeRef{}, err
	}
	// The nearest explicit target override wins only when the consumer is absent.
	if override == nil {
		override = target.Nullable
	}
	v, p := target.Schema, target.Pointer
	if target.ModelName != "" || target.Deferred != nil {
		nullable, err := a.nominalSourceNullable(target)
		if err != nil {
			return TypeRef{}, err
		}
		if override != nil {
			nullable = *override
		}
		ty := TypeRef{Kind: Named, Name: target.ModelName, deferredNominal: target.Deferred}
		if target.Deferred != nil {
			ty.Name = target.Deferred.Name
		}
		if a.aliasPlanning && steps > 1 {
			// The classifier ends recursion, not the erased chain's source work.
			// Cache before nullable wrapping so false narrowing cannot erase hops.
			ty.SchemaSteps = steps
		}
		if nullable {
			ty = ty.Null()
		}
		return ty, nil // Nominal identity is a recursion and validation barrier.
	}
	var ty TypeRef
	kind, _, err := schemaType(v, p)
	if err != nil {
		return TypeRef{}, err
	}
	if kind == "array" || kind == "object" {
		var nested []Declaration
		ty, err = a.propertyType(v, "", p, &nested, map[string]bool{}, depth+steps)
		if err != nil {
			return TypeRef{}, err
		}
		// Finite admission guarantees no nominal planning or declaration here.
		a.hintInternalCollectionLeaves(ty)
	} else if a.options.Validation && has(v, "enum") {
		ty, err = a.nativeEnumValueType(v, p, depth+steps)
		if err != nil {
			return TypeRef{}, err
		}
	} else if a.options.Validation && hasPrimitiveBounds(v) {
		value, checks, err := a.boundedPrimitiveType(v, p, depth+steps)
		if err != nil {
			return TypeRef{}, err
		}
		ty = value
		if checks != nil {
			a.validationSources[checks] = p
			a.emissionHints[checks] = true
			if ty.Kind == Nullable {
				inner := *ty.Element
				inner.Validation = checks
				ty.Element = &inner
			} else {
				ty.Validation = checks
			}
		}
	} else {
		kind, nullable, err := schemaType(v, p)
		if err != nil {
			return TypeRef{}, err
		}
		scalar, err := scalarType(kind, v, p)
		if err != nil {
			return TypeRef{}, err
		}
		ty = TypeRef{Kind: Scalar, Name: scalar}
		if nullable {
			ty = ty.Null()
		}
	}
	if override != nil {
		if ty.Kind == Nullable {
			ty = *ty.Element
		}
		if *override {
			ty = ty.Null()
		}
	}
	if a.aliasPlanning {
		ty.SchemaSteps += steps
	}
	return ty, nil
}

// Only newly root-bound dictionary consumers acquire this backing view. Direct
// historical dictionary references keep their existing explicit-false behavior.
func bindingCollectionView(ty TypeRef) TypeRef {
	if ty.Kind == Nullable {
		inner := *ty.Element
		if inner.Kind == Named && inner.Element != nil {
			inner.ArrayAliasView = NullableArrayAlias
		}
		ty.Element = &inner
	}
	return ty
}

func nullableOverride(v *document.Value) *bool {
	if value, ok := v.Get("nullable"); ok {
		flag, _ := value.Bool()
		return &flag
	}
	return nil
}

// Wrapper annotations retain their existing precedence. For new borrowed uses,
// a reference member's explicit false must also be distinguishable from absence.
func wrapperNullableOverride(v *document.Value) *bool {
	if value := nullableOverride(v); value != nil {
		return value
	}
	var ref *document.Value
	for _, member := range get(v, "allOf").Items() {
		if has(member, "$ref") {
			ref = member
		} else if value := nullableOverride(member); value != nil {
			return value
		}
	}
	return nullableOverride(ref)
}
