# Raw scalar request media

Non-null `text/plain` String and `application/octet-stream` ByteArray
request schemas are supported for required and optional request bodies when no JSON or form media is selected. Priority is
JSON, form URL encoding, plain text, then octet stream. Within a raw kind, exact
lowercase spelling wins over case/parameter variants; ties use lexical order.
Every declared media schema still undergoes the shared request validation. Raw
payload type restrictions apply only to the selected raw media, so an existing
JSON/form-preferred request may keep an otherwise valid discarded schema that
does not match the raw wire type. Invalid discarded sources still fail.

The actual planned type must be a builtin non-null String for text or ByteArray
for octets, including existing transparent component/alias mappings. Structural
or nominal enum/model wrappers, scalar-kind mismatches and schema-nullable values
are outside this profile. Existing scalar projections retain
their original approximation warnings; raw media does not add enum membership or
constraint enforcement beyond the shared type's checks. Optional raw requests expose `String? = null` or `ByteArray? = null`. Omitted
and null parameters mean no request content and set no generated Content-Type.
A non-null empty String/ByteArray sends empty content with the selected raw
Content-Type. The String `"null"` remains literal text; schema null does not
acquire a raw wire representation. The same presence policy applies to cold SSE.

The selected raw media must have valid media-type syntax. Its original spelling
and parameters are passed to Ktor `ContentType.parse`, followed by `setBody(input)`
when input is present. Required requests always supply the payload.
String text is not JSON quoted or parsed. Quotes, newlines and the literal string
`"null"` remain text; an empty String sends empty text. Ktor uses the declared
charset or its UTF-8 default for String conversion. Charset availability and
replacement behavior belong to the configured Ktor platform. The JVM contract
exercises UTF-8 and ISO-8859-1. ByteArray sends the exact original bytes, including
zero, signed-byte values and an empty array; it does not become a numeric JSON
array or Base64.

The caller needs no ContentNegotiation plugin to encode these builtin raw
payloads. Response decoding retains its own plugin requirements. Existing
supported scalar validation runs before dispatch and the final callback; callers
can still override the generated body/content type in `requestBuilder`. HTTP and
cold SSE share the same body emitter. Existing response/error selection,
cancellation and caller-owned HttpClient lifecycle retain their contracts.

The [synthetic fixture](../tests/fixtures/http/raw-request-media.json),
[core priorities and boundaries](../core/raw_request_media_test.go),
[native parity/old-output preservation](../internal/cli/raw_request_media_process_test.go),
[bounded fuzz](../core/raw_request_media_fuzz_test.go)
and [Ktor consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawRequestMediaContracts.kt)
define this A05/A08/D07 profile. Run `bash scripts/test.sh feature raw-request-media`
for fresh Data/Plain × validation, baseline/all optimization/lock outputs and four
Json settings, exact text/charset/binary/empty bytes, JSON/form/raw priorities,
namespace collisions, callback overrides, validation preflight and fragmented
SSE error/close/cancellation/reuse. The complete `http` route invokes the same
consumer. Other targets require independent evidence.

**Migration:** formerly unsupported eligible required and optional raw requests now generate.
Previously accepted JSON/form calls, wire output and persisted metadata formats
retain their contracts. Parameter null has the explicit absence convention above; schema-nullable raw
payloads, multipart encoding, automatic error dispatch or broader HTTP media guarantee.

The same fixture also covers optional direct and referenced raw payloads, absent/empty/value distinctions, callback count/override, bare HttpClient encoding and optional validation preflight. SSE checks unwrap Ktor content wrappers to assert the underlying empty-content state, without replacing it by JSON null. Native optional-profile tests retain exact source diagnostics and old output for invalid nullable schemas; bounded reference/nullable fuzz supplements the shared media fuzz.
