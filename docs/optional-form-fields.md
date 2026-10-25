# Optional flat form fields

Selected `application/x-www-form-urlencoded` request DTOs may contain optional
ordinary non-null String, Boolean, Int, Long, Float or Double source properties. A property
outside `required` has a nullable Kotlin constructor argument defaulting to null;
null omits that key from the form. Required fields retain their original API.

```json
{"type":"object","properties":{"text":{"type":"string"},
 "count":{"type":"integer"},"active":{"type":"boolean"}},
 "required":["text"]}
```

A value such as `Input(text="x", count=null, active=false)` sends
`text=x&active=false`. Empty String sends `key=`, false sends `key=false`, and zero
sends `key=0`. Fields retain original schema order and original wire names, with
Ktor's UTF-8 form encoding for reserved characters, spaces and Unicode. Optional
values are captured in local nullable-value callbacks; reserved/normalized Kotlin
property names do not affect the wire name.

A supplied all-optional DTO with every field null sends a present empty form
body and the form Content-Type. For an optional whole body, an omitted/null DTO
still sends no content and no generated Content-Type. These two levels remain
independent. HTTP and cold SSE use the same emitter, and the final
requestBuilder callback can override the generated content. ContentNegotiation
is unnecessary for form encoding; response requirements retain their contracts.

Optional properties reuse existing typed constructor/codec validation. Absent
nulls skip field checks; supplied values satisfy the retained constraints before
callback/dispatch when validation is enabled. Explicit schema-nullable fields
would need wire-null/Patch semantics and remain rejected, including `type` arrays
containing null. Default, reference, formatted nonbuiltin, structured,
projection and media encoding roles keep the [form grammar](form-request-media.md).
JSON media precedence and validation of discarded/excluded schemas remain.

`bash scripts/test.sh feature optional-form-fields` checks Data/Plain with
validation off/on, baseline and combined optimized/locked output, four Json
configurations and 240 optional fields. The
[fixture](../tests/fixtures/http/optional-form-fields.json) and consumer exercise
exact omission/empty/false/zero/Unicode bytes, reserved `class`/`it` names,
required fields, independent body presence, bare HttpClient encoding, JSON
precedence, callback override, validation before dispatch and live fragmented
SSE close/cancellation/reuse. Native tests compare core bytes and readonly/no-op
output and preserve old files for invalid source grammars. Source tests retain
excluded nullable/default/Patch roles, and bounded fuzz compares source admission
with required fields. Other platforms need independent validation.

**Migration:** previously rejected eligible optional form properties now
generate with explicit omitted-key behavior. Existing required fields/forms,
JSON/raw output and manifest/name-lock formats retain their contracts. Wider
form/media/projection/platform work remains open.

Floating values keep their [ordinary form text contract](floating-form-fields.md),
including mapped IEEE representation and existing opt-in checks.

Direct String/native enum fields have the separate [enum form contract](enum-form-fields.md);
open String serial names and opt-in mapped native membership retain their policies.
