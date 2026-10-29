# Structured const positional-array sources

OpenAPI 3.1 positional members and explicit schema tails accept original
object/array const sources, including existing named references and pure aliases.
They retain actual JsonObject/JsonArray codecs and independent nullable views.
This extends D03/D04/D06 and F08/M03/M07/Q05. General union, recursive and default
graphs retain separate contracts.

```json
{"Rows":{"type":"array","prefixItems":[{"const":{"fixed":1}}],
         "default":[{"fixed":1.0}]},
 "Arrays":{"type":"array","prefixItems":[{"type":"array","const":[1,2]}],
           "default":[[1.0,2e0]]}}
```

Rows is `List<JsonObject>` and Arrays is `List<JsonArray>`. Explicit array consts
remain JSON values; they do not become ordinary List item schemas. Selected
sources use existing `valueConstant` planning and retained component codecs,
without a new enforcing class or serializer.

Same original terminal plus effective nullable view may be uniform. Separate
inline sources, different pins or const versus ordinary types retain raw mixed
projection. Explicit tails win after every original prefix validates. Runtime
membership remains approximated: another object or array of the same selected
JSON kind can decode. `W_CONST_STRUCTURED_APPROXIMATION` and the tuple's explicit
lost-literal warning remain. Positions, exact lengths and unrestricted tails
retain their existing List approximation. Raw/tail projections may discard the
original kind restriction; typed JsonObject/JsonArray elements retain theirs.

## Strict own annotations

Outer field/root/independent alias annotations compare each supplied member with
the complete original const literal. Object key order has no meaning; arrays
retain order and kinds; numbers use bounded exact decimal identity without
Double rounding. Thus `{"fixed":1}` and `{"fixed":1.0}` compare equally, while distinct
integers above Double's exact range remain distinct. Equality uses the existing
256-character number/±512 exponent profile. Original canonical keys are cached
once per terminal and charged to the shared capture budget; supplied values
retain their original diagnostics and value/byte budget.

Admitted finite collection tuple trees reuse the same structured literal leaf
checks and final codecs for their own outer annotations.

Helpers and factories decode fresh values through the actual selected codec,
including nullable views. Null requires actual nullable intent. Required and
Patch annotations validate before initializer suppression, and targets/children
contribute no inherited default. Invalid sources cannot hide behind an empty or
null annotation, a mixed projection or an overriding tail.

## Literal data and original authority

Const/default/enum payloads use a data-only capture role. Keys such as `$ref`,
`allOf`, `properties`, `description`, `default`, `items` and `enum` remain data at
every nested level. They grant no schema/reference permission. All original data
nodes and member/item edges remain captured, including metadata-named keys.
Schema composition edges keep their original resolver and catalog checks.

Original ancestors/references/literals/catalogs, actual retained scalar or
transparent alias backings, selected element addresses and type/check snapshots
remain authoritative. Same-byte copied data, missing canonical source facts or
changed JSON kind codecs reject. Source48 and cached reference hops after
raw/tail erasure, cumulative capture16384/8MiB, default16384 values/32KiB,
checked16 and JVM240 limits remain independent. Canonical keys and newly retained
literal data edges pay their existing work/byte budgets; over-budget inputs fail
without replacing existing output.

Previously rejected eligible structured tuple sources now generate. Previously
misinterpreted payload keys are captured as data. Existing accepted API/wire and
manifest/name-lock formats retain their contracts, subject to the explicit
resource boundaries. No performance, full-platform or final-candidate acceptance
is implied.

[Core](../core/tuple_structured_const_test.go),
[source/codec/data identity](../internal/schema/tuple_structured_const_test.go),
[structured fuzz](../core/tuple_structured_const_fuzz_test.go),
[native parity and preservation](../internal/cli/tuple_structured_const_process_test.go),
[synthetic input](../tests/fixtures/http/tuple-structured-const.json) and
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleStructuredConstContracts.kt)
define this profile. `bash scripts/test.sh feature tuple-structured-const` runs
Data/Plain × validation, baseline/combined optimized output with nonempty hoist
locks, 240 independent fields, four Json settings, strict defaults, same-kind
nonliteral runtime values, required/Patch, direct/shared HTTP and fragmented live
SSE failure/closure/client reuse. The complete `http` registry includes the same
consumer. PREPARE_ONLY is preparation; full all/platform evidence is separate.
