# P1 JSON corpus

The retained historical inventory contains **154 inputs: 38 original successes and 116 original rejection expectations**, with **50 unchanged Kotlin snapshots**. Seven scalar inputs and the non-null [`array-component`](invalid/array-component.json), direct empty [`nullable-component`](invalid/nullable-component.json) and initial3.1 direct true field [`property-boolean`](invalid/property-boolean.json) have explicit positive graduation contracts in core and native CLI tests, leaving 106 current rejections. The [direct true value contract](../../../docs/boolean-values.md) verifies the complete artifact and original warning Pointer; false/3.0 value slots and Boolean component roots remain rejected. The [`root-array`](invalid/root-array.json) document (`[]`) remains invalid; no original input, manifest or snapshot bytes are rewritten. Individual contracts and descriptions live in [`cases.json`](cases.json); test execution and contribution rules are in the [fixture guide](../README.md).

## Supported behavior

| Area | Representative inputs |
|---|---|
| Dictionaries | [dictionary-typed](valid/dictionary-typed.json), [dictionary-open](valid/dictionary-open.json), [closed-object](valid/closed-object.json): typed/open fields and explicit closed-object approximation |
| Open string enums | [inline-string-enum](valid/inline-string-enum.json), [array-string-enum](valid/array-string-enum.json), [typeless-enum](valid/typeless-enum.json) |
| Scalar const body fields | [declared-const](valid/declared-const.json), [typeless-const](valid/typeless-const.json) |
| Inline primitive union | [typeless-oneof](valid/typeless-oneof.json): sibling string/Long wrappers and JSON-kind dispatch |
| Object allOf approximation | [typeless-allof](valid/typeless-allof.json): empty object composition maps to JsonElement with warnings |
| Nullable composition field | [typeless-anyof](valid/typeless-anyof.json) |
| Primitive component aliases | [primitive-component](valid/primitive-component.json) |
| Every scalar/format | [scalars-30](valid/scalars-30.json), [singleton-type-arrays](valid/singleton-type-arrays.json) |
| Required/nullable combinations | [presence-30](valid/presence-30.json), [presence-31](valid/presence-31.json), [nullable-reference](valid/nullable-reference.json), [nullable-booleans](valid/nullable-booleans.json) |
| Container/item nullability | [array-presence](valid/array-presence.json), [nested-arrays](valid/nested-arrays.json), [array-inline-object](valid/array-inline-object.json) |
| Recursive and escaped references | [self-reference](valid/self-reference.json), [mutual-reference](valid/mutual-reference.json), [pointer-escapes](valid/pointer-escapes.json) |
| Nested/empty/unconstrained objects | [inferred-objects](valid/inferred-objects.json), [nested-object-scopes](valid/nested-object-scopes.json), [empty-components](valid/empty-components.json), [unconstrained-fields](valid/unconstrained-fields.json) |
| Naming and literal safety | [identifier-forms](valid/identifier-forms.json), [wire-escaping](valid/wire-escaping.json), [builtin-shadowing](valid/builtin-shadowing.json), [suffix-reference](valid/suffix-reference.json) |
| Metadata and order | [descriptive-metadata](valid/descriptive-metadata.json), [source-order](valid/source-order.json) |
| Input detection | [explicit-json-extension](valid/explicit-json-extension.json), [extensionless-auto](valid/extensionless-auto.json), [uppercase-json-auto](valid/uppercase-json-auto.json) |
| Combined graph | [catalog-graph](valid/catalog-graph.json): six related components, required references, recursion, nullable elements, inline objects and untyped payloads |

## Historical rejection coverage

| Area | Cases |
|---|---|
| JSON syntax | Empty/truncated input, trailing content/comma, comments, non-finite numbers, duplicate document/property/metadata keys |
| Document containers | Array/null/boolean/number roots; absent/unsupported/malformed versions; absent/null components; empty/wrong-type schemas |
| Model-only scope | Non-empty or malformed paths/webhooks; non-model parameter components; array/untyped/nullable component roots |
| Schema slots | Null/boolean/array component or property slots; malformed properties; invalid type arrays; malformed nullable/format; absent or invalid items |
| Required lists | Wrong list type, unknown fields, duplicate fields, non-string entries, explicit null |
| Deferred semantics | Object enums and typeless non-string enum members, non-scalar const contexts/default, general composition and multi-variant unions, discriminator, numeric/string/collection constraints, tuples, property-name schemas, read/write-only, native-type extensions and date/binary/URI/email formats |
| Nested/unreferenced semantics | Unsupported object enum in an otherwise unreferenced component; supported string array enums are positive cases |
| References | Missing target, external HTTP/file references, nested schema pointers, percent encoding, invalid/truncated pointer escapes, empty/non-string refs, semantic siblings |
| Names and paths | Normalization/case collisions, nested owner/array-item conflicts, serialization-owned names, unrepresentable names, reserved filenames/packages, suffix traversal |
| Diagnostics/options | Escaped pointer locations, package traversal/keywords/reserved namespace/empty segments, explicit YAML and unsupported auto-detected extensions |

Original rejection inputs are retained in [`invalid/`](invalid/), including the explicitly graduated cases above. “Rejected” means outside this generator's accepted subset or invalid for this tool; some are legal OpenAPI constructs deferred to later milestones. Adding rejection fixtures does not implement those features.

## Remaining boundaries

No HTTP wire behavior, Patch/union/custom-serializer runtime behavior, YAML parsing, Android/KMP compilation, or cross-platform filesystem guarantee is established by this corpus. The file-transaction tests and generated resource-boundary tests remain separate from the JSON inventory. The suite is intentionally expandable rather than a claim that all OpenAPI inputs have been covered.
