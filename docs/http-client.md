# HTTP client subset

[Direct nested temporal DTO defaults](nested-temporal-collection-defaults.md) use the same actual LocalDate/Instant model codecs through typed request/response/SSE payloads. Missing keys use ordinary defaults, invalid temporal responses fail decoding and per-layer counts retain validation behavior; root raw/scalar HTTP rules remain separate.

Nullable `Int` JSON success responses are decoded with the generated client's configured `Json` codec, avoiding Ktor's default erased-Int transformer parsing JSON `null` as a non-null integer. Non-null Int and raw String/ByteArray behavior are unchanged. Affected clients/groups without SSE expose an additional defaulted `json` constructor argument; existing two-argument Kotlin source calls remain valid, but previously compiled consumers need rebuilding. This is not a binary ABI promise. [Nullable Int regression input](../tests/fixtures/http/nullable-int-responses.json) covers the no-SSE case; intrinsic nullable primitive roots have their separate model contract.

The native Go core and CLI generate a **limited Ktor client** when a JSON document contains supported operations. This is not complete HTTP/OpenAPI support. Model-only inputs retain their existing files; HTTP inputs may omit `components` entirely.

```bash
build/openapi-kt generate --input tests/fixtures/http/basic.json \
  --output build/generated/http --package-name org.example.http --model-name-suffix Dto
```

## Kotlin API

- Components remain in `<package>.models`; API classes are in `<package>.api`; the aggregate client is in `<package>`.
- The first operation tag selects the group; no tags selects `DefaultApi`. Tags that normalize to the same API class are combined, not overwritten. Other case-insensitive filename collisions fail. Aggregate accessors are sorted and lazy.
- The client name comes from `info.title`, removing a trailing `API` or `Api` before appending `ApiClient`; absent/blank titles use `ApiClient`.
- Dotted operation IDs drop their first segment and move a final REST verb to the front (`widgets.item.get` → `getItem`). A single REST verb gains its tag context (`widgets.delete` → `deleteWidgets`). Ordinary identifiers such as `batchGet` and `batch_get` retain spelling. Without an ID, use method plus non-parameter/non-`@` path segments.
- Conflicting method names with the same erased parameter signature receive stable numeric suffixes in path/method order; distinct signatures remain overloads. Naturally existing suffixed names are reserved first. Transparent array/dictionary aliases are compared by their backing type: distinct List/Map alias names or element types cannot distinguish JVM signatures. Nominal DTO/enum/valueclass identities remain distinct. This repairs formerly uncompilable named-dictionary alias overloads; affected later methods acquire suffixes, while exported alias spellings, payloads and valid nominal overloads remain unchanged. See the [alias-erasure fixture](../tests/fixtures/http/alias-method-erasure.json) and [fresh call-site/wire contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AliasMethodErasureContracts.kt).
- Value-expression namespace collisions use private import aliases: parameters named `io`/`kotlinx` keep their Kotlin and wire names, while HttpMethod/ContentType/Flow calls remain callable. SSE event constructors likewise avoid parameters, generated locals and members shadowing the first package segment (`org`, `frame`, `value`, `input`, `requestBuilder`, `client`, `basePath`, `json`). Aggregate API constructors also avoid `httpClient`, `basePath`, `json` and API property names such as `shadows`. Unaffected operations retain their existing representation; no model or method is renamed. [Regression input](../tests/fixtures/http/namespace-collisions.json), [core/native checks](../core/http_namespace_collisions_test.go) and the fresh `http-namespace-collisions` feature verify these cases.
- API names must resolve to safe ASCII identifiers. Unlike model properties, API parameter names use camel-case conversion. Unsupported punctuation/non-ASCII-only names and all-caps spellings that still contain hyphens/spaces are rejected rather than silently renamed.

## Parameters and responses

The [special scalar subset](special-scalars.md) also applies here. Root ByteArray bodies delegate raw bytes to Ktor; ByteArray model fields encode numeric arrays. Direct bigint model fields/Patch values encode decimal strings, while simple root/list/map Long values remain numeric. These are context-specific wire contracts, not Base64 or arbitrary-precision support.

Supported verbs are GET, PUT, POST, DELETE, OPTIONS, HEAD, PATCH and TRACE. Request bodies may be scalars, supported shared component references, or the bounded inline request forms described below.

Path-level parameters merge into operation parameters; an operation entry overrides the same wire name/location. Signatures put path before ordinary query before deep-object filters before header, with operation-local entries before non-overridden inherited entries within each location. Scalar parameters, component string-enum references and the collection forms below are supported. Optional query/header parameters default to null and are omitted. Required parameters have no default. Path parameters must be required and non-null and must match the path template.

Path values are encoded as segments, including `/`, `?`, `#`, spaces and Unicode. Scalar query/header values use native values; open enums use `serialName`, including unknown strings. Header wire names retain their spelling. Parameter allocation reserves `requestBuilder` and disambiguates same-name locations without stealing naturally suffixed names. Scalar `form` query and `simple` path/header styles are accepted.

Arrays of scalars/component or inline string enums use these wire forms, including nullable elements:

| Location/style | Wire form |
|---|---|
| query `form` (default), `explode=true` (default) | repeated `name=value` |
| query `form`, `explode=false` | comma-delimited |
| query `spaceDelimited` / `pipeDelimited` | space / pipe-delimited; defaults to `explode=false` |
| query delimited style with `explode=true` | repeated `name=value` |
| query `deepObject` array | `name[0]=value`, `name[1]=value` |
| path/header `simple` | comma-delimited; path result is then segment-encoded |

Optional collections are nullable with a null default; a required nullable query collection has no default but omits parameters when explicitly passed null. Empty repeat/indexed lists or maps send no entries; empty delimited lists send an empty value. Array enum elements use `serialName` rather than the list/object `toString`. Direct query dictionaries and raw JSON objects support `form + explode=true`: each map key becomes a query name, without an enclosing parameter-name prefix. Ktor omits a null entry or repeat/indexed element; delimited lists instead spell a null element as `null`. Nullable enum elements use safe `serialName` access; joined null enum elements also spell `null`. This fixes nullable-element access without changing non-null enum wire values.

Supported nested containers, named model values, raw parameter unions and structured map values follow Ktor's **one-value `toString()`** behavior, not recursive query or JSON encoding. For example a nested list becomes `[a, b]`, a model uses its class `toString()`, and a `JsonPrimitive` string includes JSON quotes. Map values are forwarded directly: a generated enum's class `toString()` need not equal `serialName`. Non-list header maps likewise use one textual value. `W_HTTP_PARAMETER_PROJECTION` identifies these lossy projections; branch schemas still pass the supported shared subset. Other map/style combinations and composition remain bounded.

Responses may be scalars, nullable scalars, or references to supported component types. A response without content returns `Unit`. Success selection considers 200/201/202/203/206 first, then other explicit 2xx ascending, then 2XX; the first JSON-body candidate wins over a bodyless earlier status. `default` is never selected as success. JSON selection prefers exact `application/json`, then case/parameter variants, then `application/*+json`. Unsupported content is diagnosed, not silently treated as `Unit`.

## Direct true JSON roots

OpenAPI 3.1 literal `true` JSON request/success-response schemas use unconstrained `JsonElement`, including JsonNull. Required bodies take a non-null argument without a default; optional bodies take `JsonElement? = null` according to `requestBody.required`. With the existing tested Ktor codec, a null optional argument sends JSON null; the non-null response returns JsonNull, not Kotlin null or Unit. JSON strings are quoted through the JSON carrier, unlike raw root String bodies.

Borrowed `components.requestBodies`/`components.responses` schemas retain their original target Pointer. Every media/status is still validated before preference/filtering: a discarded false or unsupported schema fails without partial output. JSON/+json preference and success ordering are unchanged. Each analyzed true source produces `W_BOOLEAN_SCHEMA_PROJECTION`; repeated HTTP object-reference uses retain the existing per-use warning policy. No DTO, custom serializer or automatic client plugin is added.

False schemas, 3.0 Boolean values, Boolean parameter/SSE-envelope roots and general Boolean composition remain unsupported. Existing true array elements, SSE data and declared error schemas use their separate value paths; this root increment does not reimplement them. Named true component aliases have the separate [root contract](boolean-roots.md).

[Whole input](../tests/fixtures/http/boolean-http.json), [core contracts](../core/boolean_http_test.go), [discarded-source guards](../core/boolean_http_guards_test.go), [method-slot limits](../core/boolean_http_limits_test.go), [native preservation](../internal/cli/boolean_http_process_test.go) and [generated runtime consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanHttpContracts.kt) exercise this profile. `bash scripts/test.sh feature boolean-http` generates fresh Data/Plain × validation projects and four Json configurations for all JSON kinds, optional null, borrowed roots/priorities, callbacks/errors and fragmented cold POST SSE closure/reuse. Reference-height48/49 and HTTP/SSE240-slot boundaries are independent Go checks, not maximum-width Kotlin/platform guarantees.

## Operation-local types

Inline response objects, direct dictionaries and arrays with inline object/dictionary items generate types in `.api.models`; component references continue to use `.models`. Objects are named `<Operation>Response`, array items `<Operation>Item`, followed by `--model-name-suffix`. Dictionary roots are transparent aliases with package-level value peers. Supported scalar/component collection responses need no extra declaration. Root/container/item nullability is retained.

String enums declared directly on path/query/header parameters or their list items generate `<Operation><Parameter>` types in `.api.models`. Parameter names `Input`, `Response`, `Filter`, `Event`, `Data` and `Item` gain `Param` before the configured suffix to avoid taking an operation model's name. They use the same unknown-value-preserving serializer and `serialName` wire encoding as component enums.

`<Operation>` uses the full declared operationId in Pascal form; without one it uses the final fallback method name. When erased-signature conflicts number a fallback method, its local types are rebound to that name too. Inherited inline parameter enums are bound independently for each operation. Only the selected success/media response contributes declarations, and excluded APIs do not leave local model files behind. Identical declarations with the same identity share one file; conflicting declarations fail explicitly instead of overwriting. This does not enable general shape deduplication or hoisting.

Top-level response enums use the scalar projection described below, not enum declarations. Nullable single-object/ref folds are supported; nominal union responses use the raw-JSON policy described below, while additional composition projections remain diagnosed. Supported nested fields inside a response model use the same model analyzer and serializer contracts as components. Plain inline request models use the separate [request-only presence/Patch contract](request-presence.md); object-union contexts have separate default rules.

## Client options

`--api-client-name ChosenClient` overrides title-derived client naming without changing model/API class names. `--exclude-api WidgetsApi` removes that exact normalized API class and its aggregate accessor; repeat the flag for multiple classes. Exclusions are case-sensitive, duplicates are harmless, and names absent from a document are ignored. All operations are still validated before filtering; exclusions are not a way to suppress unsupported semantics. Excluding every group retains the aggregate client and component models.

Both `generate` and `validate` accept these flags. Core callers use `GenerationOptions.APIClientName` and `ExcludeAPIs`. Explicit names must be safe ASCII Kotlin identifiers, without keyword or reserved-filesystem names. Model-only inputs remain unchanged by these options.

## Local HTTP references

Parameter, request-body and response objects may use local `$ref` chains. Lookup supports `#/...` JSON Pointers with `~0`/`~1` escaping and canonical array indices. References borrow the original document nodes: errors within a target retain that target's Pointer, while missing references identify the `$ref` use site. Cycles and chains longer than 48 references fail explicitly. Duplicate parameter uses identify the second occurrence, not the shared definition.

`components.parameters`, `components.requestBodies` and `components.responses` are accepted in HTTP documents. Their object slots are checked; target semantics are checked when used by an operation. Every `components.schemas` definition is still validated, including unreferenced schemas. Ordinary component **schema** references retain their existing model/scalar identities. The independent [terminal scalar schema Pointer subset](model-semantics.md#terminal-scalar-schema-pointers) can borrow builtin leaves under actual component schema edges for typed fields/parameters/bodies/responses/SSE; the separate [finite collection schema Pointer profile](model-semantics.md#finite-collection-schema-pointers) adds direct List/typed Map trees with terminal-only internal scalar-ref leaves. These retain existing parameter style/null omission, root collection body/response types, inline request Patch and shared/SSE behavior. Typed collection preflight rejects before callback/engine; nominal DTO JSON counts retain their separate later codec boundary. Neither profile enables HTTP schema target anchors.

Reference wrappers allow descriptive `summary`/`description` annotations, but reject semantic siblings rather than silently discard them. Examples are opaque: `$ref` strings inside `example`/`examples` are never resolved. External references, percent-encoded fragments, noncanonical array indices, path-item references and schema-internal targets outside the bounded terminal scalar/finite collection subsets remain unsupported. No filesystem/network lookup occurs.

Unordered path/status/media inventories are analyzed in stable key order, so their JSON object spelling order does not change warnings or generated files. Parameter arrays retain their call-site ordering.

## Primitive value-class references

The [simple component value-class subset](model-semantics.md#simple-primitive-value-classes) preserves nominal wrapper types in requests/responses. Scalar parameters, supported homogeneous lists/direct query-map values and deep-object scalar fields use `rawValue` rather than the wrapper's class `toString()`. Existing structured/reference-owned collection projections remain bounded. Root String wrapper bodies are JSON-quoted and wrapper responses are decoded through ContentNegotiation; transparent String aliases retain raw Ktor String behavior. Enabling the component flag is an explicit API/wire migration, not a change to existing unflagged inputs. Supported [wrapper bounds](model-semantics.md#primitive-value-class-bounds) run on construction and deserialization, so invalid success-response payloads fail rather than returning unchecked wrappers. Opt-in [DTO field bounds](model-semantics.md#opt-in-dto-field-bounds) also validate inline/named JSON request/response models, including supported String patterns; null/Patch presence remains unchanged. [Primitive alias constraints](model-semantics.md#primitive-alias-validation) propagate to typed arguments and bodies before dispatch, primitive/collection success responses after decoding, and typed SSE event data. Transparent aliases keep the existing Kotlin/wire types, including raw Ktor String bodies/responses. Raw projections that would erase constrained source fields are rejected; caller requestBuilder overrides are not validated. [Direct primitive collection-element bounds](model-semantics.md#primitive-collection-element-validation) also propagate through typed nested list/map arguments, bodies, responses and SSE event data; [typed collection counts](model-semantics.md#typed-collection-counts) and [bounded scalar uniqueness](model-semantics.md#bounded-scalar-uniqueness) use the same consumers; [ordinary DTO JSON property counts](model-semantics.md#ordinary-dto-json-property-counts) have a separate serializer-owned contract; [direct JSON-value uniqueness](json-unique.md) is independently supported; broader typed/projected uniqueness and inherited/reference-composed object counts remain unsupported; ordinary flat-inline counts use the independent nominal JSON codec. [Numeric exclusives/multipleOf](model-semantics.md#exclusive-bounds-and-multipleof) use the same typed consumers; floating multiples use strict mapped-IEEE remainder without a tolerance. Independently created/serialized aliases and collection serializers are inherently unchecked.

## Simple operation-value projections

Root inline string enums in request/response bodies, and enums in simple array values, use `String` rather than a generated enum declaration. [Explicit nullable String null vocabularies](string-enum-null-members.md) preserve this projection. Under the tested Ktor client, String success responses use the raw-text reader: JSON `null` remains the text `"null"`, and JSON-quoted strings retain their quotes; named enum responses still use typed enum codecs. They report `W_ENUM_NATIVE_APPROXIMATION` because membership is not enforced. This differs from model-property and parameter enums, which retain their generated types. Named component references keep their established identities. Ktor's root String reader/writer uses raw text; a JSON string response therefore retains its quotes, while list elements use JSON decoding.

The same operation-only scalar rule applies after a supported nullable-enum or known-vocabulary-plus-free-string union fold, including values below simple arrays and dictionaries. It preserves nullable element/value types without creating enum DTOs. Original branch diagnostics, approximation warnings and overlapping-null `oneOf` warnings remain. Component references retain named enum identities; DTO properties and parameter enums keep their own enum contracts.

A response containing one inline object plus a null branch gets a nullable `<Operation>Response` model; one component reference plus null retains a nullable component type. Unions that retain multiple non-null branches return raw `JsonElement`, not a sealed response DTO. A null branch in such a union is represented by `JsonNull` inside the non-null `JsonElement` type. Supported uniform primitive/open-enum normalization happens first. Branch schemas are still validated against the supported schema subset, and `W_HTTP_RESPONSE_PROJECTION` makes the lack of union membership validation explicit. Model and request-body union policies are unchanged.

An explicit `type: null` success response returns `Unit`, including the admitted [explicit null enum/const profile](explicit-null-literals.md) after source grammar validation. Named typed null aliases retain their JsonNull codec. A preferred success status with JSON media remains preferred even when its schema is void; for example, a void JSON 200 can precede a typed 201. Ktor's Unit reader does not validate that the body contains JSON null. Literal true JSON roots have the separate profile above; false/3.0 Boolean and direct `not` schemas, additional composition and structural projections remain gated.

### Empty branches in response unions

A bounded response-only compatibility projection recognizes empty schema objects (including descriptive metadata) and exact `not: {}` **union branches**. If every branch is empty, the method returns `Unit`. One component reference or one nonempty inline object beside empty branches becomes a nullable component or `<Operation>Response` type. Scalar branches and unions retaining multiple real branches fall back to raw `JsonElement`. An explicit `type: object` is not counted as void, even when it has no properties.

`W_HTTP_RESPONSE_PROJECTION` identifies each projected empty branch. This is deliberately not schema validation: `{}` admits any value, while `not: {}` is unsatisfiable; treating them as empty payload information does not preserve those constraints. A narrowed nullable DTO can still reject an unknown object or a non-object payload. Unit reading ignores the body, and a JSON-bearing preferred 200 remains preferred over a typed 201.

Only this response-union context recognizes exact `not: {}`. Direct `not` schemas, nonempty `not` schemas, Boolean union branches and unknown semantic keywords remain diagnosed; direct true JSON roots are a separate context. Model and request unions are unchanged, and borrowed object/ref diagnostics keep their original branch Pointers.

### Structural values below simple containers

A root request array of inline objects, or an inline object below an additional array layer in a response, uses `Map<String, JsonElement?>` rather than a new DTO. Map entries preserve caller-supplied keys and explicit null values, including with `explicitNulls = false`; no request-only Patch or field defaults are introduced. Dictionaries recurse through their value types; named component references remain named. Open dictionaries use `JsonObject`, while empty plain objects keep the established `JsonElement` approximation.

`W_HTTP_VALUE_PROJECTION` records that fixed fields in these structural values are not typed DTO properties and their membership/defaults are not enforced. Schema validation still runs against the supported subset. This does not change ordinary model properties, inline request objects, or direct inline-object response array items, which retain their named-model contracts.

## Deep-object query filters

A `query` parameter with `style: deepObject` and a plain object schema with known properties becomes `<Operation>Filter` plus the model suffix in `.api.models`. Scalar fields and component/inline string enums expand to `name[property]` query keys; enums send `serialName`. Constants use their model getters. Supported nested objects, collections and raw union fields each send one `name[property]` value through Ktor's `toString()`, **not** recursive bracket expansion. `W_HTTP_PARAMETER_PROJECTION` identifies this approximation. Inline primitive/mixed filter unions use `JsonElement` without payload wrappers (`W_HTTP_FILTER_UNION_PROJECTION`); null nodes can send literal `null`, while Kotlin null members are omitted. Root component-reference filter conversion and composition projections remain outside this Filter contract.

Filter models retain ordinary model defaults, **not** request-only Patch semantics. A required nullable property can default to null. Null filters and null members send no query key, while empty strings, false and zero remain values. The parameter's `required` flag determines the filter argument's nullability, independently of schema-root nullability. Positional arguments are path, ordinary query, deep-object filters, headers, then input and callback. Inherited and fallback-renamed operations bind their own filter identities. Multiple different filter shapes cannot overwrite the same operation's `Filter` model.

Generated accesses use the actual model property spelling (including underscores and escaped keywords) and null-safe enum access. This corrects formerly uncompilable accesses for such fields and required nullable enums; valid non-null values retain their wire spelling, while null values are omitted. No new serializer or client plugin is required for query expansion.

## Request bodies

Plain inline request objects and dictionary value objects now use [request-only presence/Patch](request-presence.md). Direct scalar/component lists and dictionary aliases are also supported. Shared component references do not acquire request-only field defaults.

A supported JSON request body adds `input` after the ordinary parameters, before `requestBuilder`. Required bodies have no default; optional bodies use a nullable type with `= null`. A spec parameter named `input` is disambiguated only when the method actually has a body. Body parameters count toward the JVM slot limit and overload signature planning.

Generated requests set `ContentType.Application.Json` and call `setBody(input)`, then emit other parameters, then run the caller's callback. A `+json` schema media type still uses this JSON request content type. Shared component models keep their ordinary defaults and are **not** rewritten as request-only/Patch types.

The caller's Ktor serialization configuration controls the actual encoding. With the tested Ktor 3.5.2 ContentNegotiation setup:

- Optional body omission (the Kotlin default null) sends JSON `null`, not an absent HTTP body.
- Numeric bodies serialize as JSON numbers; component bodies use the configured `encodeDefaults` / `explicitNulls` behavior.
- Ktor ignores `String` in ContentNegotiation by default, so `setBody(String)` sends raw text, **not JSON-quoted string text**. Customize the client or final request callback if quoted JSON string encoding is required.
- The final callback may replace the body. The generator does not serialize it earlier and prevent that override.

These delegated-Ktor behaviors are explicitly tested, not a promise that every schema's presence semantics are fully represented. Plain inline presence/Patch and the bounded raw `JsonElement` request-field union projection follow the separate [request-only contract](request-presence.md); other request contexts remain bounded.

### Flat inline allOf primitive bounds

With `--validation`, supported nominal ordinary DTO consumers share [flat inline same-property primitive conjunction](model-semantics.md#flat-inline-allof-primitive-bounds). Component bodies/responses and nested ordinary inline models retain their winning type/presence/wire API. Invalid construction or automatic decode fails with the existing safe primitive cause; invalid typed construction does not reach the callback or engine. JSON-owned object-count serialization still has its separate callback-before-encoding policy. No HTTP-specific validator, hidden codec or root inline allOf body support is added; raw overrides remain caller-owned.

### Ordinary DTO JSON property counts

[Direct explicitly nullable component DTO root own counts](direct-nullable-component-counts.md) use this same nominal JSON serializer and caller configuration with validation. Nullable aliases/references and false-narrowed views keep the original physical class and presence; root null skips counts through the existing nullable adapter. Nullable allOf/ref-wrapped/member-inferred component count roles remain excluded.

With `--validation`, supported direct or [flat inline](model-semantics.md#flat-inline-allof-dto-property-counts) ordinary non-inherited DTOs with constructor fields enforce [JSON wire-property counts](model-semantics.md#ordinary-dto-json-property-counts) in their nominal serializer. The HTTP ContentNegotiation Json controls actual outgoing members. Unlike existing typed preflight checks, object counts run during serialization: the final callback may already execute, but invalid encoding prevents engine dispatch. This does not validate arbitrary callback body replacements. Constructors/copy are not wire-count barriers, and no HTTP-specific counter or hidden Json is installed.

Incoming success DTOs count the raw parsed object before defaults or unknown handling. Missing const/default keys are not retroactively counted; unknown/null members count once, independently of what remains in the DTO. Valid bodies retain exact delegated codec bytes under all four default/null configurations. This opt-in accepts formerly rejected ordinary count schemas without changing unconstrained call sites; union/inherited/reference-composed/empty/const-only and raw/stringification projections remain fail-closed. [Fresh generated contracts](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOPropertyCountContracts.kt) check legal bodies, callback-before-count-error, no dispatch and response checks before defaults. The [flat-inline consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AllOfDTOPropertyCountContracts.kt) additionally verifies nested Patch legal two-member output versus a leaf-valid third member rejection. Inline HTTP root allOf is not enabled.

### Root request unions with primitive or array branches

A remaining root inline request union generates a nominal `<Operation>Input` sealed class in `.api.models`. In this **root request context**, a primitive or array branch becomes an empty `data object VariantN`, not a native payload wrapper. It is unreachable during decoding and encodes as `{}`, even if its schema declares a scalar or array payload. `W_HTTP_REQUEST_UNION_PROJECTION` and `W_UNION_UNREACHABLE_VARIANT` report this lossy compatibility mapping. Primitive enum/const pins are validated but do not add fields or enforce membership in these empty variants.

**Use the generated `Unknown<Operation>Input(rawElement)` peer to send an actual primitive, array or raw object payload.** Its serializer preserves the complete JSON value, including `JsonNull`. If no object tag/required-key checks exist, decoding any JSON value returns Unknown. If such checks exist, decoding first requires a JSON object: non-object values fail rather than falling back to Unknown, although explicit Unknown values can still encode them. An empty variant therefore need not decode back to that variant.

Object branches retain ordinary root-union defaults and no Patch, including when borrowed from a component. Only `requestBody.required` controls the method argument's nullability; nullable nominal unions do not gain a null subtype. Nested request-property unions and component primitive unions retain their separate contracts. Branch schemas, array items, references, limits and excluded operations are validated before projection.

## Declared errors

Operation KDoc lists numeric 4xx/5xx statuses in ascending order, then ranges, then `default`. Component schema references retain their model types. Selected direct ordinary inline JSON error objects generate [status-named local DTOs](inline-errors.md) and participate in ordinary model deduplication/hoisting; other inline error schemas retain the validated projection. Non-JSON or unspecified schemas are identified as having no JSON body schema. A `default` response never changes the success return type.

When an included operation declares a JSON error schema, one `api/ErrorBody.kt` adds `suspend inline fun <reified T> ResponseException.errorBody(): T?`. It uses Ktor's `response.body<T>()`, returns null on ordinary decoding exceptions, and rethrows cancellation. It needs a readable body and the caller's serializers; HTML, malformed JSON and missing required fields are not mistaken for a valid DTO. Ktor's default saved-response behavior permits decoding the error again. String readers retain Ktor's raw-text behavior, including JSON quotes.

The helper does **not** install validation, throw on declared statuses, wrap results or select an error type automatically. Configure `expectSuccess = true` when exceptions are desired, then import the generated `api.errorBody` extension and select a type from the KDoc (or a caller-provided type/`JsonElement`). Excluding the only API with error schemas removes the manifest-owned helper; bodyless-error-only clients need no helper.

## SSE responses

HTTP 200 `text/event-stream` supports typed cold Flow via Ktor `client.sse`, shared parameter/body planning, Last-Event-ID and an injectable Json payload decoder. JSON media wins when both are declared. See the separate [SSE contract](sse.md) for envelope shapes, streaming/cancellation and backend limits. SSE methods open a session only on collection and are not suspend functions.

## Caller-owned HTTP configuration

The caller provides `HttpClient` and any ContentNegotiation/authentication/default-request configuration. Generated code installs no plugins, performs no implicit downloads, and never closes the caller's client. Ktor's own error policy applies; no `Result` wrapper is introduced.

The default `basePath` resolves the first server and its declared variable defaults. Operation servers take precedence over Path Item servers, which take precedence over document servers; explicit non-null constructor prefixes override every scope. See [server defaults and precedence](http-servers.md) for nullable constructor behavior, empty arrays, shared helpers and the expansion budget. URL normalization preserves the existing contract: leading slashes are stripped, and a nonempty prefix gains `/` even when it already ends in `/`. Relative prefixes still require caller configuration supplying the host.

Each suspend method takes an optional final `HttpRequestBuilder.() -> Unit` callback, executed after generated parameters. It can replace headers, method or URL. Ktor 3.5.2's `url("...")` retains previously added query parameters: call `url.parameters.clear()` explicitly when a full query replacement is required.

## Deliberate limits

Request union branches outside the shared schema subset, common abstract union-property projections and additional request composition projections, remaining false/general Boolean-composition response projections beyond the direct true/empty-union profiles, parameter/filter shapes outside the shared subset or accepted style combinations, value classes beyond the simple component subset, error types beyond the direct ordinary inline JSON DTO profile, SSE shapes outside the explicit subset, security declarations and runtime server-variable selection remain unsupported. These require separate contracts, not silent omission. Model constraints and formats retain their existing support limits.

Unknown HTTP semantic keywords fail with `E_OPERATIONS_UNSUPPORTED`; malformed slots use structured schema diagnostics. Every declared component schema is still validated. Response schemas/media entries are validated even when another success is preferred. Nonempty `paths` containing no operations still fail rather than generating a misleading empty client. The 8 MiB input limit applies; HTTP planning additionally caps operations at 16,384, parameter lists at 240 entries and method parameter slots at 240 (non-null Long/Double use two).

## Executed verification surface

- `core/http_contracts_test.go`, `core/http_collections_test.go`, `core/http_bodies_test.go`, `core/http_references_test.go`, `core/http_options_test.go`, `core/http_operation_model*_test.go` and `core/http_request_presence*_test.go`: public generation, method/group/parameter collisions, ordering, media priorities, source pointers, all-or-nothing errors and JVM slot limits.
- `core/http_request_kind_test.go` and `core/http_request_field_union*_test.go`: separate root singleton and request-only field raw projections, source warnings, fallback ownership, exclusion, branch/depth/slot limits and fail-closed validation.
- `internal/naming/operations_test.go`: method and parameter spelling rules.
- `internal/cli/http_process_test.go`: actual CLI generation/validation/check/no-op and old-output preservation after HTTP rejection.
- `bash scripts/test.sh http`: fresh native CLI output, Kotlin compilation and Ktor MockEngine URL/header/response assertions, caller overrides/client ownership, plus generated 120-Long method and conflicting-method compilation.

The JVM smoke uses Kotlin 2.4.10, serialization 1.11.0, Ktor 3.5.2, Gradle 9.3.1 and JDK 21 (JVM 17 bytecode). The separate streaming adapter now exercises fragmented SSE input and cancellation; it does not establish Android/KMP, all HTTP verbs' wire behavior, or a minimum supported dependency version. Existing model corpus and serializer runtime suites remain separate regression gates.
