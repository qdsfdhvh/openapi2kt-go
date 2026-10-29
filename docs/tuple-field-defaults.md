# Scalar tuple defaults on ordinary DTO fields

An ordinary DTO field with an already accepted OpenAPI 3.1 `prefixItems` array
can own a scalar List `default`. Uniform prefix members select their original
scalar schema; an explicit accepted scalar `items` tail takes precedence after
every prefix source validates. The selected schema must be a direct builtin,
UUID or standard temporal scalar already supported by the
[array-field default planner](primitive-array-defaults.md). The same original
element type, nullable view, literal checks and typed default helper are reused.
No schema tree is copied or synthesized.

Uniform [false Boolean tails](tuple-boolean-tails.md) select their original scalar
prefix and use the same own-default grammar. True-tail raw projections use the [raw tuple default role](tuple-raw-defaults.md).

For example, a prefix of String and Int plus explicit Int `items` can have
`default: [1, 2]`. The default is checked against the selected `List<Int>`
approximation, without enforcing the original positional String slot. As with
existing tuples, `W_TUPLE_APPROXIMATION` reports that positions and tuple length
are not enforced and a typed List can narrow unrestricted tail values. Uniform
prefix-only defaults likewise do not gain positional or exact-length validation.
Mixed and true-tail raw projections have the separate implemented
[raw tuple default role](tuple-raw-defaults.md).

Optional fields receive an independently decoded typed default List. Missing
keys use it, supplied values override it and explicit null follows the existing
nullable-field policy. Required fields retain their required constructor argument
and receive no initializer. Optional nullable request fields retain Patch.Absent;
the default does not become an implicit update. All original default annotations
validate before required/Patch suppression, including overridden fields.

Null container defaults require root nullability and null elements require a
nullable selected element. Empty/null defaults still validate every original
prefix and the selected tail. Bad members report their actual `/default/index`
Pointer. UUID and temporal lexical rules, scalar representability, Float/Double
mapping and existing counts/checks retain their contracts.

This role does not grant a new uniqueness source. Prefix-only enforcing
`uniqueItems` keeps its existing rejection. With an explicit accepted scalar
`items` tail, the original validation-enabled unique source policy applies to
both annotations and runtime values, including UUID canonical equality and
Instant equality across offsets. Collection counts retain their ordinary opt-in
behavior. The new tuple own-default role uses the existing whole-value
**16,384-node**, **32 KiB JSON-byte** and depth budgets, and checks the canonical
mapped JSON size as well. Original source48/reference-work16384/checked16/
constructor-slots240 limits remain.

Collection-element defaults and unadmitted ref/nominal selected elements,
structured prefix declarations, tuple-position
annotations and broader default/uniqueness roles remain separate. Referenced
components' independent factories use the [target-default profile](tuple-target-defaults.md);
they do not supply a missing tuple field or grant this field's annotation role.

The [fixture](../tests/fixtures/http/tuple-field-defaults.json),
[core type/literal/budget contracts](../core/tuple_field_defaults_test.go),
[original source and scoped-permission checks](../internal/schema/tuple_field_defaults_test.go),
[bounded fuzz](../core/tuple_field_defaults_fuzz_test.go),
[native required/Patch/member failure preservation](../internal/cli/tuple_field_defaults_process_test.go)
and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleFieldDefaultsContracts.kt)
define this D03/D06/M03/M04/M07 profile. Run `bash scripts/test.sh feature tuple-field-defaults`
for fresh Data/Plain × validation, baseline/combined optimization/lock outputs,
240-field constructors, four Json settings, fresh typed helpers, temporal values,
required/Patch, count/explicit-tail uniqueness and HTTP/fragmented SSE contracts.
The complete `http` route invokes the same consumer. Other platforms need
independent evidence.

**Migration:** formerly rejected eligible scalar tuple field defaults now
generate. Existing accepted call sites, wire behavior, warnings and persisted
formats retain their contracts. General tuple defaults and D03/D06 remain open.

[Inline enum tuples](tuple-inline-enums.md) separately extend direct selected String/native enum field own defaults with original strict vocabularies and actual codecs.

[Selected admitted references](tuple-reference-defaults.md) separately support tuple field/root/alias own defaults with actual scalar/enum/ordinary DTO/allOf codecs and original source/value budgets.

[Selected inline ordinary DTO tuples](tuple-inline-dto.md) have a separate field/root/independent alias own-default contract using actual final DTO codecs, strict existing object-default walks and complete discarded-source validation. Positional own annotations and general composed/default graphs remain separate.

[Inline ordinary allOf tuple sources](tuple-inline-allof.md) have a separate actual-constructor and independent outer-default contract. Original source/nullable/required/count/default checks, isolated discarded validation and shared overlapping allOf budgets remain; general compositions stay independent.

[List/typed Map tuple sources](tuple-collection-sources.md) retain actual finite collection codecs, strict independent outer annotations, original backing/source proof and erased-only cached source depth. General tuple/default graphs stay separate.

[JsonNull tuple sources](tuple-null-sources.md) separately admit original plain/null enum/null const positions and accepted reference paths, with actual codecs, strict independent outer defaults, original source proof and existing limits. General const/graph and child default roles remain independent.

[Scalar const tuple sources](tuple-scalar-const.md) separately admit existing scalar const codecs and strict independent outer defaults, preserving original literal/ref/backing identity, nearest nullable and cached/cumulative source budgets. Retained catalog-binding records count toward cumulative capture work; structured const/union/recursive roles remain independent.

[Structured const tuple sources](tuple-structured-const.md) separately admit original JsonObject/JsonArray positions and existing references, with strict complete-literal outer annotations and data-only capture of const/default/enum payloads. Original identity, nullable/Patch and cumulative budgets remain; broader union/recursive/default graphs stay independent.

[Union tuple sources](tuple-union-sources.md) extend D02/D03/D06/F08/M03/M07/Q05 with actual inline/reference union codecs, existing normalization/context projections and independent outer defaults. Original dispatch/pins/required/null/Unknown policies, source/declaration/backing/address proof and erased-source budgets remain. General composition/recursive/default/platform and final candidate acceptance stay open.

[Product union tuple sources](tuple-product-union-sources.md) extend D02/D03/D06/F08/M03/M07/Q05 through actual Cartesian product declarations, original contribution/dispatch/null policies and strict outer own annotations. Original product/source/variant/origin/type/address/cache proof and budgets remain, including discarded validation and internal wrappers. General recursive/hierarchy/intersection/default/platform and final acceptance remain open.

[Hierarchy tuple references](tuple-hierarchy-references.md) extend D01/D02/D03/D06/F08/M03/M07/Q05 through actual named root/intermediate/leaf codecs and strict outer own defaults. Original dispatch/inherited tags/Unknown/null adapters, actual lineage/leaf/source/type/address/cache proof and budgets remain. General inline/multiple-parent/recursive/default/platform and final acceptance stay open.
