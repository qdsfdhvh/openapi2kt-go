# Named true JSON component roots

The initial **F08/F04** named-root profile supports OpenAPI 3.1 literal `true` in `components.schemas`, ordinary references to those roots, and pure named `$ref` alias chains. The initial non-null profile and the separate [nullable-alias profile](#nullable-root-aliases) share canonical JSON backing. [Direct true fields/items](boolean-values.md) have a separate profile.

For example:

```json
{
  "openapi": "3.1.0",
  "components": {
    "schemas": {
      "AnyJson": true,
      "Alias": {"$ref": "#/components/schemas/AnyJson"}
    }
  }
}
```

With a `Dto` suffix, the canonical root exports `typealias AnyJsonDto = kotlinx.serialization.json.JsonElement`; `AliasDto` targets that canonical alias. It is not a DTO, value class, constructor or independently generated serializer. Assign JSON values and use backing/reified codecs, such as `json.decodeFromString<AnyJsonDto>(text)`; do not assume `AnyJsonDto()` or `AnyJsonDto.serializer()` APIs exist. Alias and canonical reified codecs have the same descriptor and JSON-kind behavior.

## Source, null and presence

The generator retains the original literal node, catalog identity and Pointer. Each literal root emits one `W_BOOLEAN_SCHEMA_PROJECTION` at its original component Pointer, regardless of reference count. Pure aliases retain every original source hop even though their emitted spelling targets the canonical root. Filename collision, source-depth and constructor-slot guards remain unchanged; cyclic aliases fail without partial artifacts.

A non-null Kotlin JsonElement may carry **JsonNull**. This root accepts null, Boolean, number, String, array and object JSON values; a schema `true` does not create nullable Kotlin intent or implicit Patch. A required non-null field still rejects absence.

Consumer-level nullable overrides and ordinary optional fields retain the existing raw `{}` presence policy. Nullable codecs can collapse JsonNull to Kotlin null; optional fields have their existing default-null behavior. Inline request fields with explicit nullable intent retain existing Patch.Absent/Null/Value handling. A required nullable constructor parameter still permits JSON wire omission of a Kotlin null when the caller's `explicitNulls=false`; required Kotlin arguments must not be confused with always-present JSON keys. No nullable-field wire policy changes in this increment.

Root references are usable as simple JSON request/response bodies and in typed DTO/list/map/SSE payloads. Literal true JSON schemas used directly as operation body/success-response roots have a separate [HTTP contract](http-client.md#direct-true-json-roots). Boolean SSE-envelope roots are not added by either profile. Existing Ktor JSON/SSE plugin and client-ownership rules apply.

## Nullable root aliases

Pure root `$ref` aliases to a named true component may carry `nullable:true` or `nullable:false`. Absent intent inherits from the referenced alias; the nearest explicit override wins. True aliases export canonical `Root?`, false aliases export the real non-null canonical `Root`, including after a nullable intermediate. There is no extra classifier, constructor, companion or nominal validation barrier. The grammar remains `$ref`/`nullable`, not a type-array reference extension.

**False does not reject JSON null:** non-null JsonElement still carries JsonNull. Reified nullable alias codecs decode JSON null as Kotlin null, whereas false-narrowed canonical codecs preserve JsonNull. Canonical roots themselves remain non-null. Shared nullable fields retain default-null; inline request required-nullable arguments have no default, optional inherited-nullable fields use Patch, and consumer overrides follow existing presence policy. Patch.Value(JsonNull) and Patch.Null have the same JSON null wire value and decode through the nullable Patch policy as Patch.Null. Required-nullable Kotlin arguments can still be omitted from JSON with `explicitNulls=false`.

Nullable named JSON request/response bodies retain the existing JSON codec policy; false-narrowed bodies can transport JsonNull. Raw String/ByteArray HTTP projection and nullable Int handling are unchanged. Source warnings remain once per literal root, not per alias. This profile does not relax historical validation-opt-in admission for unmarked/false-only builtin primitive aliases.

Previously rejected root-alias nullable keys now generate aliases. Existing accepted inputs retain output; changing meaningful property order is not a metadata-only change. Malformed nullable values, unknown semantic keys, missing references and cycles retain structured diagnostics. General nullable shape borrowing, internal Boolean targets and Boolean composition are not added.

[Nullable fixture](../tests/fixtures/http/boolean-root-nullable.json), [core alias contracts](../core/boolean_root_nullable_test.go), [source/limits guards](../core/boolean_root_nullable_guards_test.go), [catalog/checked-edge tests](../core/boolean_root_nullable_source_test.go), [native lifecycle](../internal/cli/boolean_root_nullable_process_test.go) and [nullable consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanRootNullableContracts.kt) exercise this bounded profile. `bash scripts/test.sh feature boolean-root-nullable` generates fresh Data/Plain × validation settings and four Json configurations for codecs, DTO constructor/copy, list/map/Patch, controlled nullable/narrowed HTTP and fragmented still-open SSE closure/reuse/coldness. Independent depth48/consumer46, checked16 and slots240 boundaries are Go contract checks, not Cartesian or platform compilation guarantees.

## Limits and migration

- [Literal false named component roots and pure aliases](false-component-roots.md) have a separate uninhabited type/serializer contract. Other false source roles retain their refusal boundaries; no false schema is deleted or treated as an unconstrained JSON/object value.
- OpenAPI 3.0 Boolean roots remain rejected. Existing Boolean `additionalProperties` behavior is unchanged.
- Pure root aliases have the separate nullable intent profile above. Nullable keys do not turn a true root into a structural object definition or widen other root-binding grammars.
- Internal Boolean reference targets, Boolean union branches/hierarchy/allOf shape borrowing and general Boolean composition remain unsupported. A true root is not a structural object definition merely because its Kotlin representation can carry an object value.
- Previously rejected admitted roots now generate with a source warning. Warning denial/allowlisting uses the existing [warning policy](warning-policy.md); generation/check/validate failures preserve existing owned, unowned and manifest files.
- Source depth48, typed checked edges16, constructor/method slots240 and other budgets remain unchanged. Independent boundaries do not establish combined maximum-resource or platform support.

## Verification surfaces

[Whole fixture](../tests/fixtures/http/boolean-roots.json), [core contracts](../core/boolean_root_test.go), [source/budget guards](../core/boolean_root_guards_test.go), [cycles/internal-target/catalog-order checks](../core/boolean_root_source_test.go), and [native lifecycle/failure preservation](../internal/cli/boolean_root_process_test.go) exercise the admitted profile and exclusions. Original corpus inputs, inventory and reviewed snapshots are not rewritten.

[BooleanRootContracts.kt](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanRootContracts.kt) covers reified canonical/alias codecs, all JSON kinds, JsonNull, DTO construction, list/map/Patch, controlled JSON HTTP and fragmented still-open SSE close/failure/reuse/coldness. `bash scripts/test.sh feature boolean-root` generates fresh Data/Plain × validation off/on projects, each with four Json configurations; Data projects also check `copy`. The full HTTP smoke invokes the consumer.

This profile does not complete F08/F04, general Boolean-schema validation, optimizer combinations, minimum-toolchain, Android/KMP, all-platform or first-version delivery.

## Literal false component roots

[Literal false roots](false-component-roots.md) have a separate uninhabited nominal class/serializer contract for OpenAPI 3.1 named components and pure aliases. Nullable aliases follow existing nearest carrier intent; all non-null values reject. [Ordinary named reference fields](false-reference-fields.md) have a separate absence/null policy. This grants no inline literal-false value/container/HTTP/default/composition role, and true-root behavior remains as described above.

[Named collections with false-reference leaves](false-collection-references.md) extend F10/D06/E05 with finite original List/typed Map roots, actual nominal leaf codecs, empty-container and nullable-leaf semantics, existing referenced DTO/JSON HTTP consumers, immutable source/export/backing/cached-height authority and source48/work16384/checked16/JVM240 limits. Inline/literal-false/default/tuple/composed/internal/discarded-extra-key and direct false HTTP-root roles remain separate; broader plan/platform/final acceptance stays open.

[Named true/false tuple references](tuple-boolean-references.md) have a separate original-root/actual-codec contract, including independent outer defaults, nullable false elements and complete erased-prefix source-depth accounting. General Boolean graphs remain independent.
