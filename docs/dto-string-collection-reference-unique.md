# DTO String enum collection reference getter uniqueness

With validation, ordinary DTO `uniqueItems` can compare finite List/typed Map reference getters whose terminal is an original plain String enum. Named collection aliases, transparent root bridges and eligible internal lexical collections retain their existing Kotlin types. Internal collections containing an inline String enum now borrow the unique original enum declaration, including without validation; they do not create an enum under the consumer's namespace.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "States":{"type":"array","items":{"type":"string","enum":["ready","done"]}},
  "Item":{"properties":{"states":{"$ref":"#/components/schemas/States","default":["ready"]}}},
  "Holder":{"properties":{"items":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}
}}}
```

List order matters; Map entry order does not. Enum leaves compare their runtime `serialName`: a known member and `Unrecognized` with the same wire string have equal keys, while a future unknown string remains a distinct value. Runtime enum decoding remains open. Field-own defaults must use the original declared vocabulary, even when required/Patch handling suppresses their initializer. Defaults on targets and aliases are neither inherited nor newly admitted.

Eligibility retains original schema nodes and Pointers, lexical ownership, actual enum object and physical classifier, member names/wire values, and exactly one retained enum/source occurrence. Existing alias declarations, linked backing addresses and finalized named/borrowed type/check snapshots remain authoritative. The enum key descriptor does not grant nominal permission. Deferred internal enum names use the shared final declaration binding; defaults bind their private helper to the actual consuming type after names are resolved.

Nullable fields/leaves and existing direct dictionary views keep their typed null behavior. Constructor, decode, HTTP and fragmented SSE checks use actual getters in both Data and Plain classes. Native enum validation-off admission is unchanged.

Source depth48 (including incoming/cached alias hops), shared capture work16,384/8MiB, key/value work16,384, rendered defaults32KiB, checked depth16, vocabulary240 and JVM/member slots240 remain enforced. Referenced leaves inside the target tree, formatted/bounded/const/valueclass/native override/nominal/raw/composed/projected leaves and ambiguous or discarded owners retain separate boundaries.

[Core/default/budget controls](../core/dto_string_collection_reference_unique_test.go), [declaration/source authority](../internal/schema/dto_string_collection_reference_unique_test.go), [native failure preservation](../internal/cli/dto_string_collection_reference_unique_process_test.go) and [bounded fuzz](../core/dto_string_collection_reference_unique_fuzz_test.go) cover this role. `bash scripts/test.sh feature dto-string-collection-reference-unique` runs fresh Data/Plain validation controls, optimized locked outputs, 240 reference getters with240 referring arrays, independent depth16, four Json settings, Patch, HTTP and fragmented SSE. Optimized consumer code follows the actual enum classifier when merging emits a typealias.

**Migration:** eligible DTO uniqueness inputs and internal inline-enum collection references previously rejected and now generate. Accepted public APIs, wire behavior, output ownership and lock formats retain their contracts. General graph, platform and final candidate acceptance remain open.
