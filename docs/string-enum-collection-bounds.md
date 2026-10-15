# String enum bounds on direct DTO collections

With `--validation`, ordinary DTO fields whose direct inline List item or typed Map value is a plain String enum accept `minLength`, `maxLength` and the existing portable `pattern`. The leaf may omit `type`, declare String, or use its already supported explicit nullable/null-member grammar. An outer field default is optional.

```json
{
  "type": "array",
  "items": {
    "enum": ["ready", "x"],
    "minLength": 2,
    "maxLength": 8,
    "pattern": "^[a-z]+$"
  },
  "default": ["ready"]
}
```

The generated field remains `List<CodesInner>` (with its context-specific nullable/default view), and a dictionary remains `Map<String, LabelsValue>`. The actual lexical enum owns its entries, `fromSerialName`, `Unrecognized`, String serializer and fully qualified descriptor. No String collection projection or additional serialized state is introduced.

## Open vocabulary and field checks

The entire original vocabulary remains present, including known constants that fail a bound. The example still exposes `X`, but supplying it in the checked DTO collection fails `minLength`. `Unrecognized("future")` succeeds when its actual `serialName` satisfies the same constraints; unknowns do not acquire closed membership checks. A vocabulary whose known entries all fail bounds is still admitted: it remains open to other Strings. Unlike the native numeric enum domain contract, no finite-vocabulary intersection is inferred.

Checks use the [builtin String semantics](model-semantics.md#string-pattern-validation): Unicode codepoint length and portable Regex substring search. Anchors are needed for whole-string matching. Grammar, diagnostics and generation budgets are unchanged: 1,024 UTF-8 pattern bytes, 256 syntax nodes and quantifier bounds up to 1,000. Patterns are not a new dialect or a runtime timeout guarantee.

DTO constructors, data-class `copy` and automatic JSON decoding validate the actual enum payloads. Plain models retain getters/value equality without a copy API. Typed DTO HTTP responses and known SSE payloads use those same constructor checks; invalid fragmented live SSE data terminates and closes the input, while the caller client and a new cold stream remain reusable. Independent enum creation/lookup/serialization remains open and is not a field-validation barrier. Later mutation of caller-owned collections and arbitrary request-builder overrides keep their existing boundaries.

Container counts remain independent. Null containers bypass their own count and member checks; nullable null members bypass only their leaf checks and never access `serialName`. Non-null members still validate. Empty collections pass leaf checks vacuously, subject to their counts.

## Defaults and original sources

Every non-null annotation member must belong to the original vocabulary **and** satisfy the String constraints, even when the same unknown token would be valid at runtime. Default pattern matching retains the existing printable-ASCII profile; Unicode length-only defaults retain codepoint semantics. Default errors point to the original array index or escaped dictionary key. Dictionary validation visits original source order, then sorts successful keys for deterministic emission and identity.

Required annotations validate fully before the initializer is suppressed. Optional explicitly nullable request fields validate fully and retain `Patch.Absent`; Null and Value keep their ordinary presence behavior. Empty/null annotations still validate the entire schema, enum grammar, constraints and type before any early return. No default enables validation implicitly: with validation off, the original unsupported constraint keyword Pointer is retained.

Original discarded allOf fields, fixed extras and raw/reference projections cannot lose checks. This profile can retain a direct field in an already supported ordinary allOf DTO; it adds no collection/enum constraint aggregation across contributions. Effective field checks, defaults, nullable views and actual enum types participate in deduplication, merging, hoisting and name-lock shape identity. Equivalent fields bind final canonical lexical owners; changed checks/defaults cannot reuse a frozen different shape. Lock formats are unchanged.

## Scope and verification

Only ordinary DTO **direct inline** List/typed Map String enum leaves gain bounds. This includes supported shared/nested DTOs, inline request/response DTOs and DTO SSE data. [Direct nested String enum field defaults](nested-string-enum-collection-defaults.md) have a separate outer-field contract; they do not grant these String bounds/patterns, including for empty/null annotations. Enum roots, named/ref leaves, wrappers, nested collection trees, union/hierarchy field roles, formatted/native/value-class/const leaves, numeric bounds and enum `uniqueItems: true` retain their exclusions. Referencing a DTO that contains an admitted field retains that DTO's own constructor; it does not grant checked enum reference leaves.

This additively admits sources previously rejected with `E_SCHEMA_UNSUPPORTED`. Existing accepted unbounded outputs and String ABI/wire behavior remain unchanged. Consumers of new checked inputs should regenerate and compile; invalid field values now throw. This closes only the M03/M04/M08/M12/V05 direct String enum leaf bounds subset and its existing A03/A05/A08 DTO consumers, leaving general D03/D05 and platform support open. Source depth 48, work 16,384, vocabulary/constructor slots 240, checked collection edges 16 and input 8 MiB remain. Existing nested builtin default node/32 KiB budgets retain their independent scope; no caps or JVM heap are raised.

The [synthetic fixture](../tests/fixtures/http/string-enum-collection-bounds.json), [core admission](../core/string_enum_collection_bounds_test.go), [original-source/scope/budget guards](../core/string_enum_collection_bounds_guards_test.go), [actual optimization identities](../core/string_enum_collection_bounds_optimization_test.go), [native failure preservation](../internal/cli/string_enum_collection_bounds_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumCollectionBoundsContracts.kt) define this contract. Historical rejected inline enum-bound cases graduate verbatim; other negative cases and golden artifacts remain.

`bash scripts/test.sh feature string-enum-collection-bounds` builds fresh Data/Plain × validation on/off projects. Validation-off uses an accepted unbounded control and separately asserts rejection of the original bounded source. All four Json default/null configurations exercise constructors, JSON/String/element codecs, required/Patch/null/count behavior, valid unknowns, invalid known/unknown payloads, actual dedup/merge/combined hoist/name-lock owners and typed HTTP/fragmented SSE. The full `http` route generates the bounded fixture and calls this consumer. A separate 240 shallow checked collection-field compilation probe retains existing heap settings; independent probes do not establish Cartesian width/depth, Android/KMP, minimum dependencies or other engines. `bash scripts/kotlin/features/string-enum-collection-bounds.sh --prepare-only` performs native preparation/checks without executing JVM evidence.
