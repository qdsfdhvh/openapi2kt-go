# Raw JSON carrier field defaults

A field whose shared plan produces a raw JSON carrier may declare its own `default`. Two carriers are admitted:

- a `JsonObject` from an open dictionary (`{"type":"object","additionalProperties":true}`), which requires a JSON-object default;
- a `JsonElement` from an unconstrained schema (`{}`), an empty object (`{"type":"object"}`) or a request-context union projection, which accepts any JSON value.

```json
{
  "open":  {"type": "object", "additionalProperties": true, "default": {"a": 1}},
  "free":  {"default": [1, 2, 3]},
  "empty": {"type": "object", "default": {"x": "y"}}
}
```

Optional fields get typed initializers decoded by a file-level helper (`Json.parseToJsonElement` for `JsonElement`, plus an `as JsonObject` cast for `JsonObject`), so public qualifiers cannot be shadowed. Missing JSON uses the default, a supplied value replaces it and explicit null remains null. Required fields validate the annotation and keep their constructor argument; optional explicitly nullable inline request fields keep `Patch.Absent`.

## Admission and source fidelity

Only the consuming field's own `default` is granted, and only for the carriers above. References, composition, typed dictionaries (`additionalProperties` as a schema), arrays, enums, consts, formats and native/value-class markers keep their separate roles. The `JsonObject` default must be a JSON object; a wrong kind or a null on a non-nullable carrier is rejected at the original `/default` Pointer.

The planner borrows the original schema and value nodes. The canonical JSON is embedded as a literal and parsed at each evaluation, so optimization/hoisting does not need to relocate a classifier. Structured `const` values keep their existing separate helpers.

## Validation and resources

The existing input8MiB, source48, checked-collection16 and 240 serialized constructor-slot limits are unchanged. `JsonElement`/`JsonObject` carriers keep their existing approximation warnings (`W_JSON_ELEMENT_APPROXIMATION` for empty objects, `W_HTTP_FIELD_UNION_PROJECTION` for request-context unions); this feature adds no membership or constraint enforcement.

## Compatibility and verification

These raw JSON field defaults previously failed with `E_SCHEMA_UNSUPPORTED`. Existing no-default carriers, typed dictionaries and structured const values retain their contracts and emitted bytes. This closes the raw `JsonElement`/`JsonObject` carrier subitem of D03; typed dictionaries, DTO/union/hierarchy carriers and root/component defaults remain separate.

The [fixture](../tests/fixtures/http/raw-json-field-defaults.json), [core typed and guard tests](../core/raw_json_defaults_test.go) and feature `raw-json-field-defaults` define the increment. `bash scripts/test.sh feature raw-json-field-defaults` freshly generates Data/Plain × validation off/on projects and checks constructor/serde, four Json configurations, override/null, controlled HTTP and fragmented live SSE, plus a combined optimized/locked package. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected raw JSON carrier field defaults now generate. General D03/D04, wider platforms and release gates remain open.
