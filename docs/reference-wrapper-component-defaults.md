# Reference wrapper component defaults

A component whose body is only `$ref` plus an optional `nullable` is emitted as a canonical alias. When it also carries `default`, the **alias** owns the factory and the target gains nothing.

```json
{"Target": {"type": "object", "required": ["z"], "properties": {"z": {"type": "string"}}},
 "Wrapped": {"$ref": "#/components/schemas/Target", "default": {"z": "1"}}}
```
```kotlin
public typealias TargetDto = …
public typealias WrappedDto = TargetDto
public fun `WrappedDto default`(): WrappedDto = `openapi-kt component default … WrappedDto`("{\"z\":\"1\"}")
```

A typealias cannot own members, so the factory is a top-level function, exactly like the scalar, enum, temporal, bytes and collection component roots. It decodes through the alias's own serializer, which resolves to the target's, so the produced value is a real target instance.

## A wrapper default is not inherited

A field that references a defaulted wrapper keeps its neutral presence and does **not** pick up the value:

```kotlin
public val `object`: TargetDto? = null,     // $ref to Wrapped: no inheritance
```

This is the same rule every other component default follows, and it is asserted directly rather than inferred from a rejection.

## How the value is validated

The wrapper's shape lives on the target, while the annotation lives on the wrapper. The generator builds a **throwaway** schema node combining the target's shape with the wrapper's annotation, runs the existing default walk against it, and reports every diagnostic at the wrapper's own `.../default` Pointer. The target's original node is never modified, so its identity, Pointer and emitted output are unchanged.

## Accepted targets

A wrapper is admitted exactly when the target's own root default role exists **and** the target's alias role is itself admitted:

| Target | Result |
|---|---|
| ordinary object, List/typed Map | factory on the wrapper alias |
| String enum, standard temporal, bytes | factory on the wrapper alias |
| plain scalar (`string`/`integer`/`number`/`boolean`) | factory only with validation enabled — a `$ref` to a plain scalar is itself a validation-opt-in alias |
| chained alias (`S → M → T`) | `E_SCHEMA_UNSUPPORTED @ …/S/default` |
| `nullable` wrapper | `E_SCHEMA_UNSUPPORTED @ …/S/default` |
| target that already carries its own `default` | `E_SCHEMA_UNSUPPORTED @ …/S/default` |
| missing reference, union, `oneOf`/`anyOf`, `const` | `E_SCHEMA_UNSUPPORTED @ …/S/default` |

Every rejected row reports at the wrapper's own `.../default` — the same location as before this contract, so no shape loses diagnostic precision.

A bad value keeps the target's own grammar *and* the wrapper's location:

```
E_SCHEMA_INVALID @ /components/schemas/S/default        (object missing a required member)
E_SCHEMA_INVALID @ /components/schemas/S/default/0      (bytes element outside [-128, 127])
```

## Runtime stays per-role

The wrapper does not change any runtime policy. A String enum keeps its open vocabulary and surfaces an unknown wire member as `Unrecognized`; a DTO keeps its required-member check; a bytes element outside `[-128, 127]` is a generator-time check because `ByteArraySerializer` narrows with `toByte()` instead of rejecting.

## Nested classifiers

A typealias cannot reach nested classifiers, so `WrappedDto.Member` does not compile where `TargetDto.Member` does. Access members through the canonical owner, as documented in the [DTO reference contract](internal-dto-references.md).

## Verification

- [Fixture](../tests/fixtures/http/reference-wrapper-component-defaults.json), [core contract](../core/reference_wrapper_component_defaults_test.go) (alias-owned factory, target untouched, canonical JSON, per-role validation, excluded roles and Pointers, annotation-less alias unchanged), and the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferenceWrapperComponentDefaultsContracts.kt).
- `bash scripts/test.sh feature reference-wrapper-component-defaults` runs fresh Data/Plain × validation-off/on projects and four `Json { encodeDefaults, explicitNulls }` configurations, asserting the factory output and that no field inherits it.
- A component without the annotation emits byte-identical output; the factory is purely additive.
