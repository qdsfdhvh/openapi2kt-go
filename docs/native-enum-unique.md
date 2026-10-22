# Native integer and Boolean enum list uniqueness

With `--validation`, arrays whose original enum items map to Int, Long or Boolean accept `uniqueItems: true`. Direct items, already admitted named/internal references and their finite aliases, existing single native-scalar-plus-null folds, nullable members and retained nested List/typed Map paths share this profile. The original enum must retain its actual mapped membership plan; a matching scalar spelling or an erased projection is insufficient.

```json
{
  "type": "array",
  "items": {"type": "integer", "enum": [1, 2, 3, 4], "minimum": 2, "multipleOf": 2},
  "uniqueItems": true
}
```

This list accepts `[2,4]`. It rejects `[2,2]` as duplicate, `[6]` as outside the original vocabulary, `[1]` below the minimum and `[3]` for its factor. The vocabulary is not rewritten. Types remain native scalars/typealiases with native numeric/Boolean JSON; no String enum or wrapper API is introduced.

## Equality and retained checks

Int/Long equality is exact over the mapped integer values, including minimum/maximum values. Boolean equality uses true/false. JSON integer `0` and `-0` decode to the same value and collide; one nullable null is allowed and remains distinct from zero/false. Nullable containers skip their own checks. Empty/singleton arrays pass uniqueness unless a separate count bound excludes them.

The existing Comparable helper sorts a copy of non-null values and compares adjacent entries. It does not mutate caller order. Existing per-element membership, numeric bounds and integer factors remain active alongside container counts and uniqueness. Constructors, Data-class copy, automatic DTO decoding and Patch.Value enforce their retained checks. Typed collection HTTP inputs check before the callback/engine, decoded outputs check before returning, and known SSE events check their typed payload. Invalid or cancelled live SSE closes its input; cold collection and the caller client remain reusable.

Independent alias assignment/reified scalar or collection serialization, later caller collection mutation and arbitrary requestBuilder body replacements keep their existing transparency boundaries. These are use-site checks, not intrinsic scalar codecs or general JSON Schema validation.

## Defaults, sources and limits

Already admitted direct/nested/referenced collection default roles and component-own factories check uniqueness as well as strict original vocabulary membership and retained bounds. Integer comparison keys normalize zero spellings without changing original scalar wire types. Invalid annotations report their original `/default/index` path, including nested values. Empty/null annotations and required/Patch initializer suppression retain full original-schema validation. This grants no new default, target annotation, reference or wrapper role.

Named/internal reference admission, actual scalar export and original vocabulary/check authority remain with the existing native enum planner. The new gate requires both that retained membership metadata and an eligible original source. Copied/replaced declarations or source members and changed cached checks retain their rejection. Source depth 48, checked collection edges 16, analysis work 16,384, vocabulary/serialized slots 240, input 8 MiB and existing default value/byte budgets remain unchanged. Effective membership/bounds and uniqueness participate in existing optimization/name-lock identity; no persisted-format migration is needed.

Float/Double enum lists now follow the separate [finite floating uniqueness contract](floating-unique.md), including mapped rounding ties, signed-zero equality and nonfinite rejection; existing membership semantics remain. Const/native overrides/value classes, typed container or DTO elements, general union/allOf aggregation and new reference graphs remain outside this profile. String enum [serialName equality](string-enum-unique.md), temporal [mapped-value equality](temporal-unique.md) and direct [JSON equality](json-unique.md) retain separate contracts. `uniqueItems: false` remains exactly effect-free on otherwise supported arrays with validation enabled; both values require the opt-in.

This additively admits previously rejected D05/M03/M07/M08/M12/V05 inputs and their existing A03/A05/A08 consumers. Previously accepted APIs/wire and validation-off behavior retain their contracts; consumers of the newly admitted sources should regenerate and compile. General uniqueness and cross-platform coverage remain open.

## Verification

The [fixture](../tests/fixtures/http/native-enum-unique.json), [core source/default/budget/identity tests](../core/native_enum_unique_test.go), [actual vocabulary/export authority](../internal/schema/named_native_enums_test.go), [native failure preservation](../internal/cli/native_enum_unique_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumUniqueContracts.kt) define this profile. Original native uniqueness negatives remain verbatim opt-in positives; Float/Double have their separate profile, while const and container-element exclusions remain negative.

`bash scripts/test.sh feature native-enum-unique` builds fresh Data/Plain × validation projects; Plain also enables dedup/hoist/merge/name-lock. Validation-off runs an unconstrained scalar control, removing uniqueness, vocabularies and numeric bounds and replacing the checked-only alias paths with the same direct scalar reference. The original constrained fixture is separately verified to reject without validation, preserving its existing reference diagnostic. Four Json default/null configurations cover native equality/extrema, null, defaults/factories, membership/bounds, nested/reference/Patch, actual request JSON/preflight, HTTP responses and fragmented live SSE failure/cancellation/closure/cold collection/client reuse. The complete `http` route includes the checked consumer. Resource-boundary Go tests are separate from runtime profiles; preparation-only output, combined maxima, Android/KMP and minimum toolchains do not constitute executed JVM evidence.
