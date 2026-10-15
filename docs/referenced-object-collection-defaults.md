# Referenced ordinary DTO elements in collection field defaults

[Direct inline List/typed Map of ordinary DTO objects](nested-object-collection-defaults.md) own inline object elements. This contract extends collection field own-`default`s to elements that resolve to an already accepted plain ordinary named DTO, in two admitted forms:

- an inline `array`/typed `Map` field whose element is a direct local `$ref` to such a DTO;
- an ordinary DTO field whose direct local `$ref` resolves to a named `array`/typed `Map` component whose elements resolve to such a DTO.

```json
{
  "Item": {"type": "object", "required": ["name"], "properties": {"name": {"type": "string"}, "age": {"type": "integer", "default": 7}}},
  "Rows": {"type": "array", "items": {"$ref": "#/components/schemas/Item"}},
  "Holder": {"properties": {
    "inlineRows": {"type": "array", "items": {"$ref": "#/components/schemas/Item"}, "default": [{"name": "a"}]},
    "namedRows": {"$ref": "#/components/schemas/Rows", "default": [{"name": "d"}]}
  }}
}
```

The field keeps its actual collection type (`List<ItemDto>?`, `RowsDto?`) and the emitted helper decodes the canonical JSON through the final actual serializer, binding the final relocated element/alias type after optimization and deduplicating per wrapper shape. Each evaluation produces a fresh tree, and omitted element object keys use the element DTO's own property defaults.

## Annotation, presence and source fidelity

Only the consuming field's own `default` is accepted. The inline element form is selected by the analyzer-aware candidate only when the direct local `$ref` resolves to a plain object target; a `$ref` to a scalar, enum, composition, dictionary, union or alias chain does not become an object element. The named-collection form is selected by the existing referenced collection role and reuses its existing container/count/nullable handling.

Element objects are validated exactly like the direct object leaf: every declared property is checked, unknown keys fail, every `required` wire must be present, property values reuse the finite leaf/container planner, and a property may carry its own validated `default`. A referenced target must not own a root `default`; target-root defaults are rejected outright. Container counts and non-object element leaves keep their existing rules. Element schemas cannot own the default annotation themselves.

Nullable intent follows the existing reference contracts: nearest explicit consumer/element true or false, target intent and the historical dictionary policy. Optional ordinary fields get a typed initializer; a supplied collection replaces the default; explicitly nullable containers accept a literal `null`. Required fields validate the annotation and keep required constructor arguments. Optional explicitly nullable inline request fields retain `Patch.Absent`/`Null`/`Value`. Canonical JSON sorts dictionary keys at every level while validation keeps original member order and `/default/<index>/<escaped-key>` Pointers.

## Validation and resources

With `--validation`, default values satisfy each retained element/container count and leaf bound, and each element DTO's own `init` checks run when the decoded default is constructed. Null bypasses only its own non-null checks. The existing input8MiB, source48, checked-collection16 and 240 serialized constructor-slot limits are unchanged; the whole annotation reuses the 16,384 value-node and 32 KiB compact/canonical JSON budgets. Exceeded budgets return `E_MODEL_LIMIT` at an original Pointer with no partial artifacts.

## Compatibility and verification

Referenced DTO elements inside collection defaults previously failed (inline elements at their `$ref`, named-collection elements at the target's `properties`). Existing scalar/temporal/enum collection defaults, inline object collection defaults and single referenced object defaults retain their contracts and emitted bytes. This closes the referenced ordinary DTO element subitem of D03 and extends M01/M03/M04/M07/M08 with the existing M12/V05 and typed A03/A05/A08 consumers; target-root defaults, target-internal `$ref` properties, unions/hierarchies, root/component defaults and general D03/D04 remain open.

The [fixture](../tests/fixtures/http/referenced-object-collection-defaults.json), [core typed/presence and graduation tests](../core/referenced_object_collection_defaults_test.go), [guards/budgets](../core/referenced_object_collection_defaults_guards_test.go), [source/candidate authority tests](../internal/schema/referenced_object_collection_defaults_test.go) and [native parity/preservation/optimization tests](../internal/cli/referenced_object_collection_defaults_process_test.go) define the increment. `bash scripts/test.sh feature referenced-object-collection-defaults` freshly generates Data/Plain × validation off/on projects and checks constructor/serde/Patch/copy, required presence, four Json configurations, validation membership, controlled HTTP and fragmented live SSE closure/reuse, plus a combined optimized/locked package whose helper still binds its relocated type. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected referenced DTO elements inside collection field defaults now generate. General D03/D04, wider platforms and release gates remain open.
