# Scalar tuple component default factories

An OpenAPI 3.1 array component with `prefixItems` can own a default factory when
its selected List element uses the existing direct scalar tuple-field grammar.
A uniform prefix selects its original first member; an explicit admitted scalar
`items` tail owns the List type after every original prefix member validates.
Plain String, Boolean, Int/Long, Float/Double, Uuid, LocalDate and Instant mappings
retain their ordinary codec contracts. Null intent belongs to each source layer.

```json
{"Rows":{"type":"array","prefixItems":[{"type":"integer"},{"type":"integer"}],
         "default":[1,2]},
 "Wrapped":{"$ref":"#/components/schemas/Rows","default":[3]}}
```

The actual component remains a List typealias. Top-level functions such as
`` `RowsDto default`() `` and `` `WrappedDto default`() `` decode each independent
annotation through that alias's serializer. Factories produce fresh values;
referencing fields, required keys and Patch fields do not inherit the root
annotation. A default on a target does not grant a default role to a tuple
position or a nested child. Existing independent alias nullable/false overrides
apply without rewriting target or child schemas.

Every original prefix and tail source validates before an empty/null annotation
is consumed. Supplied members must fit the selected List scalar type and
nullable intent, including temporal lexical/precision checks. Existing
validation-on collection counts and mapped-value uniqueness apply where already
admitted; prefix-only uniqueness retains its separate rejection. The literal
walk borrows original component/selected-member nodes and Pointers, and copied
source trees receive no component permission. Invalid values report their
consuming `/default/<index>` Pointer. Original source depth and tuple work
budgets, value16384/default32KiB, checked16 and JVMslots240 remain.

Uniform [false Boolean tails](tuple-boolean-tails.md) reuse the selected direct
scalar factory grammar. Mixed and true-tail projections use the implemented
[raw tuple factory role](tuple-raw-defaults.md), including already admitted
reference prefixes. Direct inline enum selected members use the [enum tuple default contract](tuple-inline-enums.md). Selected admitted reference elements use the [reference default contract](tuple-reference-defaults.md). Member-own defaults and unadmitted source roles keep their separate contracts. A tuple default adds no positional or exact-length validation: generated
Lists retain the existing `W_TUPLE_APPROXIMATION`, scalar codecs and tail policy.
It also adds no runtime validation to standalone List alias assignment.

`bash scripts/test.sh feature tuple-root-defaults` generates Data/Plain with
validation off/on, baseline and combined optimized/locked output. The
[fixture](../tests/fixtures/http/tuple-root-defaults.json) and consumer check four
Json configurations, Long extrema, Float negative zero, temporal/Uuid codecs,
nullable/empty values, independent and narrowed aliases, no inheritance,
required/Patch, controlled HTTP and live fragmented UTF-8 SSE failure/closure/
client reuse. Supplemental consumers compile 240 fields and validation-on
count/uniqueness factories and typed fields. Native tests compare core bytes,
readonly/no-op and failure preservation; source/value tests cover original
identity, excluded sources and exact node/byte limits, with bounded fuzz for
deterministic valid/invalid/null annotations. Additional platforms require
independent evidence.

**Migration:** formerly rejected eligible scalar tuple root annotations now
produce factories. Accepted tuple declarations, source/wire behavior and
persisted manifest/name-lock formats retain their contracts. Wider tuple and
default roles remain separate.

[Boolean positional sources](tuple-boolean-members.md) add uniform false empty-array factories and true/mixed raw factories, with actual selected codecs, independent nullable intent and unchanged required/Patch/no-inheritance/budgets.

[Selected inline ordinary DTO tuples](tuple-inline-dto.md) have a separate field/root/independent alias own-default contract using actual final DTO codecs, strict existing object-default walks and complete discarded-source validation. Positional own annotations and general composed/default graphs remain separate.

[Inline ordinary allOf tuple sources](tuple-inline-allof.md) have a separate actual-constructor and independent outer-default contract. Original source/nullable/required/count/default checks, isolated discarded validation and shared overlapping allOf budgets remain; general compositions stay independent.

[List/typed Map tuple sources](tuple-collection-sources.md) retain actual finite collection codecs, strict independent outer annotations, original backing/source proof and erased-only cached source depth. General tuple/default graphs stay separate.

[JsonNull tuple sources](tuple-null-sources.md) separately admit original plain/null enum/null const positions and accepted reference paths, with actual codecs, strict independent outer defaults, original source proof and existing limits. General const/graph and child default roles remain independent.

[Scalar const tuple sources](tuple-scalar-const.md) separately admit existing scalar const codecs and strict independent outer defaults, preserving original literal/ref/backing identity, nearest nullable and cached/cumulative source budgets. Retained catalog-binding records count toward cumulative capture work; structured const/union/recursive roles remain independent.

[Structured const tuple sources](tuple-structured-const.md) separately admit original JsonObject/JsonArray positions and existing references, with strict complete-literal outer annotations and data-only capture of const/default/enum payloads. Original identity, nullable/Patch and cumulative budgets remain; broader union/recursive/default graphs stay independent.

[Union tuple sources](tuple-union-sources.md) extend D02/D03/D06/F08/M03/M07/Q05 with actual inline/reference union codecs, existing normalization/context projections and independent outer defaults. Original dispatch/pins/required/null/Unknown policies, source/declaration/backing/address proof and erased-source budgets remain. General composition/recursive/default/platform and final candidate acceptance stay open.

[Product union tuple sources](tuple-product-union-sources.md) extend D02/D03/D06/F08/M03/M07/Q05 through actual Cartesian product declarations, original contribution/dispatch/null policies and strict outer own annotations. Original product/source/variant/origin/type/address/cache proof and budgets remain, including discarded validation and internal wrappers. General recursive/hierarchy/intersection/default/platform and final acceptance remain open.

[Hierarchy tuple references](tuple-hierarchy-references.md) extend D01/D02/D03/D06/F08/M03/M07/Q05 through actual named root/intermediate/leaf codecs and strict outer own defaults. Original dispatch/inherited tags/Unknown/null adapters, actual lineage/leaf/source/type/address/cache proof and budgets remain. General inline/multiple-parent/recursive/default/platform and final acceptance stay open.
