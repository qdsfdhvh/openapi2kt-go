# Boolean positional-array members

OpenAPI 3.1 nonempty `prefixItems` accepts literal `true` and `false` members
alongside the already admitted scalar/enum/reference sources. The shared tuple
planner validates every original member before selecting a List element. This
extends D03/D06 and F08/F10/M03/M07/Q05; other Boolean reference, composition and
source roles retain their own admission rules.

| Original prefix and tail | Resulting List element |
|---|---|
| Uniform `false`, absent tail or `items:false` | Actual generated uninhabited classifier |
| Uniform `true` | `JsonElement` |
| Mixed Boolean/scalar/enum/reference sources | Existing `JsonElement` projection |
| Any prefix with `items:true` | `JsonElement` |
| Explicit admitted schema tail | Actual tail type after every prefix validates |

For a uniform false prefix, the selected original literal owns a real nested or
collection-peer sealed class and serializer. It has no constructible instances;
decoding any supplied element, including JSON null, fails. Empty arrays remain
inhabited. A nullable outer array separately permits JSON null, and optional
containers retain normal field presence. This uses a collection of an
uninhabited type rather than a direct-false field's absence carrier.

True and mixed/raw projections retain arbitrary JSON data, including JsonNull,
objects with metadata-looking keys and Boolean literals. All mappings retain
`W_TUPLE_APPROXIMATION`. When a raw or explicit-tail projection discards a false
position, the warning explicitly states that Boolean false prefix restrictions
are not enforced by the selected List projection. Positions, exact tuple length
and closed-tail maximum length remain approximated under the existing tuple
contract; the generator is not a positional JSON Schema validator.

## Declarations and consumers

Ordinary model fields and nested List/typed Map containers reuse their actual
lexical declarations. Component collection aliases export their actual peer
classifier. Direct JSON request/response arrays and inline/shared SSE data retain
that same rejecting codec; operation peers use allocated owner-qualified names.
Only original tuple-selected false peers gain the extra operation declaration
role. Arbitrary simple HTTP declarations and direct false HTTP roots keep their
existing boundaries. Boolean `additionalProperties` retains its keyword meaning.

Original schema/member/literal and physical ancestor paths are captured and
rechecked. The final plan proves each actual false helper's identity, allocated
name, unique retained declaration, selected source and actual element address.
Operation qualification explicitly carries proof to its copied type nodes;
matching type spelling or copied helpers cannot grant permission. Discarded
scratch plans grant no public declaration. Missing, duplicated, renamed or
substituted helpers/types fail before emission. No source schemas are copied or
manufactured to authorize a type.

## Own annotations and limits

Field, original component root and independent alias own defaults retain their
existing tuple roles. A selected false List admits only an empty array default;
a nullable container may independently own null. Every supplied element rejects
at its original default index. True/mixed raw defaults keep the
[raw value contract](tuple-raw-defaults.md). An explicit schema tail validates
annotations using its selected codec, with the same positional approximation.

Required arguments receive no initializer and Patch stays Absent after complete
annotation validation. Target annotations are never inherited. Factories decode
their own values; alias nullable overrides retain their nearest intent. Existing
opt-in container counts apply to annotations and typed consumers, while
standalone transparent aliases retain their existing unchecked count boundary.
Boolean members grant no prefix-only or false-tail `uniqueItems` authority;
explicit eligible tails keep their existing policy.

New Boolean sources share the existing cumulative tuple capture budget of
16,384 node/edge/catalog work and 8 MiB terminal bytes. Repeated original captures
are reused across model and HTTP contexts, including HTTP-only documents.
Existing source depth48, reference work16384, default value16384/32KiB,
checked16 and JVM240 limits remain. Ordinary non-tuple Boolean roles are not
widened by this source permission.

## Verification and migration

[Core behavior and budgets](../core/tuple_boolean_members_test.go),
[retained helper/source proof](../internal/schema/tuple_boolean_members_test.go),
[native lifecycle](../internal/cli/tuple_boolean_members_process_test.go) and
[bounded deterministic fuzz](../core/tuple_boolean_members_fuzz_test.go) define
this profile. The [fixture](../tests/fixtures/http/tuple-boolean-members.json) and
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleBooleanMembersContracts.kt)
exercise empty versus every nonempty false value, raw JSON, nullable/default/
required/Patch, nested containers, direct and referenced HTTP roots, inline/shared
fragmented SSE, failure closure and client reuse.

`bash scripts/test.sh feature tuple-boolean-members` builds fresh Data/Plain ×
validation off/on, baseline and combined optimized output with actual nonempty
hoist locks. A separate 240-field DTO compiles actual false classifiers and empty
annotation helpers. Supplemental directional consumers exercise readOnly/writeOnly
request and public-response projections with the same rejecting element codecs. The complete `http` route includes the consumer. These checks
do not establish the full option Cartesian product or other target platforms.

Former exact true-prefix and false-prefix/false-tail refusals remain positive
regressions. Previously rejected eligible Boolean members now generate; existing
accepted Kotlin APIs, wire behavior and persisted manifest/name-lock formats
retain their contracts. General structured/composed/recursive tuple sources,
Boolean graphs, enclosing default graphs and platform/final acceptance remain
separate.

[Named true/false tuple references](tuple-boolean-references.md) have a separate original-root/actual-codec contract, including independent outer defaults, nullable false elements and complete erased-prefix source-depth accounting. General Boolean graphs remain independent.
