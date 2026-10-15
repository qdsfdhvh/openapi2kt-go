# JSON-value array uniqueness

With `--validation`, `uniqueItems: true` supports arrays whose original direct item schemas are OpenAPI 3.1 `true`, unconstrained `{}`, empty ordinary objects or raw `JsonObject` dictionaries. The existing native String/Int/Long/Boolean contract keeps its previous implementation and output. This profile adds exact JSON-value comparison rather than Kotlin object identity.

The JSON comparison rules are:

- Objects compare the same key set and recursively equal values; key insertion order does not matter.
- Arrays compare length and ordered values; `[1,2]` differs from `[2,1]`.
- Strings compare parsed contents without Unicode normalization. `"a"` equals `"\u0061"`; `"é"` differs from `"e\u0301"`.
- Numbers compare exact decimal values. `1`, `1.0` and `1e0` are equal; signed zero compares equal. Numeric-looking strings remain strings.
- Booleans and null remain separate JSON kinds. Kotlin null in an admitted nullable item view compares equal to JsonNull.

Checks run in generated DTO constructors, copy, decoding and existing typed request/response/SSE consumers. Nested List/Map paths and transparent array aliases retain the check; a standalone alias constructor is not itself a validation barrier. Request Patch Absent/Null do not inspect contents, while Patch.Value checks the supplied list. Bad HTTP response or SSE data fails decoding with the retained uniqueness error.

The implementation builds a length-prefixed canonical equality key per value, sorts object keys and stores keys in a set. It does not use a pairwise scan or expand decimal exponents. Keys preserve numeric precision and distinguish every JSON kind. This is deterministic and uses only Kotlin common collections/StringBuilder APIs plus kotlinx.serialization JSON types.

## Runtime and generation limits

A checked list permits at most 16,384 traversed JSON value nodes across its elements, recursion depth 48 per element, and 8,388,608 UTF-16 code units across equality keys. Numbers permit at most 256 source characters and decimal exponents from -512 to 512. Invalid/non-finite number tokens and inputs beyond these limits fail explicitly, including singleton lists. These are runtime validation limits for the new JSON profile; existing native scalar uniqueness and schema resource limits are unchanged.

Original source eligibility is checked independently of Kotlin spelling. Caller-native types, borrowed nominal/enum/temporal/byte schemas, named raw/recursive-cover aliases, lossy unions, fixed object DTOs and typed dictionaries do not acquire JSON equality merely because a mapped type resembles JsonElement. More general typed and projected `uniqueItems` remains separate. Wrong-kind keys, malformed Boolean flags, absent validation opt-in and erased/discarded enforcing checks retain their existing rejection rules.

New JSON uniqueness participates in model equivalence and name-lock shape identity; legacy collection fingerprints remain unchanged. The original JsonObject-array rejection input is retained as an explicit positive regression, and historical fixture bytes are preserved.

The [synthetic envelope](../tests/fixtures/http/json-unique.json), core/private/native source and resource tests, and `bash scripts/test.sh feature json-unique` cover the profile. Fresh Data/Plain consumers run all four Json settings and verify duplicates, object key ordering, exact numbers, nested arrays, Unicode/null, Patch/aliases, depth/work/number limits, HTTP and SSE. This does not establish full JSON Schema uniqueness or Android/KMP acceptance.
