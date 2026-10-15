# Common dictionary properties in object unions

[Inline nominal dictionary values](union-inline-values.md) retain their real variant-owned classifiers and variant-only dictionaries; named shared values keep the common property contract below.

Object `oneOf` / `anyOf` variants and supported object union-in-`allOf` products may expose a shared dictionary property on their sealed Kotlin base. Direct dictionaries and supported references to named dictionaries use the existing typed collection planner:

```json
{
  "oneOf": [
    {"properties": {"kind": {"const": "a"}, "value": {"additionalProperties": {"type": "string"}}}, "required": ["value"]},
    {"properties": {"kind": {"const": "b"}, "value": {"additionalProperties": {"type": "string"}}}, "required": ["value"]}
  ]
}
```

The sealed type exposes `abstract val value: Map<String, String>`. Each variant retains its own constructor property and serializer. A named dictionary alias remains a transparent alias, and named DTO, enum, value-class or union values retain their actual generated classifier and codec. Common type planning inspects original source nodes and linked canonical backing; it creates no replacement model or copied specification.

Public identity compares the actual Kotlin backing, including value nullability and codec annotations. Dictionary and primitive validation metadata do not change the abstract classifier. If all variants contain the field but their value types differ, the base omits that common property and each variant keeps its own field. A missing field also prevents a common property. This does not select or convert values between incompatible dictionaries.

## Presence and validation

Common container nullability joins the existing variant required/optional and nullable policy. Nullable map values remain separate from nullable containers. Open/raw-value dictionaries expose `JsonObject`, which retains object-only decoding and JSON null entries. Known variants use their normal serializers; required keys and tag or presence dispatch keep their existing behavior.

With `--validation`, each variant retains its original dictionary count and element checks, even when two variants have different bounds. Constructor, Data-class copy, automatic deserialization and typed HTTP/SSE payload construction enforce those checks. Bare `Map` assignment or serialization does not enforce schema constraints. Bounds with validation off retain the existing unsupported-key rejection. Discarded allOf fields still obey validation noDrop.

Content-dispatch Unknown carriers preserve `rawElement` and re-encode it unchanged. A non-null common Map/JsonObject property on Unknown follows the existing unavailable-property policy and throws `IllegalStateException`; a nullable property returns the existing null fallback. Raw JSON is not interpreted as a known common property. The existing Patch and request-only presence contracts apply to nullable dictionary and union fields.

## Limits and exclusions

Original source depth 48, expansion work 16,384, checked collection edges 16 and serialized JVM constructor slots 240 remain. A pinned tag consumes a serialized slot, so the independent width probe has 239 Map fields plus its tag. Transparent alias cycles remain errors; a nominal DTO value retains its recursion barrier. Independent resource probes do not establish simultaneous maximum width/depth, minimum Kotlin versions or Android/KMP support.

Inline dictionary values requiring their own nominal declarations retain variant-only dictionaries and actual lexical owners. [Common nominal references](union-common-nominals.md) have a separate actual DTO/union/hierarchy classifier contract. Shared inline ownership, dictionary union alternatives, nullable allOf alternatives, discriminator intersections and arbitrary constraint aggregation remain separate.

## Migration and verification

Previously rejected compatible dictionary common fields now generate a typed base property. Existing accepted unrelated inputs retain their generated APIs and wire behavior; incompatible dictionary value fields are newly accepted with variant-only access. No generated product version or published package changes are implied.

The [synthetic fixture](../tests/fixtures/http/union-common-dictionaries.json), [core types and presence](../core/union_common_dictionaries_test.go), [source and validation guards](../core/union_common_dictionaries_guards_test.go), [resource limits](../core/union_common_dictionaries_resources_test.go), [private source/check planning](../internal/schema/union_common_dictionaries_test.go), [native output preservation](../internal/cli/union_common_dictionaries_process_test.go) and [fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionCommonDictionariesContracts.kt) define this F04/M03/M09/M11 increment. Run `bash scripts/test.sh feature union-common-dictionaries` for Data/Plain × validation profiles with four Json settings, public-property calls, original nominal codecs, Unknown, Patch, controlled HTTP and fragmented SSE failure/cancellation/closure/reuse. [Model](../scripts/kotlin/models.sh) and [HTTP](../scripts/kotlin/http.sh) suites provide broader integration verification.
