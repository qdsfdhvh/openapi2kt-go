# Direct inline List/typed Map of ordinary DTO objects with field defaults

[Direct inline ordinary DTO object field defaults](nested-object-defaults.md) own the object leaf directly on an ordinary field. This contract extends the same own-`default` role to a direct inline `array` or typed `Map<String, T>` tree whose element tree contains that same plain object leaf, including longer finite mixed paths such as `List<Map<String, DTO>>`.

```json
{
  "rows": {
    "type": "array",
    "items": {"type": "object", "required": ["name"], "properties": {"name": {"type": "string"}, "age": {"type": "integer", "default": 7}}},
    "default": [{"name": "a"}, {"name": "b", "age": 3}]
  },
  "maps": {
    "type": "object",
    "additionalProperties": {"type": "object", "properties": {"v": {"type": "integer"}}},
    "default": {"k": {"v": 1}}
  }
}
```

[Direct inline allOf constructor elements](allof-object-defaults.md#direct-inline-collection-field-defaults) also accept this outer annotation when the shared plan produces a nonempty ordinary constructor Model and property contributions are disjoint. The source walk grants only that consuming collection's original allOf leaf; [inline collections under object-default properties](allof-object-defaults.md#inline-collection-properties-inside-object-defaults) use their own per-property original-leaf permission. Neither role grants new reference admission. [AllOf element references under inline collection properties](allof-object-defaults.md#referenced-elements-in-collection-properties) separately reuse already admitted named/internal chains and the actual terminal constructor through original-reference permission. Required/Patch/discarded validation, independent container/element nullability and final serializer ownership remain.

The field keeps its actual collection type (`List<RowsInner>?`, `Map<String, MapsValue>?`) and the emitted helper decodes the canonical JSON through the final actual collection serializer, for example `= `openapi-kt model default List org.example.models.HolderDto RowsInner`("[{\"name\":\"a\"},{\"age\":3,\"name\":\"b\"}]")`. The helper binds the final relocated element type after optimization and is deduplicated per file; distinct wrappers (DTO, list of DTO, dictionary of DTO, longer mixed paths, nullable layers) never share one helper. Each evaluation produces a fresh tree, and omitted nested object keys use the target DTO's own property defaults.

## Admission and source fidelity

The role is granted only to the consuming field's own `default`. It is selected when the field schema is a direct inline `array` or typed dictionary whose element tree contains a plain inline object candidate from the [direct object contract](nested-object-defaults.md); scalar-only collections stay with the existing [nested builtin](nested-builtin-collection-defaults.md), [temporal](nested-temporal-collection-defaults.md) and enum collection profiles and keep their emitted helpers unchanged.

Object elements are validated exactly like the direct object leaf: every declared property is checked, unknown keys fail, every `required` wire must be present, property values reuse the finite leaf/container planner (builtin/temporal scalars, String enum membership against the original vocabulary with open runtime `Unrecognized`, native exact membership, nested builtin collections, nested plain-object recursion and per-layer nullability), and a property may carry its own validated `default`. Container counts (`minItems`/`maxItems`/`minProperties`/`maxProperties`/`uniqueItems`) and non-object element leaves keep their existing rules. An element schema cannot own the default annotation itself; container-child defaults remain rejected.

`$ref`, other composition, dictionaries/fixed extras, `const`, object-level `enum`, native/value-class/format markers, empty objects, root/component defaults and referenced named-DTO targets use the separate [referenced object collection contract](referenced-object-collection-defaults.md). Canonical JSON sorts dictionary keys at every level, while validation runs in original member order so diagnostics keep their source precedence. Changing an effective value, a nullable layer or a retained check changes the final shape seen by deduplication, merging, hoisting and name locks; key reordering and equivalent mapped Float/Double spelling preserve identity. Required/Patch annotations that emit no initializer add no runtime default identity.

## Presence and validation

Optional ordinary fields get a typed initializer; a supplied collection replaces the default, omitted object keys inside the default use the element DTO's own defaults, and an explicitly nullable container accepts a literal `null`. Required fields validate the annotation and keep required constructor arguments. Optional explicitly nullable inline request fields retain `Patch.Absent`/`Null`/`Value`, and a missing request key never becomes a default update.

With `--validation`, default values satisfy each retained element/container count and leaf bound, and each element DTO's own `init` checks run when the decoded default is constructed. Null bypasses only its own non-null checks. The existing input8MiB, source48, checked-collection16 and 240 serialized constructor-slot limits are unchanged; the whole annotation reuses the 16,384 value-node and 32 KiB compact/canonical JSON budgets. Exceeded budgets return `E_MODEL_LIMIT` at an original Pointer with no partial artifacts.

The [16,384-step object-source planning budget](nested-object-defaults.md#object-source-planning-budget) is independent of annotation size, including for empty defaults and repeated source references.

## Compatibility and verification

Object leaves inside collections previously failed under the builtin/temporal/object profiles. Existing scalar/temporal/enum collection defaults, direct object defaults and referenced object defaults retain their contracts and emitted bytes. This closes only the direct inline List/typed Map-of-DTO field-default subitem of D03 and extends M01/M03/M04/M07/M08 with the existing M12/V05 and typed A03/A05/A08 consumers; union/hierarchy fields, containers of referenced named DTOs, root/component defaults and general D03/D04 remain open.

The [fixture](../tests/fixtures/http/nested-object-collection-defaults.json), [core typed/presence and graduation tests](../core/nested_object_collection_defaults_test.go), [guards/budgets](../core/nested_object_collection_defaults_guards_test.go), [source/type-plan authority tests](../internal/schema/nested_object_collection_defaults_test.go) and [native parity/preservation/optimization tests](../internal/cli/nested_object_collection_defaults_process_test.go) define the increment. `bash scripts/test.sh feature nested-object-collection-defaults` freshly generates Data/Plain × validation off/on projects and checks constructor/serde/Patch/copy, required presence, four Json configurations, validation membership, controlled HTTP and fragmented live SSE closure/reuse, plus a combined optimized/locked package whose helper still binds its relocated element type. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected object leaves inside direct collection field defaults now generate through this role. General D03/D04, wider platforms and release gates remain open.
