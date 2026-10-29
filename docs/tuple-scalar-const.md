# Scalar const positional-array sources

OpenAPI 3.1 positional members and explicit schema tails accept existing scalar
const sources, including admitted named references and pure aliases. Builtin
scalars, supported numeric formats, UUID, date/date-time and existing native date/
bigint mappings retain their original const grammar and actual context codecs.
JsonNull uses its [singleton contract](tuple-null-sources.md); structured const,
union and general recursive sources remain independent D04/D06 work.

```json
{"Only":{"const":"fixed"},
 "Rows":{"type":"array","prefixItems":[{"const":"fixed"}],"default":["fixed"]},
 "Referenced":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Only"}]}}
```

Rows and Referenced use `List<String>`. Native const element planning borrows the
actual existing `valueConstant` codec only at an original admitted tuple slot;
ordinary native/reference grammar is unchanged. Const references retain their
actual scalar export/backing. The generator creates no enforcing wrapper.

## Projection and defaults

Same original const terminal and effective nullable view can be uniform. Distinct
inline const nodes, different pins or a const versus an ordinary scalar remain
mixed raw projections, even when scalar spellings match. An explicit tail wins
after all original prefixes validate. Source identity and lexical provenance
remain independent of later optimization.

Runtime const membership remains approximated by the scalar codec, with existing
`W_CONST_SCALAR_APPROXIMATION` for retained value sources. The tuple warning also
explicitly identifies lost scalar const prefix membership, including raw or tail
projection. A typed String list may decode a different string. Validation does
not add const membership checks. Position/length and unrestricted-tail behavior
retain the existing List approximation and warnings.

Outer field/root/independent alias defaults are strict against the original const
literal before mapped Float precision or native wire conversion. A Float pin
16777216 does not permit default16777217 merely because both map to the same
Float. Numeric equality reuses the original bounded exact-number checker, then
the value must fit the actual mapped codec. UUID/date/time defaults retain source
lexemes and codec grammar. Native bigint string pins require the exact original
string annotation and canonicalize the helper payload to the actual Long wire.
Nullability follows the actual nearest view; null is not admitted by an unrelated
optional container. Required/Patch annotations validate before initializer
suppression, and no target or child annotation is inherited.

## Original sources and limits

Null and scalar const roles share the private literal-source registry. Original
ancestor/reference/terminal/literal nodes, catalog bindings, actual scalar or
transparent alias backings, selected element addresses and immutable type/check
snapshots remain authoritative. Same-byte copied nodes, missing source proof,
changed classifiers/backings and duplicate codecs reject.

Original reference hops remain in cached source heights after raw/tail erasure.
Cumulative capture work includes each retained original catalog-binding record:
separate uses sharing a long alias chain pay their overlapping snapshots. Inputs
above this budget fail `E_MODEL_LIMIT` while preserving existing output. Ordinary
non-tuple collections retain their separate accounting. Source48, capture16384/
8MiB, default16384 values/32KiB, checked16 and JVM240 limits remain independent.

Eligible formerly rejected inputs now generate. Existing accepted API/wire and
manifest/name-lock formats retain their contracts, subject to the explicit
resource failure above. No performance, whole-platform or current-candidate
acceptance is implied.

[Core types/defaults/budgets](../core/tuple_scalar_const_test.go),
[private source/codec proof](../internal/schema/tuple_scalar_const_test.go),
[structured fuzz](../core/tuple_scalar_const_fuzz_test.go),
[native preservation](../internal/cli/tuple_scalar_const_process_test.go),
[synthetic fixture](../tests/fixtures/http/tuple-scalar-const.json) and
[Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleScalarConstContracts.kt)
define this profile. `bash scripts/test.sh feature tuple-scalar-const` compiles and
runs Data/Plain × validation, baseline/combined optimized output with nonempty
hoist locks, 240 independent fields, four Json settings, defaults, nonliteral
runtime values, required/Patch, direct/shared HTTP and fragmented live SSE
failure/closure/client reuse. The complete `http` registry includes the consumer;
PREPARE_ONLY is preparation, and final all/platform evidence remains separate.

[Structured const tuple sources](tuple-structured-const.md) separately admit original JsonObject/JsonArray positions and existing references, with strict complete-literal outer annotations and data-only capture of const/default/enum payloads. Original identity, nullable/Patch and cumulative budgets remain; broader union/recursive/default graphs stay independent.
