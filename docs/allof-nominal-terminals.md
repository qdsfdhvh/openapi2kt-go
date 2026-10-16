# References to allOf constructor models

An internal schema Pointer or finite typed reference chain may target an original object allOf node when it retains a unique constructor-bearing Model under its lexical component. Nested targets use their actual physical owner, and chains ending at a named allOf component use the actual top-level classifier. For example, `#/components/schemas/Owner/properties/node` can bind a retained `OwnerDto.Node` whose source contains allOf, while a chain through `Hops/properties/named` to `AllModel` binds `AllModelDto`.

Component-root aliases to these internal nodes or chains keep the same constructor and serializer. Forward aliases, self/mutual nominal recursion, ordinary list/map consumers, request Patch and typed HTTP/SSE data use the canonical model. An inline object-union variant built from an allOf node can also be a target if it has one actual retained constructor occurrence, such as `AllChoiceDto.All`. Nested classifier access follows the physical owner rather than the root typealias.

## Original source and physical declaration

The [composition-owned reference contract](composed-nested-owners.md) supplies original catalog/node/Pointer provenance and actual retained declaration scopes. Source allOf syntax is only a provisional candidate. Shared model construction and final retained Models authorize the physical classifier; generation never expands JSON or guesses a name. Borrowed copies cannot replace the source component's model. An allOf product that becomes a union, a raw/const-only result, a discarded override, scratch validation model or ambiguous occurrence cannot authorize a target.

The final shared ownership barrier resolves and removes transient links before alias validation, optimization, name-lock fingerprints, operation planning and emission. First late HTTP uses may bind only registered physical occurrences. Transparent dictionary/collection cycles retain their existing structured cycle diagnostics; cleanup can visit provisional cyclic types without recursively overflowing.

An ordinary direct local `$ref` field may also own a [default referencing an internal allOf constructor Model](allof-object-defaults.md#references-to-internal-allof-dtos) when the original shape has disjoint property contributions and passes the finite object default walk. The initializer binds the actual lexical serializer after optimization; target/chain defaults are not inherited. This separate default role excludes hierarchy sources/owners, broader composition/collection default roles and repeated-field aggregation. Typed recursion and full terminal admission alone do not authorize defaults.

## Nullable intent and source work

Nested allOf Models may obtain nullable intent from their own annotation or from the original source-shape merge. A root alias or typed use with no override inherits that effective intent. The nearest consumer/ref/alias true or false overrides it, and a later true can re-enable null. Explicit node-level false takes precedence over member intent. The physical Kotlin class and its direct serializer remain non-null; nullable typed views use the nullable codec.

All erased internal/named reference steps remain in cached source height after false narrowing. Nominal Models are recursion barriers, while transparent-only alias cycles still fail. Source depth stays 48, graph/ownership work stays bounded at 16,384, and constructor/serialization slots stay 240. Width and source-depth probes are independent; these limits do not certify every width/depth/name combination.

This extends typed binding to actual allOf Model terminals. Existing allOf merging keeps later property overrides and its approximation warning; this does not add exact JSON Schema intersections, shape borrowing of internal DTO references as union/allOf/hierarchy members, dictionary composition or general constraint aggregation. [Explicit intrinsic nullable ordinary allOf roots](nullable-allof-dtos.md) have a separate final constructor Model contract; member-derived unmarked root nullability and nullable product alternatives remain excluded.

## Compatibility and verification

Previously rejected unique allOf terminal inputs now generate canonical DTO types. Existing accepted Kotlin call sites and wire behavior stay under their existing contracts. Historical allOf terminal negative inputs are preserved and explicitly graduated to constructor/serializer-owner positives; unrelated scalar/raw/union/collection/shape exclusions remain.

The [synthetic fixture](../tests/fixtures/http/allof-nominal-terminals.json), [core contract](../core/allof_nominal_terminals_test.go), [owner/resource guards](../core/allof_nominal_terminals_guards_test.go), [inherited-null regression](../core/allof_nominal_inherited_null_test.go), [private source/cache tests](../internal/schema/allof_nominal_terminals_test.go) and [native preservation check](../internal/cli/allof_nominal_terminals_process_test.go) define the verification surface. `bash scripts/test.sh feature allof-nominal-terminals` generates fresh Data/Plain × validation projects, with optimization flags on Plain, four Json configurations, canonical constructors/codecs, nominal recursion, inherited null/false, Patch/controlled HTTP/live fragmented SSE closure/reuse, and independent 240-slot/48-source probes. [The runtime consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AllOfNominalTerminalsContracts.kt) also runs in the full HTTP suite.

Ambiguous/general nominal graphs, inline/multiple-parent hierarchies and unexecuted Android/KMP or additional host/engine combinations remain separate capabilities.
