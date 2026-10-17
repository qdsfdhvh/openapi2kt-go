// Package schema analyzes borrowed JSON nodes into private, typed Kotlin plans.
// It neither expands references by copying JSON nor performs any IO.
package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

const MaxSchemaDepth = 48
const MaxMembers = 240

type Options struct {
	PackageName    string
	RuntimePackage string
	Suffix         string
	ClientName     string
	ExcludeAPIs    []string
	Validation     bool
}

type TypeKind uint8

const (
	Scalar TypeKind = iota
	Named
	List
	Nullable
	JSON
	Map
	JSONObject
	PatchValue
	InlineValue
)

// ArrayAliasView retains nullable-root identity before forward alias binding.
// It also marks new root-bound dictionary views; direct historical dictionary
// consumers remain unmarked. Narrowing never changes the shared target.
type ArrayAliasView uint8

const (
	OrdinaryAlias ArrayAliasView = iota
	NullableArrayAlias
	NonNullArrayAlias
)

type TypeRef struct {
	Kind TypeKind
	Name string
	// Element also retains a transparent collection alias's validation target
	// for Named types; spelling and JVM slots remain nominal.
	Element *TypeRef
	// Serializer is a type-use override at a field use site, not a scalar identity.
	Serializer string
	// Validation on transparent scalar aliases follows typed consumers, not
	// Kotlin's inherently unchecked standalone primitive/typealias creation.
	Validation *PrimitiveValidation
	Collection *CollectionValidation
	// SchemaSteps retains transparent source hops erased by scalar/collection
	// lowering. It is resource accounting, not Kotlin or wire identity.
	SchemaSteps    int
	ArrayAliasView ArrayAliasView
	// Transient source binding is cleared after actual retained model ownership
	// is known, before semantic equality, optimization or emission.
	deferredNominal *deferredNominal
}

// WithoutRootNull preserves item nullability and lazy forward alias linkage.
func (t TypeRef) WithoutRootNull() TypeRef {
	if t.Kind == Nullable {
		t = *t.Element
	}
	if t.ArrayAliasView == NullableArrayAlias {
		t.ArrayAliasView = NonNullArrayAlias
	}
	return t
}

func (t TypeRef) Null() TypeRef {
	if t.Kind == Nullable {
		return t
	}
	return TypeRef{Kind: Nullable, Element: &t}
}

func (t TypeRef) Slots() int {
	if t.Kind == InlineValue {
		return t.Element.Slots()
	}
	if t.Kind == Scalar && (t.Name == "kotlin.Long" || t.Name == "kotlin.Double") {
		return 2
	}
	return 1
}

type Property struct {
	Name, WireName string
	Type           TypeRef
	DefaultNull    bool
	Default        string // Validated Kotlin initializer; absent when empty.
	Validation     *PrimitiveValidation
}

type Constant struct {
	Name, WireName string
	Type           TypeRef
	Initializer    string
}

// Declaration is closed to implementations in this package. Emitter dispatch
// must explicitly cover every implementation; unknown IR is a programming error.
type Declaration interface {
	Name() string
	declaration()
}

type Model struct {
	Plain      bool
	ClassName  string
	Properties []Property
	Constants  []Constant
	Nested     []Declaration
	Nullable   bool
	// JSON wire counts belong to the nominal ordinary DTO, not its constructor.
	PropertyCount *CollectionValidation
	// ComponentDefault holds the canonical JSON of this component's own root
	// `default`. It emits an opt-in factory and never an initializer, so the
	// constructor, presence classification and serializer stay unchanged.
	ComponentDefault string
}

func (m *Model) Name() string { return m.ClassName }
func (*Model) declaration()   {}

type EnumMember struct{ Name, Wire string }
type StringEnum struct {
	Plain     bool
	ClassName string
	Members   []EnumMember
	// ComponentDefault holds the canonical JSON of the component's own root
	// `default`; it emits an opt-in factory beside the existing companion.
	ComponentDefault string
}

func (m *StringEnum) Name() string { return m.ClassName }
func (*StringEnum) declaration()   {}

type ScalarAlias struct {
	ClassName string
	Scalar    string
	// ComponentDefault holds the canonical JSON of the component's own root
	// `default`; the factory decodes through the actual scalar serializer.
	ComponentDefault string
}

func (m *ScalarAlias) Name() string { return m.ClassName }
func (*ScalarAlias) declaration()   {}

// ValueAlias is a transparent collection alias. Its value declarations are
// planned as top-level peers because typealiases cannot own nested classes.
type ValueAlias struct {
	ClassName string
	Type      TypeRef
}

func (m *ValueAlias) Name() string { return m.ClassName }
func (*ValueAlias) declaration()   {}

type component struct {
	Wire, Name, Scalar, Pointer string
	Schema                      *document.Value
	ValueClass                  bool
	Validation                  *PrimitiveValidation
	Reference                   string
	// BindingTarget retains a pure root alias's canonical declaration.
	// Own schema/source stay original; the nearest explicit nullable override
	// is separate from terminal spelling, admitted for collections, ordinary DTOs and plain String enums.
	BindingTarget   *component
	BindingNullable *bool
	// A root alias may retain an existing physical nested DTO target without
	// fabricating component identity. Schema/Pointer remain the original node.
	InternalModelTarget *resolvedSchema
	// Legacy validation-opt-in builtin aliases keep their historical scalar
	// declaration/consumer behavior, while retaining canonical provenance for
	// an independently nullable alias that refers through them.
	ScalarTarget *component
	// Only validated direct ordinary DTO roots carry this intrinsic intent;
	// the Kotlin classifier itself remains the canonical non-null data class.
	// Intrinsic intent on a hierarchy source; the emitted classifier and its
	// inheritance edges remain non-null. Final hierarchy ownership authorizes it.
	NullableHierarchy       bool
	MemberNullableHierarchy bool // Derived specifically from newly admitted members.
	NullableDTO             bool
	NullableEnum            bool
	NullableNativeEnum      bool
	NullableValueClass      bool
	NullableTemporal        bool
	NullableBigint          bool
	NullableBytes           bool
	NullableCallerNative    bool
	// Plain builtin roots expose nullable aliases, never a nominal barrier.
	NullablePrimitive       bool
	NullableStructuredConst bool // Transparent nullable JSON-kind alias, not a DTO.
	// Direct empty ordinary objects expose nullable transparent JsonElement,
	// not a nominal DTO. Narrowing must use the linked non-null backing view.
	NullableRaw bool
	// BooleanRoot is an admitted 3.1 literal true named JSON value, never a
	// nominal DTO or a relaxation of arbitrary schema object slots.
	BooleanRoot bool
	// RecursiveJSON retains an exact connected complete JSON-kind cover graph.
	// Its finite original source height is cached separately from its carrier.
	RecursiveJSON    bool
	AliasHeight      int
	CollectionHeight int
	AliasType        TypeRef
}

type analyzer struct {
	options     Options
	root        *document.Value
	components  map[string]*component
	warnings    []diagnostic.Diagnostic
	usage       UsageContext
	fieldUnions FieldUnionPolicy
	// Diagnostic provenance stays outside semantic IR/declaration equality.
	emissionHints     EmissionHints
	operationRoles    map[string]string
	operationOrigins  map[string][]string
	validationSources map[*PrimitiveValidation]string
	collectionSources map[*CollectionValidation]string
	aliasUses         map[collectionAliasUse]struct{}
	aliasPlanning     bool
	// Fully validated JSON cover graphs cache heights by original node identity.
	recursiveJSONHeights map[*document.Value]int
	// Original-node identity maps to an existing physical nested DTO owner;
	// it is separate from named catalog bindings and transparent value plans.
	internalModels      map[*document.Value]string
	internalNamedModels map[*document.Value]string
	internalModelUses   map[string]string
	// Discovery only; final hierarchy declarations authorize nullable alias
	// bindings. Original nodes remain distinct from physical classifier names.
	hierarchySources map[*document.Value]bool
	hierarchyOrigins map[*Hierarchy]nominalOrigin
	modelOrigins     map[*Model]*document.Value
	modelPointers    map[*Model]string
	// Original inline object field roots in the direct DTO object default role.
	// The shared object type planner must ignore their own annotation, which the
	// dedicated default planner validates instead.
	objectDefaultSchemas map[*document.Value]bool
	// componentDefaultSchemas marks original component roots whose own `default`
	// the factory planner validates instead of the shared type plan.
	componentDefaultSchemas   map[*document.Value]bool
	objectDefaultWork         *int            // Shared only by one source validation walk.
	objectDefaultAllOfElement *document.Value // One consuming inline collection's original leaf.

	objectDefaultAllOfMemberElement *document.Value // One collection-valued object's original leaf/reference.

	// One plain-DTO collection member walk. Nested containers inside this walk
	// keep the same member profile; the flag is scoped to a shallow validation
	// copy so it cannot leak into sibling roles.
	objectDefaultPlainDTOMember bool

	declarationOwners            map[Declaration]*component
	unionOrigins                 map[Declaration]nominalOrigin
	enumOrigins                  map[Declaration]nominalOrigin
	valueClassOrigins            map[Declaration]nominalOrigin
	temporalOrigins              map[Declaration]temporalOrigin
	temporalUses                 map[*component]string
	nativeEnumOrigins            map[*component]*nativeEnumOrigin
	nativeEnumUses               map[*component]string
	builtinOrigins               map[Declaration]builtinOrigin
	builtinUses                  map[*component]string
	namedCollectionOrigins       map[Declaration]namedCollectionOrigin
	namedCollectionUses          map[*component]string
	recursiveCoverOrigins        map[Declaration]recursiveCoverOrigin
	recursiveCoverUses           map[*component]string
	rawTrueOrigins               map[Declaration]rawTrueOrigin
	rawTrueUses                  map[*component]string
	callerNativeOrigins          map[Declaration]callerNativeOrigin
	callerNativeUses             map[*component]string
	bytesOrigins                 map[Declaration]bytesOrigin
	bytesUses                    map[*component]string
	bigintOrigins                map[Declaration]bigintOrigin
	bigintUses                   map[*component]string
	borrowedScalars              map[*document.Value]*borrowedScalarSource
	borrowedRoots                map[Declaration]borrowedRootOrigin
	borrowedCollections          map[*document.Value]*borrowedCollectionSource
	borrowedCollectionRoots      map[Declaration]borrowedCollectionRoot
	borrowedDTOLeaves            map[*document.Value]*borrowedDTOLeafUse
	borrowedEnumLeaves           map[*document.Value]*borrowedEnumLeafUse
	borrowedValueClassLeaves     map[*component]*borrowedValueClassLeafUse
	borrowedAllOfLeaves          map[*document.Value]*borrowedAllOfLeafUse
	borrowedUnionLeaves          map[*document.Value]*borrowedUnionLeafUse
	borrowedHierarchyLeaves      map[*document.Value]*borrowedHierarchyLeafUse
	borrowedBuiltinLeaves        map[*component]*borrowedBuiltinLeafUse
	borrowedTemporalLeaves       map[*component]*borrowedTemporalLeafUse
	borrowedBigintLeaves         map[*component]*borrowedBigintLeafUse
	borrowedBytesLeaves          map[*component]*borrowedBytesLeafUse
	borrowedCallerNativeLeaves   map[*component]*borrowedCallerNativeLeafUse
	borrowedRawTrueLeaves        map[*component]*borrowedRawTrueLeafUse
	borrowedRecursiveCoverLeaves map[*component]*borrowedRecursiveCoverLeafUse
	borrowedEnumAliasVisit       func(*component, int) *diagnostic.Diagnostic
	borrowedDTOAliasEdges        map[*component]*borrowedDTOAliasEdge
	deferredNominals             map[*document.Value]*deferredNominal
	referencedDefaults           *[]referencedFieldDefault
	nominalsLinked               bool
}

func (a *analyzer) warn(code, message, pointer string) {
	a.warnings = append(a.warnings, *diagnostic.At(code, message, pointer))
}

func invalid(message, pointer string) *diagnostic.Diagnostic {
	return diagnostic.At("E_SCHEMA_INVALID", message, pointer)
}

func unsupported(pointer string) *diagnostic.Diagnostic {
	return diagnostic.At("E_SCHEMA_UNSUPPORTED", "This schema construct is not supported.", pointer)
}

func depthCheck(depth int, pointer string) *diagnostic.Diagnostic {
	if depth > MaxSchemaDepth {
		return diagnostic.At("E_SCHEMA_DEPTH", "Schema nesting exceeds the supported limit.", pointer)
	}
	return nil
}
