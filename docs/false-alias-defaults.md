# Own null factories on nullable false aliases

A pure named OpenAPI 3.1 `$ref` alias ending at a [literal false component](false-component-roots.md) may own `default: null` when its effective nearest nullable intent is true. It retains the actual root classifier and nullable serializer wrapper.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "MaybeDenied":{"$ref":"#/components/schemas/Denied","nullable":true,"default":null},
  "Inherited":{"$ref":"#/components/schemas/MaybeDenied"},
  "OwnInherited":{"$ref":"#/components/schemas/MaybeDenied","default":null}
}}}
```

With a `Dto` suffix, the generated top-level factory `MaybeDeniedDto default` returns the nullable alias value through the existing Json component-factory decoder. Import and call its name using Kotlin backticks. `OwnInheritedDto default` is independent; `InheritedDto` gains no factory. Null is the only value admitted by this own-annotation profile. Every non-null default fails at the original `/default` Pointer, even on an otherwise unused alias.

Nearest explicit nullable intent wins. An alias may inherit true or explicitly restore true after a false intermediate. A non-null narrowed alias has no valid null annotation and keeps its refusal boundary. Other alias annotations remain independent and must validate on their own sources; one valid null factory cannot hide an invalid target factory.

Ordinary typed fields and supported named collections keep their existing actual codecs and presence policies. [Collection root factories](false-collection-defaults.md) may reference independently valid annotated false aliases without inheriting their annotations. Consuming fields, child schemas and object-default members do not acquire own default permission here. Direct false HTTP request/response/parameter/SSE roots retain their separate boundaries. Existing ordinary DTO JSON HTTP use is unchanged.

A nullable alias has no new classifier, constructor or companion. The non-null canonical false codec still rejects every JSON value; the standard nullable wrapper admits Kotlin/JSON null. Shared required-nullable fields retain their existing missing/default-null and `explicitNulls` behavior. Root serialization of a nullable alias still writes JSON null independently of DTO key omission settings.

Generation binds the actual alias factory, canonical `null` initializer and original null literal to the original named owner, reference/nullable nodes, unique actual alias and rejecting root declarations. Copied source nodes, missing factory authority, changed code or a replacement nullable raw codec cannot gain this permission. Each original annotation and its actual factory remain separate from consumer defaults.

Source depth48 and cumulative false-source work16,384 include the new null-node proof. Existing literal node/32 KiB, checked-depth16, JVM slots240, enum and input limits remain. Data/Plain, suffixes, deterministic dedup/hoist/merge and naming-lock behavior preserve the actual codec and factory identity. Invalid generation preserves old output.

Previously rejected eligible own annotations now emit factories; existing accepted Kotlin/wire and persisted formats require no migration. General false/default graphs and platform/final plan acceptance remain open.

[Fixture](../tests/fixtures/http/false-alias-defaults.json), [core/roles/budgets](../core/false_alias_defaults_test.go), [factory/source guards](../internal/schema/false_alias_defaults_test.go), [native lifecycle](../internal/cli/false_alias_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseAliasDefaultsContracts.kt) exercise this profile. `bash scripts/test.sh feature false-alias-defaults` generates Data/Plain × validation off/on projects with baseline and optimized locked output, four Json consumer settings and controlled ordinary DTO HTTP. Factory decoding retains the existing default Json configuration. Separate probes compile/run a 48-hop alias factory and 240 fields. This is JVM evidence, not Android/KMP or combined maximum-resource acceptance.

[False-reference field own null defaults](false-reference-field-defaults.md) extend D03/E03 for actual ordinary named Shared DTO fields with logical nullable intent before presence wrapping. Only null validates; required arguments keep no initializer and optional arguments initialize null, using existing Json explicitNulls rules and actual rejecting-root nullable codecs. Original owner/property/required/default/type/initializer/omission authority, source48/work16384/value/bytes/checked16/JVM240 and no inheritance remain. Other default/object-member/child/composed/internal/directional/inline/direct false HTTP/platform/general roles stay separate.
