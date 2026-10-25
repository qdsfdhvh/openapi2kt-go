# Floating form fields

Selected flat `application/x-www-form-urlencoded` DTOs accept ordinary direct
Float and Double properties, both required and optional. An unformatted
`type: number` keeps the existing Double mapping; `format: float` and
`format: double` retain their ordinary scalar mappings. Other selected form
source restrictions remain in the [form contract](form-request-media.md).

```json
{"type":"object","properties":{
 "floatValue":{"type":"number","format":"float"},
 "doubleValue":{"type":"number"},"optionalDouble":{"type":"number"}},
 "required":["floatValue","doubleValue"]}
```

`Input(-0.0f,1.25)` sends `floatValue=-0.0&doubleValue=1.25`. A non-null optional
zero remains present; optional null omits only its key. Required/optional body
presence remains independent. Ktor encodes the standard scalar `toString()`
result using the existing UTF-8 form encoder, preserving property order and wire
names. This text profile adds no JSON number codec, formatter or decimal
precision: Float/Double rounding, negative zero and exponent spelling remain
those of the mapped runtime scalar.

Unconstrained floating constructor values retain their existing behavior:
standard text such as `NaN` and `Infinity` can be sent. Enabling validation does
not create a global finite-number requirement. A field with admitted numeric
constraints keeps the existing typed finite/bounds/multipleOf checks, including
before callback/dispatch; optional null skips those checks. No source
default, explicit schema-null, native, structured or Patch form role is granted.
The existing required/default, JSON precedence, discarded-schema validation,
callback, HTTP/SSE and failure-preservation contracts remain.

`bash scripts/test.sh feature floating-form-fields` generates Data/Plain with
validation off/on in baseline and combined optimized/locked packages. The
[fixture](../tests/fixtures/http/floating-form-fields.json) and consumer test four
Json settings, exact form text for finite values/negative zero/optional zero,
existing unconstrained nonfinite text, UTF-8 labels, independent body absence,
HTTP/cold live fragmented SSE failure/closure/cancellation/client reuse, bare
HttpClient encoding and final callback override. Validation-on feature inputs
add bounds to an optional Double and verify invalid/nonfinite supplied values
fail before callback/dispatch. A separate 120-required-Double DTO reaches
JVMslots240; Go tests reject 121 required Doubles and retain excluded grammars.
Native tests compare core bytes, readonly/no-op and old-output preservation;
bounded fuzz covers original source admission and determinism. Other platforms
and broader media require independent evidence.

**Migration:** formerly rejected ordinary floating form fields now generate
using their existing Kotlin types and ordinary textual wire representation.
Existing required/optional integer/Boolean/String forms, JSON/raw output and
persisted manifest/name-lock formats retain their contracts.

Direct String/native enum fields have the separate [enum form contract](enum-form-fields.md);
open String serial names and opt-in mapped native membership retain their policies.
