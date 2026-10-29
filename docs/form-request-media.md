# Form URL encoded requests

When a request declares no JSON media, `application/x-www-form-urlencoded` can
select an inline ordinary object as its required or optional request DTO. Exact lowercase media
spelling wins over case or parameter variants; ties use lexical media order.
Existing `application/json` and `application/*+json` selection keeps precedence.
Media parameters do not configure another charset: form output uses Ktor's UTF-8
form encoder and canonical form content type.

The selected schema must explicitly have `type: object`, a nonempty `properties`
object. Individual fields and the body itself may be required or optional.
Root keys are limited to `type`, `properties`, `required`, `title` and
`description`. Each property must resolve directly to an ordinary non-null
`String`, `Boolean`, `Int`, `Long`, `Float` or `Double` source.
Direct Uuid/LocalDate/Instant fields follow the separate
[UUID/temporal form contract](standard-form-fields.md). Required fields keep non-null
constructor arguments; optional fields use nullable arguments defaulting to null
solely for an absent form key. The field grammar
permits `type`, `format`, `title`, `description`, builtin string length/pattern
checks and numeric bounds/multipleOf already supported by the shared planner.
Direct plain String vocabularies and native scalar enums have the separate
[enum form contract](enum-form-fields.md), retaining their shared mappings and
checks. Constraints retain their existing `--validation` requirement and checks.

Direct field-own defaults use the [form default contract](form-field-defaults.md),
including required annotation validation and optional getter initializers.

References, aliases, const annotations, schema-nullable fields,
arrays/maps, structured/composed values, readOnly/writeOnly projections, object
counts and explicit media `encoding` are outside this profile. Unsupported
selected forms fail with source diagnostics and produce no partial output.
Unselected media still undergo the original shared schema validation: a JSON
request may retain an otherwise valid discarded form schema outside this selected
profile, while invalid discarded schemas and unsupported media keywords still
fail. Excluded operations also validate their sources.

The method accepts the actual `<Operation>Input` DTO, including the configured
model suffix and Data/Plain style. Wire keys come from original property names;
Kotlin identifier normalization does not rename form keys. Each field produces
one entry in property order when present. Optional null values omit the key;
empty Strings, false and numeric zero remain present values. Strings pass directly to Ktor `FormDataContent`;
Boolean and numeric values use their builtin textual representation. Floating
values retain the mapped Float/Double value, signed zero and standard exponent
spelling; see [floating form fields](floating-form-fields.md). Ktor encodes
spaces as `+`, literal plus as `%2B`, and reserved characters and Unicode as
percent-encoded UTF-8. Empty strings produce `key=`. No JSON serializer or
ContentNegotiation plugin is needed to encode the form body. Response decoding
retains its existing plugin requirements.

Ordinary HTTP and cold SSE use the same body planner and emitter. The final
`requestBuilder` callback runs after the generated body and parameters, so callers
can override them. Streaming, cancellation, error handling and ownership of the
provided `HttpClient` retain their [existing contracts](http-client.md).

An optional body exposes `<Operation>Input? = null`. An omitted or null argument
leaves request content absent and sets no generated Content-Type. A supplied DTO
keeps every required field and the existing form encoding; empty String fields
still produce `key=`, rather than omitting the body. Schema-nullable roots or
fields and explicit Patch form states remain outside this profile.
The same omission guard applies to HTTP and cold SSE, and the final callback can
still supply its own body. Required forms and JSON-media null behavior are unchanged.

The [synthetic input](../tests/fixtures/http/form-request-media.json),
[core contracts](../core/http_form_request_media_test.go),
[native preservation tests](../internal/cli/form_request_media_process_test.go)
and [Ktor consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FormRequestMediaContracts.kt)
cover this A05/A08/D07 increment. Run `bash scripts/test.sh feature form-request-media`
for fresh Data/Plain × validation projects, baseline and combined
dedup/hoist/merge/lock outputs, a 240-field constructor, four Json configurations,
exact form bytes, JSON priority, namespace collisions, callback overrides and
fragmented SSE close/cancellation/reuse. The complete `http` route includes the
same consumer. These are JVM contracts; other targets require separate evidence.

Previously unsupported eligible forms now generate. Existing accepted JSON
call sites, wire behavior and output/name-lock formats have no migration.
Broader request media, parameter structures and D07 remain open.

Optional-body consumers also cover omitted/explicit null/present DTO, empty String fields, bare HttpClient encoding, callback count/override, validation before dispatch, and live SSE failure/closure/client reuse. Native tests compare core bytes and no-op/readonly output, retain formerly rejected optional-body inputs as positives and preserve old output for invalid source grammars. Bounded fuzz compares optional and required source admission so the new body role cannot authorize an otherwise unsupported form schema. Existing input, source, work, checked and constructor budgets remain.

[Optional flat form fields](optional-form-fields.md) separately verify omitted
keys versus empty/false/zero values, all-optional empty forms, independent body
absence, reserved Kotlin names and optional field checks. The original source
schema stays non-null; optional constructor null is only a form omission marker.
