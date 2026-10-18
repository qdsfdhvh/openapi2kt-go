# Bytes field defaults

A direct ordinary DTO field whose schema is `type: string` with `format: byte` or `format: binary` may carry a `default`. The default uses the **same signed-integer array wire the existing `ByteArray` contract already produces**, so a defaulted value round-trips through `Json` unchanged.

```json
{"sortKey": {"type": "string", "format": "byte", "default": [-128, -1, 0, 127]}}
```

```kotlin
@SerialName("sortKey")
public val sortKey: kotlin.ByteArray? = byteArrayOf(-128, -1, 0, 127),
```

## Why not Base64

OpenAPI describes `format: byte` as Base64 text, but this generator does not decode Base64: `ByteArray` is serialized by `ByteArraySerializer`, whose JSON form is an integer array. A Base64 string default could not round-trip, so it is rejected at the annotation instead of producing a value the wire cannot carry. **`"AAEC"` is `E_SCHEMA_INVALID`.** Introducing a Base64 codec is a different contract and is not part of this increment.

## Accepted values

| Value | Result |
|---|---|
| `[]` | `byteArrayOf()` |
| `[-128]` … `[127]` | `byteArrayOf(n, …)` |
| `[-128, -1, 0, 127]` | `byteArrayOf(-128, -1, 0, 127)` |
| `null` on a nullable field | `null` |
| `null` on a non-nullable field | `E_SCHEMA_INVALID` |

An element outside `[-128, 127]`, a fractional number, a `null` or string element, a Base64 string, or a non-array value is rejected with `E_SCHEMA_INVALID` **at that element's own pointer** (`…/default/3`), not at the whole annotation. The diagnostic states the accepted grammar instead of the generic scalar message.

`type: ["string", "null"]` makes the field nullable, so `default: null` is meaningful there and is validated like the `nullable: true` spelling.

## What does not change

- Signatures, `required`/Patch classification, serializers, and the wire form of a field **without** a default are untouched. The change is additive.
- A `required` bytes field still emits **no initializer**; the default is validated but never silently substituted. This matches every other required field.
- An optional bytes field keeps its existing optional-null type, so the initializer is `byteArrayOf(…)` on a `kotlin.ByteArray?`.
- Each construction evaluates the default expression again, so instances own **distinct** arrays; the generated default never shares one mutable instance.
- Bytes bounds (`minLength`/`maxLength`/`pattern`) are not applied to bytes: those constraints address text, and the existing `ByteArray` contract does not read them.

## `encodeDefaults` behavior

`encodeDefaults = false` **does not suppress a non-null bytes default.** The serializer compares against a freshly evaluated `byteArrayOf(…)`, and `ByteArray` equality is referential, so the comparison never matches and the value is always written. This differs from structural defaults such as `List`, which `encodeDefaults = false` does omit.

A `null` bytes default is comparable, so it keeps the ordinary optional-null policy: present only when `encodeDefaults` and `explicitNulls` are both enabled. Both behaviors are asserted by the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BytesFieldDefaultsContracts.kt).

## Scope

Admitted: a direct ordinary DTO field. Not admitted, each keeping its own diagnostic at its own pointer:

- a **component root** bytes default — that role is a separate contract on the same grammar, see [Bytes component defaults](bytes-component-defaults.md);
- bytes leaves inside collection, object-default, or nested-default trees;
- `$ref` wrappers, aliases, `allOf`, unions, `const`, caller-native types, value classes;
- Base64 text and any non-integer-element array.

## Verification

- [Fixture](../tests/fixtures/http/bytes-field-defaults.json), [core mappings](../core/bytes_field_defaults_test.go) (wire identity, per-element rejection, ordering, component factory), [native process](../internal/cli/bytes_field_defaults_process_test.go) (read-only `--check`, per-element pointers, preserved outputs), and the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BytesFieldDefaultsContracts.kt).
- `bash scripts/test.sh feature bytes-field-defaults` runs fresh Data/Plain × validation-off/on projects and four `Json { encodeDefaults, explicitNulls }` configurations.
- The [existing bytes reference chain contract](bytes-reference-chains.md) keeps its raw-Ktor-root and null-ASCII behavior unchanged; no Base64 or custom byte codec is introduced anywhere.
