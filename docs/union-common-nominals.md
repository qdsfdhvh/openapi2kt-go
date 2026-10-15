# Common nominal references in object unions

[Inline object and union values](union-inline-values.md) have separate original variant owners and variant-only fields, even for identical shapes.

Object `oneOf` / `anyOf` variants and supported object union-in-`allOf` products may expose a common property that references an actual generated DTO, union or hierarchy. All variants must contain the field and agree on its canonical Kotlin classifier and codec. Supported aliases and typed internal reference chains use the same original declaration; they do not own copied variants or replacement DTOs.

```json
{
  "Record": {"properties": {"id": {"type": "string"}}, "required": ["id"]},
  "Choice": {
    "oneOf": [
      {"properties": {"kind": {"const": "a"}, "value": {"$ref": "#/components/schemas/Record"}}, "required": ["value"]},
      {"properties": {"kind": {"const": "b"}, "value": {"$ref": "#/components/schemas/Record"}}, "required": ["value"]}
    ]
  }
}
```

The sealed Choice base exposes `abstract val value: Record`. Each known variant retains its own constructor property and the original Record serializer. Ordinary/allOf DTOs, native and object unions, retained hierarchy classifiers and supported reference views follow the shared source/type planner and final declaration checks.

## Identity, presence and codecs

Common type comparison preserves actual nominal identity. Two different DTOs with identical fields remain different classifiers and have variant-only access. References to supported aliases of the same target share the canonical classifier. Root nullable intent and explicit false views follow ordinary typed references; common nullability joins the existing required/optional and nullable rules. A native union's non-null Null carrier remains separate from Kotlin null.

Known variants retain the original target constructor, checks and codec. With `--validation`, original DTO/collection/leaf validation still runs at its existing typed boundary. Nominal references do not erase a target's interior checked-edge limits. AllOf source fields, required order, discarded-field validation and source warnings follow their existing contracts. Field metadata probing inspects a typed reference; it does not borrow a nullable DTO's shape into the surrounding variant.

Unknown carriers preserve `rawElement` and re-encode the original JSON. A non-null common nominal property on an outer Unknown is unavailable and throws the existing `IllegalStateException`; a nullable property returns the existing null fallback. An inner unknown union value remains the original union's Unknown carrier. Patch, request-only presence, typed HTTP and SSE payloads use the canonical target serializers.

## Source authority and limits

Only original schema nodes and actual retained declarations authorize a common classifier. Shared final checks require the real DTO/union or hierarchy lineage and supported alias owners. Missing, replaced, ambiguous or scratch declarations cannot supply a classifier merely by using the same name. No expanded JSON or synthetic nominal component is introduced.

Complete recursive JSON covers and raw JSON roles keep their existing transparent projection; their union syntax does not make them nominal targets. Inline object/union values retain variant-only lexical owners under their separate contract. Direct enum common-value projection retains its existing policy. Shared inline ownership, nullable product alternatives, discriminator intersections and arbitrary reference/constraint graphs remain separate capabilities.

Source depth 48, expansion work 16,384, checked collection edges 16 and serialized JVM slots 240 remain. A pinned tag consumes a serialized slot, so the independent wide-union probe uses 239 nominal fields plus its tag. Nominal constructor/codec recursion barriers remain; transparent alias hops retain their complete source budget, including cached false views. Independent limits do not establish simultaneous maxima, minimum Kotlin versions or Android/KMP support.

## Migration and verification

Previously rejected compatible nominal-reference common fields now expose the canonical base property. Newly accepted incompatible classifiers retain variant-only fields. Existing accepted unrelated input APIs and wire behavior remain protected by contract and compatibility tests. Product versions and published packages are unchanged.

The [synthetic fixture](../tests/fixtures/http/union-common-nominals.json), [core types](../core/union_common_nominals_test.go), [identity/source guards](../core/union_common_nominals_guards_test.go), [resource guards](../core/union_common_nominals_resources_test.go), [actual ownership checks](../internal/schema/union_common_nominals_test.go), [native preservation](../internal/cli/union_common_nominals_process_test.go) and [fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionCommonNominalsContracts.kt) define this F04/F07/M09/M10/M11 increment. Run `bash scripts/test.sh feature union-common-nominals` for Data/Plain × validation, four Json settings, base-property calls, original DTO/union/hierarchy codecs, Null/Unknown/Patch, controlled HTTP and fragmented SSE failure/cancellation/closure/reuse, with independent source/checked/constructor resource probes.
