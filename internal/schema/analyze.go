package schema

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Analyze validates every component schema, including unreferenced definitions. Failure
// returns neither declarations nor warnings; no consumer can emit partial files.
func Analyze(root *document.Value, options Options) (*Plan, []diagnostic.Diagnostic, *diagnostic.Diagnostic) {
	return analyzeDocument(root, options, false, nil)
}

// The source-validation branch owns a fresh analyzer and never returns an
// emission candidate to its caller. It shares immutable document nodes only.
func analyzeDocument(root *document.Value, options Options, tupleSourceValidation bool, tupleSourceProof **tupleNativeSourceRegistry) (*Plan, []diagnostic.Diagnostic, *diagnostic.Diagnostic) {
	referencedDefaults := []referencedFieldDefault{}
	structuralUniqueUses := []structuralUniqueUse{}
	objectDefaultGraphUses := []objectDefaultGraphUse{}
	a := &analyzer{referencedDefaults: &referencedDefaults, emissionHints: EmissionHints{}, options: options, root: root, components: map[string]*component{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	a.tupleSourceValidation = tupleSourceValidation
	a.tupleObjects = &tupleInlineDTOState{Seen: map[*document.Value]string{}}
	// HTTP-only documents can bypass component preparation. Share diagnostic
	// locations before request/response contexts copy the analyzer; this grants
	// no schema-origin or annotation permission.
	a.modelPointers = map[*Model]string{}
	a.modelDiagnosticPointers = map[*Model]string{}
	a.enumDiagnosticOrigins = map[*StringEnum]nominalOrigin{}
	a.structuralUniqueUses = &structuralUniqueUses
	a.objectDefaultGraphUses = &objectDefaultGraphUses
	declarations, err := a.analyze(root)
	if err != nil {
		return nil, nil, err
	}
	client, err := a.clientPlan(root)
	if err != nil {
		return nil, nil, err
	}
	if tupleSourceValidation {
		declarations, err = a.attachTupleSourceModels(declarations, client)
		if err != nil {
			return nil, nil, err
		}
	}
	if err := a.finalizeTupleCollectionUses(); err != nil {
		return nil, nil, err
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, client); err != nil {
		return nil, nil, err
	}
	if err := a.finalizeInlineEnumCollectionChildDefaults(declarations, client); err != nil {
		return nil, nil, err
	}
	if err := a.finalizeInlineBuiltinChildDefaults(declarations, client); err != nil {
		return nil, nil, err
	}
	if err := a.validateInternalModels(declarations); err != nil {
		return nil, nil, err
	}
	if err := a.aliasConsumerDepth(); err != nil {
		return nil, nil, err
	}
	plan := &Plan{Models: declarations, Client: client, EmissionHints: a.emissionHints, operationRoles: a.operationRoles, operationOrigins: a.operationOrigins, modelDiagnosticPointers: a.modelDiagnosticPointers, enumDiagnosticOrigins: a.enumDiagnosticOrigins}
	if err := a.validateTuplePlan(plan); err != nil {
		return nil, nil, err
	}
	if err := a.validationDepth(plan); err != nil {
		return nil, nil, err
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		return nil, nil, err
	}
	if err := a.validateFalseDTOFieldDefaults(declarations); err != nil {
		return nil, nil, err
	}
	if err := a.directionalModels(plan); err != nil {
		return nil, nil, err
	}
	if err := a.requestRuntime(plan); err != nil {
		return nil, nil, err
	}
	if err := a.scalarRuntime(plan); err != nil {
		return nil, nil, err
	}
	if err := a.validateStructuralUniqueUses(); err != nil {
		return nil, nil, err
	}
	if err := a.validateDTOUniqueUses(plan); err != nil {
		return nil, nil, err
	}
	if err := a.validateObjectDefaultGraphUses(); err != nil {
		return nil, nil, err
	}
	// Field-default and object-graph validation can add scratch uses after the
	// first freeze. Recheck existing proofs before freezing only those later uses.
	if err := a.validateInlineEnumCollectionChildDefaults(declarations, true); err != nil {
		return nil, nil, err
	}
	if err := a.finalizeInlineEnumCollectionChildDefaults(declarations, client); err != nil {
		return nil, nil, err
	}
	if err := a.validateInlineEnumCollectionChildDefaults(declarations, false); err != nil {
		return nil, nil, err
	}
	if err := a.validateInlineBuiltinChildDefaults(declarations, client, true); err != nil {
		return nil, nil, err
	}
	if err := a.finalizeInlineBuiltinChildDefaults(declarations, client); err != nil {
		return nil, nil, err
	}
	if err := a.validateInlineBuiltinChildDefaults(declarations, client, false); err != nil {
		return nil, nil, err
	}
	if !tupleSourceValidation {
		if err := a.runtimeCollisions(plan); err != nil {
			return nil, nil, err
		}
		if err := a.certainJVMSignatureOverflow(plan); err != nil {
			return nil, nil, err
		}
	}
	if tupleSourceValidation {
		if tupleSourceProof != nil {
			*tupleSourceProof = a.tupleNativeSources
		}
		return nil, nil, nil
	}
	return plan, a.warnings, nil
}

func (a *analyzer) analyze(root *document.Value) ([]Declaration, *diagnostic.Diagnostic) {
	if a.collectionChildDefaults == nil {
		a.collectionChildDefaults = map[*document.Value]*collectionReferenceChildDefault{}
	}
	if a.collectionChildDefaultOrder == nil {
		a.collectionChildDefaultOrder = &[]*collectionReferenceChildDefault{}
	}
	if a.inlineEnumChildDefaults == nil {
		a.inlineEnumChildDefaults = map[*document.Value]*inlineEnumCollectionChildDefault{}
	}
	if a.inlineEnumChildDefaultOrder == nil {
		a.inlineEnumChildDefaultOrder = &[]*inlineEnumCollectionChildDefault{}
	}
	if a.inlineBuiltinChildDefaults == nil {
		a.inlineBuiltinChildDefaults = map[*document.Value]*inlineBuiltinChildDefault{}
	}
	if a.inlineBuiltinChildDefaultOrder == nil {
		a.inlineBuiltinChildDefaultOrder = &[]*inlineBuiltinChildDefault{}
	}
	if a.checkedEnumSources == nil {
		a.checkedEnumSources = map[*document.Value]*checkedEnumSource{}
	}
	// HTTP-only inputs return before component preparation. Operation models
	// still need original source authority for directional views and defaults.
	if a.modelOrigins == nil {
		a.modelOrigins = map[*Model]*document.Value{}
	}
	if a.unionOrigins == nil {
		a.unionOrigins = map[Declaration]nominalOrigin{}
	}
	a.initTupleInlineDTOs()
	if a.tupleAllOfSources == nil {
		a.tupleAllOfSources = &tupleAllOfSourceBudget{Seen: map[*document.Value]bool{}}
	}
	a.initTupleBooleanReferences()
	a.initTupleLiterals()
	a.initTupleCovers()
	a.initTupleSpecials()
	a.initTupleBooleans()
	if a.tupleNativeSources == nil {
		a.tupleNativeSources = &tupleNativeSourceRegistry{}
	}
	a.initDTOUnique()
	if a.directionalModelSources == nil {
		a.directionalModelSources = map[*Model]directionalModelSource{}
	}
	version := text(get(root, "openapi"))
	parts := strings.Split(version, ".")
	valid := len(parts) == 3 && parts[0] == "3" && (parts[1] == "0" || parts[1] == "1") && parts[2] != ""
	if valid {
		for _, digit := range []byte(parts[2]) {
			valid = valid && digit >= '0' && digit <= '9'
		}
	}
	if !valid {
		return nil, diagnostic.At("E_OPENAPI_VERSION", "Expected an OpenAPI 3.0.x or 3.1.x version.", "/openapi")
	}
	for _, key := range []string{"paths", "webhooks"} {
		if value, ok := root.Get(key); ok {
			if value.Kind() != document.Object {
				return nil, invalid("Expected an object.", "/"+key)
			}
			if key == "webhooks" && len(value.Members()) != 0 {
				return nil, diagnostic.At("E_OPERATIONS_UNSUPPORTED", "Webhooks are not supported.", "/"+key)
			}
		}
	}
	hasPaths := has(root, "paths") && len(get(root, "paths").Members()) != 0
	components := get(root, "components")
	if hasPaths && !has(root, "components") {
		return nil, nil
	}
	if components.Kind() != document.Object {
		return nil, invalid("Expected components.schemas.", "/components")
	}
	for _, member := range components.Members() {
		if hasPaths && slices.Contains([]string{"parameters", "requestBodies", "responses"}, member.Name) {
			p := naming.Pointer("/components", member.Name)
			if err := object(member.Value, p); err != nil {
				return nil, err
			}
			for _, entry := range member.Value.Members() {
				if err := object(entry.Value, naming.Pointer(p, entry.Name)); err != nil {
					return nil, err
				}
			}
			continue
		}
		if member.Name != "schemas" && (member.Value.Kind() != document.Object || len(member.Value.Members()) != 0) {
			return nil, unsupported(naming.Pointer("/components", member.Name))
		}
	}
	schemas := get(components, "schemas")
	if hasPaths && !has(components, "schemas") {
		return nil, nil
	}
	if schemas.Kind() != document.Object || !hasPaths && len(schemas.Members()) == 0 {
		return nil, invalid("Expected a non-empty schemas object.", "/components/schemas")
	}
	ordered := slices.Clone(schemas.Members())
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	used := map[string]bool{}
	a.componentSchemas = schemas
	// A component's own root `default` emits an opt-in factory and never an
	// initializer. Only plain ordinary object components qualify; every other
	// role keeps its existing rejection at the original `.../default` Pointer.
	for _, entry := range schemas.Members() {
		pointer := naming.Pointer("/components/schemas", entry.Name)
		if !has(entry.Value, "default") {
			continue
		}
		if err := a.markComponentDefaultSource(entry.Value, pointer); err != nil {
			return nil, err
		}
	}
	// Catalog validation follows source declaration order, preserving which
	// original node receives a collision diagnostic. Declaration emission and
	// warning analysis below use sorted components.
	for _, entry := range schemas.Members() {
		v, wire := entry.Value, entry.Name
		pointer := naming.Pointer("/components/schemas", wire)
		booleanRoot := a.booleanTrueValue(v)
		falseRoot := a.booleanFalseRoot(v)
		if !booleanRoot && !falseRoot {
			if err := object(v, pointer); err != nil {
				return nil, err
			}
		}
		kind, nullable, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		recursiveJSON, err := a.recursiveJSONRoot(v, pointer)
		if err != nil {
			return nil, err
		}
		if kind == "" {
			switch {
			case booleanRoot:
				kind = "boolean_raw"
			case falseRoot:
				kind = "boolean_false"
			case has(v, "$ref"):
				kind = "scalar_ref"
			case has(v, "allOf"):
				kind = "object"
			case has(v, "enum"):
				kind = "string"
			case has(v, "x-native-type"):
				kind = "native"
			case has(v, "oneOf") || has(v, "anyOf"):
				kind = "union"
			}
		}
		if hasObjectCounts(v) && (kind != "object" || nullable && !dictionarySchema(v) && !(a.options.Validation && (directNullableDTOCountCandidate(v, pointer) || nullableAllOfDTOCountCandidate(v, pointer)))) {
			return nil, rejectObjectCounts(v, pointer)
		}
		nullableStructuredConst := nullable && structuredLiteral(v)
		nullableDTO := nullable && kind == "object" && nullableDTOCandidate(v)
		nullableEnum := nullable && kind == "string" && lexicalStringEnumSchema(v)
		nullableNativeEnum := a.options.Validation && nullable && nullableNativeEnumCandidate(v, kind)
		nullableValueClass := nullable && nullableValueClassCandidate(v, kind, pointer)
		nullableTemporal := nullable && nullableTemporalCandidate(v, kind)
		nullableBigint := nullable && nullableBigintCandidate(v, kind)
		nullableBytes := nullable && nullableBytesCandidate(v, kind)
		nullableCallerNative := nullable && nullableCallerNativeCandidate(v, kind)
		nullableRaw := nullable && kind == "object" && nullableRawCandidate(v)
		nullablePrimitive := nullable && nullablePrimitiveCandidate(v, kind)
		if nullable && !nullableStructuredConst && !nullableDTO && !nullableEnum && !nullableNativeEnum && !nullableValueClass && !nullableTemporal && !nullableBigint && !nullableBytes && !nullableCallerNative && !nullableRaw && !nullablePrimitive && kind != "union" && kind != "array" && kind != "scalar_ref" && !dictionarySchema(v) {
			return nil, unsupported(pointer)
		}
		scalar := ""
		valueClass := false
		var validation *PrimitiveValidation
		reference := ""
		if kind == "scalar_ref" {
			// A marked reference wrapper owns its root default; an unmarked one keeps
			// the original rejection here.
			refKeys := []string{"$ref", "nullable"}
			if a.componentDefaultSchemas[v] {
				refKeys = append(refKeys, "default")
			}
			if err := checkKeys(v, refKeys, pointer); err != nil {
				return nil, err
			}
			var ok bool
			reference, ok = get(v, "$ref").Text()
			if !ok {
				return nil, invalid("$ref must be a string.", pointer+"/$ref")
			}
			if reference == "" {
				return nil, diagnostic.At("E_REF_UNSUPPORTED", "Only local named component references are supported.", pointer+"/$ref")
			}
		} else if booleanRoot || falseRoot {
			// A literal root has no object keywords/scalar constraints. Preserve
			// its original node and explicit JSON role, not a fabricated Model.
		} else if has(v, "x-kotlin-value-class") {
			scalar, valueClass, err = valueClassScalar(v, kind, pointer)
			if err != nil {
				return nil, err
			}
			if valueClass {
				validation, err = a.primitiveValidation(v, scalar, pointer)
				if err != nil {
					return nil, err
				}
			}
		} else if has(v, "const") {
			value, e := valueConstant("validation", wire, v, pointer, 0)
			if e != nil {
				return nil, e
			}
			scalar = bare(value.Type).Name
		} else {
			if !slices.Contains([]string{"object", "array", "string", "integer", "number", "boolean", "null", "x-void", "native", "union"}, kind) {
				return nil, unsupported(pointer)
			}
			if kind == "null" && has(v, "enum") {
				scalar, err = nullOnlyEnumProfile(v, pointer, a.componentDefaultSchemas[v])
			} else if (kind == "integer" || kind == "number" || kind == "boolean") && has(v, "enum") {
				scalar, err = scalarMapping(kind, v, pointer)
			} else if kind != "object" && kind != "array" && kind != "union" && !has(v, "enum") {
				if a.options.Validation && hasPrimitiveBounds(v) {
					ty, bounds, e := a.modelPropertyType(v, "Validation", pointer, nil, nil, 0)
					err = e
					scalar = bare(ty).Name
					validation = bounds
				} else {
					scalar, err = scalarTypeFor(a, kind, v, pointer)
				}
			}
			if err != nil {
				return nil, err
			}
		}
		name, err := naming.ClassName(wire, pointer)
		if err != nil {
			return nil, err
		}
		name += a.options.Suffix
		if used[strings.ToLower(name)] {
			return nil, diagnostic.At("E_NAME_COLLISION", "Component names resolve to the same case-insensitive Kotlin filename.", pointer)
		}
		used[strings.ToLower(name)] = true
		if naming.ReservedPathSegment(name) {
			return nil, diagnostic.At("E_NAME_INVALID", "Model name is a reserved filesystem name.", pointer)
		}
		if validation != nil {
			a.validationSources[validation] = pointer
		}
		a.components[wire] = &component{Wire: wire, Name: name, Scalar: scalar, Pointer: pointer, Schema: v, ValueClass: valueClass, Validation: validation, Reference: reference, NullableDTO: nullableDTO, NullableEnum: nullableEnum, NullableNativeEnum: nullableNativeEnum, NullableValueClass: nullableValueClass, NullableTemporal: nullableTemporal, NullableBigint: nullableBigint, NullableBytes: nullableBytes, NullableCallerNative: nullableCallerNative, NullableRaw: nullableRaw, NullablePrimitive: nullablePrimitive, NullableStructuredConst: nullableStructuredConst, BooleanRoot: booleanRoot, RecursiveJSON: recursiveJSON != 0}
		a.components[wire].FalseRoot = falseRoot
		if a.checkedStringEnumRoot(v, pointer) {
			checks, failure := a.checkedEnumValidation(v, pointer)
			if failure != nil {
				return nil, failure
			}
			a.components[wire].Validation = checks
			a.checkedEnumSources[v].Root = a.components[wire]
		}
		if booleanRoot {
			a.components[wire].AliasType = TypeRef{Kind: JSON}
		} else if recursiveJSON != 0 {
			// Retain original graph work after projection to a JSON carrier.
			a.components[wire].AliasType = TypeRef{Kind: JSON, SchemaSteps: recursiveJSON}
		}
		if a.options.Validation && has(v, "enum") && !has(v, "x-native-type") && !has(v, "const") && !has(v, "x-kotlin-value-class") && (kind == "integer" || kind == "number" || kind == "boolean") {
			// A component root's own `default` is owned by the factory planner, so
			// keep the field role's membership/validation semantics here and only
			// let the key check accept the annotation.
			_, checks, e := a.nativeEnumFieldType(field{Schema: v, Pointer: pointer}, false)
			if e != nil {
				return nil, e
			}
			a.components[wire].Validation = checks
			if a.componentDefaultSchemas[v] {
				// The factory planner re-runs this profile on a warning-free copy, so
				// emit the component's own approximation warning here exactly once.
				a.nativeEnumWarn(kind, v, pointer)
			}
		}
	}
	// Root aliases need physical target names before catalog binding. Hierarchy
	// planning may subsequently exclude an apparent owner; retain provenance and
	// require membership again before emission and final declaration validation.
	a.modelOrigins = map[*Model]*document.Value{}
	a.modelPointers = map[*Model]string{}
	a.declarationOwners = map[Declaration]*component{}
	a.unionOrigins = map[Declaration]nominalOrigin{}
	a.hierarchyOrigins = map[*Hierarchy]nominalOrigin{}
	a.enumOrigins = map[Declaration]nominalOrigin{}
	a.valueClassOrigins = map[Declaration]nominalOrigin{}
	a.temporalOrigins = map[Declaration]temporalOrigin{}
	a.temporalUses = map[*component]string{}
	a.nativeEnumOrigins = map[*component]*nativeEnumOrigin{}
	a.nativeEnumUses = map[*component]string{}
	a.builtinOrigins = map[Declaration]builtinOrigin{}
	a.builtinUses = map[*component]string{}
	a.namedCollectionOrigins = map[Declaration]namedCollectionOrigin{}
	a.namedCollectionUses = map[*component]string{}
	a.recursiveCoverOrigins = map[Declaration]recursiveCoverOrigin{}
	a.recursiveCoverUses = map[*component]string{}
	a.rawTrueOrigins = map[Declaration]rawTrueOrigin{}
	a.rawTrueUses = map[*component]string{}
	a.callerNativeOrigins = map[Declaration]callerNativeOrigin{}
	a.callerNativeUses = map[*component]string{}
	a.bytesOrigins = map[Declaration]bytesOrigin{}
	a.bytesUses = map[*component]string{}
	a.bigintOrigins = map[Declaration]bigintOrigin{}
	a.bigintUses = map[*component]string{}
	a.borrowedScalars = map[*document.Value]*borrowedScalarSource{}
	a.borrowedRoots = map[Declaration]borrowedRootOrigin{}
	a.borrowedCollections = map[*document.Value]*borrowedCollectionSource{}
	a.borrowedCollectionRoots = map[Declaration]borrowedCollectionRoot{}
	a.borrowedDTOLeaves = map[*document.Value]*borrowedDTOLeafUse{}
	a.borrowedEnumLeaves = map[*document.Value]*borrowedEnumLeafUse{}
	a.borrowedValueClassLeaves = map[*component]*borrowedValueClassLeafUse{}
	a.borrowedAllOfLeaves = map[*document.Value]*borrowedAllOfLeafUse{}
	a.falseRootSources = nil
	a.falseValueOrigins = map[*FalseValue]falseValueOrigin{}
	a.falseAliasOrigins = map[*ValueAlias]falseAliasOrigin{}
	a.falseAliasFactories = map[*ValueAlias]falseCollectionFactoryOrigin{}
	a.falseAbsentUses = map[*component]bool{}
	a.falseFieldOrigins = map[*Model]falseFieldOrigin{}
	a.literalFalseFieldOrigins = map[*Model]literalFalseFieldOrigin{}
	a.literalFalseFieldUses = map[*Model]string{}
	a.literalFalseObjectDefaultUses = map[*Model]bool{}
	a.literalFalseObjectFactories = map[*Model]falseCollectionFactoryOrigin{}
	a.falseInlineCollectionOrigins = map[*Model]falseInlineCollectionOrigin{}
	a.falseInlineCollectionUses = map[*Model]string{}
	a.falseObjectDefaultUses = map[*Model]bool{}
	a.falseObjectFactories = map[*Model]falseCollectionFactoryOrigin{}
	a.falseDTOFieldDefaults = map[falseDTOFieldDefaultKey]falseDTOFieldDefaultOrigin{}
	a.falseDTOFieldDefaultUses = map[falseDTOFieldDefaultKey]string{}
	a.falseDTOConsumerSources = map[*Model][]borrowedAllOfNode{}
	a.falseDTOOwnerDefaults = map[*Model]*falseDTOOwnerDefaultOrigin{}
	a.falseDTOOwnerDefaultUses = map[falseDTOFieldDefaultKey]string{}
	a.falseDTOOwnerAliasSources = map[*component][]borrowedAllOfNode{}
	a.falseCollectionOrigins = map[*ValueAlias]falseCollectionOrigin{}
	a.falseCollectionFactories = map[*ValueAlias]falseCollectionFactoryOrigin{}
	a.intersectionCaptures = &intersectionCaptureBudget{Seen: map[*document.Value]bool{}}
	a.borrowedUnionLeaves = map[*document.Value]*borrowedUnionLeafUse{}
	a.borrowedHierarchyLeaves = map[*document.Value]*borrowedHierarchyLeafUse{}
	a.borrowedBuiltinLeaves = map[*component]*borrowedBuiltinLeafUse{}
	a.borrowedTemporalLeaves = map[*component]*borrowedTemporalLeafUse{}
	a.borrowedBigintLeaves = map[*component]*borrowedBigintLeafUse{}
	a.borrowedBytesLeaves = map[*component]*borrowedBytesLeafUse{}
	a.borrowedCallerNativeLeaves = map[*component]*borrowedCallerNativeLeafUse{}
	a.borrowedRawTrueLeaves = map[*component]*borrowedRawTrueLeafUse{}
	a.borrowedRecursiveCoverLeaves = map[*component]*borrowedRecursiveCoverLeafUse{}
	a.borrowedDTOAliasEdges = map[*component]*borrowedDTOAliasEdge{}
	a.deferredNominals = map[*document.Value]*deferredNominal{}
	a.indexInternalModels(nil)
	// Inline hierarchy branches are registered as synthetic components before the
	// first consumer walks a.components by name.
	if err := a.hierarchyInlineLeaves(used); err != nil {
		return nil, err
	}
	// Synthetic inline leaves are appended so the declaration loop below consumes
	// them through the same path as a named component.
	for _, leaf := range a.hierarchyInlineOrder {
		ordered = append(ordered, document.Member{Name: leaf.Wire, Value: leaf.Schema})
	}
	if err := a.discoverHierarchySources(); err != nil {
		return nil, err
	}
	for _, entry := range ordered {
		c := a.components[entry.Name]
		if c != nil && has(c.Schema, "allOf") && nullableDTOCandidate(c.Schema) && !a.hierarchySources[c.Schema] && !a.containsObjectCompositionUnion(c.Schema, 0, map[*document.Value]bool{}) {
			value, err := a.compositionNullable(c.Schema, c.Pointer, 0, &compositionBudget{remaining: 16384, enabled: true}, map[*document.Value]bool{})
			if err == nil && value {
				c.NullableDTO = true
			}
		}
	}
	for _, c := range a.components {
		// A retained allOf product follows the existing standalone union
		// Null-carrier policy; it is not an ordinary nullable DTO classifier.
		if c.NullableDTO && a.containsObjectCompositionUnion(c.Schema, 0, map[*document.Value]bool{}) {
			c.NullableDTO = false
		}
		_, nullable, err := schemaType(c.Schema, c.Pointer)
		if err != nil {
			return nil, err
		}
		if (nullable || c.NullableHierarchy) && a.hierarchySources[c.Schema] {
			c.NullableHierarchy = true
			c.NullableDTO, c.NullableRaw = false, false
		}
	}
	if err := a.resolveScalarAliases(schemas); err != nil {
		return nil, err
	}
	hierarchy, err := a.planHierarchy(used)
	if err != nil {
		return nil, err
	}
	if err := a.linkDeferredNominals(hierarchy); err != nil {
		return nil, err
	}
	// The provisional ownership index depends only on immutable source nodes
	// and catalog names. Rebuild only when hierarchy ownership filters it.
	if len(hierarchy) != 0 {
		a.indexInternalModels(hierarchy)
	}
	for _, c := range a.components {
		if c.NullableHierarchy && hierarchy[c.Wire] == nil {
			return nil, unsupported(c.Pointer)
		}
	}
	for _, entry := range ordered {
		c := a.components[entry.Name]
		if target := c.InternalModelTarget; target != nil && target.Deferred == nil && target.CanonicalTemporal == nil && target.CanonicalBuiltin == nil && target.CanonicalBigint == nil && target.CanonicalBytes == nil && target.CanonicalCallerNative == nil && target.CanonicalJSON == nil && target.CanonicalCover == nil && target.CanonicalCollection == nil && target.BorrowedCollection == nil && target.BorrowedScalar == nil && a.physicalDTOName(target.Schema) != target.ModelName {
			return nil, schemaReferenceUnsupported(c.Pointer + "/$ref")
		}
	}
	var declarations []Declaration
	for _, entry := range ordered {
		if node := hierarchy[entry.Name]; node != nil {
			declarations = append(declarations, node)
			continue
		}
		component := a.components[entry.Name]
		v, pointer := component.Schema, component.Pointer
		switch {
		case component.FalseRoot:
			model := &FalseValue{ClassName: component.Name}
			if err := a.recordFalseValueOrigin(model, component); err != nil {
				return nil, err
			}
			declarations = append(declarations, model)
		case component.RecursiveJSON:
			a.warn("W_RECURSIVE_JSON_PROJECTION", "A complete recursive JSON-kind union uses JsonElement, including JsonNull, not a general schema or oneOf match-count validator.", pointer)
			alias := &ValueAlias{ClassName: component.Name, Type: component.AliasType}
			a.recordRecursiveCoverOrigin(alias, component)
			declarations = append(declarations, alias)
		case component.BooleanRoot:
			if _, err := a.booleanValueType(v, pointer, 0); err != nil {
				return nil, err
			}
			alias := &ValueAlias{ClassName: component.Name, Type: component.AliasType}
			a.recordRawTrueOrigin(alias, component)
			declarations = append(declarations, alias)
		case component.InternalModelTarget != nil:
			ty := a.componentType(component)
			// Preserve all erased named/internal hops in transparent alias caches;
			// the canonical DTO is the barrier, not the aliases leading to it.
			ty.SchemaSteps = component.AliasHeight
			if component.InternalModelTarget.CanonicalCollection != nil {
				// The linked Named view retains the final canonical edge itself.
				ty.SchemaSteps--
			}
			if cover := component.InternalModelTarget.CanonicalCover; cover != nil {
				ty.SchemaSteps += a.recursiveJSONHeights[cover.Schema]
			}
			alias := &ValueAlias{ClassName: component.Name, Type: ty}
			a.recordBorrowedRootOrigin(alias, component)
			a.recordBorrowedCollectionRoot(alias, component)
			declarations = append(declarations, alias)
		case component.BindingTarget != nil:
			ty := a.componentType(component)
			if collectionAliasSchema(component.BindingTarget.Schema) || component.BindingTarget.RecursiveJSON {
				// Canonical spelling erases intermediate alias nodes. Named type
				// traversal already counts one hop; retain the remaining work.
				ty.SchemaSteps = component.AliasHeight - 1
			}
			alias := &ValueAlias{ClassName: component.Name, Type: ty}
			if component.BindingTarget.FalseRoot {
				if err := a.recordFalseAliasOrigin(alias, component); err != nil {
					return nil, err
				}
			}
			declarations = append(declarations, alias)
		case component.ValueClass:
			model := &PrimitiveValueClass{ClassName: component.Name, Scalar: component.Scalar, Validation: component.Validation}
			a.valueClassOrigins[model] = nominalOrigin{v, pointer, component.NullableValueClass}
			declarations = append(declarations, model)
		case component.NullableStructuredConst || component.NullableTemporal || component.NullableBigint || component.NullableBytes || component.NullableCallerNative || component.NullablePrimitive || component.NullableNativeEnum:
			if component.NullableStructuredConst {
				a.warnValueConst(v, pointer)
			}
			model := &ValueAlias{ClassName: component.Name, Type: a.componentType(component)}
			a.recordNativeEnumOrigin(model, component)
			a.recordTemporalOrigin(model, component)
			a.recordBuiltinOrigin(model, component)
			a.recordBigintOrigin(model, component)
			a.recordBytesOrigin(model, component)
			a.recordCallerNativeOrigin(model, component)
			declarations = append(declarations, model)
		case component.Scalar != "":
			if has(v, "enum") && (component.Validation == nil || len(component.Validation.Enum) == 0) {
				kind, _, _ := schemaType(v, pointer)
				if _, err := a.nativeEnum(kind, v, pointer); err != nil {
					return nil, err
				}
			} else if has(v, "const") && component.Scalar != "kotlinx.serialization.json.JsonNull" {
				a.warnValueConst(v, pointer)
			}
			model := &ScalarAlias{ClassName: component.Name, Scalar: component.Scalar}
			a.recordNativeEnumOrigin(model, component)
			a.recordTemporalOrigin(model, component)
			a.recordBuiltinOrigin(model, component)
			a.recordBigintOrigin(model, component)
			a.recordBytesOrigin(model, component)
			a.recordCallerNativeOrigin(model, component)
			declarations = append(declarations, model)
		case has(v, "enum"):
			model, err := buildStringEnumKeysFor(component.Name, v, pointer, false, a)
			if err != nil {
				return nil, err
			}
			a.recordEnumOrigin(model, v, pointer, component.NullableEnum)
			if a.checkedStringEnumRoot(v, pointer) {
				a.checkedEnumSources[v].Declarations[model] = true
			}
			declarations = append(declarations, model)
		case has(v, "oneOf") || has(v, "anyOf"):
			model, err := a.rootUnion(v, component.Name, pointer, used)
			if err != nil {
				return nil, err
			}
			declarations = append(declarations, model)
		default:
			if has(v, "allOf") {
				view, err := a.objectCompositionUnion(v, pointer, 0)
				if err != nil {
					return nil, err
				}
				if view != nil {
					if view.Nullable && len(view.Branches) >= MaxMembers {
						return nil, diagnostic.At("E_UNION_LIMIT", "Root union exceeds the supported 240 branch limit, including its null variant.", view.Pointer)
					}
					model, err := a.objectUnion(view, component.Name, 1, true)
					if err != nil {
						return nil, err
					}
					if model == nil {
						return nil, unsupported(pointer)
					}
					if err := reserveObjectUnion(model, view, pointer, true, used); err != nil {
						return nil, err
					}
					a.recordUnionOrigin(model, v, pointer, false)
					declarations = append(declarations, model)
					continue
				}
			}
			if collectionAliasSchema(v) {
				var peers []Declaration
				context := *a
				context.aliasPlanning = true
				context.falseCollectionPlanning = a.falseCollectionRootSource(component)
				var ty TypeRef
				var err *diagnostic.Diagnostic
				if leaf := inlineStringEnumBoundsLeaf(v, 1); a.options.Validation && leaf != nil {
					var node *nestedDefaultNode
					// The actual original collection component owns these enum peers.
					// Forward consumers retain the same linked AliasType/check objects.
					node, err = context.nestedDefaultSchemaWithEnumBounds(v, component.Name, pointer, &peers, used, 0, true, leaf)
					if err == nil {
						ty = node.ty
					}
				} else {
					ty, err = context.propertyType(v, component.Name, pointer, &peers, used, 0)
				}
				a.warnings = context.warnings
				if err != nil {
					return nil, err
				}
				component.AliasType = ty
				for _, peer := range peers {
					a.declarationOwners[peer] = component
				}
				alias := &ValueAlias{ClassName: component.Name, Type: ty}
				a.recordNamedCollectionOrigin(alias, component)
				if context.falseCollectionPlanning != nil {
					if err := a.recordFalseCollectionOrigin(alias, *context.falseCollectionPlanning); err != nil {
						return nil, err
					}
				}
				declarations = append(declarations, alias)
				declarations = append(declarations, peers...)
				continue
			}
			model, err := a.buildModel(component.Name, v, pointer, 0)
			if err != nil {
				return nil, err
			}
			if model.Nullable && !(component.NullableDTO && len(model.Properties) != 0 || component.NullableRaw && len(model.Properties) == 0 && len(model.Constants) == 0) {
				return nil, unsupported(pointer)
			}
			a.recordRawTrueOrigin(model, component)
			declarations = append(declarations, model)
		}
	}
	if err := a.planComponentDefaultFactories(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedDTOLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedEnumLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedValueClassLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedAllOfLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedUnionLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedHierarchyLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedBuiltinLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedTemporalLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedBigintLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedBytesLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedCallerNativeLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedRawTrueLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedRecursiveCoverLeaves(declarations); err != nil {
		return nil, err
	}
	if err := a.validateNamedCollectionBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateRecursiveCoverBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateRawTrueBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateCallerNativeBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBytesBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBigintBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBorrowedRootBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateBuiltinBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.validateTemporalBindings(declarations); err != nil {
		return nil, err
	}
	if err := a.linkNominalDeclarations(hierarchy, declarations, true); err != nil {
		return nil, err
	}
	// Nullable bindings retain an eligible final ordinary DTO, plain String
	// vocabulary, primitive value class or scalar. Empty final ordinary Models
	// retain the same raw projection below; hierarchy/union identities stay out.
	byName := map[string]Declaration{}
	for _, d := range declarations {
		byName[d.Name()] = d
	}
	for _, entry := range ordered {
		c := a.components[entry.Name]
		if c.BindingTarget != nil && c.BindingNullable != nil && *c.BindingNullable && !collectionAliasSchema(c.BindingTarget.Schema) {
			if _, ok := byName[c.BindingTarget.Name].(*FalseValue); ok && c.BindingTarget.FalseRoot {
				continue
			}
			if _, ok := byName[c.BindingTarget.Name].(*Hierarchy); ok && a.hierarchySources[c.BindingTarget.Schema] {
				continue
			}
			if unionNominalCandidate(c.BindingTarget.Schema) || composedModelCandidate(c.BindingTarget.Schema) {
				switch byName[c.BindingTarget.Name].(type) {
				case *ObjectUnion, *PrimitiveUnion:
					continue
				}
			}
			if c.BindingTarget.BooleanRoot || c.BindingTarget.RecursiveJSON {
				if alias, ok := byName[c.BindingTarget.Name].(*ValueAlias); ok && alias.Type.Kind == JSON {
					continue
				}
			}
			if _, ok := byName[c.BindingTarget.Name].(*StringEnum); ok && lexicalStringEnumBinding(c.BindingTarget) {
				continue
			}
			if _, ok := byName[c.BindingTarget.Name].(*PrimitiveValueClass); ok && c.BindingTarget.ValueClass {
				continue
			}
			if temporalBinding(c.BindingTarget) || bigintBinding(c.BindingTarget) || bytesBinding(c.BindingTarget) || callerNativeBinding(c.BindingTarget) || builtinPrimitiveBinding(c.BindingTarget) || a.options.Validation && nativeEnumBinding(c.BindingTarget) {
				switch byName[c.BindingTarget.Name].(type) {
				case *ScalarAlias:
					continue
				case *ValueAlias:
					if c.BindingTarget.NullableTemporal || c.BindingTarget.NullableBigint || c.BindingTarget.NullableBytes || c.BindingTarget.NullableCallerNative || c.BindingTarget.NullablePrimitive || c.BindingTarget.NullableNativeEnum {
						continue
					}
				}
			}
			if structuredConstBinding(c.BindingTarget) {
				switch target := byName[c.BindingTarget.Name].(type) {
				case *ScalarAlias:
					if target.Scalar == c.BindingTarget.Scalar {
						continue
					}
				case *ValueAlias:
					if c.BindingTarget.NullableStructuredConst && bare(target.Type).Kind == Scalar && bare(target.Type).Name == c.BindingTarget.Scalar {
						continue
					}
				}
			}
			if _, ok := byName[c.BindingTarget.Name].(*Model); !ok {
				return nil, unsupported(c.Pointer)
			}
		}
	}
	// Final declarations, not object catalog syntax, define raw versus nominal
	// identity. Link forward raw users before transparent alias validation.
	a.linkRawProjections(declarations)
	a.finalizeTupleCollectionWork()
	sort.Slice(declarations, func(i, j int) bool { return declarations[i].Name() < declarations[j].Name() })
	if err := a.validateAliases(declarations); err != nil {
		return nil, err
	}
	// Cached source heights now own temporary lowering-work annotations. Remove
	// them before semantic model canonicalization and emission; aliases retain
	// the cached resource height separately from their Kotlin/wire identity.
	var clearWork func(*TypeRef)
	clearWork = func(ty *TypeRef) {
		ty.SchemaSteps = 0
		if ty.Element != nil {
			clearWork(ty.Element)
		}
	}
	for _, c := range a.components {
		clearWork(&c.AliasType)
	}
	for _, d := range declarations {
		if alias, ok := d.(*ValueAlias); ok {
			clearWork(&alias.Type)
		}
	}
	// Newly admitted root arrays need the same bounded constructor checks as
	// borrowed collection trees. Wait until forward bindings are complete and
	// transparent cycles rejected; dictionary-only inputs keep their old text.
	for _, declaration := range declarations {
		if alias, ok := declaration.(*ValueAlias); ok && bare(alias.Type).Kind == List {
			a.hintInternalCollectionLeaves(alias.Type)
		}
	}
	for _, c := range a.components {
		if c.BindingTarget != nil && collectionAliasSchema(c.BindingTarget.Schema) {
			a.hintInternalCollectionLeaves(a.componentType(c))
		}
	}
	// Original named backing addresses are established before consumers, but
	// forward values become stable only after all aliases have been linked.
	// Capture their actual values once here for the final/late authority barrier.
	a.finalizeNamedCollectionOrigins()
	a.finalizeReferencedEnumCollectionFactories()
	a.finalizeBorrowedCollectionSources()
	if err := a.finalizeCheckedEnumRootGetters(); err != nil {
		return nil, err
	}
	a.finalizeBorrowedDTOAliasEdges()
	a.finalizeBorrowedEnumLeaves()
	if err := a.finalizeInternalEnumFactories(); err != nil {
		return nil, err
	}
	a.finalizeBorrowedAllOfLeaves()
	a.finalizeBorrowedUnionLeaves()
	a.finalizeBorrowedHierarchyLeaves()
	a.recordDTOReferenceDeclarations(declarations)
	if err := a.finalizeTupleCollectionUses(); err != nil {
		return nil, err
	}
	return declarations, nil
}

type unionBranch struct {
	Schema  *document.Value
	Pointer string
	Shape   *shape // Borrowed union/allOf alternative, never an expanded AST.
	Depth   int
}

type unionView struct {
	Branches                            []unionBranch
	Pointer                             string
	Nullable, HasNull, OverlappingNulls bool
	OneOf                               bool
	NullSources                         int
}

func (a *analyzer) readUnion(v *document.Value, pointer string) (*unionView, *diagnostic.Diagnostic) {
	keyword := ""
	if has(v, "oneOf") {
		keyword = "oneOf"
	} else if has(v, "anyOf") {
		keyword = "anyOf"
	} else {
		return nil, nil
	}
	location := naming.Pointer(pointer, keyword)
	if has(v, "type") {
		return nil, unsupported(location)
	}
	keys := []string{keyword, "nullable"}
	if a != nil && a.objectDefaultSchemas[v] {
		// A union field own default is validated by its dedicated planner; the
		// shared type plan borrows the same node and must allow the annotation.
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	branches := get(v, keyword)
	if branches.Kind() != document.Array || len(branches.Items()) == 0 {
		return nil, invalid("Composition must be a non-empty schema array.", location)
	}
	view := &unionView{Pointer: location, Nullable: nullable, OneOf: keyword == "oneOf"}
	nullCount, memberNullCount := 0, 0
	for i, branch := range branches.Items() {
		p := naming.Pointer(location, strconv.Itoa(i))
		if err := object(branch, p); err != nil {
			return nil, err
		}
		if text(get(branch, "type")) == "null" || inferredNullLiteral(branch) {
			if err := checkKeys(branch, []string{"type"}, p); err != nil {
				return nil, err
			}
			nullCount++
		} else {
			kind, memberNullable, err := schemaType(branch, p)
			if err != nil {
				return nil, err
			}
			if memberNullable && kind == "object" && !has(branch, "allOf") && !has(branch, "oneOf") && !has(branch, "anyOf") && !has(branch, "$ref") {
				view.Nullable = true
				memberNullCount++
			}
			if len(view.Branches) >= MaxMembers {
				return nil, diagnostic.At("E_UNION_LIMIT", "Union exceeds the supported 240 non-null branch limit; simplify the composition.", location)
			}
			view.Branches = append(view.Branches, unionBranch{Schema: branch, Pointer: p})
		}
	}
	if len(view.Branches) == 0 {
		return nil, unsupported(location)
	}
	view.Nullable = view.Nullable || nullCount > 0
	view.HasNull = nullCount > 0
	view.NullSources = nullCount + memberNullCount
	view.OverlappingNulls = keyword == "oneOf" && view.NullSources > 1
	return view, nil
}

func reserveNested(used map[string]bool, name, pointer string) *diagnostic.Diagnostic {
	if used[strings.ToLower(name)] {
		return diagnostic.At("E_NAME_COLLISION", "Nested type collides with another declaration or a serializer-owned name.", pointer)
	}
	used[strings.ToLower(name)] = true
	return nil
}

func (a *analyzer) propertyType(v *document.Value, nestedName, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if ty, err := a.booleanValueType(v, pointer, depth); err != nil {
		return TypeRef{}, err
	} else if ty != nil {
		return *ty, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, err
	}
	if err := object(v, pointer); err != nil {
		return TypeRef{}, err
	}
	union, err := a.objectCompositionUnion(v, pointer, depth)
	if err != nil {
		return TypeRef{}, err
	}
	if union == nil {
		union, err = a.readUnion(v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
	}
	if union != nil {
		start := len(*nested)
		ty, err := a.unionType(union, nestedName, pointer, nested, used, depth, false)
		if err == nil && bare(ty).Kind == Named {
			for _, d := range (*nested)[start:] {
				if d.Name() == bare(ty).Name {
					a.recordUnionOrigin(d, v, pointer, ty.Kind == Nullable)
					if a.tupleObjects != nil && a.tupleObjects.Unions[v] != nil {
						a.recordTupleUnionDeclaration(d, v, pointer)
					}
				}
			}
		}
		return ty, err
	}
	return a.nonUnionPropertyType(v, nestedName, pointer, nested, used, depth)
}

// unionType shares normalization and ownership rules, while keeping root
// request projection separate from nested/component payload wrappers.
func (a *analyzer) unionType(union *unionView, nestedName, pointer string, nested *[]Declaration, used map[string]bool, depth int, requestRoot bool) (TypeRef, *diagnostic.Diagnostic) {
	if err := a.deriveObjectReferenceNullability(union, depth+1); err != nil {
		return TypeRef{}, err
	}
	decision, err := a.normalizeNested(union, depth+1)
	if err != nil {
		return TypeRef{}, err
	}
	var ty TypeRef
	if decision != nil {
		if decision.Enum != nil {
			model, e := stringEnumValues(nestedName, decision.Enum, union.Pointer)
			if e != nil {
				return TypeRef{}, e
			}
			if e := reserveNested(used, nestedName, union.Pointer); e != nil {
				return TypeRef{}, e
			}
			*nested = append(*nested, model)
			ty = TypeRef{Kind: Named, Name: nestedName}
		} else {
			ty = TypeRef{Kind: Scalar, Name: decision.Scalar}
		}
	} else if len(union.Branches) == 1 && union.HasNull {
		branch := union.Branches[0]
		ty, err = a.propertyType(branch.Schema, nestedName, branch.Pointer, nested, used, depth+1)
	} else {
		model, e := a.objectUnionContext(union, nestedName, depth+1, false, requestRoot)
		if e != nil {
			return TypeRef{}, e
		}
		if model != nil {
			if e := reserveObjectUnion(model, union, pointer, false, used); e != nil {
				return TypeRef{}, e
			}
			*nested = append(*nested, model)
			ty = TypeRef{Kind: Named, Name: nestedName}
		} else if a.usage != SharedUsage || a.fieldUnions != NativeFieldUnions {
			ty, err = a.operationFieldUnion(union, nestedName, depth+1)
		} else {
			ty, err = a.primitiveUnion(union, nestedName, depth+1, false, nested, used)
		}
	}
	if err != nil {
		return TypeRef{}, err
	}
	if union.OverlappingNulls {
		a.warn("W_ONEOF_NULL_OVERLAP", "Nullable folding does not enforce the exact oneOf match count for overlapping null branches.", naming.Pointer(pointer, "oneOf"))
	}
	if union.Nullable {
		ty = ty.Null()
	}
	if a.aliasPlanning {
		// Branch traversal is source work even when normalization folds it to
		// a scalar or a zero-cost nullable type wrapper. Cache before clearing.
		ty.SchemaSteps++
	}
	return ty, nil
}

func (a *analyzer) nonUnionPropertyType(v *document.Value, nestedName, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	wrapper, err := referenceWrapper(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if wrapper != nil {
		if depth == MaxSchemaDepth {
			return TypeRef{}, diagnostic.At("E_SCHEMA_DEPTH", "Schema nesting exceeds the supported limit.", wrapper.Pointer)
		}
		target, err := a.resolveSchema(wrapper.Reference, wrapper.Pointer)
		if err != nil {
			return TypeRef{}, err
		}
		override := &wrapper.Nullable
		if target.Catalog == nil || target.CollectionBinding || target.NullableBinding || nullableArraySchema(target.Schema) {
			override = wrapperNullableOverride(v)
		}
		return a.referenceType(target, override, depth, wrapper.Pointer)
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if has(v, "allOf") {
		if kind != "" && kind != "object" {
			return TypeRef{}, unsupported(naming.Pointer(pointer, "allOf"))
		}
		kind = "object"
	}
	ownChild, err := a.inlineEnumCollectionChildDefault(v, pointer, depth)
	if err != nil {
		return TypeRef{}, err
	}
	ownBuiltinChild, err := a.inlineBuiltinCollectionChildDefault(v, pointer, depth)
	if err != nil {
		return TypeRef{}, err
	}
	var ty TypeRef
	switch {
	case has(v, "$ref"):
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return TypeRef{}, err
		}
		reference, ok := get(v, "$ref").Text()
		if !ok {
			return TypeRef{}, invalid("$ref must be a string.", pointer)
		}
		target, err := a.resolveSchema(reference, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		ty, err = a.referenceType(target, nullableOverride(v), depth, pointer)
		if err != nil {
			return TypeRef{}, err
		}
	case has(v, "x-native-type") && kind == "" && !has(v, "enum"):
		scalar, err := scalarType("native", v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		ty = TypeRef{Kind: Scalar, Name: scalar}
	case has(v, "enum") && kind == "null":
		scalar, err := nullOnlyEnumType(v, pointer)
		if err != nil {
			return TypeRef{}, err
		}
		ty = TypeRef{Kind: Scalar, Name: scalar}
	case has(v, "enum") && (kind == "integer" || kind == "number" || kind == "boolean"):
		if a.options.Validation {
			return a.nativeEnumValueTypeOwnDefault(v, pointer, depth, ownChild)
		}
		scalar, err := a.nativeEnumKeys(kind, v, pointer, ownChild)
		if err != nil {
			return TypeRef{}, err
		}
		ty = TypeRef{Kind: Scalar, Name: scalar}
	case has(v, "const"):
		value, err := valueConstant("validation", nestedName, v, pointer, depth)
		if err != nil {
			return TypeRef{}, err
		}
		if value.Type.Name != "kotlinx.serialization.json.JsonNull" {
			a.warnValueConst(v, pointer)
		}
		ty = value.Type
	case has(v, "enum"):
		model, err := buildStringEnumKeys(nestedName, v, pointer, ownChild)
		if err != nil {
			return TypeRef{}, err
		}
		if err := reserveNested(used, nestedName, pointer); err != nil {
			return TypeRef{}, err
		}
		a.recordEnumOrigin(model, v, pointer, nullable)
		*nested = append(*nested, model)
		ty = TypeRef{Kind: Named, Name: nestedName}
	default:
		switch kind {
		case "string", "integer", "number", "boolean", "null", "x-void":
			var scalar string
			var err *diagnostic.Diagnostic
			if ownBuiltinChild {
				scalar, err = scalarMapping(kind, v, pointer)
			} else {
				scalar, err = scalarType(kind, v, pointer)
			}
			if err != nil {
				return TypeRef{}, err
			}
			ty = TypeRef{Kind: Scalar, Name: scalar}
		case "array":
			decision, err := a.arrayItems(v, pointer, depth)
			if err != nil {
				return TypeRef{}, err
			}
			var item TypeRef
			if decision.Projection != nil {
				item = *decision.Projection
			} else {
				item, err = a.collectionElementType(decision.Selected, nestedName+"Inner", decision.Pointer, nested, used, depth+1)
				if err != nil {
					return TypeRef{}, err
				}
			}
			ty, err = a.checkedArrayItems(item, v, pointer, decision)
			if err != nil {
				return TypeRef{}, err
			}
		case "object":
			if dictionarySchema(v) {
				ty, err = a.dictionaryType(v, nestedName, pointer, nested, used, depth)
				if err != nil {
					return TypeRef{}, err
				}
				break
			}
			model, err := a.buildModel(nestedName, v, pointer, depth)
			if err != nil {
				return TypeRef{}, err
			}
			nullable = model.Nullable
			if len(model.Properties) == 0 && len(model.Constants) == 0 {
				ty = TypeRef{Kind: JSON}
			} else {
				if err := reserveNested(used, nestedName, pointer); err != nil {
					return TypeRef{}, err
				}
				*nested = append(*nested, model)
				ty = TypeRef{Kind: Named, Name: nestedName}
			}
		case "":
			keys := []string{"nullable"}
			if a.objectDefaultSchemas[v] {
				// A raw JsonElement field own default is validated separately.
				keys = append(keys, "default")
			}
			if err := checkKeys(v, keys, pointer); err != nil {
				return TypeRef{}, err
			}
			ty = TypeRef{Kind: JSON}
		default:
			return TypeRef{}, unsupported(naming.Pointer(pointer, "type"))
		}
	}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}

func (a *analyzer) componentType(component *component) TypeRef {
	if target := component.InternalModelTarget; target != nil {
		if canonical := target.CanonicalTemporal; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalCollection; canonical != nil {
			ty := a.namedCollectionType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if target.CanonicalCover != nil {
			ty := TypeRef{Kind: JSON}
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalJSON; canonical != nil {
			ty := a.rawTrueType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalCallerNative; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalBytes; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalBigint; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if source := target.BorrowedCollection; source != nil {
			ty := source.Type.WithoutRootNull()
			ty.SchemaSteps = 0
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if source := target.BorrowedScalar; source != nil {
			ty := source.Type.WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalNativeEnum; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.CanonicalBuiltin; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		if canonical := target.PhysicalValueClass; canonical != nil {
			ty := a.componentType(canonical).WithoutRootNull()
			if component.BindingNullable != nil && *component.BindingNullable {
				ty = ty.Null()
			}
			return ty
		}
		ty := TypeRef{Kind: Named, Name: target.ModelName, deferredNominal: target.Deferred}
		if a.checkedEnumReferenceCandidate(target.Schema) {
			// Errors are reported by original source planning; valid forward
			// aliases retain the same source check object before name linking.
			ty.Validation, _ = a.checkedEnumValidation(target.Schema, target.Pointer)
		}
		if target.Deferred != nil {
			ty.Name = target.Deferred.Name
		}
		if component.BindingNullable != nil && *component.BindingNullable {
			ty = ty.Null()
		}
		return ty
	}
	if component.BindingTarget != nil {
		ty := a.componentType(component.BindingTarget)
		if collectionAliasSchema(component.BindingTarget.Schema) {
			ty = bindingCollectionView(ty)
		}
		if component.BindingNullable != nil {
			if *component.BindingNullable {
				ty = ty.Null()
			} else {
				ty = ty.WithoutRootNull()
			}
		}
		return ty
	}
	if component.ValueClass {
		scalar := TypeRef{Kind: Scalar, Name: component.Scalar}
		ty := TypeRef{Kind: InlineValue, Name: a.options.PackageName + ".models." + component.Name, Element: &scalar}
		if component.NullableValueClass {
			ty = ty.Null()
		}
		return ty
	}
	if component.Scalar != "" {
		ty := TypeRef{Kind: Scalar, Name: component.Scalar, Validation: component.Validation}
		if component.NullableStructuredConst || component.NullableTemporal || component.NullableBigint || component.NullableBytes || component.NullableCallerNative || component.NullablePrimitive || component.NullableNativeEnum {
			ty = ty.Null()
		}
		return ty
	}
	ty := TypeRef{Kind: Named, Name: a.options.PackageName + ".models." + component.Name}
	if a.checkedStringEnumRoot(component.Schema, component.Pointer) {
		ty.Validation = component.Validation
	}
	if component.NullableDTO || component.NullableEnum || component.NullableHierarchy {
		ty = ty.Null()
	}
	if component.NullableRaw {
		ty.Element = &component.AliasType
		ty.ArrayAliasView = NullableArrayAlias
		return ty.Null()
	}
	if collectionAliasSchema(component.Schema) || component.AliasType.Kind == JSON {
		ty.Element = &component.AliasType
		_, nullable, _ := schemaType(component.Schema, component.Pointer)
		if nullable {
			if nullableArraySchema(component.Schema) {
				ty.ArrayAliasView = NullableArrayAlias
			}
			ty = ty.Null()
		}
	}
	return ty
}
