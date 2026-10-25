# Validating discarded structured const declarations

Fixed-field DTOs, object-union variants and hierarchy leaves accept valid object/array `const` declarations in ignored `additionalProperties` schemas. They also validate structured const fields overridden by a later allOf contribution. These declarations retain their original source nodes and diagnostics, but produce no property, initializer, helper, constructor parameter or typed-value warning.

```json
{"properties":{"value":{"type":"string"}},
 "additionalProperties":{"const":{"type":"literal data","$ref":"literal data"}}}
```

This generates the same Kotlin model as the fixed-field projection with `additionalProperties: true`. `W_ADDITIONAL_PROPERTIES_APPROXIMATION` remains at the original additional-properties schema: the DTO does not collect extra keys or enforce their declared restriction. The generated serializer keeps its existing behavior. Ordinary class serializers reject unknown keys with the default JSON configuration and discard them with `ignoreUnknownKeys = true`; scalar-only singleton serializers retain their existing consume-all behavior. An unrecognized union variant still preserves raw JSON in Unknown.

For an overridden field, the final winning schema determines its Kotlin type, constructor and codec. Required-field union and original allOf ordering remain: a required non-null String winner still needs its constructor/JSON value, even if an earlier nullable const body would have supplied a default. A discarded object const cannot supply a default for a winning String field. Discarded contributions emit no const kind/body warning because their value is not retained.

These roles are distinct from a real [structured body constant](structured-const-fields.md), [pinned data body](pinned-structured-const.md) or [dictionary value](structured-const-values.md). A real body property still has its declared non-null JSON default. A dictionary value still retains its JsonObject/JsonArray codec and rejects other JSON kinds. Ignoring an extra-key declaration does not authorize unsupported schema syntax elsewhere.

Validation runs before discard. Type/kind mismatches, malformed metadata/nullable, unsupported default, format/native/value-class mappings, constraints and sibling schemas still fail at their original Pointers. Literal JSON keys are data, not schema keywords or references. Source48, JSON/document, member240, checked-container and other existing budgets remain. An ignored literal is not a Kotlin constant or a default factory and therefore does not consume a serialized slot or acquire an initializer limit.

The [validation-only path](discarded-const-validation.md) avoids encoding and quoting unused structured initializers. Real body/value initializers, original diagnostic precedence, generated files and name-lock identities remain unchanged; the linked measurements distinguish native generation and core allocations from downstream Kotlin costs.

Migration: these direct extra-key and overridden pinned literals previously failed generation. Original negative inputs remain as explicit acceptance contracts. Previously accepted Kotlin APIs and wire behavior remain. Warnings-as-errors can reject the existing extra-key projection warning before changing output. No persisted-format change, strict const membership or extra-key storage is introduced. Broader const/enum reference/composition/constraint roles and platform/final acceptance remain separate.

Run `bash scripts/test.sh feature discarded-structured-const` for fresh Data/Plain × validation, four JSON settings, optimized name locks, actual 240-member constructors, strict/ignored unknown keys, winning-field codecs, typed body/dictionary controls, controlled HTTP and fragmented live SSE. See the [fixture](../tests/fixtures/http/discarded-structured-const.json), [core contracts](../core/discarded_structured_const_test.go), [native preservation and warning policy](../internal/cli/discarded_structured_const_process_test.go), [fuzz contract](../core/discarded_structured_const_fuzz_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DiscardedStructuredConstContracts.kt).
