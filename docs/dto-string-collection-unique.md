# DTO String enum collection getter uniqueness

Under validation, ordinary DTO uniqueItems accepts direct original finite inline
List/typed Map getters with plain unformatted String enum leaves, including
typeless vocabularies. Getter keys compare actual serialName values: a known
member and same-name Unrecognized are equal, while runtime unknown strings
remain open. List order matters; Map key order does not. Null and String "null"
remain distinct. Existing nullable codecs and constructor/Patch APIs remain.

Preflight builds an abstract serialName tree over original inline schema
edges, with no provisional enum declaration, classifier guess, inserted type
node or checker metadata. After actual model planning, the recursive proof
requires each real nested enum's original schema/Pointer, retained owner,
classifier, member names/wire values and declaration identity. A foreign,
missing, repeated or same-byte copied enum cannot grant equality. Relocation
uses actual getter types; the key descriptor stores no stale classifier name.

Own collection defaults retain strict original vocabulary membership, even
though runtime strings are open. Omitted fields use their actual constructor
initializer; explicit equal trees are duplicates at the second outer default
value Pointer. Required/Patch suppression happens after validation, with no
inheritance. Malformed vocabulary/defaults keep their original source Pointers.
The original raw240 member policy and null-token rules remain.

This is a DTO-local extension. Field/leaf references or aliases, formatted or
bounded String enum collection leaves, const/nominal/raw/native-override/
value-class/composed leaves and child defaults retain separate boundaries.
Other getter families keep their own contracts. Global structural/source
uniqueness permissions, output ownership and name-lock formats are unchanged.

Shared checked16 includes current outer edges plus deepest getter edges; a DTO
boundary does not reset it. Source48, serialized JVMslots240, source capture
16,384work/8MiB, default-key16,384work and whole-value budgets remain. Container
nodes, values/map keys and omitted initializer work are charged; identical
original literal/descriptor reuse is cached. Canonical keys encode each tree
once and keep source/checker metadata outside enforcing descriptor identity.

`bash scripts/test.sh feature dto-string-collection-unique` freshly generates
Data/Plain baseline/validation and combined optimized/locked output. Four Json
settings check actual collection enum classifiers, known/Unrecognized equality,
open unknown values, List order, nested nullable Map defaults, named/inline
DTOs, existing getter families, Patch, duplicate HTTP responses and fragmented
SSE closure/client reuse. Independent probes compile240 enum List getters with
240 referring arrays, and15 getter edges plus one outer edge with runtime
JSON/duplicate assertions. Core/source tests check actual enum identity,
vocabulary/name/wire mutations, source replacement, pure descriptor planning,
strict defaults, exact key work and shared depth. Native CLI failures preserve
old output; bounded fuzz checks deterministic options and omitted/explicit
initializer equality. Broader graph/platform/final candidate gates remain open.

**Migration:** eligible DTO uniqueItems inputs with original String enum
collection getters formerly rejected now generate under validation. Existing
accepted APIs, open enum wire/codecs and manifest/name-lock formats retain
their contracts.

[DTO scalar reference getter uniqueness](dto-scalar-reference-unique.md) separately admits already-supported builtin/temporal scalar reference fields under validation, with original chain/owner/actual declaration/type/initializer authority. Field-own defaults reuse the existing deferred default planner and source-role restrictions; consumers inherit no component defaults.

[DTO String enum reference getters](dto-string-reference-unique.md) separately admit already-supported plain named/internal/bridge enum sources under validation. Actual original enum and intermediate alias identity/physical types, strict own defaults and open serialName runtime behavior remain; other reference roles retain their contracts.
