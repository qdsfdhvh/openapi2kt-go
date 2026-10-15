# References to actual union classifiers

[Nullable ordinary DTO reference members](nullable-object-reference-members.md) have original-source authority and nearest member/alias null intent; physical constructors remain distinct from root carrier or inline nullable codecs.

[Nullable ordinary object members](nullable-object-members.md) retain their actual physical variants while deriving the surrounding union null view; member intent is distinct from literal null-only branch folding.

Local schema-internal Pointers and finite typed chains may bind actual retained object or JSON-kind union declarations. A nested target uses its physical containing scope; a named terminal uses its canonical top-level classifier. Component-root bridges and forward aliases preserve those same types. For example, `#/components/schemas/UnionBox/properties/choice` binds the retained `UnionBoxDto.Choice`, and a chain ending at `Choice` binds `ChoiceDto`.

Root array and typed dictionary union peers use their actual top-level declarations. Borrowed allOf copies keep their existing declarations, while references to the original source remain bound to its lexical owner's union. No copied union or alias-owned variant namespace is created. Use the canonical owner for nested variants, and the containing scope for native union sibling wrappers.

## Retained identity

Shared union planning records the original source node, Pointer, emitted declaration and returned typed view in an analyzer sidecar. Final shared ownership traverses actual Model/union trees and collection peers. Exactly one retained occurrence under the original lexical catalog can authorize a classifier. Source syntax and a scratch validation union cannot authorize it. Original catalog entries and schema nodes remain unchanged, with no expanded schema JSON or alternate name predictor.

Self/mutual nominal recursion, aliases, list/map consumers, request Patch and typed HTTP/SSE data use the canonical union. First late HTTP uses may bind only registered actual occurrences. Discarded overrides, fixed-object additionalProperties scratch models, ambiguous repeated occurrences and scalar/raw/request projections fail with `E_REF_UNSUPPORTED`. A single-object nullable fold that emits a Model has no union classifier to bind through this profile.

## Null carriers and nullable views

Standalone unions keep their existing physical dispatch and non-null named Null singleton carriers. For a root string/boolean/null union, decoding `null` as the canonical root returns its Null variant. Internal chains and root bridges to that root preserve this behavior.

Inline unions can instead emit a nullable typed view of a physical non-null union. An internal reference inherits the actual view: an inline string/boolean/null union alias is nullable, and decoding `null` returns Kotlin null. A nearest `nullable: false` view restores the non-null inline classifier, whose codec rejects null when it has no emitted Null subtype. Root collection peers follow the view returned by their actual planner, rather than inferring a scope or codec from source null branches.

Named root aliases may independently apply nearest true/false intent to an actual canonical object or JSON-kind union. A nullable adapter decodes JSON null as Kotlin null, even if the canonical root has a Null variant. Its false view restores the canonical backing, including that Null singleton. Thus encoding a named Null variant through a nullable alias and decoding it can collapse to Kotlin null. This is not carrier-identity round-trip preservation. Direct physical serializers retain their existing non-null descriptor and dispatch behavior.

Absent alias intent inherits the nearest alias/ref intent; an explicit consumer wins. Optional nullable request properties use Patch, including direct internal union targets whose nullable view comes from a null member. `Patch.Value(NullRoot)` and `Patch.Null` share JSON null and decode as Patch.Null under the existing presence contract.

## Limits and boundaries

Every erased named/internal hop remains in cached source height after false narrowing. Nominal declarations are recursion barriers; transparent-only cycles still fail. Source depth stays 48, graph/ownership work stays bounded at 16,384, union branches and serialization slots stay bounded at 240. Original constraints, malformed source and discarded nodes must still validate before artifacts are returned.

This extends typed classifier binding and nullable alias views. It preserves existing union tag/key/native-kind dispatch, Unknown and default-polymorphic behavior, rather than introducing exact oneOf match-count validation. Effective nullable aliases cannot be borrowed as non-null union/allOf/hierarchy shapes. Internal root bridges remain excluded from shape borrowing. Common structured fields, dictionary intersections, multiple-parent/inline hierarchies and general ambiguous graphs keep separate contracts.

## Compatibility and verification

Previously rejected internal/named union terminals and own-nullable union aliases now generate canonical types. Historical negative inputs are preserved and explicitly graduated to precise owner/null-view positives; unrelated scalar/raw/fold/shape exclusions remain. Existing accepted Kotlin output and wire behavior stay under their original contracts.

The [synthetic fixture](../tests/fixtures/http/union-nominal-owners.json), [core contracts](../core/union_nominal_owners_test.go), [owner/source/resource guards](../core/union_nominal_owners_guards_test.go), [nullable/late-use guards](../core/union_nominal_nullable_guards_test.go), [private identity tests](../internal/schema/union_nominal_owners_test.go) and [native preservation tests](../internal/cli/union_nominal_owners_process_test.go) define the verification surface. `bash scripts/test.sh feature union-nominal-owners` generates fresh Data/Plain × validation projects, with optimization flags on Plain, four Json configurations, constructors/codecs, nominal recursion, nullable/false and Null carriers, Patch/controlled HTTP/live fragmented SSE closure/reuse, and independent 240-field/240-branch/48-source probes. [The Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionNominalOwnersContracts.kt) also runs in the full HTTP suite.

This profile does not establish full JSON Schema graphs or additional Android/KMP, host or engine acceptance.
