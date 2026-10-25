# Structured const fields

Plain object and array `const` values in ordinary DTO fields become `JsonObject` and `JsonArray` body properties. Explicit matching `type` and omitted `type` are supported. Literal keys such as `$ref`, `type`, `enum` and `const` remain data. The original schema and value nodes are retained; no inferred schema is substituted.

```json
{
  "properties": {
    "label": {"type": "string"},
    "payload": {"type": "object", "const": {"version": 1}},
    "rows": {"const": [null, true]}
  }
}
```

The generated body exposes `val payload: JsonObject` and `val rows: JsonArray`. Private file helpers initialize the values with the kotlinx.serialization JSON parser. Constants do not become constructor or `copy` parameters, nullable fields or Patch values, even when required or annotated nullable. Const-only ordinary models remain plain classes under both model styles.

Missing JSON keys use the constant defaults. Encoding always includes the constants under every `encodeDefaults`/`explicitNulls` combination. A supplied object or array of the correct kind is retained and re-encoded, even when its contents differ from the declared literal. Wrong kinds and JSON null fail the typed serializer. `W_CONST_STRUCTURED_APPROXIMATION`, located at each retained original `/const`, documents this approximation. `--validation` does not add literal-membership enforcement. This matches ordinary scalar body constants rather than promising exact JSON Schema `const` validation.

Nested ordinary DTOs, object `allOf` field sources, retained nominal DTO references, typed HTTP bodies/results and SSE DTO payloads share this behavior. Literal values participate in shape merging and name-lock identity; changing a locked retained constant changes the shape. Source checks still run for overridden/discarded fields, while discarded scratch constants do not leak warnings.

The field profile excludes root/reference/list/map values, which have the separate [structured value contract](structured-const-values.md). Union/hierarchy data bodies have the separate [pinned body contract](pinned-structured-const.md); discarded extra-key declarations have the separate [validation-only contract](discarded-structured-const.md). `const` plus `default`, native/format/value-class mappings, sibling object/array schemas and validation constraints remain diagnosed at original pointers. General nullable/reference/composition graphs and structured defaults have separate contracts.

Document JSON depth and 8 MiB input limits, schema depth 48, composition work 16,384, checked collection depth 16 and 240 serialized/method slots remain unchanged. Literal container depth uses the document loader's budget; literal keys are not traversed as schema edges. Numeric tokens and resulting encoding inherit the backing JSON codec's representation/precision behavior. This does not promise arbitrary-precision numbers or preservation of every numeric spelling through decode/re-encode.

Migration: these ordinary field profiles were previously rejected. Existing accepted schemas retain generated APIs, warnings, file bytes and name-lock identities. The new warning can reject a newly admitted schema when warnings are configured as errors. Independent boundary probes do not establish Cartesian maximum-size behavior, minimum Kotlin/Gradle compatibility, Android or KMP runtime support.

Run `bash scripts/test.sh feature structured-const-fields` for fresh Data/Plain × validation projects and four JSON configurations with kind/default/Patch, controlled HTTP and fragmented live SSE error/close/recovery assertions. The [fixture](../tests/fixtures/http/structured-const-fields.json), [core contracts](../core/structured_const_fields_test.go), [source guards](../core/structured_const_fields_guards_test.go), [resource/lock checks](../core/structured_const_fields_resources_test.go), [original-node checks](../internal/schema/structured_consts_test.go), [native preservation](../internal/cli/structured_const_fields_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StructuredConstFieldsContracts.kt) define the bounded surface.
