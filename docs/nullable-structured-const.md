# Nullable structured const values

Plain structured `const` declarations can expose `JsonObject?` or `JsonArray?` through effective `nullable: true` or a supported object/array-plus-null type. The JSON classifier remains non-null; the generated root is a transparent nullable typealias, rather than a DTO or new codec identity. Unmarked aliases inherit the nearest nullable view, explicit false narrows to the real non-null JSON classifier, and a later true reopens null. The same view applies to ordinary value references, collection leaves, HTTP JSON roots and SSE data.

```json
{
  "Object": {"type": ["object", "null"], "const": {"version": 1}},
  "Alias": {"$ref": "#/components/schemas/Object"},
  "Narrow": {"$ref": "#/components/schemas/Alias", "nullable": false},
  "Reopen": {"$ref": "#/components/schemas/Narrow", "nullable": true}
}
```

Nullable values accept and encode JSON null. A non-null value must retain the declared object/array kind, while other values of that kind retain their contents. Narrowed views reject null and wrong kinds. Root alias assignment and standalone serialization use the backing classifier/nullable adapter; `const` is still an approximation without literal membership or a root/value-position default. The original declaration's `W_CONST_STRUCTURED_APPROXIMATION` remains located at `/const`. Native numeric spelling/precision behavior follows the caller's JSON codec.

Ordinary DTO fields referencing these const declarations retain the [body contract](structured-const-fields.md): a non-null, always-encoded constant body default with no constructor/copy argument, even when the referenced value root is nullable. Missing keys use that declared literal; explicit null fails this non-null field codec. Value uses are distinct: list/map leaves can accept null, and a nearest false item/reference rejects it. Nullable roots do not create Patch fields or infer missing/default behavior for root HTTP values.

HTTP JSON bodies/results retain actual null versus object/array values. SSE data text `null` produces a present event whose data value is Kotlin null for a nullable view; it is distinct from an absent data frame. Narrowed data rejects null, and wrong JSON kinds fail typed decoding. Flow error/close/recovery policy is unchanged. String/ByteArray raw HTTP projections and broader protocol handling have separate contracts.

Original nodes and warnings retain source authority. Alias/root intent cannot erase malformed type/nullable/metadata or unsupported default/native/format/constraint keywords, even through explicit false. Union/hierarchy data bodies use the [pinned body contract](pinned-structured-const.md). Discarded extra-key and overridden declarations use their [validation-only contract](discarded-structured-const.md); general nullable/composition/internal-reference graphs retain their separate exclusions. Const arrays remain JSON classifiers rather than transparent List aliases. Full cached alias/source work, depth 48, checked collection edges 16, serialized/method slots 240, composition work 16,384 and document JSON/input limits remain unchanged. Retained body defaults still participate in optimizer/name-lock identity.

Migration: these nullable root/alias/value views were previously rejected. Original nullable counterexamples are preserved verbatim as explicit admissions, while malformed discarded sources remain negative coverage. Previously accepted non-null values and direct ordinary body fields retain generated APIs, bytes and warnings. Independent boundary probes do not establish Cartesian maxima, minimum toolchain compatibility, Android or KMP runtime support.

Run `bash scripts/test.sh feature nullable-structured-const` for fresh Data/Plain × validation projects and all four JSON settings. See the [fixture](../tests/fixtures/http/nullable-structured-const.json), [root/alias contract](../core/nullable_structured_const_test.go), [source guards](../core/nullable_structured_const_guards_test.go), [depth/ownership limits](../core/nullable_structured_const_resources_test.go), [native preservation](../internal/cli/nullable_structured_const_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableStructuredConstContracts.kt).
