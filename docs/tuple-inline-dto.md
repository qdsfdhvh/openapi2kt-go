# Inline ordinary DTO positional-array sources

OpenAPI 3.1 `prefixItems` members and schema-valued `items` tails accept direct
ordinary DTO objects with actual constructor properties. Their top-level schema
permits `type`, `nullable`, `properties`, `required` and existing property counts.
The normal model/operation collection planner owns the selected declaration,
its nested types and its serializer. This extends D03/D06 and F08/M03/M07/Q05.

```json
{"Rows":{"type":"array",
         "prefixItems":[{"properties":{"name":{"type":"string"},
                                      "n":{"type":"integer","default":7}},
                         "required":["name"]}],
         "default":[{"name":"root"}]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":[]}}
```

`RowsDto` is a List of its actual peer DTO element declaration. A DTO field's
inline tuple instead retains its actual nested element declaration. DTO property
references, defaults, enums, checks and supported request/response directional
views keep their ordinary planner contracts. In the example, the root factory
returns an element named `root` with `n = 7`; the alias factory returns an empty
List. A consuming field referencing Rows acquires no default from either factory.

A single inline DTO prefix supplies its original element type. Distinct inline
DTO nodes remain different sources even if their properties are equal, so their
prefix-only projection uses `List<JsonElement>`. An explicit admitted `items`
schema wins after every original prefix validates. `items:true` selects raw JSON;
`items:false` retains the prefix decision. Typed Lists accept empty or longer
Lists and may narrow unrestricted tails. `W_TUPLE_APPROXIMATION` retains these
positional/length limitations, including discarded false-prefix restrictions.

Simple inline HTTP object values keep their existing `Map<String, JsonElement?>`
projection and `W_HTTP_VALUE_PROJECTION`. That projection does not apply DTO
field defaults, required keys or field validation. Shared model references and
ordinary DTO-owned fields retain actual DTO codecs in HTTP and SSE. This source
extension does not replace an existing context's projection policy.

## Own annotations

Outer field, original component root and independent component alias defaults
may use the selected inline DTO's existing ordinary object-default walk. Required
keys, unknown keys, original kinds, admitted property-reference/default roles,
nullable layers and validation-enabled counts/checks are validated before
required/Patch initializer suppression. Required arguments retain no initializer;
Patch remains Absent. Nested child annotations retain their independent ordinary
roles; a positional DTO's own top-level `default` is not admitted.

Factories and optional field initializers decode fresh Lists and nested objects
through the actual final type after optimization or relocation. No default is
inherited from a component, alias, positional member or container child. Raw
projections use the [raw tuple annotation contract](tuple-raw-defaults.md), so
metadata-looking keys in default JSON remain data rather than schema keywords.
Empty/null outer annotations cannot hide invalid source annotations.

## Source validation and limits

Every original DTO source, including a mixed or tail-discarded member, completes
ordinary source analysis and deferred reference/default validation in an isolated
analyzer. Discarded validation declarations, registrations and warnings do not
enter the emitted plan. Original document, physical ancestor/member/item nodes,
terminal bytes and diagnostic Pointers remain authoritative; a same-byte copied
source supplies no permission. Selected DTOs are planned again in their actual
emission context rather than copied from the validation plan.

New sources share the existing cumulative tuple capture limits of 16,384
node/edge/catalog work and 8 MiB terminal bytes, with original captures reused.
Source depth48, default value16384/32KiB, checked-depth16 and JVM240 limits remain.
An extra isolated analysis pass occurs once per document using this source role;
this contract makes no performance improvement claim.

Inline allOf/union/hierarchy/dictionary/empty-object/const carriers, positional
own defaults, general nominal uniqueness and arbitrary tuple/reference/recursive
graphs retain their independent admission rules. Existing admitted DTO/allOf
references use [their source contract](primitive-tuples.md#referenced-ordinary-dto-members-and-tails).

Formerly rejected eligible tuple inputs now generate. Existing accepted tuple
APIs, wire behavior and manifest/name-lock formats retain their contracts.

The shared source initialization also repairs ordinary inline HTTP DTOs in
**documents without a components section**. Their response planning now applies
`writeOnly` omission through the ordinary operation-owned public response type;
previously this source metadata was missing and the canonical response retained
that field. Affected methods can therefore return `<Operation>PublicResponseDto`
in place of `<Operation>ResponseDto`, with numbered peers for nested DTOs. Migrate
caller response type references and stop reading omitted properties. Strict Json
still rejects an unexpected write-only response key; `ignoreUnknownKeys=true`
uses the usual discard policy. Request read-only fields keep their existing
Patch carrier. Component-bearing inputs retain their existing directional
contract. The tuple and non-tuple HTTP-only Go regressions and fresh Kotlin
HTTP-only consumer protect this intentional API/wire correction. Enabling
merge/hoist separately retains its existing classifier and nested-access migration
rules; nested enum constants must use the actual physical canonical classifier.

[Core tests](../core/tuple_inline_dto_test.go),
[bounded fuzz](../core/tuple_inline_dto_fuzz_test.go),
[source isolation and identity](../internal/schema/tuple_inline_dto_test.go),
[native lifecycle](../internal/cli/tuple_inline_dto_process_test.go) and the
[synthetic fixture](../tests/fixtures/http/tuple-inline-dto.json) define this
profile. `bash scripts/test.sh feature tuple-inline-dto` generates fresh
Data/Plain × validation off/on projects, baseline and combined optimization
profiles with nonempty hoist locks and same-options `--check`. The
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleInlineDtoContracts.kt)
covers four Json settings, fresh field/root/alias defaults, open enum values,
nullable/required/Patch, nested containers, raw/simple HTTP projection, typed
HTTP and fragmented still-open SSE failure/closure/reuse. An independent shallow
240-field probe compiles in each profile. The complete `http` route includes the
consumer. These entrypoints do not prove additional platforms, full option
Cartesian products or final candidate acceptance.

[Inline ordinary allOf tuple sources](tuple-inline-allof.md) have a separate actual-constructor and independent outer-default contract. Original source/nullable/required/count/default checks, isolated discarded validation and shared overlapping allOf budgets remain; general compositions stay independent.
