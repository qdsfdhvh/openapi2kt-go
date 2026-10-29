# Own defaults with selected tuple references

[Model String enum metadata](metadata-string-enums.md) extends nullable roots/aliases, actual internal lexical references, existing independent scalar/collection/object default roles and serialName uniqueness. Original ownership, source/null-vocabulary boundaries, presence and budgets remain. Equivalent enum merges now retain each component-owned factory; generic graphs, platforms and final acceptance remain open.


OpenAPI 3.1 positional arrays may own field, original component root and
independent component alias defaults when the selected List element is an
already admitted finite local reference. This extends D03/D06 and
F04/F07/F08/M03/M04/M07/M08/Q05. The actual shared reference and collection
planners retain the Kotlin element type and serializer; default annotations
supply values independently of type admission.

```json
{"Choice":{"enum":["ready","paused"],"default":"ready"},
 "Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Choice"}],
         "default":["paused"]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":[]}}
```

The selected reference may reach an existing builtin/UUID/LocalDate/Instant
scalar, plain String enum, validation-on native numeric/Boolean enum, ordinary
DTO or ordinary allOf constructor through existing named/internal/alias paths.
Those paths keep their original opt-in, source, nullable and ownership gates.
A new default never grants a new reference target, declaration or wrapper role.

Every prefix validates before tail selection and before consuming an empty/null
annotation. A uniform prefix selects its original reference; a schema `items`
tail wins after that source check. Boolean tails and raw projections keep their
[existing contracts](tuple-boolean-tails.md). The mandatory
`W_TUPLE_APPROXIMATION` still describes lost positions, length constraints and
possible narrowing of an unrestricted tail. Standalone transparent aliases do
not become runtime constructor/check barriers.

## Literal and Kotlin behavior

An outer annotation must be an array unless the actual container source admits
null. Element null follows the actual selected reference's nearest nullable
intent independently. Scalar values use their mapped codecs and existing
lexical checks; temporal precision and UUID validation remain strict. String
enum annotations must belong to the original vocabulary despite open runtime
Unrecognized strings. Native enum annotations compare exact original membership
before floating mapping; typed consumers keep their existing validation checks.

DTO values validate declared keys, required fields, existing leaf/default
walks, constraints and ordinary whole-object counts. allOf values reuse the
existing winning constructor/default/constraint aggregation. An omitted required
property still fails even if that property has a default annotation. Existing
object-default source roles remain authoritative for nested properties; this is
not a general recursive/composed default graph extension.

Optional fields initialize from their own annotations. Required fields retain
explicit arguments and Patch remains Absent after annotation validation. Root
and alias factories use their actual serializers and decode fresh state on each
call. Independently admitted target/alias annotations validate under their own
factory roles. They supply neither a missing tuple member nor an outer default.
For example, `Rows` above uses paused, its target factory uses ready, and a field
that only references Rows inherits neither annotation.

Field helpers bind the final actual nominal leaf after lexical linking and
optional hoist/merge. Root collection factories decode the real emitted alias.
The existing opt-in classifier migration still applies: use canonical owners
for nested enum constants, or retained companion lookup APIs on old aliases.

## Original sources and resource limits

The complete original tuple decision is rechecked before borrowing a selected
reference; a fabricated typed selection cannot replace its raw projection.
The selected reference must be the actual original first prefix or schema tail
at its physical Pointer. Every finite hop and terminal is source-validated;
original ancestor/schema/member/literal identities and terminal bytes are
captured and rechecked, including independent target annotations. Same-byte
copies and fabricated selected paths acquire no source permission. The shared
final linker still requires each actual retained nominal declaration/codec.
No expanded or synthetic source schemas are created.

New selected-reference default source captures share the existing tuple source
registry's cumulative 16,384 node/edge/catalog work and 8 MiB terminal bytes;
repeated original captures are reused. Existing source depth48, per-array
reference work16384, default value16384/32KiB, checked16, enum240 and JVM240
limits remain. Source planning and complete value budgets run before
required/Patch suppression or empty/null literal shortcuts.

Position-owned/child annotations, unadmitted reference and target roles,
inline DTO/composed positional sources, recursive default graphs and
broader whole-object/constraint aggregation retain independent contracts.
`uniqueItems` retains its existing explicit-tail source eligibility; an own
default adds no prefix-only uniqueness role. Invalid source/default diagnostics
retain original Pointers and old output remains intact.

## Verification and migration

[Core](../core/tuple_reference_defaults_test.go),
[original source/type guards](../internal/schema/tuple_reference_defaults_test.go),
[native lifecycle](../internal/cli/tuple_reference_defaults_process_test.go) and
[bounded structured fuzz](../core/tuple_reference_defaults_fuzz_test.go) define
this role. The [fixture](../tests/fixtures/http/tuple-reference-defaults.json) and
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleReferenceDefaultsContracts.kt)
exercise fields, aliases, internal enum/DTO/allOf sources, independent factories,
nullable/required/Patch, four Json settings, UUID/temporal/native codecs,
controlled HTTP and fragmented still-open SSE failure/closure/client reuse.
`bash scripts/test.sh feature tuple-reference-defaults` builds Data/Plain ×
validation off/on, baseline and combined optimized output with nonempty hoist
locks; a separate 240-field referenced-default DTO compiles in each profile.
The complete `http` route includes the consumer. These profiles do not establish
the full option Cartesian product or additional target platforms.

Previously rejected eligible annotations now generate initializers/factories.
Former exact DTO/allOf/native reference default refusals remain positive
regressions; unadmitted unchecked native references retain their rejection.
Existing accepted Kotlin APIs, wire behavior and manifest/name-lock formats
retain their contracts. General D03/D06 and platform/final acceptance remain open.

[Boolean positional literals](tuple-boolean-members.md) separately admit true/false direct sources and their own tuple annotations.

[Named true/false tuple references](tuple-boolean-references.md) have a separate original-root/actual-codec contract, including independent outer defaults, nullable false elements and complete erased-prefix source-depth accounting. General Boolean graphs remain independent.

[Inline ordinary allOf tuple sources](tuple-inline-allof.md) have a separate actual-constructor and independent outer-default contract. Original source/nullable/required/count/default checks, isolated discarded validation and shared overlapping allOf budgets remain; general compositions stay independent.

[List/typed Map tuple sources](tuple-collection-sources.md) retain actual finite collection codecs, strict independent outer annotations, original backing/source proof and erased-only cached source depth. General tuple/default graphs stay separate.

[JsonNull tuple sources](tuple-null-sources.md) separately admit original plain/null enum/null const positions and accepted reference paths, with actual codecs, strict independent outer defaults, original source proof and existing limits. General const/graph and child default roles remain independent.

[Scalar const tuple sources](tuple-scalar-const.md) separately admit existing scalar const codecs and strict independent outer defaults, preserving original literal/ref/backing identity, nearest nullable and cached/cumulative source budgets. Retained catalog-binding records count toward cumulative capture work; structured const/union/recursive roles remain independent.

[Structured const tuple sources](tuple-structured-const.md) separately admit original JsonObject/JsonArray positions and existing references, with strict complete-literal outer annotations and data-only capture of const/default/enum payloads. Original identity, nullable/Patch and cumulative budgets remain; broader union/recursive/default graphs stay independent.

[Metadata String enum tuple defaults](tuple-inline-enums.md#string-format-and-native-metadata) separately admit actual named lexical enums and pure named aliases. Outer defaults retain original String membership; target/chain/member defaults and generic nullable/internal enum admission remain independent.
