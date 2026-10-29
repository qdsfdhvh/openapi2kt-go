# Enum form fields

Selected flat `application/x-www-form-urlencoded` DTOs accept direct plain
String enum properties and direct native Boolean/Int/Long/Float/Double enum
properties, both required and optional. String vocabularies keep their actual
retained local generated declaration; native enums keep their existing scalar
mapping. The surrounding [closed form grammar](form-request-media.md), explicit
object root and actual DTO ownership remain.

```json
{"type":"object","properties":{
 "code":{"type":"string","enum":["ready","done"]},
 "count":{"type":"integer","enum":[1,2]}},"required":["code"]}
```

String enum fields encode `serialName`, including `Unrecognized` values. They
preserve the existing open vocabulary rather than using object `toString()` or
adding membership rejection. Native enum fields encode their ordinary mapped
scalar text. Native runtime membership/bounds remain opt-in with validation;
unknown native values retain the existing unchecked approximation and warning.
Mapped floating rounding/signed-zero rules remain those of the native checker.
Optional null omits only the key; required fields, empty serial names, false and
zero remain present. Original property order and wire names use Ktor UTF-8 form
encoding, independently of Json defaults/null settings.

Every original vocabulary and field source is validated before encoding.
The selected form role caps each raw String/native vocabulary at 240 members
with validation on or off, without changing the separate unchecked native
policy for other media/contexts. String caps report E_ENUM_LIMIT; native caps
report E_MODEL_LIMIT at the original `/enum` Pointer. Native field checks retain
the original raw source and actual constructor validation. No schema or enum
is synthesized for form encoding; the emitter uses only the retained fields.

Own defaults follow the [form default contract](form-field-defaults.md), with
strict source vocabulary membership and ordinary required suppression.
This grants no reference/alias, formatted String enum, const,
schema-nullable, structured or explicit Patch form role. A null enum member does
not infer source nullability. Source-depth/work, checked16 and JVMslots240 retain
their contracts. HTTP/cold SSE share encoding, body absence and final callback;
validation runs before dispatch/callback, failure closes streams and the client
remains reusable.

`bash scripts/test.sh feature enum-form-fields` generates Data/Plain with
validation off/on, baseline and combined optimized/locked output. The
[fixture](../tests/fixtures/http/enum-form-fields.json) and consumer test four Json
configurations, original/open String serial names, reserved/Unicode encoding,
Long extrema, mapped floating negative zero, optional null/empty/false, native
membership before dispatch, independent body absence, callback override, bare
HttpClient and live fragmented UTF-8 SSE failure/closure/cancellation/reuse.
Supplemental consumers compile String/native vocabularies of exactly 240;
source tests reject 241 and retain excluded roles and original model/AST binding.
Native tests preserve raw Long numbers while mutating sources, compare core
bytes/readonly/no-op and preserve old output on failure. Bounded fuzz covers
admission and determinism.

Combined optimization may expose an input DTO as an alias of a retained shared
model. Constructor aliases stay usable, while nested enum access follows the
actual classifier recorded in generated output; Kotlin aliases do not re-export
nested classifiers. The optimized consumer resolves that existing owner contract
and records the binding, without renaming production declarations. Additional
platforms and broader nominal form roles need independent evidence.

**Migration:** formerly rejected eligible direct enum forms now generate with
their existing Kotlin enum/scalar types and original serial/text wire values.
Accepted non-enum forms, JSON/raw output and persisted formats retain their
contracts. String openness, native opt-in checks and wider media/platform
boundaries remain.
