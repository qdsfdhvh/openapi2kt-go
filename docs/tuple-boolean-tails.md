# Boolean tuple tails

OpenAPI 3.1 arrays with `prefixItems` accept explicit Boolean `items` tails.
This extends the existing [positional-array List approximation](primitive-tuples.md)
across component aliases, DTO fields, nested containers, JSON HTTP bodies and
typed SSE data. It does not change Boolean schema roles without `prefixItems`.

| Tail | Kotlin element mapping |
|---|---|
| `items: true` | `JsonElement`, including a uniform typed prefix, because the tail permits every JSON kind |
| `items: false` | The original common prefix type when all member signatures match; otherwise `JsonElement` |

The prefix grammar includes [Boolean positional literals](tuple-boolean-members.md), existing builtin/UUID/temporal scalar and
admitted String/native enum or ordinary/allOf constructor reference grammar.
Effective nullable intent and original nominal source identity still determine
whether members are uniform. A false tail does not synthesize a false class or
invent a positional union. A true tail does not erase source validation or grant
new inline member, reference or declaration roles.

Both mappings emit `W_TUPLE_APPROXIMATION` at the original `/prefixItems` Pointer.
The Boolean-tail message identifies the lost constraint: true-tail raw lists do
not validate positional prefix types; false-tail lists do not enforce positions
or the tail's maximum length. For example, a String prefix with `items: false`
produces `List<String>` and still accepts extra Strings; with `items: true` it
produces `List<JsonElement>` and can decode objects, arrays, null or Booleans at
any index. This remains a documented approximation rather than a tuple validator.
Warning denial rejects it before output mutation; explicitly allowing the warning
retains the ordinary policy.

Every original prefix member validates before typed/raw selection, even when
`items: true` discards its final Kotlin type. Referenced vocabularies/constructors
still need their actual original declarations and source ownership. Malformed
formats, unknown keys, unsupported annotations, invalid reference chains and
resource failures retain their original Pointers. No source node is copied or
rewritten. Source depth48, tuple reference work16384, checked16 and constructor
slots240 retain their existing limits.

A uniform false-tail tuple selecting an already eligible direct scalar also
uses the existing [field default](tuple-field-defaults.md) and
[component factory](tuple-root-defaults.md) grammars. It validates literals through
the selected scalar codec, preserves required/Patch suppression, fresh values,
independent alias annotations and the original value/default budgets. Defaults
retain the same List approximation and gain no closed-tail length check. Raw
true-tail or mixed projections now use the [raw default role](tuple-raw-defaults.md).
Nominal selected leaves and positional defaults keep their separate eligibility.
A default does not grant new source
permission or inheritance.

Existing opt-in collection counts apply to the resulting typed consumers.
A literal true tail also retains its existing exact JSON-value `uniqueItems`
authority: duplicate nulls and equal objects with reordered keys fail through
typed consumer checks. A false tail does not grant a positional uniqueness
source. Standalone aliases and their ordinary serializers remain unchecked
where the existing collection contract supplies no validation barrier.

`bash scripts/test.sh feature tuple-boolean-tails` compiles/runs fresh Data/Plain
sources with validation off/on, baseline and all optimization/locked output,
four Json settings, typed/raw/nullable/enum/DTO leaves, independent factories,
field defaults, request Patch, actual HTTP requests/responses and fragmented
still-open SSE failure/cancellation/closure/client reuse. The
[fixture](../tests/fixtures/http/tuple-boolean-tails.json),
[core contracts](../core/tuple_boolean_tails_test.go),
[original source checks](../internal/schema/tuple_boolean_tails_test.go),
[native policy/output checks](../internal/cli/tuple_boolean_tails_process_test.go)
and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleBooleanTailsContracts.kt)
define this D06/F08/Q05 increment. The existing tuple fuzz route includes Boolean
tail and closed scalar factory seeds. The complete HTTP route invokes the same
consumer. Other platforms require separate evidence.

**Migration:** previously rejected Boolean tails now generate with an explicit
approximation warning. The historical `unsupported-prefixitems` corpus input
retains its original bytes and now has a reviewed successful snapshot. Prior
accepted tuple mappings, warning text, Kotlin APIs, wire codecs and persisted
formats are unchanged. General structured/composed
tuples and the remaining D06 graph roles remain open.

[Named true/false tuple references](tuple-boolean-references.md) have a separate original-root/actual-codec contract, including independent outer defaults, nullable false elements and complete erased-prefix source-depth accounting. General Boolean graphs remain independent.
