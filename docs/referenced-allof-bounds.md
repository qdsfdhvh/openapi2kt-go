# Referenced and nested object allOf primitive bounds

With `--validation`, an ordinary non-inherited DTO can retain the supported primitive bounds of repeated same-wire fields from admitted object `allOf` references, named object aliases and nested object compositions. This extends the [flat inline primitive conjunction](model-semantics.md#flat-inline-allof-primitive-bounds) using the existing object-shape traversal. It does not introduce a general schema intersection validator.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Lower":{"properties":{"value":{"type":"string","minLength":2}},"required":["value"]},
  "Alias":{"$ref":"#/components/schemas/Lower"},
  "Record":{"allOf":[
    {"$ref":"#/components/schemas/Alias"},
    {"type":"object","allOf":[{"properties":{"value":{"type":"string","maxLength":3}}}]}
  ]}
}}}
```

The generated `Record.value` remains a required String. `Record("ab")` passes, while `"a"` and `"abcd"` fail construction or automatic JSON decoding. With DataClass output, `copy` enforces the same checks. PlainClass retains its existing API. Nullable properties and request-only `Patch.Absent`/`Patch.Null` skip non-null checks; `Patch.Value` retains the conjunction. Required, optional and default presence semantics keep their existing contracts.

Each original contribution must directly declare the same string/integer/number kind, identical format presence and value, and identical effective field nullability, mapping to String, Int, Long, Float or Double. At least one overridden contribution must contain an enforcing primitive check. All contributions are independently validated at their original node, Pointer and source depth. Object references retain their existing admissibility and alias rules; a pure single-reference wrapper does not become a new object expansion grammar.

Traversal follows each `allOf` member, recursively, then the object's own properties. The complete original contribution sequence is retained separately from winner/discarded lowering. For example, an early `minLength:4` followed by nested `maxLength:3` and `maxLength:6` fails at the nested `maxLength:3`; the later bound cannot repair the earlier empty interval. Diagnostic Pointers identify the original referenced source rather than a copied or expanded schema.

String length and pattern rules, numeric exact-source consolidation, OpenAPI 3.0/3.1 exclusives and mapped floating comparisons reuse the flat profile. Different String patterns and unequal exact `multipleOf` factors remain unsupported. Numeric empty-domain diagnostics retain their existing restrictive-upper/factor attribution. Equal bounds retain the first original contribution. No regex satisfiability, LCM, floating tolerance or numeric-source-count limit is added.

Property references, scalar wrappers/unions/collections, enum/const/native/date/byte/value-class/custom-serializer aggregation and differing scalar/nullability signatures remain outside this profile. Own scalar defaults on eligible repeated fields follow the bounded contract below. Union variants, hierarchy/common/inherited/typed shape planning do not acquire ordinary DTO aggregation permissions. Ignored extras, raw projections and incomplete discarded groups still reject checks they cannot retain. Object `minProperties`/`maxProperties` retain their separate [whole-object JSON contract](referenced-allof-counts.md), including the admitted referenced/nested ordinary DTO profile. Inline HTTP root `allOf` projection remains unsupported.

Supported nominal DTO request/response and SSE payloads keep their existing codecs. Invalid construction precedes request callbacks/dispatch; invalid success payloads fail decoding. Invalid known SSE data closes the live input while preserving caller-client reuse. No new serialized fields, constructor slots, regex caches or hidden Json settings are introduced. Standalone alias assignment, later mutation and caller requestBuilder overrides keep their existing validation boundaries.

Only effective checks participate in declaration reuse and owned shape-lock identity. Original source provenance does not prevent equal declarations from sharing, while changed effective owned bounds fail an existing name lock. Persisted state formats remain unchanged. `W_ALL_OF_APPROXIMATION`, winning field order, required union and all other object approximation semantics remain.

## Defaults on repeated primitive fields

With validation enabled, the same direct String/Int/Long/Float/Double contributions may carry their own scalar `default` annotations. Every original annotation is validated against its own original field before override or required/Patch suppression. Only the final contribution supplies a constructor default; an earlier default is not inherited when the final field omits it.

The final annotation must also satisfy every retained constraint. For example, a referenced field with `minLength:2` followed by `maxLength:3, default:"abc"` generates an optional String initialized to `"abc"`; a final default `"a"` fails at the final field's `/default`. An earlier `default:"longer"` remains valid against its own `minLength:2` and does not become the final initializer. Required fields still require a constructor argument, and request-only nullable fields retain `Patch.Absent`.

Numeric checks read each original bound node, preserving exact integer constraints, OpenAPI 3.0 exclusive modifiers and mapped Float/Double comparisons. String checks use the merged lengths and the single admitted pattern. Floating `multipleOf` with defaults and the existing differing-pattern/factor cases remain unsupported. Empty intersections still fail at their original closing constraint. Source depth, work, checked edges and constructor limits are unchanged.

Only emitted defaults and effective checks enter reuse and lock identity. Changing a discarded default can change its source component's own initializer without changing an operation's owned shape; changing the final owned initializer triggers the existing name-lock shape diagnostic. Existing inputs without defaults retain their output and lock semantics. This accepts previously rejected combinations; it does not enable root/component defaults, property refs, object-valued defaults with overlapping contributions, enum/default intersection or general composition validation.

The [core tests](../core/allof_primitive_defaults_test.go), [reuse/lock tests](../core/allof_primitive_defaults_optimization_test.go), native CLI preservation and fresh Data/Plain consumer in the existing `referenced-allof-bounds` feature cover this extension. Runtime checks include four Json settings, default omission, missing/null/value, required/Patch, copy, Long extrema, floating ties, controlled HTTP and live fragmented SSE decoding/failure closure and client reuse.

**Migration:** previously rejected eligible referenced/nested schemas generate when validation is enabled; invalid constructed/decoded values now throw. Default-off rejection and historical accepted output remain unchanged. F04/F06/F07/M12 gain this bounded profile; broader composition and V06 remain open. The limits remain source depth 48, expansion work 16,384, checked collection edges 16 and JVM slots 240. Required non-null Long/Double use two slots; nullable values use one. Kotlin/JVM evidence does not establish Android/KMP, other-host runtime support or complete JSON Schema validation.

See the [fixture](../tests/fixtures/http/referenced-allof-bounds.json), [core contracts](../core/referenced_allof_bounds_test.go), [source guards](../core/referenced_allof_bounds_guards_test.go), [resource boundaries](../core/referenced_allof_bounds_resources_test.go), [reuse/lock checks](../core/referenced_allof_bounds_optimization_test.go), [native CLI preservation](../internal/cli/referenced_allof_bounds_process_test.go) and [Kotlin contracts](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedAllOfBoundsContracts.kt). Run `bash scripts/test.sh feature referenced-allof-bounds` for fresh Data/Plain compilation and four Json configurations per style, including constructor/serde, nullable/Patch, numeric extrema/ties/remainder and actual fragmented HTTP/SSE consumption.

### Repeated String contributions in outer defaults

[Outer allOf defaults with retained repeated plain String checks](allof-string-defaults.md) reuse complete original ordinary primitive groups under validation. Source-own defaults validate before override, the winning annotation satisfies retained checks, supplied outer members satisfy the same conjunction, and omitted members use the actual constructor default. Original source/Pointer/final codec, required/Patch, work/value/width limits and independent default roles remain. Numeric/ref/enum/unchecked duplicate aggregation and broad D03/D05/platform completion stay separate.
