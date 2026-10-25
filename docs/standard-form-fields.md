# UUID and temporal form fields

Selected flat `application/x-www-form-urlencoded` request DTOs accept direct
`string/uuid`, `string/date` and `string/date-time` properties. They use the
existing `kotlin.uuid.Uuid`, `kotlinx.datetime.LocalDate` and `kotlin.time.Instant`
models and codecs, with required or optional fields and independent required or
optional bodies. The [closed form grammar](form-request-media.md) and original
source validation still apply.

Each present field uses its actual value's `toString()` followed by Ktor UTF-8
form encoding. UUID text is lowercase with hyphens; Instant text represents the
parsed instant in UTC, so original offset and fractional spelling are not retained.
LocalDate keeps its typed date representation. For example, a UUID parsed from
uppercase input sends lowercase text, and `2026-10-22T20:30:40.125+08:00` sends
`2026-10-22T12%3A30%3A40.125Z`. These are the existing scalar mappings, not a new
lexeme-preserving format or a restriction on values constructible by the typed
Kotlin APIs.

An optional field's constructor null omits its key. A missing or null optional
body produces no content or generated Content-Type. A present DTO uses the
ordinary form media and original field order/keys. HTTP and cold SSE share the
encoder and final callback override; encoding needs no ContentNegotiation
plugin. JSON default/null settings do not control form omission. JSON model and
SSE response codecs retain their separate existing contracts.

UUID length/pattern checks retain their existing `--validation` gate and inspect
canonical UUID text. Temporal string bounds remain unsupported. References,
aliases, formatted String enums, native overrides, default/const annotations,
schema-nullable fields/roots, structured/composed values, projections, explicit
Patch states and media encoding directives gain no permission. All original
source/work/value/checked/JVM-slot caps remain, including 240 constructor slots.
No source node, property binding or persisted identity is rewritten.

`bash scripts/test.sh feature standard-form-fields` generates Data/Plain with
validation off/on, baseline and combined dedup/hoist/merge/locked output, plus a
240-field mixed UUID/date/Instant constructor. The
[fixture](../tests/fixtures/http/standard-form-fields.json) and
[consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StandardFormFieldsMediaContracts.kt)
exercise four Json configurations, typed JSON round-trips and invalid dates,
UUID canonicalization, timestamp normalization, reserved names, exact form bytes,
optional field/body absence, UUID checks before dispatch, callback overrides,
bare HttpClient and fragmented UTF-8 SSE failure/closure/cancellation/reuse.
Core/source/native regressions retain the former UUID refusals as positives,
verify the original AST/retained fields, reject excluded sources, compare CLI/core
bytes and readonly/no-op behavior, and preserve old output on source failure.
Bounded fuzz checks selected-source admission and determinism. Other platforms
and broader media/nominal roles require independent evidence.

**Migration:** previously rejected eligible UUID/temporal forms now generate
with their existing Kotlin types and canonical mapped text. Existing accepted
forms, JSON/raw API/wire and manifest/name-lock formats retain their contracts.
The earlier UUID String-to-Uuid migration remains documented in the
[UUID contract](nullable-uuid.md); this form role does not redefine it.
