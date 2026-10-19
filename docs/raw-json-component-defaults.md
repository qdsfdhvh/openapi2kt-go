# Raw JSON carrier component defaults

A component whose root is a raw JSON carrier may carry its own `default`. The
grammar is the same one the [field role](raw-json-field-defaults.md) already
defines; this adds the component entry point.

Two roots are admitted:

- an **open dictionary** (`{"type":"object","additionalProperties":true}`), whose
  declared type is `JsonObject` and whose annotation must be a JSON object;
- an **unconstrained / empty object** (`{"type":"object"}` or a schema whose
  `properties` is absent or empty), whose declared type is `JsonElement` and
  which accepts any JSON value.

```json
{"Open": {"type": "object", "additionalProperties": true, "default": {"a": 1}}}
{"Free": {"type": "object", "default": [1, "two", false]}}
```
```kotlin
public typealias OpenDto = kotlinx.serialization.json.JsonObject
public fun `OpenDto default`(): OpenDto = `openapi-kt component default …`("{\"a\":1}")

public typealias FreeDto = kotlinx.serialization.json.JsonElement
public fun `FreeDto default`(): FreeDto = `openapi-kt component default …`("[1,\"two\",false]")
```

Because these roots are `typealias` projections, the factory is a top-level
function. The emitter previously wrote the alias and dropped any component
default on the empty-object projection; that is fixed here, so a rejected
annotation can no longer disappear silently.

## Admission and exclusions

Admission mirrors the field role exactly: `$ref`, `allOf`, `oneOf`/`anyOf`,
`enum`, `const`, `format`, `items`, `x-native-type` and `x-kotlin-value-class`
keep their own roles. In particular a **typed** dictionary
(`additionalProperties` as a schema) is the collection role, not a raw carrier,
and a closed object with declared properties is the ordinary object role.

A `default: null` annotation keeps the nullable-root contract: these raw roots
are not nullable roles, so a null literal stays rejected at `.../default`.

## Diagnostics and resources

A wrong-kind annotation reports at the original `.../default` Pointer, and the
open dictionary requires a JSON object there. The shared input 8 MiB, source 48,
480-slot and value-tree 16,384-node / 32 KiB budgets are unchanged; the
`W_JSON_ELEMENT_APPROXIMATION` warning is emitted once by the shared type plan
and is not duplicated.

## Compatibility

This is an additive role for previously rejected input. The carrier declarations,
their warning and every document without the annotation are unchanged; only the
opt-in factory is added. A field referencing the carrier keeps its neutral
`= null` and never inherits the default.

## Verification

- [Core admission, guards, non-inheritance](../core/raw_json_component_defaults_test.go)
- [Native parity, readonly check and failure preservation](../internal/cli/raw_json_component_defaults_process_test.go)
- [Fixture](../tests/fixtures/http/raw-json-component-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawJSONComponentDefaultsContracts.kt); run `bash scripts/test.sh feature raw-json-component-defaults` for fresh Data/Plain × validation-off/on projects under four `Json { encodeDefaults, explicitNulls }` configurations.
