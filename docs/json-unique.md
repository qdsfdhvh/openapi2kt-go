# JSON-value array uniqueness

With `--validation`, `uniqueItems: true` supports arrays whose original direct item schemas are OpenAPI 3.1 `true`, unconstrained `{}`, empty ordinary objects or raw `JsonObject` dictionaries. Existing finite references to named literal-true or direct unconstrained ordinary `type: object` components also support this comparison, including admitted named aliases and internal ref bridges. The existing native String/Int/Long/Boolean contract keeps its previous implementation and output. This profile adds exact JSON-value comparison rather than Kotlin object identity.

The JSON comparison rules are:

- Objects compare the same key set and recursively equal values; key insertion order does not matter.
- Arrays compare length and ordered values; `[1,2]` differs from `[2,1]`.
- Strings compare parsed contents without Unicode normalization. `"a"` equals `"\u0061"`; `"é"` differs from `"e\u0301"`.
- Numbers compare exact decimal values. `1`, `1.0` and `1e0` are equal; signed zero compares equal. Numeric-looking strings remain strings.
- Booleans and null remain separate JSON kinds. Kotlin null in an admitted nullable item view compares equal to JsonNull.

Checks run in generated DTO constructors, copy, decoding and existing typed request/response/SSE consumers. Nested List/Map paths and transparent array aliases retain the check; a standalone alias constructor is not itself a validation barrier. Request Patch Absent/Null do not inspect contents, while Patch.Value checks the supplied list. Bad HTTP response or SSE data fails decoding with the retained uniqueness error.

The implementation builds a length-prefixed canonical equality key per value, sorts object keys and stores keys in a set. It does not use a pairwise scan or expand decimal exponents. Keys preserve numeric precision and distinguish every JSON kind. This is deterministic and uses only Kotlin common collections/StringBuilder APIs plus kotlinx.serialization JSON types.

## Empty ordinary allOf item references

Existing item references to named empty ordinary `allOf` components also support `uniqueItems: true`. Eligible direct, nested and referenced ordinary object contributions must produce the original empty Model with no constructor properties, constants, nested declarations or property-count codec. Its original source nodes and unique retained declaration authorize the JSON leaf through the [raw reference-source contract](raw-true-reference-chains.md). A copied same-byte member or replacement/repeated Model cannot substitute for that source.

Existing aliases and internal ref bridges retain nearest nullable intent, independently of container nullability. The same JSON equality helper checks DTO constructors/decoding, nested List/Map consumers, collection aliases, Patch.Value, typed HTTP root arrays and SSE DTO payloads. Kotlin null and JsonNull conflict. Absent/Null Patch states and standalone alias creation retain their existing boundaries.

This adds equality to the existing raw carrier approximation. It does not add object-kind/intersection enforcement or alter `W_ALL_OF_APPROXIMATION` and `W_JSON_ELEMENT_APPROXIMATION` ownership. Nonempty/const-only/count-bearing/union/hierarchy/dictionary compositions, unsupported intrinsically nullable empty roots and inline raw compositions remain outside this profile; explicit nullable consuming aliases remain supported. Original grammar, contribution/source/cache/checked-depth/work/slot limits and failure preservation still apply. No new public wrapper, serializer, default role or persisted format is introduced. Previously rejected eligible inputs now generate with validation enabled; accepted API/wire behavior retains its contract.

## Complete recursive-cover item references

Arrays whose item references already resolve to a named [complete recursive JSON-kind cover](recursive-json.md) also support `uniqueItems: true`. Self-recursive covers, mutually recursive covers, transparent named graph refs and admitted incoming aliases/internal ref bridges use the same exact JSON equality keys. Nearest nullable intent controls the item codec; Kotlin null and JsonNull count as the same value. Existing nested containers, collection aliases, Patch, HTTP root arrays and SSE DTO payloads retain the check.

Permission requires the complete original reachable graph and its unique retained JSON ValueAlias, using the [existing cover-source contract](recursive-cover-reference-chains.md). Incoming reference work and graph/SCC suffix height remain separate, including after caching and explicit false narrowing. Replaced source nodes, changed descendant grammar or work caches, missing/replaced/repeated exports and changed backing codecs reject the binding. Partial/constrained covers, inline cover nodes, nullable recursive graph-edge siblings and reference-only cycles remain outside the profile. This does not authorize general DTO, typed container or composed-object equality.

Equality checks inspect the JSON values present at their existing validation boundaries. They do not change the caller's JsonElement codec: encoding may change large-number precision or exponent spelling as described in [the numeric codec contract](recursive-json.md#codec-and-numeric-boundaries). No new wrapper, serializer, default role or persisted-format version is introduced. Previously rejected eligible cover-reference arrays now succeed with validation enabled; already accepted artifacts keep their equality policy.

## Runtime and generation limits

A checked list permits at most 16,384 traversed JSON value nodes across its elements, recursion depth 48 per element, and 8,388,608 UTF-16 code units across equality keys. Numbers permit at most 256 source characters and decimal exponents from -512 to 512. Invalid/non-finite number tokens and inputs beyond these limits fail explicitly, including singleton lists. These are runtime validation limits for the new JSON profile; existing native scalar uniqueness and schema resource limits are unchanged.

Original source eligibility is checked independently of Kotlin spelling. Caller-native types, borrowed nominal/enum/temporal/byte schemas, nonempty or otherwise ineligible composed objects, incomplete or constrained recursive covers, lossy unions, fixed object DTOs and typed dictionaries do not acquire JSON equality merely because a mapped type resembles JsonElement. Direct typed dictionary elements with builtin String/Int/Long/Boolean/UUID/LocalDate/Instant values have a [separate mapped equality contract](map-unique.md). More general typed and projected `uniqueItems` remains separate. Wrong-kind keys, malformed Boolean flags, absent validation opt-in and erased/discarded enforcing checks retain their existing rejection rules.

Raw/true referenced leaves require the original named raw/true source and its unique retained declaration. An empty ordinary Model or literal-true JSON ValueAlias grants permission; copied source nodes, replacement/repeated declarations, changed nullable or codec backing do not. The same source checks apply before and after forward binding. Nearest explicit nullable intent still controls the mapped item type: Kotlin null and JsonNull compare equal. This does not extend reference resolution or authorize internal raw terminals or typed collections as JSON leaves. Empty ordinary named allOf roots follow the source authority described above. Named complete recursive covers follow the graph authority described above.

Reference paths keep their existing depth/work/cache, checked-container and JVM-slot budgets. No new serializer, public classifier, default role or name-lock/output format is introduced. Existing accepted artifacts retain their equality policy; formerly rejected eligible referenced arrays now generate successfully with validation enabled. Standalone aliases and later caller mutation keep their existing transparency boundaries.

New JSON uniqueness participates in model equivalence and name-lock shape identity; legacy collection fingerprints remain unchanged. The original JsonObject-array rejection input is retained as an explicit positive regression, and the original named-true and complete self-cover reference rejections are also retained as explicit positive regressions.

The [synthetic envelope](../tests/fixtures/http/json-unique.json), core/private/native source and resource tests, and `bash scripts/test.sh feature json-unique` cover the profile. Fresh Data/Plain consumers run all four Json settings and verify duplicates, object key ordering, exact numbers, nested arrays, Unicode/null, Patch/aliases, depth/work/number limits, HTTP and SSE. This does not establish full JSON Schema uniqueness or Android/KMP acceptance.
