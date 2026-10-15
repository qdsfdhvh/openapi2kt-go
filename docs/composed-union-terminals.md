# References to retained allOf product unions

[Nullable product roots](nullable-product-unions.md) add explicit original-root null intent with standalone Null carriers and inline nullable typed views; original unique product ownership remains.

Typed schema-internal references and finite internal/named-alias chains may bind an existing object union produced by the supported [object union/allOf composition](union-allof.md). Component-root aliases can bind the same canonical root, nested property, list/map value or root collection peer declaration. This extends reference binding to an already generated sealed type; it does not add intersection syntax.

For example, `Bridge → Hops/properties/value → ProductAlias → Product` exports a typealias to the original `ProductDto`. A reference to `Box/properties/choice` binds `BoxDto.Choice`, and references to root collection items/values bind their actual `Inner`/`Value` peers. No schema, catalog entry, wrapper or serializer is copied.

## Physical ownership and typed views

The successful planner records the original source node and Pointer of the retained `ObjectUnion`. Each product variant borrows that same allOf node to preserve field provenance. Those variant models are not additional physical owners of the sealed product declaration. Their actual nested declarations are still visited: a source used in multiple retained variant scopes remains ambiguous and cannot authorize an internal reference.

A binding requires exactly one retained declaration under the original lexical catalog. Borrowed copies do not own the source; discarded fields, scratch additionalProperties plans and missing physical declarations cannot authorize a classifier. Constructor and variant names, common fields, required/override order, tag conjunctions, first-claim dispatch, Unknown and serialization remain those of the original product union.

Own-nullable root aliases are independent typed views over the non-null physical union. Consumer overrides take precedence over the nearest explicit internal or named alias true/false. Absent intent inherits the emitted source view. The canonical direct serializer remains non-null, while reified nullable codecs accept Kotlin null; explicit false restores the physical non-null view. This also applies to Patch, component HTTP bodies/responses and SSE data. Optional shared properties and inline request presence retain their existing contracts.

Each internal, named alias and final named edge contributes to the 48-level source budget, including erased cached hops after false narrowing. The physical sealed declaration is a nominal recursion barrier. The 16,384 composition-work, 240 product-branch and 240 serialized JVM-slot limits remain independent and unchanged.

## Boundaries and migration

This profile does not authorize shape borrowing through internal or root-bridge bindings. Existing supported named product composition keeps its established shape rules. Nullable product/intersection alternatives, primitive/dictionary/discriminator products, exact oneOf match-count validation, general common-value projection and general constraint aggregation remain separate. Malformed and discarded source fields still fail at original Pointers before artifacts are returned; native generation preserves previous output on failure.

Previously rejected references to these retained product unions and own-nullable aliases now generate canonical bindings. Existing accepted constructors, codecs and wire behavior remain unchanged. The original negative product-chain input is preserved as an explicit canonical-binding regression; the other excluded-owner cases remain negative. This is a bounded F04/F07/M09/M11 extension, not complete graph/intersection or additional Kotlin-platform acceptance.

The [synthetic fixture](../tests/fixtures/http/composed-union-terminals.json), [core binding tests](../core/composed_union_terminals_test.go), [ownership/source/product/slot guards](../core/composed_union_terminals_guards_test.go), [private source identity tests](../internal/schema/composed_union_terminals_test.go) and [native preservation tests](../internal/cli/composed_union_terminals_process_test.go) define the contract. Run `bash scripts/test.sh feature composed-union-terminals` for fresh Data/Plain × validation projects, Plain optimization options, four Json configurations, all four known variants and Unknown, nullable/false codecs, Patch, controlled HTTP and fragmented still-open SSE closure/reuse/coldness. Independent 240-branch/240-slot/48-hop inputs compile and run in each project. The [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComposedUnionTerminalsContracts.kt) also runs in the full HTTP suite.
