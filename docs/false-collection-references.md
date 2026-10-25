# Named collections with false-reference leaves

OpenAPI 3.1 named List and typed Map component roots may retain finite inline collection trees whose leaf is a direct named `$ref` to a [false component](false-component-roots.md) or a pure named alias. The generated typealias uses the actual uninhabited nominal classifier and its rejecting serializer.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Rows":{"type":"array","items":{"$ref":"#/components/schemas/Denied"}},
  "Mapping":{"additionalProperties":{"$ref":"#/components/schemas/Denied"}}
}}}
```

`Rows` admits the empty list, and `Mapping` admits the empty map. Every supplied element/value, including JSON null, invokes the false-root codec and rejects. Finite nested List/Map trees can contain empty inner collections, such as a List of Maps of empty Lists. Empty containers are real typed values; the false schema itself still has no valid JSON value.

Nullable leaf aliases and explicit nullable overrides follow the existing nearest-intent policy. A nullable false leaf admits Kotlin null and JSON null, so lists of nulls and maps with null values are available. An explicit false override narrows the leaf again. Root/container nullability is independent. Map entries with nullable values retain JSON null even with `explicitNulls=false`, as in existing typed maps.

Existing references to the retained collection roots, pure collection aliases, ordinary DTO fields and JSON HTTP request/response codecs use those same types. The existing field presence policy applies to the inhabited container; no false-field absence serializer is added. Standalone transparent typealiases retain the established typed-consumer boundary for runtime collection bounds. Count constraints still require validation; uniqueItems admission keeps its existing source/type checks and does not gain a false-element shortcut.

Admission belongs to an actual named collection root and its original inline collection path. The generator verifies original namespace/root/container/leaf/reference/nullable nodes and member lookup consistency, actual unique alias and backing type trees, immutable classifiers and original cached source height, plus the actual rejecting root declaration. Source capture shares the existing cumulative 16,384 false-source work budget. Original reference hops survive transparent lowering and incoming cached aliases; source depth48, checked depth16, JVM slots240 and other limits remain.

This profile does not admit literal false `items`, inline container owners, internal false targets, selected tuple members, arbitrary referenced container paths, defaults, discarded additionalProperties schemas, unions, hierarchy or direct false HTTP/SSE roots. Boolean additionalProperties and literal true keep their existing contracts. No schema is dropped or replaced with raw JSON to make generation succeed.

Data/Plain, suffixes and existing dedup/hoist/merge/name-lock behavior preserve the nominal leaf codec. Previously rejected eligible collection roots now generate. Existing accepted output, configuration and persisted state formats require no migration. This bounded F10/D06 extension does not close general Boolean schemas or platform acceptance.

[Fixture](../tests/fixtures/http/false-collection-references.json), [core/source budgets](../core/false_collection_references_test.go), [actual source/export/backing guards](../internal/schema/false_collection_references_test.go), [native lifecycle](../internal/cli/false_collection_references_process_test.go), and [Kotlin/HTTP consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseCollectionReferencesContracts.kt) cover the profile. `bash scripts/test.sh feature false-collection-references` generates Data/Plain × validation off/on projects with baseline and optimized locked output and four Json configurations. Separate 240-field and 15-level consumers exercise actual Kotlin compilation/runtime; validation-on probes retain count constraints. These JVM probes do not establish Android/KMP or combined maximum-resource support.

[Own defaults on named false collections](false-collection-defaults.md) extend D03/E03 for finite original named List/typed Map trees. Actual root-owned factories validate empty/nested/nullable-leaf literals through the actual nominal codec, retain original literal nodes even under metadata-named JSON keys, and bind actual factory/canonical initializer/type/source/cached-height identity. Source48/work16384/value16384/32KiB/checked16/JVM240 budgets and no inheritance remain. Consuming field/child/wrapper/object-member/composed/internal and broader default/platform/final acceptance remain separate.

[Inline collection fields with false-reference leaves](false-inline-collection-fields.md) separately admit ordinary named Shared DTO direct finite List/Map fields without field/owner defaults. Empty containers remain inhabited, logical nullable leaves admit null, non-null false leaves reject every supplied value, and optional containers retain normal presence rather than direct-false absence policy. Actual retained field/type/check/source/codec proof and existing depth/work/checked/JVM limits remain; other roles and general Boolean/platform acceptance stay separate.
