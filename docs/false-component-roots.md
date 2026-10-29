# Literal false component roots

An OpenAPI 3.1 named component whose schema is literal `false` exports an uninhabited Kotlin sealed class with a private constructor and its own serializer. The non-null root has no valid JSON value. Reified serializer/encoder call sites compile, and decoding null, scalar, array or object values throws `SerializationException`. The generated serializer contains explicit serialization and deserialization refusal rather than a placeholder value or raw JSON projection.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Alias":{"$ref":"#/components/schemas/Denied"},
  "MaybeDenied":{"$ref":"#/components/schemas/Alias","nullable":true}
}}}
```

Pure named component aliases retain the actual root type/serializer and independent original alias nodes. Nearest explicit nullable intent follows the existing root-alias policy: a nullable alias admits Kotlin null/JSON null through the nullable serializer wrapper, absent intent inherits that view, and explicit false narrows it again. Every non-null value still rejects. Separate false components keep separate nominal classifiers/descriptors even though both have empty non-null domains.

This role accepts only actual named literal-false component roots and eligible pure local named component aliases. [Ordinary direct named reference fields](false-reference-fields.md) have a separate presence/codec contract. [Named collections](false-collection-references.md) have a separate retained leaf-codec contract. Other parameter/direct false request/response/SSE/default/const/union/hierarchy/internal-source and inline literal false roles retain their existing refusal boundaries. OpenAPI 3.0 false remains excluded. Literal true and Boolean additionalProperties retain their own contracts.

Data/Plain presentation leaves these classes sealed and uninhabited. Deduplication, hoisting and merging keep their nominal type, descriptor and aliases opaque. Suffixes and existing naming/filename reservations apply; serializer member names and fully qualified self types preserve roots named Serializer/Companion. Original source/catalog/namespace/member/lookup identities, unique actual root/alias declarations, alias nullable intent and cached source heights authorize the output. Source captures share a 16,384 work budget across this new role; original document, source depth48, default, checked-depth and JVM member limits remain.

[Core style/options/depth/resource/refusal checks](../core/false_component_roots_test.go), [source/declaration/alias guards](../internal/schema/false_component_roots_test.go), [native parser/parity/old-output checks](../internal/cli/false_component_roots_process_test.go) and [bounded deterministic fuzz](../core/false_component_roots_fuzz_test.go) define this contract. Original root-false negative inputs and the `component-boolean` corpus fixture remain in place with explicit executable positive checks; other false contexts remain refusals.

`bash scripts/test.sh feature false-component-roots` runs Data/Plain × validation off/on, each with an unsuffixed baseline and optimized/locked Dto output. Its [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseComponentRootsContracts.kt) compiles reified encoder call sites, rejects every JSON kind through actual root/alias codecs, verifies nullable alias null behavior and distinct descriptors, and exercises Serializer/Serializer2/Companion names across four Json settings. PREPARE_ONLY prepares source without compiling or executing it. The complete `http` route also executes the standalone codec consumer; that is not false HTTP-body admission.

**Migration:** eligible roots/aliases previously rejected and now generate. Existing accepted API/wire/warning behavior, output ownership and name-lock formats retain their contracts. General Boolean false values, typed consumer roles, broader composition/default/recursive graphs and platform/final candidate acceptance remain separate.

[Named collections with false-reference leaves](false-collection-references.md) extend F10/D06/E05 with finite original List/typed Map roots, actual nominal leaf codecs, empty-container and nullable-leaf semantics, existing referenced DTO/JSON HTTP consumers, immutable source/export/backing/cached-height authority and source48/work16384/checked16/JVM240 limits. Inline/literal-false/default/tuple/composed/internal/discarded-extra-key and direct false HTTP-root roles remain separate; broader plan/platform/final acceptance stays open.

[Own defaults on named false collections](false-collection-defaults.md) extend D03/E03 for finite original named List/typed Map trees. Actual root-owned factories validate empty/nested/nullable-leaf literals through the actual nominal codec, retain original literal nodes even under metadata-named JSON keys, and bind actual factory/canonical initializer/type/source/cached-height identity. Source48/work16384/value16384/32KiB/checked16/JVM240 budgets and no inheritance remain. Consuming field/child/wrapper/object-member/composed/internal and broader default/platform/final acceptance remain separate.

[Nullable false alias own null factories](false-alias-defaults.md) extend D03/E03 with original pure named aliases whose nearest nullable intent is true. Each own annotation validates independently and binds the actual nullable-root codec, canonical null factory, original owner/reference/nullable/value nodes and actual declarations. Existing collection factories may consume independently annotated aliases without inheritance. Source48/work16384/value/bytes/checked16/JVM240 budgets, field presence, API/wire and persisted state remain. Consuming field/child/object-member/default and direct false HTTP/composed/internal/general/platform roles remain separate.

[Named true/false tuple references](tuple-boolean-references.md) have a separate original-root/actual-codec contract, including independent outer defaults, nullable false elements and complete erased-prefix source-depth accounting. General Boolean graphs remain independent.
