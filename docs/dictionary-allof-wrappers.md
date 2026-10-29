# Component-root dictionary `allOf` reference wrappers

A component root may contain exactly one local `allOf` reference to an
admitted named dictionary, together with any number of neutral members. A
neutral member is an empty object or an object containing only string
`title`, `description` and `$comment` metadata. These members may appear
before or after the reference; they do not change its value or nullability.
The reference may pass through finite named `$ref` aliases or other such
wrappers before reaching the original direct dictionary. For example:

```json
{
  "Texts": {"type": "object", "additionalProperties": {"type": "string"}},
  "TextView": {
    "allOf": [{"description": "A view"}, {"$ref": "#/components/schemas/Texts"}, {}]
  }
}
```

`TextView` is a Kotlin typealias to the existing dictionary backing; it does
not introduce an object envelope or a new serializer. Open dictionaries keep
their `JsonObject` representation. Typed dictionaries retain their actual value
types and any enum or DTO peers produced for the original target.

The wrapper's explicit `nullable` wins over the member's explicit
`nullable`, which wins over the nearest target or alias intent. A false view
of a nullable dictionary uses the non-null backing map or `JsonObject`, while
values within the dictionary retain their own nullable intent. Existing
consumer-level presence and Patch rules apply. An admitted target's own
default factory remains attached to that target; the wrapper and its fields
do not inherit it.

An eligible wrapper may also own a `default` and emit its own opt-in
`<Wrapper> default` factory. Finite named `$ref` aliases through such wrappers
may likewise own independent factories. Each annotation is validated against
the original terminal dictionary's existing component-default grammar, with
the nearest wrapper/member/alias nullable intent applied at the root. Target,
intermediate, wrapper and alias annotations are checked separately and are
never inherited by each other or by consuming fields. Invalid values report
the original owner's `/default` pointer, including escaped dictionary keys.
Typed Map and open `JsonObject` factories decode through the actual alias
serializer and return fresh structured values. Nullable dictionary terminals
with property-count constraints retain their existing default-role rejection.
The original 16,384-node and 32 KiB default budgets apply independently of
the wrapper source-capture limits below.

With `--validation`, existing dictionary counts and value checks propagate to
supported DTO construction/serialization, request Patch values and typed
HTTP request/response checks. A bare typealias assignment or standalone
reified dictionary serializer is not a validation barrier. Existing nullable
dictionary behavior outside this component-root wrapper profile is unchanged.

The generator retains the original wrapper, every member and its order, the
reference, target, nullable intent, declaration and checked backing. Each
wrapper contributes two levels to the existing 48-level source depth; checked
collection paths retain their 16-edge limit. Capturing the original wrapper
chains has an additional cumulative limit of 16,384 binding/node capture steps
and 8 MiB, including neutral members and overlapping targets. Transparent
cycles, excessive depth or capture work and changed source or declaration
bindings fail without generated output.

This profile does not implement general dictionary intersections: inline
dictionary members, additional references, members with explicit `nullable`,
other metadata or constraints, internal or remote
terminal references and other component kinds retain their separate contracts.
Previously rejected eligible wrappers now generate; already accepted schemas
keep their existing API, wire and persisted output formats. No dictionary
entries are initialized from a target default.

The [synthetic fixture](../tests/fixtures/http/root-dictionary-allof-wrappers.json),
[Go contracts](../core/root_dictionary_allof_wrapper_test.go) and
[independent defaults](../core/root_dictionary_allof_wrapper_defaults_test.go),
[native CLI preservation test](../internal/cli/root_dictionary_allof_wrapper_process_test.go)
and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RootDictionaryAllOfWrapperContracts.kt)
cover forward aliases, target factories, value peers, nullable views, checked
DTO/JSON/Patch and controlled HTTP behavior. Run
`bash scripts/test.sh feature root-dictionary-allof-wrappers` for fresh
Data/Plain × validation profiles and four Json configurations. This bounded
JVM evidence is not complete E02, platform or release acceptance.
