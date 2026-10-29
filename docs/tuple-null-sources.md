# JsonNull positional-array sources

OpenAPI 3.1 `prefixItems` members and explicit schema tails accept existing plain
`type:"null"`, singleton null enums and null consts, with explicit or inferred
type. Existing named references and pure aliases retain their resolver policy.
This extends D03/D06 and F08/M03/M07/Q05; general const/union/recursive graphs and
nullable component-root admission remain independent.

```json
{"Rows":{"type":"array","prefixItems":[{"type":"null"},{"enum":[null]},{"const":null}],
         "default":[null,null]},
 "Only":{"type":"null"},
 "Referenced":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Only"}]}}
```

Plain, enum and const null share the existing singleton JsonNull codec. With the
same effective nullable view, they form `List<JsonNull>` without a new classifier,
enum, wrapper or serializer. A nullable member gives `List<JsonNull?>`. Original
sources remain independently validated and captured, including named aliases;
matching type spelling cannot grant source permission. Mixed non-null domains
retain `List<JsonElement>` projection. An explicit tail wins only after complete
prefix validation, and true/false tails retain their existing approximation.

Lists do not enforce tuple position or exact length and may narrow unrestricted
tails. Existing `W_TUPLE_APPROXIMATION` remains. JsonNull accepts only wire null;
wrong numbers, strings, objects, arrays and booleans fail its existing serializer.
Kotlin `JsonNull`, Kotlin `null` and absence remain distinct source states. A
nullable JsonNull serializer decodes wire null to Kotlin null; no implicit
nullable intent is inferred from the singleton domain.

## Defaults and consumers

Eligible outer field, component-root and independent alias annotations validate
against the original source-authorized null domain and actual selected codec.
`[null]` is valid for `List<JsonNull>`; `["null"]` and `[1]` are invalid. Required
annotations validate before initializer suppression, and nullable request fields
retain `Patch.Absent`. Targets and aliases contribute no inherited annotation.
Actual helpers/factories decode fresh lists; explicit null and omitted fields
retain the existing Json settings and presence rules.

Nested List/typed Map fields, simple inline operation arrays, shared HTTP DTOs
and typed SSE use their actual context codec. A direct null success root keeps
its existing Unit projection; this increment does not change that separate role.
An empty/null outer annotation or a scalar tail cannot hide malformed original
vocabularies, unknown keys, positional defaults or excluded format/native roles.
Duplicate null enum entries and the raw240 cap retain their original diagnostics.
General null/const component and child default permissions remain independent.

## Sources, bounds and compatibility

Original reference/ancestor/terminal/vocabulary nodes, catalog bindings and
actual retained element addresses/type/check snapshots remain authoritative.
Named null exports retain their existing ScalarAlias or transparent ValueAlias
backing; missing, changed or duplicate actual codecs reject. Original reference
hops still contribute to cached collection height after raw/tail projection.
Source48, cumulative capture16384/8MiB, default16384 values/32KiB, checked16 and
JVM240 limits remain independent. No source schema is rewritten or copied.

Formerly rejected eligible null tuple inputs now generate. Existing accepted
Kotlin API/wire and manifest/name-lock formats retain their contracts. No
performance, whole-platform or final-candidate acceptance is implied.

[Core types/defaults/budgets](../core/tuple_null_sources_test.go),
[private source/codec proof](../internal/schema/tuple_null_sources_test.go),
[structured fuzz](../core/tuple_null_sources_fuzz_test.go),
[native parity and failure preservation](../internal/cli/tuple_null_sources_process_test.go),
[synthetic fixture](../tests/fixtures/http/tuple-null-sources.json) and
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleNullSourcesContracts.kt)
define this profile. `bash scripts/test.sh feature tuple-null-sources` runs
Data/Plain × validation, baseline/combined optimized output, nonempty hoist locks,
240 independent fields, four Json settings, defaults, required/Patch, direct and
shared HTTP and fragmented live SSE failure/closure/client reuse. The complete
`http` registry includes the same consumer. PREPARE_ONLY is preparation only;
other platforms and complete current-source `all` require separate evidence.

[Scalar const tuple sources](tuple-scalar-const.md) separately admit existing scalar const codecs and strict independent outer defaults, preserving original literal/ref/backing identity, nearest nullable and cached/cumulative source budgets. Retained catalog-binding records count toward cumulative capture work; structured const/union/recursive roles remain independent.
