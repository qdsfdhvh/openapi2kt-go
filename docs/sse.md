# SSE client subset

[Direct nested temporal DTO defaults](nested-temporal-collection-defaults.md) use ordinary model defaults and actual LocalDate/Instant codecs in data payloads. Invalid known temporal/deep-count payloads fail collection through the existing lifecycle; transport envelopes acquire no default role.

A supported HTTP 200 response with `text/event-stream` generates a typed, **cold** `Flow<OperationEvent>`. SSE methods are ordinary functions, not suspend functions: creating a Flow performs no request; each collection opens its own session. Methods reuse HTTP path/query/header encoding, JSON request bodies and the final request callback, including POST.

The backend is Ktor 3.5.2 `client.sse`. The caller supplies an SSE-capable engine, installs the SSE plugin, and owns the HttpClient. Generated code installs no plugins and never closes that client. ContentNegotiation remains caller-owned for JSON request bodies.

## Schema and generated events

The media schema is an object envelope, a local component reference to one, or a bounded `oneOf`/`anyOf` of envelopes. Envelope properties are limited to `event`, `data` and `id`:

```json
{
  "oneOf": [
    {
      "type": "object",
      "required": ["data"],
      "properties": {
        "event": {"type": "string", "const": "update"},
        "data": {"$ref": "#/components/schemas/Payload"},
        "id": {"type": "string"}
      }
    },
    {
      "type": "object",
      "properties": {
        "event": {"type": "string", "const": "done"}
      }
    }
  ]
}
```

- A declared event uses a nonempty string const. Without an event property the variant is `message`; missing/empty transport event names also mean `message`. Duplicate event names fail generation.
- `<Operation>Event` is a sealed class in `.api.models`, with one data-class variant per event and public `event` / nullable `id` metadata. These transport classes are not JSON envelope serializers.
- `data` uses the shared model resolver: component references retain their model identities; the [terminal scalar schema Pointer subset](model-semantics.md#terminal-scalar-schema-pointers) retains original builtin leaf checks through typed data consumers without internal nominal declarations. [Finite collection reference chains](model-semantics.md#internal-collection-reference-chains) also retain original leaf/count checks and independent inner nullability through typed data consumers, including mixed List/Map hops. Inline object/enum/union and array/dictionary value declarations are operation-local. The event model and decoder use the same resolved type, including container peers. Payloads use ordinary shared-model defaults, not request-only Patch. Ordinary typed payloads with write-only fields use the separate [directional response DTO](directional-models.md); write-only envelope fields remain rejected.
- Required data missing from a frame skips that frame. Optional data is nullable and becomes null when absent. A variant without a data property can emit from an event-only frame. Present data is decoded as JSON, including string payloads: send `data: "text"`, not bare `data: text`.
- Unknown events are ignored without decoding their data. Malformed data for a known event fails collection; it is not silently dropped. Declared metadata identifies transport fields rather than validating JSON-envelope membership or requiring a fresh id on every frame.

The [synthetic fixture](../tests/fixtures/http/sse.json) covers referenced/inline/primitive/union/list/dictionary payloads, optional data, default message events and event-only variants.

## Json, cursor and lifecycle

Only clients/API groups with included SSE operations add a trailing `Json` constructor argument. Its default has `ignoreUnknownKeys = true`. Pass another Json to control payload decoding; this is separate from the HttpClient's ContentNegotiation codec. Non-SSE generated client constructors remain unchanged.

A method has one `lastEventId` header argument. An explicit `Last-Event-ID` header parameter supplies its required/nullability contract; otherwise an optional String argument is added. Duplicate case-insensitive cursor declarations are rejected, and collisions with query/path argument names are disambiguated. A non-null value sends one header before the callback. To override it, remove/set it in the final callback rather than append a second value.

Finishing collection, `take(...)`, downstream cancellation or payload failure ends the session. Cancelling while waiting for more bytes closes the stream input. The caller's client remains usable for subsequent HTTP calls. Cancellation is not converted into a successful payload or swallowed by generated code.

## Ordinary DTO JSON property counts

[Direct explicitly nullable component DTO root own counts](direct-nullable-component-counts.md) also use this nominal data codec with validation. Root null skips counts through the existing nullable adapter, while non-null data retains actual object counts. SSE envelopes keep their non-null policy and acquire no count permission.

With `--validation`, supported direct or [flat-inline](model-semantics.md#flat-inline-allof-dto-property-counts) ordinary non-inherited DTO data payloads share the [nominal JSON count serializer](model-semantics.md#ordinary-dto-json-property-counts), including existing nullable/nested DTOs and typed list/map consumers. Counts apply only to raw JSON data members, never transport `event` / `id`. The API's injectable Json, independently of HTTP ContentNegotiation, supplies unknown/default behavior; counts run before defaults and unknown loss. Constructors are not object wire-count barriers. Envelope classes, union variants, inherited/reference-composed aggregation, empty/const-only and raw/discarded projections do not acquire count support.

A counted known payload failure terminates collection and closes even a still-open fragmented input; subsequent collections and HTTP requests can reuse the caller client. The [fresh generated DTO count consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOPropertyCountContracts.kt) verifies distinct strict SSE Json versus permissive HTTP Json, cold recollection, data-only counts and failure closure/reuse. This newly accepted opt-in subset materializes a JSON AST and makes no stream-location, frame-size or performance guarantee; non-JSON codecs reject and parser/delegate errors are not promised sanitized.

## Flat inline allOf primitive bounds

Opt-in [ordinary-model primitive conjunction](model-semantics.md#flat-inline-allof-primitive-bounds) is enforced by the same nominal payload constructor after decoding with the API's injectable Json. Invalid known live data closes still-open fragmented input; subsequent HTTP calls and cold Flow collections can reuse the caller client. Transport envelopes are not intersection validators, and union/hierarchy/common-field aggregation remain unsupported; flat-inline ordinary object counts independently use the nominal JSON serializer.

## Backend boundaries

- JSON media takes priority when a response declares both JSON and SSE; all media schemas are validated, but only selected media contributes models. Typed SSE schemas are currently accepted only under status `200`, matching the tested backend handshake. A `204` can terminate a Ktor session, but is not a typed stream declaration.
- Framing delegates to Ktor: CRLF and multiline data are accepted, comments/retry-only frames are hidden, ids carry forward and an empty id clears them. An EOF frame without its terminating blank separator is discarded.
- Ktor's default `maxReconnectionAttempts` is zero. The generator provides no automatic reconnect, backoff, retry/id persistence across collections or resume guarantee. Caller-enabled reconnection follows the selected engine/Ktor configuration and is outside this verified subset.
- Envelope nullability, arbitrary event selectors, dynamic event enums, extra envelope fields and unsupported payload schemas fail explicitly. Limits include 240 events, bounded schema depth and the shared method-slot budget, including the cursor argument.
- Failures follow Ktor's SSE exception policy. The ordinary `ResponseException.errorBody` helper does not automatically wrap or decode SSE failures.
- JVM evidence does not establish Android/KMP, other engines or minimum dependency versions. Stream frame size is governed by Ktor/engine behavior; no generated frame-size limit is promised.

## Verification

`core/http_sse_test.go` exercises source diagnostics, selection, ownership, fallback naming and limits. `internal/cli/http_process_test.go` covers fresh generation/check/no-op and failure preservation.

`bash scripts/test.sh http` generates fresh sources and runs [SSEContracts](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/SSEContracts.kt) through a test engine adapter feeding fragmented byte channels into Ktor's actual SSE parser. It checks split UTF-8/CRLF, multiline data, delivery before EOF, cold recollection, typed payloads, POST/header wire behavior, incomplete-frame EOF, strict Json failure and cancellation of still-open inputs. This is streaming evidence, not a preassembled fake event Flow. Wide Flow closure/240-event dispatch compilation supplements the small fixture.
