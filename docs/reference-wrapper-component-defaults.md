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
| ordinary object, finite inline List/typed Map tree (including supported nullable layers) | factory on the wrapper alias |
| String enum, native enum under its existing admission rules, standard temporal, bytes | factory on the wrapper alias |
| admitted raw `JsonElement` / `JsonObject` carrier | factory on the wrapper alias, using the target's raw-value and null grammar |
| plain scalar (`string`/`integer`/`number`/`boolean`) | factory only with validation enabled — a `$ref` to a plain scalar is itself a validation-opt-in alias |
| finite pure component alias chain (`S → M → T`) | factory on S when every intermediate node meets the rules below |
| `nullable` wrapper | `E_SCHEMA_UNSUPPORTED @ …/S/default` |
| target that already carries its own `default` | `E_SCHEMA_UNSUPPORTED @ …/S/default` |
| missing reference, union, `oneOf`/`anyOf`, `const` | `E_SCHEMA_UNSUPPORTED @ …/S/default` |

Every rejected row reports at the wrapper's own `.../default` — the same location as before this contract, so no shape loses diagnostic precision.

A bad value keeps the target's own grammar *and* the wrapper's location:

```
E_SCHEMA_INVALID @ /components/schemas/S/default        (object missing a required member)
E_SCHEMA_INVALID @ /components/schemas/S/default/0      (bytes element outside [-128, 127])
```

## Finite pure component alias chains

A wrapper may own a default through a finite chain of local component-root references:

```json
{"T": {"type": "array", "items": {"additionalProperties": {"type": "integer"}}},
 "B": {"$ref": "#/components/schemas/T"},
 "M": {"$ref": "#/components/schemas/B"},
 "S": {"$ref": "#/components/schemas/M", "default": [{"z": 1}]}}
```

Only S receives a factory. T, B, M and referencing fields retain their original output; the factory decodes through `serializer<SDto>()`. Each intermediate component must be a pure `$ref` with only otherwise accepted metadata. Any intermediate `default` or explicit `nullable` (including `false`) is rejected at S's own `/default`. The outer wrapper also cannot carry explicit `nullable`, and the terminal target cannot carry its own default. The target's existing grammar, null permission and validation gates apply; for example, plain scalar wrappers still require validation, and this does not add a null factory for a nullable plain scalar wrapper.

Every hop must name `#/components/schemas/<Name>`. Missing targets, cycles, malformed references, internal/property references and chains over 48 hops are rejected. The ordinary resolver still counts the combined alias, target and consumer depth against its source-depth limit, including cached targets: 48 hops is a traversal cap, not a promise that a chain of that length is usable in every context. This adds no general reference graph or nullable-override support.

Target and intermediate source nodes are never rewritten. Value errors, including escaped dictionary keys, retain the outer annotation's Pointer. Original/canonical 32 KiB JSON and 16,384-node default budgets remain shared with direct wrappers. Factories return fresh structured values; runtime codecs and required/missing/null/supplied behavior remain the target's own.

This additive D03/M03/M04/M07 profile accepts previously rejected annotations. Previously accepted inputs and persisted lock formats require no migration.

## Raw JSON carrier targets

Pure `$ref` wrappers over admitted [raw component carriers](raw-json-component-defaults.md) use the raw carrier validator. A `JsonElement` target accepts non-null object, array and scalar annotations; a `JsonObject` target requires an object. A null annotation requires explicit nullability on the target and decodes to Kotlin null through the wrapper's actual nullable alias serializer. A non-nullable raw target keeps `E_SCHEMA_UNSUPPORTED` at the wrapper's own `.../default`, even though the standalone non-null `JsonElement` codec can represent JSON null as `JsonNull`.

```json
{"Raw": {"type": "object"},
 "Open": {"type": ["object", "null"], "additionalProperties": true},
 "RawDefault": {"$ref": "#/components/schemas/Raw", "default": [1, null]},
 "NullOpenDefault": {"$ref": "#/components/schemas/Open", "default": null}}
```

The wrapper owns the factory, and the target keeps its original source and generated bytes. Raw object members retain `JsonNull` under every `explicitNulls` setting; the String `"null"` remains a String. Required fields keep required constructor arguments, optional fields do not inherit the wrapper annotation, and structured factories decode fresh values on each call. Source validation, original annotation Pointers, target warnings and the existing 16,384-node / 32 KiB default budgets remain unchanged. Explicit `nullable` on the wrapper or intermediate aliases, intermediate/target defaults and other excluded targets retain their restrictions. Finite pure component alias chains follow the rules above.

This adds factories for previously rejected raw wrapper annotations (D03/M03/M04/M07). Previously accepted input, serializers and lock formats require no migration.

## Runtime stays per-role

The wrapper does not change any runtime policy. A String enum keeps its open vocabulary and surfaces an unknown wire member as `Unrecognized`; a DTO keeps its required-member check; a bytes element outside `[-128, 127]` is a generator-time check because `ByteArraySerializer` narrows with `toByte()` instead of rejecting.

## Nested classifiers

A typealias cannot reach nested classifiers, so `WrappedDto.Member` does not compile where `TargetDto.Member` does. Access members through the canonical owner, as documented in the [DTO reference contract](internal-dto-references.md).

## Verification

- [Fixture](../tests/fixtures/http/reference-wrapper-component-defaults.json), [core contract](../core/reference_wrapper_component_defaults_test.go) (alias-owned factory, target untouched, canonical JSON, per-role validation, excluded roles and Pointers, annotation-less alias unchanged), and the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferenceWrapperComponentDefaultsContracts.kt).
- `bash scripts/test.sh feature reference-wrapper-component-defaults` runs fresh Data/Plain × validation-off/on projects and four `Json { encodeDefaults, explicitNulls }` configurations, asserting the factory output and that no field inherits it.
- Chain coverage includes [core admission/non-inheritance](../core/chained_wrapper_defaults_test.go), [original Pointers/exclusions/cached depth](../core/chained_wrapper_defaults_guards_test.go), [source identity and hop bounds](../internal/schema/chained_wrapper_defaults_test.go), and [native parity, readonly checks and failure preservation](../internal/cli/reference_wrapper_component_defaults_process_test.go). The wrapper feature route covers chains ending in objects, nested List/Map, enum, date, bytes and raw/nullable targets under four Json configurations, including fresh values and DataClass copy.
- A component without the annotation emits byte-identical output; the factory is purely additive.
- Raw targets have [core admission/guards/budgets](../core/raw_json_wrapper_defaults_test.go), [original source/target checks](../internal/schema/raw_json_wrapper_defaults_test.go), [native parity/check/preservation](../internal/cli/raw_json_component_defaults_process_test.go), and [runtime consumers](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawJSONComponentDefaultsContracts.kt). `bash scripts/test.sh feature raw-json-component-defaults` runs fresh Data/Plain × validation-off/on projects under four Json settings, including nullable/narrowed codecs, missing/null/supplied/required fields, fresh raw values and DataClass copy.
