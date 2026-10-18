# Nullable component root defaults

A component whose root is **explicitly nullable** may carry a **non-null**
`default`. The annotation describes the value used when the whole value is
absent; presence and null stay independent of it.

```json
{"Name": {"type": ["string", "null"], "default": "fallback"}}
```
```kotlin
public typealias NameDto = kotlin.String?

// The OpenAPI component default for this schema.
public fun `NameDto default`(): NameDto = `openapi-kt component default …`("\"fallback\"")
```

The declaration keeps its nullable public type, so the factory returns a
non-null instance of it. A supplied `null` stays `null`; the factory never forces
the value to be present.

## Admitted shapes

This is the same set the non-nullable component root already admits, with its
per-shape grammar unchanged:

| Root shape | Factory form |
|---|---|
| plain scalar (`type: ["string", "null"]` and siblings) | top-level function on the nullable alias |
| standard temporal (`date` / `date-time`, nullable) | top-level function on the nullable alias |
| bytes (`format: byte` / `binary`, nullable) | top-level function on the nullable alias |
| single-layer List / typed Map (nullable) | top-level function on the nullable alias |
| plain String enum (nullable) | top-level function beside the existing companion |
| numeric/Boolean native enum (nullable, validation opt-in) | top-level function on the nullable alias |
| ordinary object (`nullable: true`) | companion `default()` returning the nullable type |
| disjoint ordinary `allOf` constructor (`nullable` true) | companion `default()` returning the nullable type |

Validation reuses the existing planner for each shape: required members,
declared-property coverage, leaf kinds, enum vocabularies, counts and bounds all
apply, and failures report at the original `.../default/...` Pointer.

## The null literal stays separate

`default: null` keeps its existing rule: it is admitted only for an ordinary
nullable **object** root, where the factory returns `null` directly without a
decode helper. Every other nullable shape keeps its rejection at `.../default`:

```json
{"N": {"type": ["string", "null"], "default": null}}   // E_SCHEMA_UNSUPPORTED
```

The reason is that a `typealias` factory must produce a value of its own declared
type; a null literal there would need a separately designed nullable factory
contract, so it is not part of this increment.

## Whole-object counts

A nullable root that also declares `minProperties` / `maxProperties` keeps its
separate nullable-component-counts role, with or without a non-null annotation.
This contract does not aggregate counts into the factory.

## Non-inheritance

A field referencing the component keeps its neutral `= null` and never picks up
the factory, matching every other component default role. A schema without the
annotation emits exactly its previous bytes.

## Compatibility

This is an additive role for previously rejected input. `default: null` on a
nullable object root, non-nullable component roots, references, and every
document without a nullable component annotation retain their existing output.
The declaration's Kotlin type, constructor, serializer and wire behavior are
unchanged; only the opt-in factory is added.

## Verification

- [Core admission, guards, non-inheritance](../core/nullable_component_defaults_test.go)
- [Original-node authority and canonical value](../internal/schema/component_default_factories_test.go)
- [Native parity, readonly check and failure preservation](../internal/cli/nullable_component_defaults_process_test.go)
- [Fixture](../tests/fixtures/http/nullable-component-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableComponentDefaultsContracts.kt); run `bash scripts/test.sh feature nullable-component-defaults` for fresh Data/Plain × validation-off/on projects under four `Json { encodeDefaults, explicitNulls }` configurations.
