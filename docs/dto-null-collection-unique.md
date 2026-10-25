# DTO JsonNull collection getter uniqueness

Under validation, the ordinary DTO uniqueItems profile accepts direct original
finite inline List/typed Map getters with plain `type: null` or null-only
`enum: [null]` leaves. These fields can coexist with the existing scalar,
enum, const, direct JsonNull and builtin collection getters. Named/inline DTO
ownership and actual retained constructor authority remain unchanged.

Keys use actual typed getters. List order/length matter; Map key order does
not. JsonNull and Kotlin null getter values remain distinct, including inside
a nullable leaf collection. The existing nullable JsonNull codec collapses
JSON null to Kotlin null: distinct caller-created getter keys can therefore
become duplicates after a JSON round trip. Decoding then applies the same DTO
uniqueness check. This extension does not promise carrier-identity round trips
or alter null codecs, constructors, copy, Patch or HTTP/SSE APIs.

Own collection field defaults reuse the admitted original field-default
planner. An omitted optional field contributes its actual constructor
initializer; an explicit equal tree is a duplicate at the second original
outer annotation value Pointer. Null-only vocabularies remain strict, even
for empty defaults. Invalid vocabulary/default errors retain their original
leaf Pointer during DTO-key preflight instead of being masked by the
consuming uniqueItems location. The raw240 enum cap precedes duplicate checks.

Original inline source edges, nodes, vocabulary, Pointer and the final actual
getter type/initializer must agree. Null-only leaves create no enum or type
node. The DTO-local tree proof is separate from global scalar/structural/raw
uniqueness. References/aliases, const elements, other enum/nominal/raw/native/
value-class/composed leaves and child defaults retain their boundaries.
Standalone List<JsonNull> uniqueItems is not admitted by this role.

The existing shared checked16 includes outer collection edges and deepest
getter collection edges; a DTO boundary does not reset it. Source48, enum240,
serialized JVMslots240, shared source capture16,384work/8MiB and whole-value
budgets remain. Default keys share16,384work: container/null nodes and map keys,
including omitted initializer work, are counted. Identical original
literal/descriptor reuse is cached; same-byte distinct literals are charged.
Descriptor identities, ownership and persisted lock formats retain their rules.

`bash scripts/test.sh feature dto-null-collection-unique` freshly generates
Data/Plain baseline and validation, with combined optimized/locked output.
Four Json configurations check plain/null-enum List and nested Map getters,
constructor/default/Map-order equality, nullable carrier collapse, invalid
codecs, named/inline DTOs, Patch, duplicate HTTP responses and fragmented SSE
failure/closure/client reuse. Independent probes compile240 List<JsonNull>
constructor fields with240 referring arrays, and15 getter edges plus one
outer edge with runtime duplicate/JSON assertions. Core/source tests cover
actual type/initializer/node authority, original diagnostics, shared depth
and exact default-key work. Native CLI checks readonly/no-op, parity, original
errors and failure preservation; bounded fuzz checks deterministic options
and omitted/explicit default equality. Wider platform and final candidate
acceptance remain separate.

**Migration:** eligible DTO uniqueItems inputs with JsonNull collection
getters formerly rejected now generate under validation. Existing accepted
APIs, wire/codecs and persisted formats retain their contracts. Newly admitted
malformed null leaves fail at their original source location.

[Native enum collection getters](dto-native-collection-unique.md) separately admit original finite inline List/typed Map fields with existing numeric/Boolean enum leaves under validation. Actual native checks, vocabulary/default/initializer/source authority and mapped equality remain; other enum/ref/nominal leaf roles stay separate.
