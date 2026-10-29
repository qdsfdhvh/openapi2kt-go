# Raw tuple defaults

An OpenAPI 3.1 positional array whose shared planner already selects
`List<JsonElement>` may own a field default or component default factory.
This includes mixed signatures without `items`, mixed signatures with
`items: false`, and every admitted prefix with `items: true`. Existing scalar,
enum and ordinary/allOf constructor reference members retain their original
source and unique declaration requirements, even when the final raw projection
discards their positional types. Direct inline member and graph admission is
unchanged.

```json
{"Rows":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],
         "default":[null,{"description":"data"},[1,true]]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":[false]}}
```

The actual generated List codec validates the annotation's carrier. A non-null
default must be an array; each element may be any JSON value, including JsonNull.
Literal keys such as `$ref`, `default` and `description` are data. Original JSON
tokens and object order are retained in the initializer. The annotation adds no
positional membership, enum membership or closed-tail length enforcement to the
existing raw projection. `W_TUPLE_APPROXIMATION` retains its original location and
tail-specific explanation. Numeric serialization retains the existing
[JsonElement codec limits](recursive-json.md#codec-and-numeric-boundaries).

Optional fields initialize from their own annotation through a typed private
List helper; required fields keep an explicit argument and Patch fields keep
Absent. Every own annotation validates before those initializers are suppressed.
Component and independently annotated alias factories return fresh decoded
values. Consumers and aliases without their own annotation inherit no default.
Nearest alias/field nullable intent retains the existing policy: a null container
default needs source nullability, while null inside the non-null JsonElement List
is a JSON value. Explicitly supplied optional null retains ordinary field policy.

Existing opt-in counts check both annotations and typed runtime consumers.
A literal true tail retains its [exact JSON uniqueness](json-unique.md) source
authority: objects ignore key order, arrays keep order, numeric decimal values
compare exactly, and strings/Booleans/null remain distinct. Annotation duplicates
fail at the second original `/default/index`; numeric precision/exponent limits
also apply to singleton annotations. Mixed prefix-only or false-tail projections
gain no new uniqueness source. Standalone aliases remain transparent rather than
runtime validation barriers.

Each whole literal retains the existing 16,384-node, 32 KiB JSON-byte and depth48
budgets before canonical containers or equality keys are allocated. Original
prefix/reference/source budgets, checked-depth16, JVM slots240 and output safety
remain. Every original prefix validates before an empty or null annotation is
consumed. Component factories require the original component node and Pointer;
same-byte copies gain no authority. Typed selected nominal defaults, nested child
annotations and broader default graphs keep their separate roles.

`bash scripts/test.sh feature tuple-raw-defaults` compiles/runs fresh Data/Plain
with validation off/on, baseline and combined dedup/hoist/merge with nonempty
name-lock entries. The [fixture](../tests/fixtures/http/tuple-raw-defaults.json),
[core checks](../core/tuple_raw_defaults_test.go),
[source checks](../internal/schema/tuple_raw_defaults_test.go),
[native checks](../internal/cli/tuple_raw_defaults_process_test.go) and
[consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleRawDefaultsContracts.kt)
cover four Json configurations, actual helpers/factories, null/data keys,
required/Patch/no inheritance, counts/uniqueness, controlled HTTP and fragmented
live SSE failure/closure/client reuse. Original rejection cases remain explicit
positive migrations. Bounded fuzz compares root and required-field annotation
validation, determinism and failure without partial artifacts. Other platforms
need independent execution.

**Migration:** previously rejected eligible raw tuple annotations now generate.
Already accepted tuple APIs, wire codecs, warnings, scalar default roles and
manifest/name-lock formats retain their contracts. This closes raw projected
tuple own annotations; general D03/D06 remains open.

[Inline enum prefixes](tuple-inline-enums.md) also validate before raw tuple defaults, with the same original vocabulary/path and cumulative enum-source budgets.

[Boolean tuple prefixes](tuple-boolean-members.md) retain raw data when the shared decision selects a true/mixed projection. Uniform false Lists instead use their actual rejecting classifier and admit only empty own array annotations.
