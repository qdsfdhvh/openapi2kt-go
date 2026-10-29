# List and typed Map positional-array sources

OpenAPI 3.1 positional members and explicit schema tails accept finite List and
typed Map sources admitted by the ordinary collection planner, including existing
named/internal collection reference paths. This extends D03/D06 and F08/M03/M07/Q05.
Source admission is context-specific; this does not introduce new resolver graphs,
recursive collection aliases or general union/const/Boolean leaf roles.

```json
{"Numbers":{"type":"array","items":{"type":"integer"}},
 "Rows":{"type":"array","prefixItems":[{"type":"array","items":{"type":"integer"}}],
         "default":[[1,2],[]]},
 "Referenced":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Numbers"}],
               "default":[[3]]},
 "Maps":{"type":"array","prefixItems":[{"additionalProperties":{"type":"integer"}}],
         "default":[{"default":7}]}}
```

The selected element uses the actual collection type: Rows is `List<List<Int>>`,
Referenced retains `List<NumbersDto>`, and Maps is `List<Map<String, Int>>`. Existing
nested collection, enum/DTO leaf, nullable, checks and codec policies remain.
Separate inline collection nodes stay different sources even with equal shapes;
references reaching the same original collection with the same nullable view can
be uniform. An explicit tail wins only after complete prefix validation. Mixed
and true-tail cases keep the raw JSON List projection and approximation warning.
Lists do not enforce tuple position or exact length and may narrow unrestricted
tails. Existing uniqueness eligibility remains separate.

## Own defaults and presence

Outer field, original component-root and independent component-alias annotations
validate through the actual selected collection tree and its final codec. Leaves
retain original kinds, enum vocabularies and eligible ordinary object/allOf walks;
counts, constraints and nullable intent stay independent at every layer. Default
JSON keys such as `default` or `items` remain data. Unsupported child/member
annotations are not granted permission by an empty/null outer default.

Named collection backings can be forward declarations. Such field annotations
wait for actual alias linking before literal validation and initializer binding.
The source and type remain original; no schema copy or guessed nominal classifier
is substituted. Required and Patch annotations still validate before initializer
suppression; required arguments retain no initializer and Patch stays Absent.
Targets, aliases and container children contribute no inherited annotation.

Helpers and factories decode fresh values through their actual final serializers,
including nested DTO/enum classifiers after optimization or relocation. Ordinary
DTO-owned fields, simple tuple-array HTTP values and typed SSE retain their
existing context-specific leaf codecs and projections. A collection source does
not turn a plain simple-HTTP object projection into a DTO or add a new direct
operation-root composition role.

## Original sources and limits

Every source, including a mixed or tail-discarded collection, completes ordinary
and deferred validation in an isolated analyzer. Temporary declarations/defaults/
registrations are not emitted. The main planner owns actual selected declarations
and collection backings. Original document paths, member/item nodes, reference
catalogs, retained element/backing addresses, type/check snapshots and final
source facts remain authoritative; same-byte copies, missing facts and changed
backings do not acquire permission.

Original finite container/reference edges remain in cached source heights after
projection. Preserved typed edges already pay their ordinary height; only erased
work is added after forward aliases link, before heights are cached. Direct and
cached/forward source48 boundaries, cumulative source16384 work/8MiB capture,
default16384 values/32KiB, checked16 and JVM240 limits remain. These are independent
bounds, not a promise that all combined maxima compile on every target.

Separate positional uses pay cumulative work and bytes for overlapping referenced
schema graphs, including nominal DTO leaves. Original node identity terminates
local sharing and nominal recursion; it does not reset the document-wide budget.
Only schema edges grant reference authority. A `$ref` key inside default/const/
enum data or metadata remains literal data.

Formerly rejected eligible collection tuple inputs now generate. Existing accepted
API/wire and manifest/name-lock formats retain their contracts. Source validation
is shared with the existing inline DTO/allOf tuple pass; no performance claim is
made. General tuple/Boolean/union/recursive/default graphs, platform matrices and
final candidate acceptance remain independent work.

[Core](../core/tuple_collection_sources_test.go),
[private identity/isolation](../internal/schema/tuple_collection_sources_test.go),
[structured fuzz](../core/tuple_collection_sources_fuzz_test.go),
[native process checks](../internal/cli/tuple_collection_sources_process_test.go),
[synthetic input](../tests/fixtures/http/tuple-collection-sources.json) and the
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleCollectionSourcesContracts.kt)
define this profile. `bash scripts/test.sh feature tuple-collection-sources` runs
Data/Plain × validation off/on, baseline and combined optimized output with
nonempty hoist locks, four Json settings, 240 independent fields, own defaults,
required/Patch, unknown enum values, controlled HTTP and fragmented still-open SSE
failure/closure/reuse. The complete `http` registry includes the consumer. Actual
Kotlin compilation/runtime, source scans, final all and platform evidence remain
distinct; PREPARE_ONLY is not runtime acceptance.
