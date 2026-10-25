# Intrinsic nullable ordinary allOf DTO roots

Nullable allOf roots containing supported object unions use the separate [product union contract](nullable-product-unions.md); their physical type is a union with original variants, not an ordinary DTO.

An ordinary component object assembled with `allOf` may declare `nullable: true` or `type: ["object", "null"]` when its final model retains at least one constructor property. The generator keeps the same non-null physical Kotlin class, constructor, Data-class copy and companion serializer. Typed references expose `T?`; explicit false views expose the canonical `T`.

```json
{
  "nullable": true,
  "allOf": [
    {"properties": {"id": {"type": "string"}}, "required": ["id"]},
    {"properties": {"label": {"type": "string"}}}
  ]
}
```

This produces the existing ordinary serializable DTO rather than a new nullable wrapper or duplicated model. Root aliases and supported typed internal chains retain original source identity and nearest nullable intent. List items and dictionary values retain their independent nullability. Supported single-reference wrappers follow the same typed reference policy.

## Constructors, codecs and consumers

The companion `Dto.serializer()` remains non-null, including when called through a nullable typealias. Use `serializer<Dto?>()`, a reified nullable alias or `Dto.serializer().nullable` to handle root null. Constructors and copies create non-null instances and retain their original field checks. With `--validation`, constructor, copy and automatic decoding enforce supported primitive/collection checks. Bare nullable references and aliases do not create a new validation barrier.

Member order, later field replacement, source required order and generated const fields follow the existing allOf approximation. Original fields, schema nodes, reference hops and diagnostic Pointers remain unchanged. Discarded sources still validate; composition warnings still describe field merging rather than exact JSON Schema intersection validation.

Shared component nullable properties keep their existing default-null policy. Inline request required-nullable fields have no default; optional-nullable fields use Patch. Typed JSON HTTP roots and SSE data use the actual nullable codec, while narrowed views reject JSON null. Request presence, nullable list/map entries and Patch are governed by their existing contracts.

## Limits and exclusions

The final declaration must be an actual ordinary Model with constructor fields. Empty/raw allOf roots, const-only roots, union-containing products, dictionary alternatives and discriminator composition retain their separate policy. [Nullable allOf operands](nullable-intersections.md) additionally admit constructor-bearing component roots whose operands all allow null, preserving the canonical physical DTO and aliases. Mixed operands narrow null unless an explicit root override applies.

[Nullable ordinary object operands](nullable-intersections.md) can be borrowed as allOf shapes, with consuming/alias intent controlling the intersection result while the original constructor and source remain canonical. [Nullable ordinary DTO union members](nullable-object-reference-members.md#composed-and-internal-constructor-sources) separately accept eligible allOf constructor sources, preserving physical fields and nearest null intent. Nullable SSE envelopes and unsupported mappings retain their diagnostics. Whole-object count bounds on explicit nullable ordinary allOf components use the [nullable count contract](direct-nullable-component-counts.md#explicit-nullable-allof-component-counts); direct explicitly nullable plain-object roots retain their own admission. Supported field/container/leaf bounds are independent.

Source depth 48, expansion work 16,384, checked collection edges 16 and serialized JVM slots 240 remain. Nominal DTOs retain their recursion/validation barrier; source aliases retain all incoming work. Independent maximum probes do not establish simultaneous maximum width/depth, arbitrary Kotlin toolchains, Android or KMP support.

## Migration and verification

Previously rejected explicit nullable ordinary allOf components with real constructor fields now generate their canonical DTO and nullable typed uses. Existing accepted unrelated inputs keep their APIs, codecs and wire behavior. The historical intrinsic-nullable DTO exclusion input containing real allOf fields is retained as a positive regression; empty and const-only inputs remain rejected. Published packages and product versions are unchanged.

The [synthetic fixture](../tests/fixtures/http/nullable-allof-dtos.json), [core types](../core/nullable_allof_dto_test.go), [source and resource guards](../core/nullable_allof_dto_guards_test.go), [original Model/source checks](../internal/schema/nullable_allof_dto_test.go), [native preservation](../internal/cli/nullable_allof_dto_process_test.go) and [fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableAllOfDTOContracts.kt) define this F04/F07/M07/M09 increment. Run `bash scripts/test.sh feature nullable-allof-dto` for Data/Plain × validation profiles, four Json settings, canonical and nullable serializers, aliases/copy, Patch, controlled HTTP and fragmented SSE failure/cancellation/closure/reuse. General nullable composition and the full compatibility matrix remain separate.

Explicit nullable ordinary allOf component roots may additionally use [whole-object JSON property counts](direct-nullable-component-counts.md#explicit-nullable-allof-component-counts) with validation enabled. This does not permit borrowing counted component shapes. Derived nullable components keep count-role restrictions separate.
