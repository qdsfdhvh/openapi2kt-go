# Nullable component root defaults

An explicitly nullable component root may carry its own `default` in the existing component-factory shapes. The generator adds an opt-in factory; consuming fields do not acquire an initializer from the component. A non-null annotation produces that value, and an admitted `default: null` produces Kotlin null.

```json
{
  "Name": {"type": ["string", "null"], "default": "fallback"},
  "MissingName": {"type": ["string", "null"], "default": null},
  "Mode": {"type": ["string", "null"], "enum": ["a", "b"], "default": null}
}
```

`NameDto` and `MissingNameDto` retain their `String?` aliases. Their top-level factories return those aliases using the existing typed JSON helper. The physical String-enum classifier remains non-null, so its null factory has a nullable return type:

```kotlin
public fun `ModeDto default`(): ModeDto? = null
```

This function needs no decode helper. The enum's existing serializer, companion, known vocabulary and runtime `Unrecognized` values retain their behavior. A non-null String-enum factory keeps its existing non-null return type.

## Admitted shapes

The source grammar is the existing component factory grammar, including its scalar format, leaf and validation restrictions:

| Root shape | Non-null annotation | Null annotation | Factory form |
|---|---|---|---|
| Builtin scalar (`string`, `integer`, `number`, `boolean`, including int32/int64/float/double) | accepted | accepted | top-level function on its nullable alias |
| Standard `date` / `date-time` | accepted | accepted | top-level function on its nullable alias |
| `byte` / `binary` | accepted | accepted | top-level function on its nullable alias |
| Finite inline List / typed Map tree | accepted | accepted | top-level function on its nullable alias |
| Plain String enum, including typeless String vocabulary | accepted | accepted | top-level function beside the existing companion |
| Numeric/Boolean native enum | existing validation-on role | existing validation-on role | top-level function on its nullable alias |
| Ordinary object | accepted | accepted | companion `default(): X?` |
| Explicitly nullable disjoint ordinary `allOf` constructor | accepted | accepted | companion `default(): X?` |
| Raw `JsonElement` / `JsonObject` carrier | accepted | accepted | top-level function on its nullable alias |

Explicit nullability comes from the schema, including `nullable: true` or a supported type array containing `null`. An optional consuming property does not authorize annotation null on a non-nullable component.

A null annotation still validates the entire source shape: enum vocabulary/kinds/duplicates, allowed collection leaves, counts and bounds keep their existing admission and diagnostics. Finite inline collection factories admit independently nullable containers and leaves under the [component factory contract](component-default-factories.md). Outer and leaf nullability are independent: `List<String?>` admits null elements but rejects a null collection, while `List<String?>?` admits both. Nested List/typed Map elements follow the [nested component contract](nested-component-defaults.md); eligible scalar references and independent child defaults follow their [separate contract](scalar-reference-collection-child-defaults.md). Other reference/composition elements and child roles retain their restrictions. Whole-object counts on nullable roots, caller-native/value-class/const roots and broader union/hierarchy roles remain separate. Raw carrier factories follow the [raw JSON component contract](raw-json-component-defaults.md): explicit nullable aliases decode their null annotations to Kotlin null through their actual nullable serializer; non-nullable `JsonElement` still retains its separate `JsonNull` value semantics.

An explicitly nullable ordinary `allOf` constructor also accepts `default: null`. Its companion emits `default(): X? = null` directly, without a decode helper. The shared object-default walk still validates every original contribution and child schema before inspecting the annotation value: empty constructors, discarded/repeated fields, additional-property contributions and unsupported child roles remain excluded. Admitted whole-object counts use the counted factory contract below. Required constructor properties and their own defaults retain their existing behavior; null does not construct an instance or bypass source budgets.

Native-enum factories retain the existing factory approximation warning exactly once, including when the annotation is null; a bare checked alias has its own warning behavior. No enum vocabulary entry or implicit nullable intent is introduced by a default.

## Presence, serializers and references

Factories do not change component types, constructors, `copy`, required/Patch classification, descriptors or wire behavior. Nullable aliases use their actual nullable serializer. String-enum and object factories may return null while the physical classifiers and their non-null serializers remain unchanged. Explicit values continue to override missing values, and `encodeDefaults` / `explicitNulls` retain their existing property behavior.

A field referencing a defaulted component keeps its neutral presence and never calls the factory automatically. A document without the annotation emits its previous bytes. Existing non-null factories retain their signatures and output. Reference-wrapper factories retain their [own contract](reference-wrapper-component-defaults.md): finite component alias chains may carry outer/intermediate nullable booleans, with the nearest explicit intent determining factory null permission; admitted intermediate/target defaults each retain their independent owner and factory. The direct-root role does not widen a wrapper's null permission or change already admitted wrapper output.

[Numeric builtin formats](component-default-factories.md#numeric-builtin-formats) use the same explicit/effective nullable rules and mapped scalar checks.

[UUID factories](component-default-factories.md#uuid-roots) also follow the [nullable UUID contract](nullable-uuid.md): intrinsic and own-nullable aliases retain Uuid? intent, independent factory ownership and original source/value checks; UUID parsing and migration apply.

[Constrained unformatted scalar root factories](component-default-factories.md#constrained-plain-scalar-roots) are validation-only and retain the same nullable rules. Null annotations still validate the complete constraint source; malformed bounds report at their actual source keyword.

## Bounds and compatibility

The original input 8 MiB, source depth 48, source work/value nodes 16,384, constructor slots 240 and original/canonical default JSON 32 KiB limits remain. String and checked native enum vocabularies retain their independent 240-member limits even for null annotations.

Previously rejected null defaults on the eligible direct scalar/enum/collection roots, explicitly nullable ordinary `allOf` constructors and direct nullable raw carriers now generate. The newly emitted factory returns the declared nullable value; adopting it remains explicit. Existing ordinary-object null factories, non-null annotations and unannotated declarations keep their behavior. This closes only this bounded D03/M03/M04/M07 subitem; general defaults and platform/release acceptance remain separate.

## Verification

- [Finite nested component factories](nested-component-defaults.md).
- [Nullable collection leaves, independent nullability and original diagnostics](../core/nullable_collection_component_defaults_test.go), plus [source/type/wrapper ownership](../internal/schema/nullable_collection_component_defaults_test.go).
- [Nullable raw alias null factories, guards and non-inheritance](../core/nullable_raw_component_defaults_test.go).
- [Nullable allOf null factories, non-inheritance and source guards](../core/nullable_allof_component_null_defaults_test.go).
- [Core admission and non-inheritance](../core/nullable_component_defaults_test.go), [null roles, exact original rejections, guards, warnings and vocabulary limits](../core/nullable_component_null_defaults_test.go).
- [Original-node and backing checks](../internal/schema/component_null_defaults_test.go) and [existing factory authority](../internal/schema/component_default_factories_test.go).
- [Native parity, readonly check and failure preservation](../internal/cli/nullable_component_defaults_process_test.go).
- [Fixture](../tests/fixtures/http/nullable-component-defaults.json), [non-null consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableComponentDefaultsContracts.kt) and [null consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComponentNullDefaultsContracts.kt).

`bash scripts/test.sh feature nullable-component-defaults` generates fresh Data/Plain × validation-off/on projects under four Json settings, including finite nested factories, nullable collection layers, fresh bytes, wrapper ownership, recursive checks and required/non-inherited fields, preserving the non-null factory checks and adding nullable aliases, enum classifiers/codecs, signed bytes, collection/allOf/null/absence/value and copy assertions. Independent 240-member String-enum and validation-on native-enum probes are included. CI's `profiles` job selects this route; the full `http` registry also invokes the consumers. `OPENAPI_KT_PREPARE_ONLY=1` prepares generation/check projects without Kotlin compilation/runtime evidence.

Admitted direct/disjoint allOf ordinary constructor components may also combine their own nullable default factory with [whole-object counts](component-default-factories.md#counted-dto-component-factories), with validation enabled. Null skips only the value count, never source validation; a non-null annotation must satisfy the retained interval.

### Ordinary DTO collection root factories

[DTO List/typed Map component factories](object-collection-factories.md) reuse existing ordinary object/allOf walks on finite inline container paths, including direct named plain DTO elements. Actual root alias/serializer, original source Pointers, field defaults, required/Patch, nullable layers, independent wrapper annotations and no inheritance remain. Probe permissions are isolated and factories cannot silently drop eligible annotations. Existing source/value/width caps and wider D03/platform/source-role boundaries remain.
