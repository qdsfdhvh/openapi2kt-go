# Outer object defaults with repeated numeric constraints

With `--validation`, an ordinary allOf object default may consume a complete
repeated plain Int, Long, Float or Double property group already admitted by
[primitive aggregation](referenced-allof-bounds.md). Existing scalar mappings
apply, including unformatted integer/number. Enum, ref, differing format/type,
hierarchy/common and unchecked duplicate groups keep their separate roles.

```json
{"Record":{"allOf":[
  {"properties":{"value":{"type":"integer","minimum":-10,"default":-5}}},
  {"properties":{"value":{"type":"integer","maximum":10,"default":5}}}
],"default":{"value":0}}}
```

`RecordDto()` uses the winning field annotation 5; `RecordDto.default()` supplies
0 through the actual model serializer. Every original own default validates
before being overridden, including when the outer annotation is empty, null,
required or Patch-suppressed. The winning own annotation satisfies the complete
retained conjunction. Supplied outer members satisfy the same constraints;
omitted members use actual constructor defaults and required keys remain
required. A factory on a target is never inherited by referencing fields.

This applies through existing ordinary allOf default roles: direct component
factories, independently defaulted aliases, consuming direct/ref fields, nested
plain object members and admitted List/typed Map constructor leaves. Source,
nullability, final declaration/codec and ownership gates retain their contracts.

The existing primitive merge selects the strongest original bound in each of
four inclusive/exclusive lower/upper categories and the admitted single factor.
The literal plan borrows those original nodes, exact numbers and Pointers; it
does not synthesize schemas or enter declaration/name-lock identity. Each
supplied value checks at most those four endpoints and one factor, independently
of the number of original contributions. Every discarded original property is
still validated and charged to the shared source work budget.

Int/Long comparisons and integer multipleOf use exact original numbers,
including Long extrema and integers above 2^53. Float/Double comparisons retain
the existing mapped IEEE semantics: rounded-equivalent numbers may satisfy an
inclusive bound, while an exclusive mapped-equal bound rejects. OpenAPI 3.0
exclusive booleans retain their original modifier attribution; 3.1 bounds remain
numeric. Floating multipleOf defaults keep their existing exclusion. A bad
supplied member reports its consuming `/default/member` Pointer. Source48,
work16384, value16384/default32KiB, checked16 and JVMslots240 remain unchanged.

`bash scripts/test.sh feature allof-numeric-defaults` generates Data/Plain with
validation enabled in baseline and combined optimized/locked packages. The
[fixture](../tests/fixtures/http/allof-numeric-defaults.json) and consumer exercise
four Json configurations, exact Long/mapped Float/Double/negative zero,
source/winner/outer defaults, nullable aliases and values, required/Patch,
nested/ref/List/Map bindings, a 240-property factory, controlled HTTP and live
byte-fragmented UTF-8 SSE failure/closure/client reuse. Native tests cover
core/file parity, readonly/no-op and old-output preservation. Source tests retain
original node identity, selected-bound equivalence and exact shared-work limits;
bounded fuzz covers repeated contributions and nullable outer annotations.
Preparation-only mode is separate from compilation/runtime evidence; additional
platforms need their own validation.

**Migration:** eligible formerly rejected numeric outer defaults now generate.
Already accepted declarations, constructor validation, wire behavior and
persisted formats retain their contracts. General defaults, broader numeric or
nominal compositions and platform completion remain separate.

[Outer allOf UUID defaults](allof-uuid-defaults.md) extend D03/D05 for complete original ordinary string/uuid contribution groups under validation. Retained length/pattern checks inspect canonical Uuid text, while original own/winning/outer annotations, actual constructor/codec, required/Patch, source Pointers and shared budgets remain. Invalid values fail at original annotations; no schema copying, new codec or persisted identity is introduced. Wider references/enum/formats/defaults/aggregation/platforms stay separate.
