# readOnly request presence

`readOnly: true` on a property means the server owns that value. A request must therefore not send it unless the caller explicitly asks to.

```json
"/r": {"post": {
  "requestBody": {"content": {"application/json": {"schema": {
    "type": "object", "required": ["id", "name"],
    "properties": {"id": {"type": "string", "readOnly": true}, "name": {"type": "string"}}
  }}}},
  "responses": {"204": {}}
}}
```
```kotlin
public data class SubmitInputDto(
    public val id: demo.models.Patch<kotlin.String> = demo.models.Patch.Absent,
    public val name: kotlin.String,
)
```

`Patch.Absent` is the point: the field is **omitted** by default instead of silently included, while staying overridable when a caller really must send it. `id` keeps that default even though the schema marks it `required`, because `required` describes the response, not the request.

## Where it applies, and where it cannot

The projection needs a **dedicated request model**. That exists for an inline request body, which the generator emits as its own `…InputDto`.

A request body that is a `$ref` reuses the **component model**, which is shared with responses. There is no separate request type to project into, so `readOnly` is accepted and **inert**: the field keeps its declared requiredness.

| Shape | Result |
|---|---|
| inline request body | `Patch<T> = Patch.Absent` |
| `$ref` request body / response component | accepted, inert; requiredness unchanged |
| `readOnly: false` | no-op, identical to omission |
| non-boolean `readOnly` | `E_SCHEMA_INVALID` at `…/readOnly` |

The inert row is a real limitation, not a hidden one: projecting it would require splitting every `$ref`-referenced request body into its own request model, which is a separate change.

`writeOnly` stays **rejected**. Honouring it needs the mirror projection — dropping the field from a *response* model — and responses currently reuse the same component model, so there is nowhere to apply it.

## Verification

- [Core contract](../core/read_only_request_presence_test.go): request projection across Data/Plain × validation, inertness on shared components, `false` as a no-op, non-boolean rejection.
- [Corpus fixture](../tests/fixtures/p1/valid/readonly-annotation.json) with a reviewed snapshot records the inert shared-component case, replacing the former `unsupported-readonly` guard.
- `writeOnly` keeps its rejection fixture, so the boundary between the two is executable rather than prose only.
