# DTO builtin collection reference getter uniqueness

With validation enabled, the existing ordinary DTO `uniqueItems` role also admits fields referencing already-supported finite List/typed Map trees with plain String, Boolean, Int, Long, Float or Double leaves. Existing named aliases, internal collection sources and finite bridges retain their canonical Kotlin types. The outer uniqueness key compares the actual getter values structurally in both Data and Plain class styles, including nested collections, Map key-order independence and mapped Float/Double zero equality.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Numbers":{"type":"array","items":{"type":"number","format":"float"}},
  "Item":{"properties":{"values":{"$ref":"#/components/schemas/Numbers","default":[0,1]}}},
  "Holder":{"properties":{"items":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}
}}}
```

Here an omitted `Item.values` and an explicit `[-0.0,1]` contribute equal mapped getter keys. The original annotation is still checked before required-field initializer suppression; supplying a malformed value never becomes valid merely because the field is required. Field-own defaults use the existing [referenced collection grammar](referenced-collection-field-defaults.md), original value/depth/byte limits and actual alias default helper. Target/alias defaults are not inherited. This source role excludes target/alias annotations rather than granting new component-factory permissions.

Nearest explicit nullable intent, existing direct dictionary views, optional presence, required fields and Patch contracts stay intact. A null getter and an empty collection getter remain distinct keys. Kotlin constructors and decoders enforce the same admitted uniqueness/checks; JSON defaults and mapped key normalization remain separate. HTTP request/response and cold SSE consumers use those same actual types and validators.

## Original source and physical type authority

Forward collection aliases have no stable backing during early DTO-key discovery. The analyzer therefore builds a separate abstract mapped key descriptor from the original terminal schema through the shared field planner, with isolated probe metadata. This never installs a replacement backing or changes the source AST. It retains the real reference type and original backing address, raw chain trees, physical document paths and original owners.

After binding, the final getter must match that descriptor and its captured physical type. Named getters retain their original backing address; original named/borrowed collection snapshots preserve actual declaration identity, nested type/check objects, source nodes and alias views. Same-byte replacement declarations, source nodes, detached getter backings and copied check objects cannot authorize a different plan. The first final barrier has the same authority as later checks.

Source depth remains48, including incoming reference/object/collection hops and cached suffixes. Capture work16,384/bytes8MiB is shared across sources and references; each reference/path traversal is charged even when node captures are reused. DTO default key work and annotation value work retain their existing16,384 limits, rendered defaults32KiB, checked traversal16, and member/JVM slot240 limits. Retained initializer trees count completely when an object omits a defaulted getter. These are existing limits, not memory or speed claims.

## Boundaries and verification

This increment does not admit reference leaves within the target collection tree, enum/const/null-only/temporal/nominal/raw/native/valueclass leaves, composed targets, projected DTOs or target/alias defaults. Existing independently documented roles remain available. General typed/nominal/projected uniqueness and wider reference graphs remain open.

[Core acceptance/default/depth/refusal tests](../core/dto_builtin_collection_reference_unique_test.go), [original source/type/backing/check/budget tests](../internal/schema/dto_builtin_collection_reference_unique_test.go), [bounded deterministic/default-key fuzz](../core/dto_builtin_collection_reference_unique_fuzz_test.go) and [native parity/failure preservation](../internal/cli/dto_builtin_collection_reference_unique_process_test.go) exercise the role. `bash scripts/test.sh feature dto-builtin-collection-reference-unique` freshly generates Data/Plain validation-off/on controls, optimized/locked output, 240 referring arrays with 240 referenced collection getters and a separate full checked-depth profile. It executes four Json settings, defaults/nulls, constructors, Map order/Float keys, Patch, HTTP and fragmented SSE. Full HTTP compilation retains the new consumer in its complete source inventory.

**Migration:** eligible validated DTO uniqueness inputs that previously rejected a referenced builtin collection getter now generate. Existing accepted Kotlin APIs, JSON/HTTP/SSE behavior, ownership and lock formats keep their contracts. Wider roadmap/platform/final candidate acceptance remains separate.

[Existing temporal/UUID mapped leaves](dto-special-collection-reference-unique.md) have a separate extension using the same original source/alias/backing authority, strict annotation grammar and existing mapped equality. Other leaf/source roles retain their boundaries.

[Native enum collection reference getters](dto-native-collection-reference-unique.md) have a separate validation-only contract preserving the original vocabulary and actual alias/backing/check authority, strict own annotations and existing unchecked-mode gates.
