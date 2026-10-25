# Outer object defaults with repeated String constraints

With `--validation`, an ordinary allOf object default may reuse the existing
retained conjunction for repeated plain String properties. Each repeated group
must contain every original contribution and at least one discarded enforcing
primitive check, with the same scalar/format/nullability signature. Effective
lengths and the single admitted portable pattern follow the existing
[primitive aggregation contract](referenced-allof-bounds.md).

```json
{"Record":{"allOf":[
  {"properties":{"value":{"type":"string","minLength":2,"default":"longer"}}},
  {"properties":{"value":{"type":"string","maxLength":3,"default":"ab"}}}
],"default":{"value":"abc"}}}
```

`RecordDto()` keeps the winning field default ab. `RecordDto.default()` supplies
abc. The earlier annotation longer remains valid against its own original
minLength and is not inherited. Every original own annotation is checked before
override; the winning annotation also satisfies all retained checks. Supplied
outer members satisfy that same conjunction. Missing optional members use the
actual constructor/serializer defaults; required members remain required.
Required/Patch annotations validate before initializer suppression.

This applies through the existing ordinary allOf default roles: direct component
factories and independently defaulted component aliases, consuming direct/ref
fields, nested ordinary object members and already admitted List/typed Map
allOf constructor leaves. Existing source/nominal/final binding gates still own
each role. Root and property nullable intent remain independent; omitted values
and null enclosing annotations still validate all contribution sources.

The walk keeps original nodes, Pointers and contribution order. Bad schema/own
annotations point to their original contribution; bad supplied members point to
the consuming `/default/member`. No schema tree or bound is synthesized. Literal
validation creates no emission hints or declarations. The final actual DTO keeps
its existing constructor checks and codec under optimization and locked names.
Every discarded and winning source is charged to the enclosing shared default
work budget. Source48/work16384/value16384/default32KiB/checked16/JVMslots240
remain unchanged.

Repeated plain String groups use this role; numeric groups have the separate
[retained numeric default contract](allof-numeric-defaults.md). Unchecked duplicate
groups, enum/ref/format aggregation, different patterns, hierarchy/common
sources and other excluded default roles retain their own contracts. Runtime
validation is still opt-in; pattern annotation matching retains the existing
ASCII limitation. Whole-object counts retain their separately admitted roles.

`bash scripts/test.sh feature allof-string-defaults` generates Data/Plain with
validation enabled, each in baseline and combined optimized/locked packages.
The [fixture](../tests/fixtures/http/allof-string-defaults.json) and consumer
exercise four Json configurations, distinct source/winner/factory defaults,
nullable aliases and values, required/Patch, original nested/ref/container
binding, a 240-property factory, controlled HTTP and byte-fragmented UTF-8 SSE
failure/closure/client reuse. Validation-off remains rejected by the original
bound profile. Native tests compare core bytes, successful reports and
readonly/no-op behavior, then preserve old output on invalid own/outer/alias
values. Source tests cover copied-contribution rejection, original AST identity
and exact shared-work limits; bounded fuzz covers multiple original contributions
and nullable enclosing defaults. `OPENAPI_KT_PREPARE_ONLY=1` is preparation only.

**Migration:** previously rejected eligible outer defaults now generate. Existing
accepted declarations, constructor checks, wire behavior and persisted formats
retain their contracts. General D03/D05, wider aggregation and additional
platforms remain open.

[Outer allOf UUID defaults](allof-uuid-defaults.md) extend D03/D05 for complete original ordinary string/uuid contribution groups under validation. Retained length/pattern checks inspect canonical Uuid text, while original own/winning/outer annotations, actual constructor/codec, required/Patch, source Pointers and shared budgets remain. Invalid values fail at original annotations; no schema copying, new codec or persisted identity is introduced. Wider references/enum/formats/defaults/aggregation/platforms stay separate.
