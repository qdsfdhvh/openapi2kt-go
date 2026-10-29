# Form field defaults

Flat `application/x-www-form-urlencoded` request DTO properties may declare their
own `default` annotation on the existing direct String, Boolean, Int, Long,
Float, Double, Uuid, LocalDate, Instant and plain String/native enum sources.
The [form grammar](form-request-media.md) still determines source admission;
adding a default does not admit references, nullable schema fields, const,
structured fields or a different body shape.

Each annotation uses the same literal parser, original vocabulary, mapped
scalar codec and opt-in checks as the existing model field default. Invalid
types, null on a non-null source, invalid dates/UUIDs, out-of-range numbers and
nonmember enum defaults fail at the original `/default` Pointer. Enum annotations
use strict original membership even when runtime String enums remain open or
native membership is unchecked. Required fields and excluded operations still
validate their annotations before suppressing initializers or emitted APIs.

For example:

```json
{"type":"object","properties":{
  "token":{"type":"string","default":"annotation-only"},
  "count":{"type":"integer","default":0}
},"required":["token"]}
```

The constructor still requires `token`; its annotation supplies no argument.
Omitting `count` initializes the actual getter to `0`, which sends `count=0`.
Explicitly passing Kotlin null for `count` omits that key. A field with no own
default retains its existing null/omitted-key behavior. A missing or null optional
whole body remains absent, independently of field defaults. No annotation is
inherited from another source.

The HTTP/SSE encoder uses the actual retained getters and unchanged UTF-8 form
encoding. It does not inspect source annotations or the caller's Json settings.
`encodeDefaults=false` does not omit a default-valued form field; `explicitNulls`
does not send a null form value. UUID/temporal defaults use their actual codecs
and canonical text, String enums use `serialName`, and native scalars use their
mapped text. JSON serialization retains its existing default/null rules.
The final request callback may still replace generated content.

`bash scripts/test.sh feature form-field-defaults` compiles and runs freshly
generated Data/Plain sources with validation off/on, baseline and combined
dedup/hoist/merge/locked output, four Json configurations, all admitted scalar
families, missing/explicit-null/explicit-value fields, required suppression,
optional-body absence, callback replacement and cold SSE requests/client reuse.
The [fixture](../tests/fixtures/http/form-field-defaults.json),
[core regressions](../core/form_field_defaults_test.go),
[native checks](../internal/cli/form_field_defaults_process_test.go) and
[consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FormFieldDefaultsContracts.kt)
also cover strict diagnostics, determinism, core/CLI parity and failure preserving
existing output. Former default refusals remain executable positive migrations;
other form source guards remain. Original source, work, default and JVM limits
and output/name-lock formats are unchanged.

**Migration:** previously rejected eligible form annotations now generate.
Previously accepted form APIs/wire behavior and JSON/raw bodies retain their
contracts. This closes the direct field-own default role within D03/D07; wider
default graphs, HTTP projections and platform acceptance remain separate.
