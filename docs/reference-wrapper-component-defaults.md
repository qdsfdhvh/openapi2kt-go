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
| ordinary object/disjoint ordinary `allOf`, finite inline List/typed Map tree (including supported nullable layers) | factory on the wrapper alias |
| String enum, native enum under its existing admission rules, standard temporal, bytes | factory on the wrapper alias |
| admitted raw `JsonElement` / `JsonObject` carrier | factory on the wrapper alias, using the target's raw-value and null grammar |
| builtin scalar (`string`/`integer`/`number`/`boolean`, including int32/int64/float/double) | factory only with validation enabled — a `$ref` to a plain scalar is itself a validation-opt-in alias |
| `string` / `format: uuid` | Uuid/Uuid? factory with validation enabled; [UUID migration](nullable-uuid.md#migration) applies, and nearest nullable intent remains |
| finite pure component alias chain (`S → M → T`) | factory on S when every intermediate node meets the rules below |
| explicit outer or intermediate `nullable: true` / `false` | factory follows the nearest explicit alias intent under the profile below |
| target or intermediate alias with its own admitted `default` | separate factory on each original owner; no inheritance |
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

In this example only S carries a default and receives a factory. T, B, M and referencing fields retain their original output; the factory decodes through `serializer<SDto>()`. Each intermediate component may contain `$ref`, optional boolean `nullable`, its own admitted `default`, and otherwise accepted metadata. Intermediate and terminal defaults are validated independently at their original owners, as described below. The nearest explicit alias nullable intent follows the profile below. Existing target grammar and validation gates remain; plain scalar factories still require validation, and a chain without any explicit alias nullability retains its separate plain-scalar null-default rejection.

Every hop must name `#/components/schemas/<Name>`. Missing targets, cycles, malformed references, internal/property references and chains over 48 hops are rejected. The ordinary resolver still counts the combined alias, target and consumer depth against its source-depth limit, including cached targets: 48 hops is a traversal cap, not a promise that a chain of that length is usable in every context. This adds no general reference graph or internal-reference support.

Target and intermediate source nodes are never rewritten. Value errors, including escaped dictionary keys, retain the outer annotation's Pointer. Original/canonical 32 KiB JSON and 16,384-node default budgets remain shared with direct wrappers. Factories return fresh structured values; runtime codecs and required/missing/null/supplied behavior remain the target's own.

This additive D03/M03/M04/M07 profile accepts previously rejected annotations. Previously accepted inputs and persisted lock formats require no migration.

## Explicitly nullable outer wrappers

An outer `nullable: true` wrapper may own either a null or non-null default through the same finite component alias chains. This is also the nearest possible override under the rules below.

## Nullable intent across component aliases

Outer and intermediate aliases may declare either boolean `nullable` value. Starting at the factory-owning wrapper, the first explicit intent wins; if every alias omits it, the terminal retains its previous grammar and null permission. This matches the existing alias resolver rather than changing generated types.

```json
{"T": {"type": ["array", "null"], "items": {"type": ["string", "null"]}},
 "Wide": {"$ref": "#/components/schemas/T", "nullable": true},
 "Narrow": {"$ref": "#/components/schemas/Wide", "nullable": false},
 "S": {"$ref": "#/components/schemas/Narrow", "default": [null, "given"]},
 "R": {"$ref": "#/components/schemas/Narrow", "nullable": true, "default": null}}
```

S receives a non-null `List<String?>` factory and rejects a null annotation. R receives a nullable list factory returning Kotlin null. Only the temporary validation root is narrowed or widened; for OpenAPI 3.1 narrowing removes the root type-array null member as well as applying `nullable: false`. Item/value nullability, original source nodes, terminal/intermediate declarations, warnings, consumer signatures and bytes remain unchanged.

Eligible terminals remain the existing ordinary object/disjoint ordinary `allOf`, finite inline List/typed Map, plain String enum, standard temporal, signed bytes and raw JSON roles. Plain scalar factories retain validation opt-in; native enum aliases retain their existing validation and format gates. Null still validates the complete source schema, vocabulary, child grammar and source/value budgets. Non-null annotations still satisfy required fields, kinds, original enum vocabulary, bounds and counts. A null collection does not authorize null elements.

Factories return the actual alias through its serializer. Referencing fields do not inherit the annotation; shared nullable fields retain their existing default/required behavior, while required non-null aliases remain required. A narrowed raw `JsonElement` factory rejects an annotation of null, but its non-null runtime codec still represents JSON null as `JsonNull`; a nullable raw alias decodes it to Kotlin null.

Cycles, internal/property references, unsupported siblings and excluded terminal roles keep their rejections. Independent intermediate/terminal defaults follow their own roles below. An outer override cannot hide malformed intermediate source. The source-depth cap remains 48 combined alias/terminal/consumer steps, including cached heights; original/canonical 32 KiB and 16,384-node value budgets remain. Previously valid input, Kotlin call sites, wire behavior and persisted lock formats require no migration. Bounded plain-scalar wrapper factories, including chains without explicit alias nullable intent, enforce retained primitive bounds on non-null annotations: values previously accepted because only the source was checked now fail with `E_SCHEMA_INVALID` at the wrapper's original `/default`. Correct those invalid annotations; valid defaults keep their bytes. This closes only this D03/M03/M04/M07 nullable-chain profile, not general reference graphs or defaults.

## Raw JSON carrier targets

Pure `$ref` wrappers over admitted [raw component carriers](raw-json-component-defaults.md) use the raw carrier validator. A `JsonElement` target accepts non-null object, array and scalar annotations; a `JsonObject` target requires an object. A null annotation requires effective nullability from the target or nearest explicit alias and decodes to Kotlin null through the wrapper's actual nullable alias serializer. A non-nullable raw target keeps `E_SCHEMA_UNSUPPORTED` at the wrapper's own `.../default`, even though the standalone non-null `JsonElement` codec can represent JSON null as `JsonNull`.

```json
{"Raw": {"type": "object"},
 "Open": {"type": ["object", "null"], "additionalProperties": true},
 "RawDefault": {"$ref": "#/components/schemas/Raw", "default": [1, null]},
 "NullOpenDefault": {"$ref": "#/components/schemas/Open", "default": null}}
```

The wrapper owns the factory, and the target keeps its original source and generated bytes. Raw object members retain `JsonNull` under every `explicitNulls` setting; the String `"null"` remains a String. Required fields keep required constructor arguments, optional fields do not inherit the wrapper annotation, and structured factories decode fresh values on each call. Source validation, original annotation Pointers, target warnings and the existing 16,384-node / 32 KiB default budgets remain unchanged. Independent target/intermediate defaults follow their own roles below; excluded target shapes retain their restrictions, and explicit nullable booleans follow the chain rules above. Finite pure component alias chains follow the rules above.

This adds factories for previously rejected raw wrapper annotations (D03/M03/M04/M07). Previously accepted input, serializers and lock formats require no migration.

## Independent defaults on target and intermediate components

Every already-admitted component on a finite named-root alias chain may carry its own `default`. Each annotation emits only its owner's opt-in factory and is validated against that owner's actual nullable view:

```json
{"T": {"type": "object", "required": ["z"], "properties": {"z": {"type": "string"}}, "default": {"z": "target"}},
 "M": {"$ref": "#/components/schemas/T", "nullable": true, "default": {"z": "middle"}},
 "S": {"$ref": "#/components/schemas/M", "nullable": false, "default": {"z": "outer"}},
 "P": {"$ref": "#/components/schemas/S"}}
```

T's companion factory creates `z = "target"`; M's alias factory creates `z = "middle"`; S's alias factory creates `z = "outer"`. P has no factory. Fields referencing any component keep their existing required/optional/null/Patch behavior and do not adopt an annotation automatically. A nullable target may own a null factory while a narrowed outer alias owns a non-null factory, and the reverse widening may own a null factory over a non-null target.

Traversal reads original sources and ignores another owner's value only when resolving shape. Each original component still passes its own source/annotation grammar and original/canonical 32 KiB / 16,384-node budgets. The throwaway validation view replaces only its root default with the current owner's annotation; original target/intermediate nodes, declarations, warnings and factories are unchanged. Bad defaults report at their original `/components/schemas/<Owner>/default/...`, including when an outer annotation is valid or null. Failed generation returns no artifacts; the CLI preserves prior output. No annotation may authorize a target role that was previously unsupported, such as a value-class/union root default or unsupported element/child defaults. Constrained plain-scalar roots follow their [validation-only factory profile](component-default-factories.md#constrained-plain-scalar-roots). [Numeric builtin formats](component-default-factories.md#numeric-builtin-formats) retain mapped range/finite checks, even without bounds or explicit alias nullable intent; null uses effective target/nearest alias nullability. Numeric aliases keep the same validation opt-in. [Native UUID factories](component-default-factories.md#uuid-roots) also admit defaults on finite pure named aliases with validation enabled, retaining canonical-text constraints, original annotation Pointers and nearest explicit nullable intent. Intrinsic nullable UUID roots and own-nullable aliases use transparent Uuid? codecs; null defaults on narrowed non-null aliases remain invalid.

Existing alias depth/cycle checks and nearest nullable precedence remain. This additive D03/M03/M04/M07 profile requires no migration for previously accepted input and changes no persisted lock format. It does not add default inheritance, internal references or general default roles.

## Runtime stays per-role

The wrapper does not change any runtime policy. A String enum keeps its open vocabulary and surfaces an unknown wire member as `Unrecognized`; a DTO keeps its required-member check; a bytes element outside `[-128, 127]` is a generator-time check because `ByteArraySerializer` narrows with `toByte()` instead of rejecting.

## Nested classifiers

A typealias cannot reach nested classifiers, so `WrappedDto.Member` does not compile where `TargetDto.Member` does. Access members through the canonical owner, as documented in the [DTO reference contract](internal-dto-references.md).

## Verification

- [Fixture](../tests/fixtures/http/reference-wrapper-component-defaults.json), [core contract](../core/reference_wrapper_component_defaults_test.go) (alias-owned factory, target untouched, canonical JSON, per-role validation, excluded roles and Pointers, annotation-less alias unchanged), and the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferenceWrapperComponentDefaultsContracts.kt).
- `bash scripts/test.sh feature reference-wrapper-component-defaults` runs fresh Data/Plain × validation-off/on projects and four `Json { encodeDefaults, explicitNulls }` configurations, asserting the factory output and that no field inherits it.
- Chain coverage includes [core admission/non-inheritance](../core/chained_wrapper_defaults_test.go), [original Pointers/exclusions/cached depth](../core/chained_wrapper_defaults_guards_test.go), [source identity and hop bounds](../internal/schema/chained_wrapper_defaults_test.go), and [native parity, readonly checks and failure preservation](../internal/cli/reference_wrapper_component_defaults_process_test.go). The wrapper feature route covers chains ending in objects, nested List/Map, enum, date, bytes and raw/nullable targets under four Json configurations, including fresh values and DataClass copy.
- Explicit outer nullability has [core acceptance/guards](../core/nullable_wrapper_defaults_test.go) and [original source/alias ownership](../internal/schema/nullable_wrapper_defaults_test.go) coverage; the existing native parity and four-profile JVM wrapper route include null/non-null factories and non-inheritance. The validation-on profiles additionally compile [scalar/native-enum wrapper factories](../tests/fixtures/http/nullable-scalar-wrapper-defaults.json) and run their [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableScalarWrapperDefaultsContracts.kt). Both wrapper consumers are also called by the full `http` registry.
- The [nullable-chain matrix and constraints/depth guards](../core/wrapper_chain_nullability_defaults_test.go), [source identity/intent traversal](../internal/schema/chained_wrapper_defaults_test.go), [native parity/check/failure preservation](../internal/cli/wrapper_chain_nullability_defaults_process_test.go), [synthetic fixture](../tests/fixtures/http/wrapper-chain-nullability-defaults.json) and [JVM consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/WrapperChainNullabilityDefaultsContracts.kt) cover nearest true/false, OpenAPI 3.0/3.1 roots, child nullability, fresh values, required/absence/null/supplied states and raw JsonNull. The existing four-profile wrapper feature and full HTTP registry execute this consumer.
- Independent defaults have [core ownership/single-owner controls/Pointers/budgets](../core/wrapper_owned_defaults_test.go), [original source/IR ownership](../internal/schema/wrapper_owned_defaults_test.go), [native parity/read-only/failure preservation](../internal/cli/wrapper_owned_defaults_process_test.go), [fixture](../tests/fixtures/http/wrapper-owned-defaults.json) and [JVM consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/WrapperOwnedDefaultsContracts.kt). The existing four-profile wrapper route and full HTTP registry execute target/middle/outer/null/fresh/absence/supplied/required assertions.
- A component without the annotation emits byte-identical output; the factory is purely additive.
- Raw targets have [core admission/guards/budgets](../core/raw_json_wrapper_defaults_test.go), [original source/target checks](../internal/schema/raw_json_wrapper_defaults_test.go), [native parity/check/preservation](../internal/cli/raw_json_component_defaults_process_test.go), and [runtime consumers](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawJSONComponentDefaultsContracts.kt). `bash scripts/test.sh feature raw-json-component-defaults` runs fresh Data/Plain × validation-off/on projects under four Json settings, including nullable/narrowed codecs, missing/null/supplied/required fields, fresh raw values and DataClass copy.
