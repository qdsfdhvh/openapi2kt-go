# Referenced String enum collection field defaults

[Null vocabulary members](referenced-string-enum-null-member-defaults.md) separately compose this role with already admitted explicitly nullable leaves, preserving actual narrowed typed views and independent container nullability.

Scalar references have a separate [String enum scalar own-default contract](referenced-string-enum-scalar-defaults.md), preserving actual lexical classifiers, strict annotation vocabulary and open supplied Strings.

An ordinary DTO property with a direct local `$ref` may declare its own `default` when the existing reference planner already accepts the path and its terminal is a finite List/typed `Map<String, T>` tree whose innermost leaves are plain String enum schemas: explicit `type: "string"` or typeless enum arrays of strings, matching the [direct inline](string-enum-collection-defaults.md) and [nested inline](nested-string-enum-collection-defaults.md) terminal grammar. The referenced target may be a named collection component, an internal chain or a root bridge, and inner layers may traverse named aliases and internal property edges exactly like the [builtin](referenced-collection-field-defaults.md) and [temporal](referenced-temporal-collection-field-defaults.md) extensions. A default never makes an otherwise rejected pure schema or reference graph admissible.

```json
{
  "Status": {"type": "string", "enum": ["on", "off"]},
  "Statuses": {"type": "array", "items": {"$ref": "#/components/schemas/Status"}},
  "Holder": {"properties": {"rows": {"$ref": "#/components/schemas/Statuses", "default": ["on"]}}}
}
```

The property retains the resolver's actual public alias type and linked backing. The leaf classifier stays the actual lexical enum declaration — a named component keeps its own sealed class, an inline collection leaf keeps its `...Inner` declaration — and no declaration is copied, renamed or injected by the annotation. Helpers bind to the final emitted property type after optimization, and each evaluation decodes a fresh tree through its actual reified serializer, so defaults materialize real enum entries while supplied unknown values keep the open `Unrecognized` runtime.

## Annotation and presence

Only the consuming field's own default is accepted. A default on a referenced target, any intermediate reference, a container child or an enum leaf remains rejected, including when the consumer has its own default. No target value is inherited. Component/root defaults and `allOf` reference-wrapper defaults remain separate roles. Nullable intent, required presence, `Patch.Absent`/Null/Value wire states and `explicitNulls = false` missing-required-null inference follow the builtin extension's contract unchanged; a default never creates an update for a missing request key.

The role is available in ordinary shared/nested DTOs and ordinary inline request/response/SSE payload DTOs. Native numeric/Boolean enum leaves use the separate [referenced native contract](referenced-native-enum-collection-defaults.md). Union/hierarchy fields, formatted or bounded enum leaves, and wider default roles remain outside this profile. Nested enum trees through referenced paths are admitted by the same walk; the direct nested role keeps its own independent scope.

## Source, checks and identity

The original schema nodes, catalog membership, TypeRef addresses and Pointers authorize the type. Default members are strict against the original enum vocabulary at generation time with original `/default/<index>/<escaped-key>` diagnostics; runtime stays open. Map keys sort for canonical output identity, array order stays significant, and equivalent lexemes share identity. The source `$ref` is not replaced by a flattened JSON schema identity. Required/Patch annotations that emit no initializer add no runtime default identity. Persisted lock formats and no-default fingerprints are unchanged.

With `--validation`, default values must satisfy each retained container count. Referenced collections with original plain String enum leaves additionally retain [serialName uniqueness](string-enum-unique.md), including their already admitted named/internal enum references and aliases. String bounds and patterns on enum leaves stay rejected with and without validation; typed construction/copy/automatic serde/Patch.Value/HTTP/SSE keep the original count checks and open enum vocabulary. The existing input8MiB, source48, checked-collection16, source-work16,384, serialization-slot240, enum240 and 32 KiB default budgets remain unchanged.

The [synthetic fixture](../tests/fixtures/http/referenced-string-enum-collection-defaults.json), [core API](../core/referenced_string_enum_collection_defaults_test.go), [guards](../core/referenced_string_enum_collection_defaults_guards_test.go), [identity tests](../core/referenced_string_enum_collection_defaults_identity_test.go), [source authority](../internal/schema/referenced_string_enum_collection_defaults_test.go), [native preservation](../internal/cli/referenced_string_enum_collection_defaults_process_test.go) and [native CAS/readonly parity](../internal/cli/referenced_string_enum_collection_defaults_unix_process_test.go) define this increment. Formerly rejected named-collection enum-leaf inputs are retained as explicit graduations in the [builtin guards](referenced-collection-field-defaults.md) and [nested scope](nested-string-enum-collection-defaults.md) tests.

`bash scripts/test.sh feature referenced-string-enum-collection-defaults` generates fresh Data/Plain × validation off/on projects covering named/typeless/inline leaves, per-layer nullable and null members, alias chains, internal bridges, required/Patch fields, four Json settings, optimized aliases/dedup/merge identity, typed HTTP and fragmented live SSE failure closure and reuse, plus independent 240-field, source48, checked16 and 32 KiB compilation probes. The suite is also part of `models`/`http` registry coverage. `OPENAPI_KT_PREPARE_ONLY=1` prepares and checks native output without running Gradle; it is not Kotlin evidence.

**Migration:** previously rejected defaults on this bounded direct-reference field role now generate. Accepted pure-schema Kotlin APIs/wire and historical artifacts retain their contracts. This closes only the referenced String enum collection **field** subitem of D03 (M03/M04/M07/M08, existing M12/V05 and A03/A05/A08 consumers); native leaves, general D03 and the release gates remain open.
