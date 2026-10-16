# Ordinary allOf object field defaults

An ordinary DTO field may declare its own `default` on an inline object `allOf` whose contributions have distinct wire property names. The shared object planner still owns its nested Kotlin DTO, required arguments and serializers. For example:

```json
{
  "allOf": [
    {"properties": {"label": {"type": "string"}}, "required": ["label"]},
    {"properties": {"count": {"type": "integer"}}, "required": ["count"]}
  ],
  "default": {"count": 2, "label": "initial"}
}
```

An optional field uses a fresh typed object decoded through the final nested DTO serializer. A missing key uses the default; a supplied object replaces it. Required fields validate the complete annotation and retain their constructor argument. Optional nullable inline request fields retain `Patch.Absent`, `Patch.Null` and `Patch.Value`; a schema default never makes a missing key an update.

## Source and value contract

The field must produce an ordinary constructor Model with at least one constructor property. Its `allOf` members may contain ordinary objects, nested object compositions and already accepted object-member references. Root own properties contribute after the members. Every wire property has exactly one contribution; repeated names, even with identical schemas, remain outside this default role.

The default walk borrows the shared merged shape. Each field retains its original schema node, Pointer and source depth, including fields expanded from named members. All supplied keys must name declared fields; all merged required names must be present. Leaf membership, nullable values, supported bounds and collection counts follow the existing [object default contract](nested-object-defaults.md). A plain object member may carry its own validated default; omitting that member from the outer default uses the nested field's initializer. Schema errors point to their source; supplied-value errors point to the original `/default/...` member. Validation visits default keys in source order before sorting the encoded keys.

Effective shape nullability authorizes `default: null`; optional Kotlin presence alone does not. Nested objects, supported collections and scalar/enum/temporal leaves retain their own nullable intent and validation. Required/Patch/discarded annotations undergo the same full validation before initializer suppression or a later parent `allOf` override.

The decode helper binds the final actual DTO type after deduplication, merging and hoisting. Adding the annotation preserves the public type and declaration ownership. The existing `W_ALL_OF_APPROXIMATION` remains owned by type planning, with no duplicate warnings from the default walk.

## References to named allOf DTOs

An ordinary direct local `$ref` field may also own a default when its original terminal is an already accepted named ordinary allOf constructor DTO with the same disjoint contributions. For example, place the `allOf` above in component `Value` without a root default, then consume it as:

```json
{"$ref": "#/components/schemas/Value", "default": {"count": 2, "label": "initial"}}
```

Transparent named aliases and already accepted finite internal reference hops may reach that named component. The source walk retains every intermediate annotation and original Pointer. Final retained declaration ownership authorizes the actual component and serializer; a provisional or coincident type name does not. The field retains the resolver's existing public type, and adding a consumer default leaves target declaration bytes and warnings unchanged.

Reference nullability follows the shared reference planner before optional presence: explicit consumer intent, nearest alias/internal hop intent, then terminal intent. Explicit false narrows even an intrinsically nullable allOf component. A valid null annotation emits literal `null`; non-null values use the final actual DTO serializer and create fresh objects, including omitted members with their own defaults. Required, Patch and discarded defaults undergo complete validation before suppression.

The same referenced role also covers eligible [internal allOf DTOs](#references-to-internal-allof-dtos). Annotation-only allOf reference wrappers and hierarchy/common/inherited projections retain separate roles. [Referenced collection fields](#referenced-collection-field-defaults) separately admit DTO element references. Defaults on the target component or an intermediate reference are rejected rather than inherited. Repeated properties, fixed extras and whole-object counts remain excluded.

## References to internal allOf DTOs

A consuming ordinary direct local `$ref` field may own a default when an already admitted [internal allOf terminal](allof-nominal-terminals.md) retains a unique constructor Model under its original lexical component. For example, a reference to `#/components/schemas/Owner/properties/node` can use `default: {"label":"initial","count":2}` when `node` contains the disjoint allOf above. A root alias or finite internal/named reference chain may reach that same node. The public type stays the physical constructor, such as `OwnerDto.Node`, rather than an alias-owned or copied class.

Eligible existing scopes include properties/nested DTOs, List item and typed Map value Models, root collection peers and unique composition-owned constructor Models. A reference to an allOf object-union variant uses that concrete Model's serializer and ordinary business payload; it does not add a closed-root class discriminator. The source must still pass the disjoint ordinary object default walk. Typed nominal recursion or wider typed admission alone does not grant a default role.

Source authorization combines the final retained nominal binding with the original node, Pointer, current component identity and schema lookup's lexical `SourceCatalog`. A generated name or a scratch/borrowed occurrence cannot grant ownership. Hierarchy sources and hierarchy lexical owners remain excluded. No resolver or declaration admission is added by the annotation.

Original field nodes, required names, leaf checks and source depths are reused. Nearest consumer/alias/internal nullable intent precedes terminal effective shape intent, including member-derived nullable on internal Models; explicit false narrows it and a later true may re-enable null. The helper binds the final actual type after optimization and produces fresh values. Target/chain annotations remain rejected, even when the target field's own initializer is otherwise valid. Required/Patch/discarded annotations and original/canonical budgets follow the same full validation before suppression.

The consuming field defaults a single DTO value. [Referenced collection fields](#referenced-collection-field-defaults) separately support references to these Models as elements; allOf-valued members inside another object default remain outside the walk.

## Direct inline collection field defaults

An ordinary direct inline finite List/typed Map field may own an outer default when its original terminal element is an ordinary disjoint allOf constructor DTO. For example:

```json
{
  "type": "array",
  "items": {
    "allOf": [
      {"properties": {"label": {"type": "string"}}, "required": ["label"]},
      {"properties": {"count": {"type": "integer", "default": 2}}}
    ]
  },
  "default": [{"label": "initial"}]
}
```

List, typed Map and finite mixed paths such as List of Map values keep their original nested constructor and public collection type. The shared type plan must create exactly one constructor Model with at least one property on that path. Empty/raw, const-only and union projections do not obtain the role. The value walk borrows the original leaf and its merged fields; no component, source schema copy or borrowed classifier is added.

Permission is scoped to that consuming field's original allOf element node in a shallow validation copy. A structurally equal copy or another object member cannot acquire it. It follows only inline container edges and stops at reference edges. Direct inline collection fields with referenced allOf elements, allOf-valued members inside another object default and bare allOf elements inside referenced collection trees retain separate roles. Primitive allOf compositions keep their existing default/diagnostic path.

The merged required names, leaf checks, original Pointers/depths, source planning and original/canonical value budgets apply even to empty/null annotations. Container and element nullable intent remain independent; effective element intent may come from an allOf member, and an explicit root false narrows it. Optional presence alone does not authorize annotation null. Required/Patch/discarded annotations validate before suppression, and existing no-drop guards still reject discarded runtime constraints. Element/container-child own defaults remain rejected; ordinary fields inside the element retain their own admitted defaults.

The helper decodes the whole collection through its final actual serializer after optimization. Each evaluation creates fresh containers, element DTOs and supplied nested objects; missing fields inside an element use its existing field initializers. Supplied collections replace the outer default and preserve existing HTTP/SSE/Patch behavior.

[Core admission/nullable/diagnostic/budget regressions](../core/inline_allof_collection_defaults_test.go), [original leaf identity/scoping/type/depth tests](../internal/schema/inline_allof_collection_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/inline_allof_collection_defaults_process_test.go) define the extension. The [nested collection fixture](../tests/fixtures/http/nested-object-collection-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectCollectionDefaultsContracts.kt) and `bash scripts/test.sh feature nested-object-collection-defaults` exercise Data/Plain × validation off/on, four Json configurations, fresh nested values, nullable/required/Patch, HTTP/live SSE and optimized/locked helpers. The checked profile adds admitted collection counts to its generated input; preparation alone is not Kotlin compilation/runtime evidence.

## Referenced collection field defaults

An ordinary DTO field whose direct local `$ref` resolves to an already admitted finite named/internal List/typed Map tree may own an outer default when its element references an eligible named or internal ordinary allOf constructor Model. Accepted collection aliases and element reference chains retain their existing physical types and source limits. For example, `Rows` may declare `items: {"$ref":"#/components/schemas/Value"}` for the disjoint allOf above, and a consuming field may use `{"$ref":"#/components/schemas/Rows","default":[{"label":"initial","count":2}]}`. Mixed finite paths such as List of typed Map values are supported by the same walk.

Only crossing an actual element reference edge grants this role. The final named element type, retained nominal binding, original source node/Pointer/owner and hierarchy exclusions are checked by the same referenced DTO authority above. In this referenced-field role, a bare inline allOf collection element remains excluded, including when another typed use has independently registered its nominal source. [Direct inline collection fields](#direct-inline-collection-field-defaults) have their own source role; referenced elements of direct inline fields and allOf-valued members inside object defaults remain separate.

Every element validates the original disjoint shape, required names and leaves. Container and element nullable intent remain independent; an explicit false element reference narrows a nullable terminal, and optional field presence alone does not authorize annotation null. Source planning and original/canonical value budgets apply even to empty/null collections. Required/Patch/discarded annotations fully validate before initializer suppression; existing no-drop checks still reject discarded runtime constraints. Source annotations are not inherited. Named collection count constraints retain their existing validation opt-in.

The helper decodes the complete collection using its final actual collection/alias serializer. Each default evaluation creates fresh containers, DTOs and supplied nested objects; omitted DTO fields retain their own declared initializers. Optimization and locks retain the effective type/default/check identity.

[Core scope/nullable/suppression/diagnostic/budget regressions](../core/referenced_allof_collection_defaults_test.go), [original source/final element authority checks](../internal/schema/referenced_allof_collection_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/referenced_allof_collection_defaults_process_test.go) define this extension. The [referenced collection fixture](../tests/fixtures/http/referenced-object-collection-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectCollectionDefaultsContracts.kt) and `bash scripts/test.sh feature referenced-object-collection-defaults` cover Data/Plain × validation off/on, four Json configurations, concrete internal serializers, fresh nested values, required/Patch, HTTP/live SSE and optimized/locked helpers. The checked profile adds admitted named collection counts to its generated input; preparation alone is not compilation/runtime evidence.

## Bounded admission

Inline fields and references to eligible named or internal constructor Models apply in ordinary shared/response/request DTOs. These remain separate:

- annotation-only reference wrappers: a single component ref plus metadata/nullable members must retain its canonical referenced type;
- target/root/component defaults and default inheritance;
- repeated-property/default/constraint aggregation, fixed extras (`additionalProperties`) and whole-object property counts;
- empty/raw and const-only projections, hierarchy/common/inherited fields, bare allOf elements inside referenced collection trees and nested defaulted-object members whose own schema is `allOf`;
- object-union products, which use the separate [composed union default contract](union-field-defaults.md).

Existing input8MiB, source48, composition-work16,384, checked-collection16 and constructor-slot240 limits remain. Object default walks also share the independent [16,384-step source planning budget](nested-object-defaults.md#object-source-planning-budget). The annotation retains the shared 16,384 JSON value-node and 32 KiB original/canonical JSON budgets. Exceeding a value budget returns `E_MODEL_LIMIT` at the original member; generation failure leaves existing output intact.

## Verification and compatibility

This extends M01/M03/M04/M07/M08 and the bounded D03 default profile; general D03/D05 remain open. Previously rejected eligible inline and referenced named/internal ordinary allOf object field defaults now generate. Fields without defaults retain existing source, type and wire behavior. Repeated-field aggregation, hierarchy and root/component default semantics are not added.

The [core contract tests](../core/allof_object_defaults_test.go), [source authority tests](../internal/schema/allof_object_defaults_test.go) and [native parity/preservation tests](../internal/cli/allof_object_defaults_process_test.go) define admission and diagnostics. The existing [nested object fixture](../tests/fixtures/http/nested-object-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectDefaultsContracts.kt) cover ordinary allOf defaults, member refs, fresh nested instances, missing/replacement/null, Patch states and HTTP/SSE consumption under four Json configurations. Run `bash scripts/test.sh feature nested-object-defaults` for Data/Plain × validation off/on compilation/runtime and optimized/locked helper binding.

The referenced role is defined by [core admission/nullable/bridge regressions](../core/referenced_allof_object_defaults_test.go), [source and final nominal authority tests](../internal/schema/referenced_allof_object_defaults_test.go) and [CLI parity/readonly check/failure preservation](../internal/cli/referenced_allof_object_defaults_process_test.go). Its [referenced-object fixture](../tests/fixtures/http/referenced-object-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectDefaultsContracts.kt) and `bash scripts/test.sh feature referenced-object-defaults` exercise actual named/alias/internal-bridge defaults, intrinsic nullability, required/Patch, optimized helper binding and controlled HTTP/live SSE under the same four profiles.

The internal extension has [core scope/nullable/presence regressions](../core/internal_allof_defaults_test.go), [source/owner/depth tests](../internal/schema/internal_allof_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/internal_allof_defaults_process_test.go). The same fixture/consumer exercises property and collection-owned Models, root peers, aliases/bridges, concrete composition variants, inherited nullability, fresh nested member defaults, Patch wire states and original validation through HTTP/live SSE. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation only. JVM verification does not establish Android/KMP or release acceptance.
