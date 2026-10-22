# Referenced native enum collection field defaults

Scalar references use the separate [referenced native enum scalar field default contract](referenced-native-enum-scalar-defaults.md), with scalar literal initializers and existing resolver admission.

An ordinary DTO property with a direct local `$ref` may declare its own `default` when the existing reference planner already accepts the path and its terminal is a finite List/typed `Map<String, T>` tree whose innermost leaves are native numeric or Boolean enum schemas: `type: integer` (int32/int64), `type: number` (float/double) or `type: boolean` with an `enum` vocabulary. The walk follows named collection components, internal chains and root bridges exactly like the [String enum extension](referenced-string-enum-collection-defaults.md), and native leaves keep their scalar wire type (`typealias IntsDto = kotlin.collections.List<kotlin.Int>`). A default never makes an otherwise rejected pure schema or reference graph admissible.

```json
{
  "Code": {"type": "integer", "format": "int32", "enum": [1, 2, 3]},
  "Codes": {"type": "array", "items": {"$ref": "#/components/schemas/Code"}},
  "Holder": {"properties": {"rows": {"$ref": "#/components/schemas/Codes", "default": [2]}}}
}
```

The property retains the resolver's actual scalar alias type and linked backing; no wrapper, classifier or declaration is created by the annotation. Helpers bind to the final emitted property type after optimization, and each evaluation decodes a fresh tree through its actual reified serializer. The terminal joins the existing native collection-leaf membership channel: exact numeric membership is checked in both validation modes before Float/Double rounding, and the original `W_ENUM_NATIVE_APPROXIMATION` warning is unchanged.

## Annotation and presence

Only the consuming field's own default is accepted. A default on a referenced target, any intermediate reference, a container child or an enum leaf remains rejected. Nullable intent, required presence, `Patch.Absent`/Null/Value wire states and `explicitNulls = false` missing-required-null inference follow the referenced collection extensions unchanged. Null members require nullable leaf layers in the actual backing; a null container default requires explicit container nullability.

Union/hierarchy fields, String enum leaves (separate contract), formatted temporal leaves (separate contract), const/native-marker/value-class leaves and wider default roles remain outside this profile. The mapped scalar and the actual backing must agree after the reference planner's final authority checks; floating `multipleOf` never joins the walk.

## Source, checks and identity

The original schema nodes, catalog membership, TypeRef addresses and Pointers authorize the type. Default members are strict against the original enum vocabulary at generation time with original `/default/<index>/<escaped-key>` diagnostics, exact numeric comparison before scalar mapping and canonical Map key sorting. Supplied values use native scalar codecs. Without `--validation`, typed consumers do not enforce enum membership; with it, typed DTO constructors and decoding enforce the mapped enum vocabulary as well as retained container counts and numeric bounds (including the existing empty mapped-intersection rejection). Standalone scalar typealias codecs retain scalar decoding in both modes. Without validation, constrained sources stay rejected. Int/Long/Boolean enum lists retain [native uniqueness](native-enum-unique.md) through existing eligible references and default roles. Float/Double enum lists additionally follow [finite floating uniqueness](floating-unique.md); container-element `uniqueItems:true` remains rejected, and `uniqueItems:false` stays a no-op. The existing input8MiB, source48, checked-collection16, source-work16,384, serialization-slot240, enum240 and 32 KiB default budgets remain unchanged, and persisted lock formats and no-default fingerprints are unchanged.

The [synthetic fixture](../tests/fixtures/http/referenced-native-enum-collection-defaults.json), [core API](../core/referenced_native_enum_collection_defaults_test.go), [guards](../core/referenced_native_enum_collection_defaults_guards_test.go), [identity tests](../core/referenced_native_enum_collection_defaults_identity_test.go), [source authority](../internal/schema/referenced_native_enum_collection_defaults_test.go), [native preservation](../internal/cli/referenced_native_enum_collection_defaults_process_test.go) and [native CAS/readonly parity](../internal/cli/referenced_native_enum_collection_defaults_unix_process_test.go) define this increment. Formerly rejected native-leaf inputs are retained as explicit graduations in the [builtin guards](referenced-collection-field-defaults.md), [String enum guards](referenced-string-enum-collection-defaults.md) and [nested native scope](nested-native-enum-collection-defaults.md) tests.

`bash scripts/test.sh feature referenced-native-enum-collection-defaults` generates fresh Data/Plain × validation off/on projects covering int32/int64/float/double/Boolean leaves, typed Map values, nullable root containers and scalar leaves, alias chains, internal bridges, required/Patch fields, four Json settings, optimized aliases/dedup/merge identity, typed HTTP and fragmented live SSE failure closure and reuse, plus independent 240-field, source48, checked16 and 32 KiB compilation probes. The suite is also part of the `http` registry coverage. `OPENAPI_KT_PREPARE_ONLY=1` prepares and checks native output without running Gradle; it is not Kotlin evidence.

**Migration:** previously rejected defaults on this bounded direct-reference field role now generate. Accepted pure-schema Kotlin APIs/wire and historical artifacts retain their contracts. This closes only the referenced native enum collection **field** subitem of D03 (M03/M04/M07/M08, existing M12/V05 and A03/A05/A08 consumers); general D03 and the release gates remain open.
