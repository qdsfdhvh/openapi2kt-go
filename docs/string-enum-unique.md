# String enum list uniqueness

[Model String enum metadata](metadata-string-enums.md) extends nullable roots/aliases, actual internal lexical references, existing independent scalar/collection/object default roles and serialName uniqueness. Original ownership, source/null-vocabulary boundaries, presence and budgets remain. Equivalent enum merges now retain each component-owned factory; generic graphs, platforms and final acceptance remain open.


With `--validation`, arrays with original plain String enum items accept `uniqueItems: true`. Items may declare `type: string` or use an already admitted typeless String vocabulary, including its existing explicit nullable/null-member grammar. Root collection aliases, ordinary DTO fields, supported nested List/typed Map paths and existing typed HTTP/SSE consumers share the check. A reference to an eligible collection retains that collection's check. Already admitted local named/internal enum leaf references and their existing aliases also retain serialName equality; the reference graph itself is unchanged.

```json
{
  "type": "array",
  "items": {"enum": ["ready", "done", "null", null], "nullable": true},
  "uniqueItems": true
}
```

## Wire equality and open vocabularies

Equality uses each actual enum instance's `serialName`. A known `READY` entry and `Unrecognized("ready")` collide, as do two unknown entries with the same payload. Runtime vocabularies remain open: `Unrecognized("future")` is valid when its payload is unique and satisfies any separately admitted leaf checks. Null and String `"null"` are distinct; at most one null is allowed. Parsed strings compare exactly without Unicode normalization, so escaped spellings of the same token collide while canonically equivalent but different strings remain distinct.

The generated check maps nominal entries to nullable String payloads, then uses the existing scalar uniqueness helper. It sorts a copy of non-null values and compares adjacent entries without mutating the caller's list. Enum classifiers, lexical owners, entries, lookup, serializers, descriptors and JSON String wire remain unchanged. No membership validator, serialized state or custom enum codec is added.

DTO constructors, Data-class copy and automatic decoding enforce retained field checks. Patch.Value is checked while Absent/Null preserve their existing presence semantics. Typed collection requests check before the request callback/engine, invalid responses fail decoding, and invalid known SSE payloads terminate and close their input. Cold collection, cancellation and caller-client reuse retain the existing lifecycle contract. Independent alias assignment/reified collection serialization, later collection mutation and arbitrary requestBuilder body replacements retain their existing boundaries.

## Referenced enum leaves

Existing local named component references, finite accepted named/internal alias chains, internal lexical enum properties and root bridges may supply list elements. The original retained enum declaration owns its codec and members; a consuming list creates no copied vocabulary or new classifier. Nullable intent follows the existing nearest explicit override: an intrinsically nullable enum can be narrowed by an admitted `nullable: false` hop or use, while the canonical enum serializer remains non-null. Container nullability stays independent.

The uniqueness source check reuses the existing borrowed enum authority records and final declaration validation. Missing, repeated or same-name replacement declarations, copied schema/member nodes, changed vocabulary/nullable provenance and changed alias cache/source identity reject. References to bounded/formatted/native/const/value-class/folded vocabularies retain their prior exclusions. The new check does not authorize a previously rejected reference, wrapper, default or intermediate annotation.

Existing referenced collection field own-defaults may contain these enum leaves and check uniqueness against their original vocabulary. This does not grant defaults to direct inline arrays whose enum reference elements previously excluded that annotation role. Required/Patch suppression and target/chain/leaf default exclusions keep their original contracts. Under explicit merging, equivalent eligible enum roots may share a canonical classifier; access singleton and nested members through that actual owner, since Kotlin typealiases do not export them. Deduplication/hoisting keep their existing nominal comparison policy, and changing a retained uniqueness check cannot reuse a frozen different name-lock shape.

## Defaults and sources

Already admitted direct, nested and referenced collection default roles also check uniqueness. Non-null annotation members must still belong to the original vocabulary, even though runtime unknown Strings remain open. Duplicate/default-membership errors identify the original `/default/index` path, including nested paths. Empty/null annotations, required fields and suppressed Patch initializers still validate the full original schema before suppression. Nullable intent, raw vocabulary indices and original default budgets remain unchanged.

Only an original plain String vocabulary and its actual nominal TypeRef authorize the check. Composition/folded vocabularies, formatted/native/const/value-class projections, DTO/union elements and lists whose elements are containers remain outside this increment. Inline HTTP collection projections that erase the enum classifier cannot silently gain or lose checks. Direct String bounds/patterns compose only where the separate [enum collection bounds contract](string-enum-collection-bounds.md) already admits them; nested bounds and general constraint aggregation remain excluded. `uniqueItems: false` remains exactly effect-free on otherwise supported arrays with validation enabled; both values require validation.

Input 8 MiB, source depth 48, checked collection edges 16, analysis work 16,384, raw enum members/constructor slots 240 and existing default node/byte budgets are unchanged. Existing collection-check and actual element identities feed deduplication, merging, hoisting and name-lock. Persisted formats do not change. Previously rejected eligible inputs now generate checks; previously accepted APIs/wire retain their contracts. New consumers should regenerate and compile. This is the D05/M03/M08/M12/V05 subset and existing A03/A05/A08 consumers, not general uniqueness or full JSON Schema validation.

## Verification

The [fixture](../tests/fixtures/http/string-enum-unique.json), [core source/default/resource regressions](../core/string_enum_unique_test.go), [reference/default/cache/optimizer regressions](../core/referenced_string_enum_unique_test.go), [actual vocabulary authority](../internal/schema/borrowed_collection_enum_leaves_test.go), [native failure preservation](../internal/cli/string_enum_unique_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumUniqueContracts.kt) define this profile. Historical exact direct and referenced enum uniqueness negatives graduate to explicit opt-in positives; other source exclusions remain.

`bash scripts/test.sh feature string-enum-unique` builds fresh Data/Plain × validation on/off projects. The unbounded validation-off control executes the same consumer and separately verifies that the constrained original rejects. Plain also enables dedup/hoist/merge/name-lock. Four Json default/null combinations cover known/unknown wire equality, nulls, defaults, nested/alias fields, named/internal/bridge enum references, nullable narrowing, Patch, request JSON/preflight, HTTP decoding and fragmented still-open SSE failure/cancellation/closure/cold collection/client reuse. The full `http` route includes the checked consumer. Resource-boundary Go tests are separate from these runtime profiles; Android/KMP, combined resource maxima and broader reference admission remain unverified.
