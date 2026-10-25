# Ordinary allOf DTO getter uniqueness

With validation, DTO array uniqueness also admits existing ordinary allOf constructors with supported scalar, enum, const and finite collection getters. Named component members and eligible original nested allOf DTOs keep their actual constructor, codec, nullable policy and field order. This extends getter equality; it does not turn the generator into an exact allOf intersection validator.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Base":{"properties":{"id":{"type":"string"}},"required":["id"]},
  "Item":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"version":{"type":"integer","default":0}}}]},
  "Holder":{"properties":{"items":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}
}}}
```

The key uses the complete original winning fields and required set from the same ordinary model-shape planner as the actual constructor. Inherited fields retain their original nodes, Pointers and defaults. Existing repeated-property selection and admitted primitive constraint aggregation remain authoritative; this adds no general default or discarded-contribution exemption. For example, an outer default on a repeated-field combination outside the existing aggregation contract can still reject even without uniqueness.

Keys compare actual runtime getters in Data and Plain classes. Existing mapped Float/Double, UUID/Instant, open enum serialName, List order and Map order rules remain. Omitted default members use actual constructor initializers; annotations validate against original sources before required/Patch suppression. Nullable allOf keeps its existing member-intersection and explicit-root policy. Const getters use decoded body values under the existing const approximation rather than assuming every instance has the source literal.

The original allOf source walk retains branch/reference targets, catalog bindings, alias heights and every winning field/required source. Final checks require the actual unique retained Model and original field mapping. Original property/TypeRef/validation object identities and immutable check values are recorded after nominal linking, including retained aggregate checks and inherited reference collections. Same-byte constructor/check replacements, changed bound values, changed source bindings or field Pointers cannot authorize a key. The shared deferred collection-default path carries the actual default-key registry.

A typed tuple List with an explicit admitted allOf DTO tail can use the same equality role. Positional or exact-length validation is still absent, and W_TUPLE_APPROXIMATION remains. Prefix-only tuple/default roles do not gain permission from the tail profile.

Whole-object counts, directional/common/hierarchy/variant projections, const-only or raw/dictionary/product-union compositions and unsupported nested nominal getters keep separate boundaries. Source48/cached/incoming reference limits, shared capture16,384/8MiB, default-key/value work16,384, default32KiB, checked depth16 and member/JVM slots240 remain.

[Core and exact former refusals](../core/dto_allof_unique_test.go), [original constructor/field/aggregate authority](../internal/schema/dto_allof_unique_test.go), [native parity and failure preservation](../internal/cli/dto_allof_unique_process_test.go) and [bounded deterministic fuzz](../core/dto_allof_unique_fuzz_test.go) cover this profile. `bash scripts/test.sh feature dto-allof-unique` generates fresh Data/Plain validation controls, optimized locked output, actual allOf240-field constructors with240 referring arrays, depth16, four Json settings, inherited defaults/enum/const/reference getters, Patch, HTTP and fragmented SSE.

**Migration:** eligible allOf DTO uniqueness and explicit-tail List inputs previously rejected and now generate. Existing accepted APIs, wire behavior, output ownership and lock formats retain their contracts. General composition/default/ownership graphs, platforms and final candidate acceptance remain open.
