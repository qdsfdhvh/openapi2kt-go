# Signed-byte leaves in default trees

[Referenced byte field own defaults](referenced-bytes-field-defaults.md) separately accept ordinary `$ref` fields' annotations through existing byte/binary scalar and finite List/typed Map paths, retaining actual sources, required/Patch and whole-annotation budgets.

Existing finite collection and ordinary object default trees accept direct `type: string` leaves with `format: byte` or `binary`. They use the same `ByteArray` serializer as [direct bytes fields](bytes-field-defaults.md): the default is a JSON array of signed integers in `[-128,127]`.

```json
{"type":"array","items":{"type":"string","format":"byte"},"default":[[-128,0,127],[]]}
```

The field remains a `List<ByteArray>` with its existing optional-null view. A private typed JSON helper creates its initializer. Each construction and each factory call decodes fresh mutable arrays; defaults do not share a mutable byte buffer. No Base64 codec is introduced.

[Finite byte/binary alias chains in object-default members](nested-object-defaults.md#bytes-scalar-alias-members) additionally preserve original hops and nearest nullable intent through already admitted named byte roots.

## Accepted roles and presence

- An ordinary direct inline DTO collection field's own outer default can contain byte leaves through its existing finite List/typed Map tree. Root containers and byte leaves have independent explicit nullable intent.
- Existing ordinary object-default walks can contain byte-valued properties, including a direct reference to an already accepted byte component. Actual DTO/alias serializers and original source nodes remain authoritative.
- The [finite nested component List/Map factory role](nested-component-defaults.md) accepts byte leaves with independently nullable containers and leaves. Other component and reference-wrapper admission stays separate. Consumer fields never inherit a target factory's default.

Required fields still have no constructor initializer. A default on a request-only nullable collection does not replace `Patch.Absent`; `Patch.Null` and `Patch.Value` retain their wire states. Earlier allOf annotations are still validated before being discarded. A supplied value overrides a default; an explicit null is retained where the public optional-null field permits it. Annotation null requires explicit schema nullability.

A tree containing a non-null ByteArray is normally written even with `encodeDefaults=false`: arrays compare by reference, including when contained in Lists, Maps or DTOs. Empty Lists/Maps and null defaults retain their existing comparable-default omission behavior. Data and Plain styles keep the same array equality policy. Mutating a default after construction changes that instance's value only; no new mutation validation is provided.

## Validation and bounds

Each byte uses the existing integer-token/range grammar. Base64 strings, fractional/exponent tokens, null/string byte elements and values outside `[-128,127]` fail with `E_SCHEMA_INVALID` at the original value Pointer, such as `/default/0/2` or `/default/a~1~0/0`. A nullable byte leaf can be null; an array containing null bytes cannot.

Schema validation precedes empty/null data. Child/element own defaults, byte `minLength`/`maxLength`/pattern, native/enum/const/composition permission and ByteArray `uniqueItems:true` are not acquired from this extension. Existing collection counts remain opt-in and apply independently at their own container.

The whole annotation retains the original 16,384 JSON value-node and 32 KiB compact JSON budgets, including byte arrays and every integer inside them. Canonical JSON is independently bounded. Required/Patch/discarded defaults use the same budgets. Original input/source-depth 48, checked collection-depth 16 and constructor-slot 240 limits are unchanged. Named/referenced collection own defaults, arbitrary alias chains, root nullable-null factories and broader default roles still use their separate contracts.

## Migration and evidence

Previously rejected eligible byte-leaf trees now generate. Existing schemas without defaults keep their APIs, serializers and output; direct byte defaults keep their dedicated initializer and diagnostics. Typed HTTP/SSE DTOs continue using signed arrays through the caller's Json codec; raw byte response policy is unchanged.

[Core contracts](../core/bytes_default_trees_test.go) preserve the exact original array/Map/component rejection inputs as positive cases and exercise original pointers, counts and exact value/byte boundaries. [Native CLI tests](../internal/cli/bytes_field_defaults_process_test.go) cover check read-only behavior and output preservation on bad tree elements. The [fixture](../tests/fixtures/http/bytes-field-defaults.json) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BytesDefaultTreeContracts.kt) cover fresh arrays/factories, missing/null/value, required/Patch, four Json settings, copy, typed HTTP, real fragmented SSE failure closure and client reuse.

Run `bash scripts/test.sh feature bytes-field-defaults` for fresh Data/Plain × validation-off/on compilation and runtime. Validation profiles add independent outer container counts; the complete `http` registry also invokes the consumer. These JVM checks do not establish other-platform or release-candidate acceptance.
