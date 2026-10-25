# Raw HTTP success response media

The HTTP client accepts two additional success-response media profiles:

| Selected media | Required resolved operation type | Wire behavior |
|---|---|---|
| `text/plain` | builtin `String` or `String?` | Ktor's raw text reader; preserves quotes, whitespace and the response charset |
| `application/octet-stream` | builtin `ByteArray` or `ByteArray?` | Ktor's raw byte reader; no JSON/Base64 conversion |

Direct schemas and existing references use the shared operation resolver, original source diagnostics and validation rules. A named String enum or value class remains a nominal codec and is not a raw builtin String. Inline scalar enum/const approximations retain their existing warnings; this profile does not add membership enforcement. Unsupported resolved types fail at the selected media's original `schema` Pointer with `E_OPERATIONS_UNSUPPORTED`.

Media selection keeps the existing JSON preference and then SSE. If neither exists, `text/plain` precedes `application/octet-stream`. Within each new media kind, exact spelling wins before case/parameter variants; equal-rank candidates use stable media-key ordering. Success status ordering is unchanged: the first body-bearing preferred success wins over an earlier bodyless status. `default` never becomes a success response.

Every declared schema is still validated, including unselected statuses/media and excluded operations. The new builtin-type restriction applies only when the new media is selected; existing JSON-preferred documents retain their previous treatment of other valid media schemas. Request bodies use their independent [form](form-request-media.md) and [raw scalar](raw-request-media.md) profiles; declared-error media selection keeps its existing policy.

Nullable raw responses follow the already established Ktor contract: a body containing ASCII `null` returns the String `"null"` or those four bytes, rather than JSON null. An empty text body returns `""`. The generator installs no plugin, changes no Accept header, and does not add runtime content negotiation. The server and caller must agree on the selected representation. JSON bodies still require the caller's configured ContentNegotiation codec.

Opt-in validation checks retained constraints on the resolved raw value after Ktor reads it. Standalone aliases remain unchecked. This adds support for previously rejected raw-only success documents without changing existing JSON/SSE artifacts, manifest or lock formats.

[Core contracts](../core/raw_response_media_test.go), [native parity](../internal/cli/raw_response_media_process_test.go), [synthetic fixture](../tests/fixtures/http/raw-response-media.json) and [fresh Kotlin wire consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawResponseMediaContracts.kt) cover text/charset/empty/bytes/nullable/status/media priorities and retained validation. Run `bash scripts/test.sh feature raw-response-media` for Data/Plain × validation off/on and four Json configurations. The consumer is also included in the full HTTP gate. These JVM checks do not establish other Ktor engines or Android/KMP support.
