# readOnly request presence

`readOnly: true` on an ordinary property identifies a value produced by the server. An inline request DTO uses `Patch<T> = Patch.Absent`, even if the schema marks the field required, so callers omit it by default and can explicitly override it.

```json
{
  "type": "object",
  "required": ["id", "name"],
  "properties": {
    "id": {"type": "string", "readOnly": true},
    "name": {"type": "string"}
  }
}
```

```kotlin
public data class SubmitInputDto(
    public val id: org.example.models.Patch<kotlin.String> = org.example.models.Patch.Absent,
    public val name: kotlin.String,
)
```

Canonical shared components retain their original constructor, requiredness and serializers. HTTP bodies referencing ordinary DTOs with read-only fields now obtain a separate [directional request DTO](directional-models.md), including affected nested/collection references and nominal recursion. The original shared type remains available; callers of affected HTTP methods migrate to the operation-specific Request type. Unannotated references remain shared.

| Shape | Result |
|---|---|
| Inline ordinary request field | Existing Input DTO uses Patch.Absent |
| Referenced ordinary request DTO needing projection | Separate Request DTO uses Patch.Absent |
| Canonical shared component / read-only response field | Original field and requiredness |
| `readOnly: false` | Equivalent to omission |
| Non-boolean `readOnly` | `E_SCHEMA_INVALID` at `…/readOnly` |
| Both `readOnly` and `writeOnly` true | `E_SCHEMA_INVALID` at `…/writeOnly` |

`writeOnly` is supported under the separate [directional DTO contract](directional-models.md): requests retain the field, while dedicated response DTOs omit it. Composition, whole-object counts, projected reference defaults and raw HTTP representations have explicit rejection boundaries in that contract.

[Core read-only contracts](../core/read_only_request_presence_test.go) retain the original inline and canonical-component behavior. [Directional core/native/Kotlin contracts](directional-models.md#verification) cover the added HTTP reference projection. The reviewed `readonly-annotation` and `writeonly-annotation` corpus snapshots record canonical shared-model output.
