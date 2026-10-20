# Kotlin UUID scalars and nullable components

`type: string`, `format: uuid` maps to `kotlin.uuid.Uuid`; nullable use sites map to `Uuid?`. This applies to direct fields, transparent components and aliases, List/typed Map leaves, scalar consts, admitted defaults, HTTP parameters/bodies/responses and SSE payloads. `x-kotlin-value-class: true` retains its nominal wrapper with a Uuid `rawValue`.

The builtin Kotlin serializer uses a JSON string. Parsing accepts 32 ASCII hexadecimal digits, either compact or separated by the four standard UUID hyphens; uppercase and lowercase are accepted. Encoding and `toString()` produce lowercase, hyphenated text. Empty/arbitrary strings, braces, URN prefixes, whitespace, misplaced hyphens and non-hex digits are rejected. No UUID version or variant restriction is added. Null remains Kotlin null only where the actual type is nullable.

```json
{"Maybe": {"type": ["string", "null"], "format": "uuid", "default": null},
 "Narrow": {"$ref": "#/components/schemas/Maybe", "nullable": false},
 "Again": {"$ref": "#/components/schemas/Narrow", "nullable": true}}
```

`Maybe` and `Again` export `Uuid?`; `Narrow` exports `Uuid`. Nearest explicit alias intent wins, followed by the consumer's own override. Supported named/internal scalar chains and borrowed List/typed Map trees retain original source nodes, declaration authority and source-depth accounting. UUID scalars cannot be borrowed as object/composition/hierarchy or SSE-envelope shapes.

Unconstrained intrinsic roots and own-nullable bindings work with validation off or on. Otherwise unmarked/false-only aliases to non-null scalar components retain their existing validation opt-in. String length and portable pattern constraints also require validation. These checks apply to the canonical UUID text, including when a default or JSON input uses uppercase or compact text. Typed DTO constructors/copy/decode, collections/Patch, HTTP pre/postflight and SSE retain eligible checks. Bare alias assignment and standalone builtin serialization remain outside this constraint barrier. Existing `uniqueItems` exclusions for UUID schemas remain.

## Defaults and presence

Non-null scalar `default` and `const` annotations must parse as UUIDs. Scalar initializers use a typed `Uuid.parse(...)` helper; List/typed Map and component factories decode with their actual serializers. Invalid annotations fail at their original value Pointer before output changes. Nullable roots may own null or UUID factories under the [component contract](component-default-factories.md#uuid-roots). Named alias factories retain their validation opt-in. Target/intermediate/outer annotations remain independent; fields and unannotated aliases never inherit a component default.

Shared nullable fields keep their existing default-null policy, including required shared fields. Inline required-nullable request fields have no initializer; optional nullable request fields use `Patch<Uuid>`. Absent omits the key, Null writes JSON null, and Value writes the UUID string under all `encodeDefaults` / `explicitNulls` combinations. Required nullable-key decoding keeps the existing `explicitNulls` policy; required narrowed fields reject JSON null. Scalar consts preserve the existing approximation: they always encode, but decoding another valid UUID does not enforce const membership.

## HTTP and SSE

UUID roots delegate to JSON serialization: a request UUID is quoted JSON text, a response must be a JSON UUID string, and a nullable root response containing JSON `null` returns Kotlin null. Invalid UUID text and non-null root nulls fail decoding. This differs from bare String roots, which Ktor handles as raw text.

Path/query/header UUID values use canonical `toString()` text without JSON quotes; collection parameter styles retain their existing rules. SSE payloads use actual JSON UUID/null semantics. Invalid kind, UUID text, nullability or retained bounds fail, cancel the active channel and leave the client usable. Flows remain cold and reusable.

String enum vocabularies with `format: uuid` retain their open `serialName` API in model/parameter contexts; mixed primitive unions retain their established kind carriers. A simple HTTP string enum uses the scalar mapper and therefore Uuid. An explicit `x-native-type` still takes precedence; the caller owns that type and serializer. No general graph, enum-factory or native default profile is added.

## Migration

This intentionally changes previously accepted generated Kotlin and wire behavior (F04/F07/M01/M03/M04/M07/M12/D03):

- Replace UUID String arguments with `Uuid.parse(text)` and use `.toString()` when handing values to String APIs. Nullable values and nominal wrapper `rawValue` types follow the same change.
- Supply valid UUID defaults/consts and server data. UUID spellings normalize; do not depend on input case or compact formatting being preserved. Length/pattern constraints observe canonical text.
- Send/receive JSON strings for root UUID HTTP bodies instead of raw UUID text. Nullable root JSON null now becomes Kotlin null.
- For opaque identifiers, remove `format: uuid`. A supported explicit `x-native-type: kotlin.String` retains caller-native String mapping and raw String root codecs; native defaults and other native-role restrictions still apply.
- Regenerate owned output normally. `--check` reports drift (exit 3); no manifest/name-lock reset or format bump is required. Historical P1 goldens stay unchanged, with explicit current UUID expectations and native migration checks for the two affected corpus cases.

The executed JVM combination is Kotlin **2.4.10**, kotlinx.serialization **1.11.0**, and the existing Ktor **3.5.2** dependencies. Uuid compiles without an experimental opt-in on this combination. Older dependency combinations, Android/KMP and a minimum-version matrix are not established by these checks. This breaking change requires a MINOR release before 1.0 under the [version policy](versioning.md); it is not included in already-published tags.

[Core mapping/grammar contracts](../core/native_uuid_test.go), [nullable/alias contracts](../core/nullable_uuid_test.go), [original-source guards](../internal/schema/builtin_reference_chains_test.go), [native parity/failure preservation](../internal/cli/nullable_uuid_process_test.go), [fixture](../tests/fixtures/http/nullable-uuid.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableUuidContracts.kt) cover this profile. `feature builtin-reference-chains` runs Data/Plain × validation, including Plain optimization options and four Json configurations. `feature reference-wrapper-component-defaults` covers independent factories, while the complete `http` registry exercises fresh output and value-class parameters. Original depth, checked-edge, slot and default-budget limits remain.
