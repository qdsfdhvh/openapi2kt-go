# Referenced String enum defaults with null vocabulary members

The [referenced String enum scalar](referenced-string-enum-scalar-defaults.md) and [referenced String enum collection](referenced-string-enum-collection-defaults.md) field-own default roles also accept an already admitted plain String/typeless enum vocabulary with one null member. The original enum schema must explicitly be nullable and retain at least one String member, following the [String enum null-member contract](string-enum-null-members.md). Both validation modes use the same grammar.

```json
{
  "Status": {"type": ["string", "null"], "enum": [null, "on", "null"]},
  "Rows": {"type": "array", "items": {"$ref": "#/components/schemas/Status"}},
  "Holder": {"properties": {
    "status": {"$ref": "#/components/schemas/Status", "default": "on"},
    "rows": {"$ref": "#/components/schemas/Rows", "default": [null, "on", "null"]}
  }}
}
```

Only ordinary direct local `$ref` fields' own annotations gain this composition, through already accepted scalar or finite List/typed Map paths. Named components, internal references, aliases and actual lexical classifiers retain their resolver contracts. The physical enum contains only String members and `Unrecognized`; its serializer remains non-null. Null uses the actual nullable typed view and is not an enum object. The String literal `"null"` remains a separate known member only when present in the original vocabulary.

Non-null annotations remain strict against the original String vocabulary, even though supplied unknown runtime Strings stay open. A null vocabulary token does not infer nullable intent or widen a narrowed use: scalar `nullable:false` rejects a null annotation, and a narrowed List element rejects null default members. Container and leaf nullability remain independent. Duplicate null/String values, malformed/non-null source vocabularies and raw 240-member limits retain original diagnostics, including original `/enum/<index>` locations. The consuming default retains its original escaped `/default` member Pointer.

Optional missing fields use their known member or fresh typed tree; explicit null remains null when decoded. Required and Patch annotations fully validate before initializer suppression, preserving required arguments and Patch.Absent/Null/Value. Discarded ordinary allOf annotations validate before override. With `explicitNulls=false`, encoding may omit an explicit null and later decoding can reapply a non-null default; this existing behavior does not imply an object round-trip identity guarantee. Validation retains existing container counts and does not close runtime String vocabularies.

Source nodes and actual declarations retain their existing authority and nominal fingerprint boundaries. Removing a redundant source null from an already nullable enum retains the same classifier/default output. Canonical effective defaults participate in existing optimization/name-lock identity, while suppressed annotations add no runtime default identity. Persisted formats, output preservation and input8MiB/source48/checked16/work16,384/slots240/raw-enum240/default32KiB limits remain unchanged. No new resolver admission, emitter behavior, implicit-null inference, formats/bounds/patterns, target/chain/root/wrapper/child defaults or union/hierarchy field roles are added.

The [fixture](../tests/fixtures/http/referenced-string-enum-null-member-defaults.json), [core regression/guards/identity/budgets](../core/referenced_string_enum_null_member_defaults_test.go), [original source and narrowed view test](../internal/schema/referenced_string_enum_null_member_defaults_test.go) and [native parity/lock/preservation test](../internal/cli/referenced_string_enum_null_member_defaults_process_test.go) define the increment. `bash scripts/test.sh feature referenced-string-enum-null-member-defaults` generates fresh Data/Plain × validation projects for four Json settings, actual classifiers/aliases and descriptors, scalar/List/Map/nested defaults, fresh trees, required/Patch/copy, open unknown Strings, controlled HTTP and finite SSE decoding/reuse, combined optimization/locks, merged nested enum owner rebinding and independent 240-field/239-String-plus-null compilation probes. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected field-own defaults in these existing reference roles now generate. Historical accepted inputs retain their contracts; general D03/D04, wider platform acceptance and release gates remain open.
