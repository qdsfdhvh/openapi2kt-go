# Raw JSON carrier component defaults

A component whose root is a raw JSON carrier may carry its own `default`. The
grammar is the same one the [field role](raw-json-field-defaults.md) already
defines; this adds the component entry point.

Two roots are admitted:

- an **open dictionary** (`{"type":"object","additionalProperties":true}`), whose
  declared type is `JsonObject` and whose annotation must be a JSON object;
- an **unconstrained / empty object** (`{"type":"object"}` or a schema whose
  object type is inferred from empty `properties`), whose declared type is `JsonElement` and
  which accepts any non-null JSON value; null uses the explicit nullable role below.

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

An explicitly nullable direct carrier root also accepts `default: null`. Its
alias remains `JsonElement?` or `JsonObject?`, and the top-level factory decodes
`"null"` through that actual nullable alias serializer, returning Kotlin null.
Both `nullable: true` and the supported `type: ["object", "null"]` form apply;
optional consuming fields alone do not authorize null. Non-nullable raw roots
keep `E_SCHEMA_UNSUPPORTED` at their original `.../default` Pointer.

```json
{"Missing": {"type": ["object", "null"], "additionalProperties": true, "default": null}}
```
```kotlin
public typealias MissingDto = kotlinx.serialization.json.JsonObject?
public fun `MissingDto default`(): MissingDto = `openapi-kt component default …`("null")
```

The factory does not change the alias or serializer. Nullable raw codecs decode
JSON null as Kotlin null; a narrowed non-null `JsonElement` still represents it
as `JsonNull`. The String `"null"` remains a JSON String, and null members inside
a raw object remain `JsonNull` under all `explicitNulls` settings. Existing
non-null factories retain their behavior. Malformed source declarations,
whole-object counts and const/union/hierarchy roles retain their separate restrictions.

Pure `$ref` wrappers over these admitted carriers also support factories under
the [reference wrapper contract](reference-wrapper-component-defaults.md#raw-json-carrier-targets).
The wrapper owns the annotation and uses the target's actual alias/null intent;
the target and consuming fields never inherit it. Finite pure component alias
chains are admitted under the [chain rules](reference-wrapper-component-defaults.md#finite-pure-component-alias-chains).
Outer and intermediate nullable booleans follow the [nearest-intent wrapper profile](reference-wrapper-component-defaults.md#nullable-intent-across-component-aliases). Admitted target/intermediate defaults are [independently owned](reference-wrapper-component-defaults.md#independent-defaults-on-target-and-intermediate-components); a referencing alias or field never inherits them.
A completely untyped `{}` component does not acquire a root factory role here.

## Diagnostics and resources

A wrong-kind annotation reports at the original `.../default` Pointer, and the
open dictionary requires a JSON object there. The shared 8 MiB input, source depth 48,
constructor limits and value-tree 16,384-node / 32 KiB budgets are unchanged; the
`W_JSON_ELEMENT_APPROXIMATION` warning is emitted once by the shared type plan
and is not duplicated.

## Compatibility

This is an additive role for previously rejected input. The carrier declarations,
their warning and every document without the annotation are unchanged; only the
opt-in factory is added. A field referencing the carrier keeps its neutral
`= null` and never inherits the default.

## Verification

- [Core admission, guards, non-inheritance](../core/raw_json_component_defaults_test.go), [nullable null factories and original diagnostics](../core/nullable_raw_component_defaults_test.go), and [original source/carrier identity](../internal/schema/nullable_raw_component_defaults_test.go)
- [Raw wrapper admission, exclusions and budgets](../core/raw_json_wrapper_defaults_test.go), and [original wrapper/target authority](../internal/schema/raw_json_wrapper_defaults_test.go)
- [Native parity, readonly check and failure preservation](../internal/cli/raw_json_component_defaults_process_test.go)
- [Fixture](../tests/fixtures/http/raw-json-component-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawJSONComponentDefaultsContracts.kt); run `bash scripts/test.sh feature raw-json-component-defaults` for fresh Data/Plain × validation-off/on projects under four `Json { encodeDefaults, explicitNulls }` configurations.
