# Enum-reference collection factories

Use this reference when a finite List/typed Map component references an existing
String or numeric/Boolean enum and declares its own `default`, or when a pure
named alias of that collection declares another default.

```json
{"openapi":"3.1.0","components":{"schemas":{
 "Word":{"enum":["one","two"],"default":"one"},
 "Rows":{"type":"array","items":{"$ref":"#/components/schemas/Word"},
         "default":["one"]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":["two"]},
 "Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null}
}}}
```

With suffix `Dto`, call `` `RowsDto default`() `` or
`` `AliasDto default`() `` explicitly. Each decodes its own canonical JSON into
fresh containers using the actual retained codec. The nullable factory returns
null. An unannotated DTO field keeps its normal presence/default behavior;
component factories are never inherited. Field and object defaults validate
independently, including before required/Patch initializer suppression.

Check non-null defaults against the original enum vocabulary. String runtime
codecs remain open, and native numeric/Boolean codecs keep scalar wire values.
Use `--validation` for supported String/native/container checks; ordinary
unchecked named native roots retain scalar factories without granting new
internal or borrowed graph admission. Nearest nullable intent controls each
layer. Eligible physical enum-reference child annotations follow the separate
[child-default guidance](collection-reference-child-defaults.md).

Run `validate`, then `generate` and `generate --check` with the same input/options.
Compile the freshly generated Kotlin and exercise the actual factories and
missing/null serialization settings. Report generation and compilation/runtime
results separately. General recursive/composed/default graphs, complete platform
coverage and release acceptance do not follow from this bounded profile.
