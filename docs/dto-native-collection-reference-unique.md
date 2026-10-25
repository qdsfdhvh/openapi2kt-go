# DTO native enum collection reference getter uniqueness

With validation, ordinary DTO `uniqueItems` keys also admit already-supported finite List/typed Map references with Int/Long/Float/Double/Boolean enum leaves. Existing named aliases, internal collection sources and finite bridges retain their actual Kotlin types and canonical ownership. This extends the [collection reference role](dto-builtin-collection-reference-unique.md); String enum classifiers and other leaf/source roles remain separate.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Numbers":{"type":"array","items":{"type":"number","format":"float","enum":[0,1]}},
  "Item":{"properties":{"values":{"$ref":"#/components/schemas/Numbers","default":[0]}}},
  "Holder":{"properties":{"items":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}
}}}
```

The key compares actual getter values structurally in Data and Plain classes. Native enums keep their existing scalar wire types; membership and retained bounds remain opt-in runtime checks. Map key order and mapped Float/Double zero equality follow the existing key contract. Nullable/required/Patch, direct dictionary views and strict field-own defaults keep their existing behavior; target/alias defaults are not inherited or newly admitted.

## Original vocabulary and actual checks

Eligibility requires the existing original native-enum source predicate under validation. A primitive Kotlin spelling alone grants no enum source authority. The isolated shared planner builds an abstract mapped key from the original source; the real physical reference remains linked to its original backing. Final original alias/declaration/backing identity, named/borrowed nested type/check snapshots, raw chains, physical source paths and original vocabulary nodes must all agree. Replacing an enum value with a same-byte node, editing membership, copying/dropping actual check objects or substituting a same-byte declaration rejects, including before the first final barrier.

Original annotation kind, vocabulary and numeric precision rules run before required initializer suppression and mapped key comparison. A Float annotation that merely rounds to an enum member remains invalid if it fails the original vocabulary rule. Strict own default diagnostics retain their original value Pointers. Constructors, decode, HTTP and cold SSE use the actual retained checks.

The established validation-off type-admission rules are unchanged. Named unchecked native-enum collection references retain their scalar approximation/warnings; borrowed internal native-enum references still reject without validation. The unchecked feature control uses the existing primitive borrowed path; checked profiles preserve the full native vocabulary and borrowed paths. This adds no implicit membership enforcement to standalone aliases or unchecked code.

Raw vocabulary240, source48 including cached/incoming hops, shared capture work16,384/bytes8MiB, key/value work16,384, rendered defaults32KiB, checked traversal16 and member/JVM slots240 retain their original limits. No source AST, catalog backing, scalar spelling, persisted ownership or name-lock format is replaced.

## Boundaries and verification

String enum/const/null-only/nominal/raw/native override/valueclass leaves, references inside the target tree, composed/projected sources and target/alias factories retain their separate boundaries. General typed/nominal/projected uniqueness and wider graph/default/platform completion remain open.

[Core controls/defaults/raw vocabulary/mode gates](../core/dto_native_collection_reference_unique_test.go), [original vocabulary/check/alias identity](../internal/schema/dto_native_collection_reference_unique_test.go), [bounded deterministic fuzz](../core/dto_native_collection_reference_unique_fuzz_test.go) and [native parity/strict-default failure preservation](../internal/cli/dto_native_collection_reference_unique_process_test.go) exercise this role. `bash scripts/test.sh feature dto-native-collection-reference-unique` freshly runs Data/Plain validation controls, optimized/locked output, 240 native collection reference getters with240 referring arrays, an independent full checked-depth native Float profile, four Json settings, constructors/decoders/default keys, Patch, HTTP and fragmented SSE. The full HTTP compilation retains the new consumer in its complete inventory. Preparation-only runs do not prove Kotlin execution.

**Migration:** these eligible validated DTO uniqueness inputs previously rejected their native-enum collection references and now generate. Existing accepted APIs, serialization/HTTP/SSE behavior, ownership and locks retain their contracts. This is a bounded D05/M03/M07/M08/M12/V05 increment; broad E04/platform/final candidate acceptance remains open.
