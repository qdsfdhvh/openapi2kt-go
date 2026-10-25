# References inside DTO collection getters

With validation, ordinary DTO `uniqueItems` keys admit existing finite List/typed Map getters whose original tree contains supported scalar or plain String/native enum reference leaves. The consuming field may be inline or a named/internal collection reference. Original finite reference chains can cross container edges; the existing type resolver still decides whether every target is supported. Ordinary DTO, raw/projected, composed, const, valueclass and caller-native terminals do not gain this equality role.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "State":{"type":"string","enum":["ready","done"]},
  "Item":{"properties":{"states":{"type":"array","items":{"$ref":"#/components/schemas/State"},"default":["ready"]}}},
  "Holder":{"properties":{"items":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}
}}}
```

Keys compare actual getters structurally: List order matters and Map entry order does not. Scalars retain mapped Float/Double zero, UUID and Instant equality; String enums compare open runtime `serialName`, including known versus same-name `Unrecognized`. Existing typed nullability and nearest explicit reference override remain. Enum defaults are strict against the original vocabulary; required/Patch suppression happens after annotation validation.

Ordinary consuming inline collection fields also gain their own strict default role for these finite scalar-reference trees. They use the shared deferred reference/default planner and final actual typed helpers. Target, alias and child defaults are not inherited or newly admitted. The original unchecked native enum inline-default gate stays closed; use validation for that role. Existing standalone alias and unchecked runtime enforcement contracts are unchanged.

Original reference/container paths, source schema nodes and Pointers, raw annotations, real catalog/lexical owners and source work are retained. Final declaration/alias/backing/check identity remains required. An original retained-getter registry records actual linked TypeRef nodes, validation objects and immutable check values after nominal linking, including sources first used during later HTTP planning. A changed shared bound or a copied same-value validation object cannot authorize the final key. String enums additionally require the original actual declaration, physical name, vocabulary/member identity and unique retained source/declaration occurrence.

The original source48/cached/incoming-hop limits, shared capture16,384/8MiB, key/value work16,384, default32KiB, checked depth16 and member/JVM slots240 remain. Cyclic transparent trees, unsupported projected/nominal leaves, ambiguous/discarded declaration owners and target/alias/child annotations retain their boundaries. No AST expansion, synthetic catalog or declaration is used.

[Core and exact former refusals](../core/dto_collection_reference_leaves_unique_test.go), [actual getter/check authority](../internal/schema/dto_collection_reference_leaves_unique_test.go), [native parity and failure preservation](../internal/cli/dto_collection_reference_leaves_unique_process_test.go) and [bounded deterministic fuzz](../core/dto_collection_reference_leaves_unique_fuzz_test.go) cover this role. `bash scripts/test.sh feature dto-collection-reference-leaves-unique` runs fresh Data/Plain validation controls, actual merged enum classifiers, optimized locks, 240 collection reference getters ×240 referring arrays, depth16, four Json settings, Patch, HTTP and fragmented SSE.

**Migration:** eligible DTO getter uniqueness and inline field-own default inputs previously rejected and now generate. Existing accepted APIs, wire behavior, output ownership and lock formats retain their contracts. Wider nominal/composed/default graphs, platforms and final candidate acceptance remain open.
