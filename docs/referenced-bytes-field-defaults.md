# Referenced byte field own defaults

Ordinary DTO fields with a direct local `$ref` accept their own `default` when the existing type plan resolves to builtin `ByteArray`, or to a finite List/typed Map tree with byte leaves. The terminal source must be `type: string` with `format: byte` or `binary`. Existing accepted named/internal aliases, nearest nullable intent, source nodes and retained declarations remain authoritative; a coincident native/provider type name does not grant this role.

```json
{
  "openapi": "3.1.0",
  "components": {"schemas": {
    "Blob": {"type": "string", "format": "byte", "nullable": true},
    "Rows": {"type": "array", "items": {"$ref": "#/components/schemas/Blob"}},
    "Holder": {"properties": {
      "blob": {"$ref": "#/components/schemas/Blob", "default": [-128, 0, 127]},
      "rows": {"$ref": "#/components/schemas/Rows", "default": [[], null]}
    }}
  }}
}
```

`blob` retains its `ByteArray?` property type; `rows` retains its actual collection alias and element codec. Private typed JSON helpers supply the initializers, including scalar bytes, so large accepted annotations do not expand into unbounded `byteArrayOf` expressions. Helpers are shared within a file, follow relocated collection types through optimization, and produce fresh mutable arrays for every constructor or missing-key decode.

Object-default properties that follow existing finite byte/binary reference paths use the separate [bytes alias member contract](nested-object-defaults.md#bytes-scalar-alias-members).

## Values and presence

Defaults use the existing signed numeric-array JSON wire: each byte must be an integer token in `[-128,127]`. Base64 text, fractional/exponent tokens, `-0`, null byte elements and out-of-range integers fail with `E_SCHEMA_INVALID` at their original value Pointer. A Map key such as `a/~` retains JSON Pointer escaping; `$ref` inside default data is an ordinary key. Map keys are canonicalized without changing array order.

Missing keys receive the field's own default. Explicit values override it, and explicit null is retained where the public optional-null property allows it. Annotation null requires explicit effective schema nullability; optional presence alone does not grant it. Container and byte-leaf nullability remain independent.

Required fields carrying these annotations receive no constructor initializer and still require arguments. Request nullable fields retain `Patch.Absent`; null and supplied values retain `Patch.Null` and `Patch.Value`. All annotations are checked before required/Patch suppression or later allOf override, including their source schema and resource budgets.

Data and Plain styles retain array reference equality. Non-null byte defaults, including empty byte arrays, are emitted with `encodeDefaults=false`; Lists/Maps containing byte arrays follow their existing element equality. Empty Lists/Maps retain comparable-default omission, and null defaults follow `encodeDefaults` plus `explicitNulls`. A data-class `copy()` keeps the existing array references unless replaced. No mutation-time validation is added.

## Scope and limits

Only the consuming ordinary field's own annotation is admitted here. Target/intermediate defaults are neither inherited nor accepted through this role. Component and reference-wrapper factories, direct inline defaults and object-default members retain their [separate](bytes-component-defaults.md) [contracts](bytes-default-trees.md). Eligible byte reference children have a [separate independent annotation contract](scalar-reference-collection-child-defaults.md). Single-ref composition wrappers, caller-native/value-class/enum/const terminals and arbitrary new reference graphs receive no new permission.

Existing collection counts remain opt-in and apply at their own container. Byte `minLength`/`maxLength`/pattern and ByteArray `uniqueItems:true` remain unsupported. Schema checks run even for empty/null annotations. Actual alias assignment and independent builtin codecs retain their existing validation behavior.

Each entire annotation retains 16,384 JSON value nodes and 32 KiB of compact original JSON, with a separate 32 KiB canonical JSON check. Bytes arrays and their integer elements share that budget with every containing List/Map. Original input 8 MiB, source depth 48, checked collection depth 16, source work 16,384 and constructor-slot 240 limits remain unchanged. Independent limit probes do not establish simultaneous maximum width/depth acceptance.

Typed DTO HTTP/SSE payloads keep signed-array JSON through the caller's codec. Root byte request/response policies remain unchanged: raw bytes remain raw, and a raw response containing ASCII `null` remains four bytes.

## Migration and verification

Previously rejected own defaults in this bounded reference role now generate, and missing keys may decode to the declared arrays. Accepted schemas without annotations retain their Kotlin APIs, serializers and wire behavior. This extends only the bytes field/collection own-default subitem of D03 (M01/M02/M03/M04/M07 with existing M12/V05 and A03/A05/A08 consumers).

The [fixture](../tests/fixtures/http/referenced-bytes-field-defaults.json), [core tests](../core/referenced_bytes_field_defaults_test.go), [original source/backing checks](../internal/schema/referenced_bytes_field_defaults_test.go) and [native preservation/lock parity tests](../internal/cli/referenced_bytes_field_defaults_process_test.go) define the contract.

`bash scripts/test.sh feature referenced-bytes-field-defaults` generates fresh Data/Plain × validation-off/on projects. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedBytesFieldDefaultsContracts.kt) checks four Json settings, actual aliases/ABI/descriptors, independent arrays, missing/null/value, required/Patch, copy, optimized/locked models, controlled HTTP and fragmented live SSE failure closure/reuse. Independent source48/checked16/slots240 and exact value-node/32 KiB scalar compilation probes retain existing caps. The full `http` route generates/checks the fixture and calls the consumer; CI's `profiles` job also selects the four-profile feature route. `OPENAPI_KT_PREPARE_ONLY=1` prepares native projects without compiling/running Kotlin. Other-platform and release acceptance remain separate.
