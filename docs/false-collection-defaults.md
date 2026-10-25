# Own defaults on named false collections

An OpenAPI 3.1 [named collection with false-reference leaves](false-collection-references.md) may declare its own root `default`. The finite inline List/typed Map tree keeps its original named false leaf and actual serializer. The generator validates the entire source before checking the annotation, including when the value is empty or null.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Rows":{"type":"array","items":{"$ref":"#/components/schemas/Denied"},"default":[]}
}}}
```

With a `Dto` suffix, import and call the top-level factory `RowsDto default` using Kotlin backticks. It decodes canonical JSON through the collection's actual serializer, following the [existing component factory API](component-default-factories.md). Each call supplies fresh collection state. A typealias has no constructor or companion factory.

A non-null false leaf admits no default value: `[null]`, `[1]` and nonempty maps with such values fail at the original annotation member Pointer. Empty containers and nested empty containers are valid when they satisfy the existing collection checks. Nullable leaf aliases admit null values; an outer nullable collection may independently have a null default. Nearest explicit nullable overrides, count validation opt-in and strict literal checks remain unchanged.

The annotation belongs only to that actual named collection root. References, pure aliases and required/optional DTO fields do not inherit it. This increment does not grant own defaults to consuming fields, children, reference wrappers, object-default members or false component aliases. Literal false `items`, tuples, internal targets, unions and hierarchy keep their separate boundaries. A false schema itself still acquires no valid JSON value.

The generator binds the actual emitted factory to its original declaration, canonical initializer and original annotation nodes. JSON map keys that resemble schema metadata, such as `description`, remain literal data during this proof. Replacing or changing those values, factory code or actual type/backing cannot reuse equal-text source authority. Original containers, references, alias nodes, cached heights and actual root codec keep the named-collection contract.

Source captures share the cumulative 16,384 false-source work budget. Literal validation retains 16,384 value nodes and 32 KiB canonical JSON, with source depth48, checked depth16 and JVM slots240 unchanged. These independent limits do not promise acceptance at every combined maximum. Invalid sources or defaults return structured errors and preserve existing output.

Previously rejected eligible own annotations now emit factories. Existing accepted Kotlin/wire behavior, naming locks, configuration and manifest formats remain unchanged. Data/Plain and existing dedup/hoist/merge settings preserve the factory's own identity and codec. General defaults, Boolean schemas, platforms and final plan acceptance remain open.

[Fixture](../tests/fixtures/http/false-collection-defaults.json), [core/value/role/budget contracts](../core/false_collection_defaults_test.go), [factory and literal source guards](../internal/schema/false_collection_defaults_test.go), [native parity/failure preservation](../internal/cli/false_collection_defaults_process_test.go), and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseCollectionDefaultsContracts.kt) cover this profile. `bash scripts/test.sh feature false-collection-defaults` generates Data/Plain × validation off/on projects with baseline and optimized locked output. Factory decoding uses the established default Json configuration; consumers exercise four Json encoding/decoding settings and controlled JSON HTTP. Additional probes compile and run a 32766-byte factory, 240 reference fields, and a 15-level typed model with count checks when enabled. This is JVM evidence, not Android/KMP or a combined maximum-resource guarantee.

[Nullable false alias own null factories](false-alias-defaults.md) extend D03/E03 with original pure named aliases whose nearest nullable intent is true. Each own annotation validates independently and binds the actual nullable-root codec, canonical null factory, original owner/reference/nullable/value nodes and actual declarations. Existing collection factories may consume independently annotated aliases without inheritance. Source48/work16384/value/bytes/checked16/JVM240 budgets, field presence, API/wire and persisted state remain. Consuming field/child/object-member/default and direct false HTTP/composed/internal/general/platform roles remain separate.
