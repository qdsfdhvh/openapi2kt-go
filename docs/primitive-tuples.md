# Scalar positional-array approximation

This is an initial **F08/Q05** subset, not complete JSON Schema tuple or boolean-schema support. OpenAPI 3.1 nonempty `prefixItems` may appear on otherwise-supported array component roots and typed fields/list-map values, inline request/response containers and typed SSE payloads. OpenAPI 3.0 remains unsupported.

## Mapping and diagnostics

Every original prefix member must be an ordinary builtin String/Boolean/Int/Long/Float/Double schema, including supported ordinary numeric formats, string `uuid` (mapped to [Kotlin Uuid](nullable-uuid.md)) and string `date`/`date-time` (mapped to LocalDate/Instant). Supported nullable spellings and descriptive metadata remain available. Direct inline enum/object, const, native, value-class, container/composition, default, validation-bearing and unknown member keywords are rejected; admitted reference profiles are described below—even when explicit `items` would override them. Byte/binary formats retain their separate wire profile and remain excluded here.

- Without `items`, identical scalar kind, format presence/value and effective nullable intent map to `List<T>`.
- Different supported signatures map to `List<JsonElement>`, not a positional union. Absent versus explicitly empty `format` is a distinct signature, even if both map to String.
- Explicit ordinary scalar `items` wins **after every prefix member has been validated**. Unsupported/boolean tails remain rejected; the historical mixed-prefix plus `items: false` fixture still fails at `prefixItems`.
- Empty/malformed prefix arrays and malformed members fail with original source diagnostics. No source nodes are rewritten or manufactured.

Each projected source emits `W_TUPLE_APPROXIMATION` at its original `prefixItems` Pointer. Lists do **not** enforce positions, tuple lengths or positional member types. A typed List also **narrows the schema's unrestricted tail** when no `items` restricts that tail. For example, two string prefix members produce `List<String>`: an empty list and three strings work, but a Boolean in the unrestricted tail cannot decode. Mixed `List<JsonElement>` accepts arbitrary JSON kinds at any position. These intentional approximations are not a strict JSON Schema validator.

`--deny-warnings` rejects this warning before output mutation; `--allow-warning W_TUPLE_APPROXIMATION` explicitly permits it. Default warning policy still generates files.

## Temporal prefix members

Uniform `type: string, format: date` members generate `List<kotlinx.datetime.LocalDate>`; uniform `date-time` members generate `List<kotlin.time.Instant>`. Effective element nullable intent remains part of the signature, independently of a nullable outer array or alias. The shared scalar mapper and existing Kotlin codecs operate on the original selected member rather than a synthetic schema. The same temporal formats are accepted for explicit scalar `items` tails, which win only after every original prefix member validates.

Different temporal formats, temporal versus plain String, or different nullability still produce the existing `List<JsonElement>` projection without an explicit tail. Raw mixed lists do not enforce positional ISO formats: an invalid date spelling, Boolean or other JSON kind can occur anywhere. Typed LocalDate/Instant lists instead parse ISO strings, reject invalid dates/timestamps and can narrow an unrestricted source tail. They still accept an empty list and do not enforce tuple length or position. Instant encoding retains the existing normalized timestamp representation rather than the input offset spelling.

Component aliases, DTO/list/map fields, inline request Patch, JSON HTTP roots and SSE data reuse those same typed codecs. Invalid temporal response/event values fail decoding; still-open SSE input closes on failure or cancellation and the client remains reusable. [Temporal list uniqueness](temporal-unique.md) has a separate mapped-value contract; a prefix-only projection does not acquire that eligibility. Temporal bounds, defaults, composition members and Boolean tails remain excluded here; finite builtin scalar references have the separate profile below. Invalid original temporal keywords/defaults must fail even when a later `items` source would replace the prefix.

This D06/F08/Q05 extension accepts formerly rejected direct temporal members, including the original single-date prefix negative now retained as a positive regression. Existing accepted call sites, wire behavior, warning policy and output/name-lock formats are unchanged. The fixture and runtime route below include both builtin and temporal profiles; broader tuple graphs and additional platforms remain separate.

## Kotlin and wire

Component roots export native collection typealiases with the configured suffix. Use ordinary `listOf(...)` construction and reified serializers/JSON codecs, not alias constructors or `Alias.serializer()`. The wire remains a JSON array. Root and element nullable intent remain independent. Shared nullable fields, required-nullable inline request arguments and optional request Patch follow the existing [presence contract](request-presence.md).

Existing opt-in array count checks apply on typed consumers; standalone aliases/codecs are not validation barriers. `uniqueItems: true` remains restricted to an explicitly declared eligible scalar `items` source; a prefix-only uniform projection does not implicitly gain uniqueness eligibility. Existing source depth48, checked edges16, constructor/method slots240 and graph budgets are unchanged. Original members are visited linearly before any typed/raw projection; they cannot hide errors or reset source depth.

This increment does not admit boolean value schemas, general structured/ref/composed tuples, schema-internal reference targets or other previously unsupported common-union/hierarchy projections. Ordinary non-prefix arrays and boolean `additionalProperties` are unchanged. Previously rejected admitted tuples now generate with an explicit approximation warning; this does not guarantee binary ABI or broader platform coverage.

## Reproducible checks

- [Synthetic envelope](../tests/fixtures/http/primitive-tuples.json): builtin/temporal component aliases, mixed/items priority, nullable/list-map, inline Patch/simple HTTP and SSE.
- [Core contracts](../core/primitive_tuple_test.go), [source/limit guards](../core/primitive_tuple_guards_test.go), [native lifecycle and output preservation](../internal/cli/primitive_tuple_process_test.go).
- [Fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/PrimitiveTupleContracts.kt): aliases, constructors, four Json profiles, native JSON values and temporal ISO codecs, required/null states, explicit approximation behavior, Patch, controlled HTTP and fragmented still-open SSE temporal failure/cancellation/reuse/coldness.
- `bash scripts/test.sh feature primitive-tuple`: fresh Data/Plain × validation off/on generation/check and Kotlin runtime projects, with cached offline dependencies. Data projects also test `copy`; Plain projects do not acquire it.
- `bash scripts/test.sh http`: includes this consumer alongside existing HTTP contracts.

Independent profiles are not maximum-width/depth Cartesian, minimum-toolchain, Android/KMP or complete first-version evidence.

## Referenced builtin scalar members and tails

An otherwise accepted tuple prefix member or explicit scalar `items` tail may follow a finite local named/internal reference chain to an already accepted ordinary builtin scalar, including UUID, LocalDate and Instant. Each original hop permits only the existing `$ref`/nullable role and every terminal retains the plain scalar grammar above. The actual shared resolver must admit the reference and its scalar mapping; the terminal's kind, format presence/value and effective reference nullability determine the signature. Aliases do not create a new tuple wrapper or scalar codec. The nearest nullable override follows the existing reference contract.

Every prefix source validates before an explicit tail replaces its projected element type. Unknown keywords, malformed nullable, unsupported default/enum/const/bounds/native/value-class terminals and structured/composed targets cannot be hidden by an explicit tail or mixed raw projection. Independently supported original component factories have the separate target-default profile below. Invalid annotations report their original source Pointer. Reference source depth remains 48; reference hops additionally share a 16,384-step budget per positional array, including its explicit tail. Repeated references consume work each time. This new bound applies to the newly admitted reference role and does not reduce accepted direct scalar prefix lengths. Existing checked-depth16 and constructor-slot240 limits remain.

The [separate reference fixture](../tests/fixtures/http/tuple-references.json) preserves the original primitive tuple fixture. `bash scripts/test.sh feature tuple-references` and its [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleReferencesContracts.kt) cover Data/Plain × validation off/on, typed/mixed/tail/nullable scalar references, four Json settings, request Patch, temporal failures and controlled HTTP/live SSE close/cancel/reuse. It is also registered in `http`. [Core](../core/tuple_references_test.go), [original source/budget](../internal/schema/tuple_references_test.go) and [native parity/failed-output preservation](../internal/cli/tuple_references_process_test.go) checks define this D06 increment.

**Migration:** formerly rejected finite builtin scalar reference members/tails now generate. Existing accepted call sites, wire behavior, original `W_TUPLE_APPROXIMATION` policy and persisted lock formats are unchanged. Structured/composed tuple members, Boolean tails, positional length enforcement, general false and nullable recursive graphs remain separate.

## Referenced String enum members and tails

A positional member or explicit `items` tail may reference an already admitted plain String enum declaration through finite local named/internal aliases. Its original terminal grammar permits only type/enum/nullable and existing descriptive metadata. The actual retained vocabulary declaration and original lexical source authorize the codec; no enum is synthesized from a positional slot. Direct inline enums, formatted/native/value-class enums, bounds/default annotations, composed vocabularies and other nominal leaves remain separate.

Uniform members with the **same original enum source and effective nullability** map to a typed List of that enum. Aliases reaching that same source count as uniform. A different source remains distinct even with the same vocabulary; enum versus ordinary String, distinct enums or different nullable views project to `List<JsonElement>`. Explicit admitted scalar or enum `items` wins only after all original prefix members validate, including actual declaration ownership for members discarded by the projection. Final dedup/merge may relocate the physical enum without replacing its source authority.

Typed lists retain canonical enum classifiers/descriptors, known values and open `Unrecognized` String values. Nullable enum lists distinguish JSON null from the String `"null"`; explicit false narrows the existing nullable view. A typed enum List accepts any length at its standalone codec and may narrow an unrestricted tail to Strings. Mixed raw projection validates neither positions nor enum membership. The original `W_TUPLE_APPROXIMATION` remains mandatory; no strict tuple or vocabulary validator is added.

All original reference hops share the existing depth48/work16384 limits. Actual vocabulary nodes, owner catalogs, unique retained declarations and member identities are rechecked by the shared nominal authority, including mixed or tail-overridden prefixes. Existing checked-depth16, enum240 and constructor-slot240 limits remain; opt-in collection checks retain their typed-consumer boundaries. Own tuple/default roles do not gain new admission.

The [synthetic fixture](../tests/fixtures/http/tuple-enum-references.json), [core acceptance/guards](../core/tuple_enum_references_test.go), [original ownership/budgets](../internal/schema/tuple_enum_references_test.go), [bounded fuzz](../core/tuple_enum_references_fuzz_test.go), [native parity/warning/failure preservation](../internal/cli/tuple_enum_references_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleEnumReferencesContracts.kt) define this profile. `bash scripts/test.sh feature tuple-enum-references` compiles and runs Data/Plain × validation off/on, baseline and all optimization options with locking, four Json settings, 240 fields, typed/mixed/tail/internal/nullable vocabularies, required/Patch, controlled HTTP and fragmented still-open SSE/error/cancellation/reuse. PREPARE_ONLY does not execute Kotlin. The complete `http` route includes the same consumer.

**Migration:** formerly rejected eligible String enum references now generate; the exact prior enum-prefix-with-integer-tail negative remains an accepted regression. Existing accepted Kotlin APIs, wire behavior, warning policy and persisted metadata encodings are unchanged. Structured/composed tuple members, Boolean tails, general nullable recursive edges and wider platforms remain open.

## Referenced ordinary DTO members and tails

A positional member or explicit `items` tail may reference an already admitted ordinary DTO constructor through finite local named/internal aliases. The original terminal must have real constructor properties and the plain type/nullable/properties/required grammar, with existing opt-in root property counts. Existing property defaults/checks and nested declarations remain on that actual DTO. Independently supported source root factories use the separate target-default profile below. Other default roles, dictionaries/fixed extras, raw or const-only carriers, union/hierarchy targets and direct inline positional DTO declarations remain separate; eligible allOf constructors use the profile below. No constructor is synthesized from a tuple position or from a `Named` type spelling.

Uniform members reaching the **same original DTO source and effective nullability** produce `List<actual DTO>`. Aliases to that source can be uniform. Distinct source DTOs remain distinct for this decision even if optimization later merges their equivalent classes; DTO/scalar or nullable mismatches project to `List<JsonElement>`. Explicit admitted scalar/enum/DTO `items` wins only after every original prefix validates, including the actual retained declaration and original alias sources for members discarded by the final projection.

The actual constructor, nested enum/default helpers, codec and supported directional views are reused. Required fields, optional/null/own defaults, nullable reference overrides, existing property/container checks and JSON property counts keep their original boundaries. DTO field defaults do not become tuple-member defaults or authorize a new own tuple-default role. Prefix-only and DTO-tail nominal `uniqueItems` also retain their independent rejection policy. Mixed raw projection does not enforce object fields or positions; typed List codecs still accept any length unless a separate typed consumer enforces accepted count constraints, and they can narrow an unrestricted schema tail. `W_TUPLE_APPROXIMATION` remains.

An already admitted nominal DTO, including its existing nominal recursion, ends transparent source traversal; schemas are not unfolded. Original reference hops still consume depth48/work16384. The shared source authority verifies original catalogs/nodes and exactly one retained constructor, including copied/missing/ambiguous declarations and replaced named-alias nodes. Checked-depth16, constructor-slot240, default node/byte and all existing graph budgets remain unchanged.

The [fixture](../tests/fixtures/http/tuple-dto-references.json), [core types/guards/directional byte comparison](../core/tuple_dto_references_test.go), [source, alias, constructor and budget contracts](../internal/schema/tuple_dto_references_test.go), [bounded fuzz](../core/tuple_dto_references_fuzz_test.go), [native parity/policy/output preservation](../internal/cli/tuple_dto_references_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleDTOReferencesContracts.kt) define this profile. `bash scripts/test.sh feature tuple-dto-references` compiles/runs Data/Plain × validation off/on, baseline and all optimization/locked packages with nonempty lock state, 240 fields, four Json settings, original fresh defaults/open enums/checks/counts, nullable/internal/self-recursive DTO lists, required/Patch, controlled HTTP and live fragmented SSE error/cancel/close/reuse. PREPARE_ONLY is preparation only. The `http` route includes the same consumer.

**Migration:** formerly rejected eligible DTO reference members/tails now generate; existing accepted APIs/wire, original tuple warning policy and persisted formats remain. General structured/composed tuples, broader nominal targets, false, nullable recursive JSON covers and additional platform/resource combinations remain separate.

## Referenced allOf constructor members and tails

A positional member or explicit `items` tail may also reference an already
admitted allOf **constructor Model** through finite named/internal aliases. Its
terminal grammar permits type/nullable/allOf/properties/required and existing
property-count keywords only when the original shared planner already accepts
them. Existing allOf approximation, property overrides, field defaults/checks,
nested declaration ownership and nominal recursion are retained. This does not
add exact intersection validation or new composed root property-count/default roles,
directional projections, tuple-owned defaults or nominal `uniqueItems`.

As with ordinary DTO references, same original terminal source and effective
nullability produce a typed List; different sources, scalars or nullability
mismatches produce raw JsonElement elements. An explicit accepted tail wins after
all prefixes validate. Both `W_TUPLE_APPROXIMATION` and the original
`W_ALL_OF_APPROXIMATION` remain. A raw projection does not enforce DTO fields,
while a typed List codec keeps its actual model's behavior and existing length
approximation. It never synthesizes a constructor for a tuple slot.

Provisional allOf syntax must resolve to exactly one original retained physical
constructor under its original lexical owner. Raw/empty, const-only, dictionary,
product union and hierarchy carriers do not acquire this permission. A source
that collapses to an alias without its own constructor is likewise insufficient.
Original semantic nodes/edges, shape-reference catalogs and intermediate aliases
are retained and rechecked even for a prefix discarded by a scalar tail. Nominal
property references are not unfolded into a copied schema tree.

This new role shares one cumulative source-snapshot budget across arrays,
discarded prefixes and analyzer contexts in a document generation: **16,384**
captured nodes plus member/item edges plus shape-reference records, and **8 MiB**
of original target snapshot JSON bytes. Each distinct original target is charged
once; repeated uses share its snapshot. Separate overlapping target captures are
charged separately. Fresh capture checks the work budget before extending its
snapshot slices, including shape alias records. Snapshots previously admitted by
another role are checked against this budget before tuple permission. This keeps
many individually bounded overlapping sources from multiplying retained state.
The existing per-target source depth48/work16384, per-array reference-hop work,
checked-depth16, constructor-slot240 and default budgets remain. Existing
non-tuple allOf roles retain their original budgets.

The [fixture](../tests/fixtures/http/tuple-allof-references.json),
[core types, cumulative budget and rejection contracts](../core/tuple_allof_references_test.go),
[original source and constructor guards](../internal/schema/tuple_allof_references_test.go),
[bounded fuzz](../core/tuple_allof_references_fuzz_test.go),
[native CLI parity/policy/preservation](../internal/cli/tuple_allof_references_process_test.go)
and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleAllOfReferencesContracts.kt)
define this profile. `bash scripts/test.sh feature tuple-allof-references`
compiles/runs Data/Plain × validation, baseline/all-optimization/locked packages,
240 fields, four Json settings, actual fresh defaults/open enums/checks,
nullable/internal/self-recursive constructor lists, Patch, HTTP and fragmented
live SSE error/close/cancellation/reuse. PREPARE_ONLY remains preparation only.
The complete `http` route invokes the same consumer. Other targets require their
own evidence.

**Migration:** eligible allOf constructor references that previously failed now
generate; the exact original inline-property allOf regression is retained as a
positive. Existing accepted APIs/wire, warnings and persisted formats retain their
contracts. General composed/structured tuples, broader nominal targets, false
and general nullable recursive JSON covers remain separate.

## Independently owned component defaults

Existing tuple reference targets and eligible named aliases may retain their own already-supported component factories; those defaults do not become positional/list/missing-key defaults. All original factories validate before discarded-prefix projection, with strict enum membership, required values, nullable and budgets preserved. Original component identity grants annotation permission; positional/internal annotations gain nothing. See the [target-default contract](tuple-target-defaults.md) and four-profile `tuple-target-defaults` route. General tuple-owned defaults remain separate.

## Scalar field own defaults

[Ordinary scalar tuple field defaults](tuple-field-defaults.md) reuse the selected direct scalar List planner for a uniform prefix or explicit accepted scalar items tail after complete original-prefix validation. Required/Patch suppression, source/nullable/literal/default budgets and the tuple approximation warning remain. Mixed raw/ref-selected/root/position default roles stay separate; prefix-only enforcing uniqueness remains rejected, while an explicit scalar tail retains its existing source policy.

[Scalar tuple component factories](tuple-root-defaults.md) extend D03/D06 for original direct scalar prefix/tail List sources and independent component aliases. Original nodes/Pointers, selected codec/nullability, strict annotations/temporal checks, no inheritance, required/Patch and source/value/slot budgets remain. Positional/length approximation and excluded nominal/raw/member-own roles retain their contracts. Formerly rejected eligible root annotations gain factories; accepted API/wire/state formats remain stable, and wider/default/platform work stays open.

[Mapped scalar-field DTO uniqueness](dto-unique.md) extends D05/M03/M07/M12/V05 to actual ordinary named/original-inline DTO constructors with direct builtin scalar getters. Typed API/codec/nullable/default/required/Patch behavior remains; constructor initializer-aware default keys match runtime Float/UUID/Instant equality and retain original value Pointers. Actual source/member/constructor proof and independent shared16384work/8MiB source plus16384 default-key work budgets remain; enforcing getter/type fingerprints use a bounded cached reference while provenance/default keys stay private. General nested/enum/alias/composed/hierarchy/projected/count/native roles and broader platform/graph completion remain separate.

## Referenced native numeric and Boolean enum members and tails

[Native enum tuple references](tuple-native-enum-references.md) extend validation-on finite named/internal source paths to existing Int/Long/Float/Double/Boolean enum codecs and checks. Same original source/effective nullability gives a typed List; distinct sources/plain scalars/mismatched nullability remain raw projections. Every prefix validates before tail selection, preserving original paths/vocabularies/actual scalar exports, aliases, default independence, cumulative capture and original budgets. Unchecked/direct positional enum/default/bounds/composed roles remain separate, and the original approximation warning and standalone alias/typed-consumer check boundaries remain.
