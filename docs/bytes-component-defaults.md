# Bytes component defaults

A component schema whose root is `type: string` with `format: byte` or `format: binary` may carry its own `default`. Because a component has no call site, the annotation never becomes a constructor initializer: the generator emits an **opt-in factory** on the same signed-integer array grammar as [the field role](bytes-field-defaults.md).

```json
{"SortKey": {"type": "string", "format": "byte", "default": [-128, -1, 0, 127]}}
```
```kotlin
public typealias SortKeyDto = kotlin.ByteArray
public fun `SortKeyDto default`(): SortKeyDto = `openapi-kt component default … SortKeyDto`("[-128,-1,0,127]")
private fun `openapi-kt component default … SortKeyDto`(source: kotlin.String): SortKeyDto =
    kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<SortKeyDto>(), source)
```

A `ByteArray` alias cannot own members, so the factory is a top-level function, exactly like the plain-scalar, String-enum and temporal component roots. It decodes through the component's own serializer, so the produced value is a real `ByteArray` and not a re-encoded copy.

## A component default is not inherited

A field that references a defaulted component keeps its neutral presence — `= null` for an optional field — and does **not** pick up the factory. This is the same rule every other component default follows: adopting the default stays an explicit caller decision.

```kotlin
public val key: kotlin.ByteArray? = null,      // $ref to SortKey: no inheritance
public val inline: kotlin.ByteArray? = byteArrayOf(1, 2),  // inline field role
```

## Scope

Admitted: a `byte`/`binary` component **root**. Rejected at the original Pointer, each keeping its own role:

| Input | Result |
|---|---|
| `default: "AAEC"` (base64 text) | `E_SCHEMA_INVALID @ …/default` |
| `default: [128]` / `[-129]` | `E_SCHEMA_INVALID @ …/default/0` |
| bytes element inside a `List`/`Map` root | `E_SCHEMA_UNSUPPORTED @ …/default` |
| nullable bytes root with `default: null` | `E_SCHEMA_UNSUPPORTED @ …/default` |
| `$ref` wrapper, `allOf`, union, `const`, caller-native, value class | its own role |

The shared nested collection default walk deliberately excludes `ByteArray`, so that bytes stays out of the collection-element and object-default-leaf roles. The component root builds its node directly instead of widening that walk.

## Diagnostics point at the component root

Admitting the bytes root exposed a pre-existing defect: the temporal scalar root validated with an empty schema pointer, so a bad lexeme reported a bare `/default` instead of `/components/schemas/X/default`. Both roles now report the absolute component Pointer, and a regression test pins it.

## Verification

- [Fixture](../tests/fixtures/http/bytes-component-defaults.json), [core contract](../core/bytes_field_defaults_test.go) (`TestBytesComponentDefaults*`: factory shape, canonical JSON, no inheritance, role boundaries, per-element Pointers), and the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BytesComponentDefaultsContracts.kt).
- `bash scripts/test.sh feature bytes-component-defaults` runs fresh Data/Plain × validation-off/on projects and four `Json { encodeDefaults, explicitNulls }` configurations, asserting the factory wire, fresh instances, and that no field inherits the default.
- A schema without the annotation emits byte-identical output; the factory is purely additive.
