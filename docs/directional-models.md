# Directional request and response DTOs

Ordinary DTO properties accept boolean `readOnly` and `writeOnly` annotations. Both cannot be `true` on the same schema: that reports `E_SCHEMA_INVALID` at its original `writeOnly` Pointer. `false` is equivalent to omission. Examples and default JSON are data, not schema annotations.

The generator preserves canonical component models and creates operation-owned DTOs when a body needs a different field view. This applies to ordinary inline objects and existing named/internal ordinary DTO references, including finite List/typed Map paths and nominal self/mutual recursion. It does not add new reference admission.

| Use | Behavior |
|---|---|
| Canonical component model | Original fields, constructor, defaults and codec remain available, including annotated fields |
| Request containing an ordinary `readOnly` field | An operation-specific `<Operation>Request<Suffix>` uses `Patch<T> = Patch.Absent` for that field, including schema-required fields; callers can explicitly send a value |
| Request `writeOnly` field | Remains writable; required fields remain required |
| Response containing an ordinary `writeOnly` field | An operation-specific `<Operation>PublicResponse<Suffix>` omits that field from the public properties and serializer descriptor, including schema-required fields |
| Response `readOnly` field | Retains its original response type and requiredness |
| Response with every field omitted | A serializable empty class, never a raw `JsonElement` alias |

Existing inline request `readOnly` fields keep their original Input/Patch behavior. Additional affected models receive deterministic numbered peers under `.api.models`. Recursive edges point to the corresponding directional DTO. Unchanged enum and other nominal codecs keep their original canonical owner. Canonical inline declarations may remain as the owners of those nested classifiers.

A request clone also uses the ordinary request presence table: required nullable fields have no default, optional explicitly nullable fields use Patch, and ordinary optional fields retain their supported defaults. References that need no directional change continue to use their shared models. Each response clone retains the response presence rules. Typed error-body declarations and SSE data use the same response projection; status dispatch and SSE reconnect remain caller-owned.

## JSON and validation

Projection runs after complete source/default/reference validation. A hidden field with an unsupported type, malformed annotation or invalid default still fails at its original Pointer. Existing source depth, checked collection edges and constructor limits remain. An individual fallback annotation scan and the total additional DTO declarations are each bounded at 16,384; TypeRef nullable/alias wrappers do not lower the original schema-depth budget. Nominal recursion is memoized rather than expanded.

Response DTOs follow the caller's `Json` unknown-key policy. A response without the write-only key decodes normally even when that key was schema-required. A server that sends it causes the normal unknown-key error with strict JSON; `ignoreUnknownKeys=true` discards it. Encoding the typed response cannot reproduce the discarded property. This does not promise removal from caller-supplied raw JSON or from canonical shared component values.

`--validation` keeps the original scalar and collection checks on retained fields. Data/plain styles and existing opt-in dedup/hoist/merge are applied after projection. Empty directional DTOs are kept as classes and are not folded into raw projections.

## Current boundary

This is an ordinary DTO profile. A directional operation requiring an allOf/union/hierarchy projection, whole-object property counts, or a default whose referenced DTO would change is rejected instead of silently losing constraints or forwarding raw JSON. A read-only body constant in a request clone also remains rejected. Ordinary retained constants keep their original value semantics; write-only response constants are omitted. Raw structural operation projections containing write-only fields are rejected. A write-only SSE envelope field is rejected; ordinary typed SSE payload DTOs use response projection.

Annotations on canonical models alone are metadata; standalone serializers of those models retain both directions' fields. No global serializer mode or automatic application authentication is introduced.

## Compatibility

`writeOnly` inputs were previously rejected and are now accepted within this profile. Existing unannotated and `false`-annotated generation retains its output. Shared component call sites remain valid. HTTP requests through ordinary component references with `readOnly:true` intentionally change their body parameter from the canonical component to the generated request DTO, so callers must construct that operation DTO. Previously inert `readOnly` annotations on such requests now select Patch omission; inline request behavior is unchanged. Annotated unsupported composition/count/default roles fail with a source diagnostic rather than keeping an inert HTTP view.

The former `unsupported-writeonly` corpus case is now the reviewed `writeonly-annotation` positive fixture; the original input content is retained. This newly accepted case is excluded from evidence about unchanged legacy goldens.

## Verification

- [Core contracts](../core/directional_models_test.go): shared/directional types, recursion/collections, deterministic optimization, false/no-op annotations, opaque examples and source diagnostics.
- [Native contracts](../internal/cli/directional_models_process_test.go): core/CLI parity, check/no-op bytes and mtime, rejection preserves old output.
- [Synthetic input](../tests/fixtures/http/directional-models.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DirectionalModelContracts.kt): fresh source compilation and JSON/HTTP assertions under Data/Plain × validation off/on, four Json configurations and default/all-optimization profiles.
- `bash scripts/test.sh feature directional-models`; the consumer is also included in `bash scripts/test.sh http` and the feature route in `all`. These JVM checks do not establish Android/KMP or minimum dependency versions.
