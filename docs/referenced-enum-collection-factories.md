# Enum-reference collection component factories

An already accepted finite List/typed Map component whose source tree references
an existing lexical String or native numeric/Boolean enum may own `default`.
Pure named collection aliases may own independent annotations, including a null
factory when their nearest nullable view permits null. Inter-container named
references compose with these existing finite paths.

```json
{"Word":{"enum":["one","two"],"default":"one"},
 "WordAlias":{"$ref":"#/components/schemas/Word","default":"two"},
 "Rows":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias"},
         "default":["one"]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":["two"]},
 "Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},
 "Use":{"properties":{"rows":{"$ref":"#/components/schemas/Alias",
                               "default":["one"]},
                      "unannotated":{"$ref":"#/components/schemas/Rows"}}}}
```

With suffix `Dto`, the actual collection aliases gain top-level
`` `RowsDto default`() `` and `` `AliasDto default`() `` functions. They decode
their own canonical JSON through the retained collection/leaf serializer and
return fresh container state on each call. The nullable factory returns null.
The ordinary field initializer uses its own annotation; the unannotated field
keeps normal optional presence and inherits no factory value. Required and Patch
annotations validate before initializer suppression. Object/default-tree
consumers retain their own strict annotation and actual enclosing codec.

Each original enum component, enum alias, collection root, collection alias and
consumer annotation validates independently. Non-null String values must belong
to the original vocabulary; native numeric membership uses exact source values
before Float/Double rounding. Nullable intent controls each actual layer.
A String enum's supported format/native metadata remains lexical. Checked String
bounds and native/container checks retain their existing validation opt-in.

Ordinary named native enum roots with no checks keep their existing scalar
exports in both validation modes. An unchecked collection factory proves that
actual original scalar declaration rather than acquiring permission for a
borrowed or internal native enum graph. Internal, intrinsically nullable and
other reference profiles retain their existing admission requirements.

Runtime String codecs remain open to `Unrecognized`; native alias codecs remain
scalar codecs. Defaults do not make these codecs intrinsically strict. Actual
typed consumers continue to enforce the checks admitted by validation. Empty or
null outer annotations never bypass malformed original sources or an invalid
independent factory. Value errors retain the annotating owner's escaped
`/default/...` Pointer; source errors retain their original keyword Pointer.

A factory requires the original root/leaf schema paths, unique retained export,
canonical binding, actual backing/check addresses, canonical initializer,
nullable intent and cached alias height. Unchecked native leaves additionally
retain the actual scalar export, original vocabulary and scalar mapping.
Missing proof records, duplicate or replaced declarations, changed bindings,
backings or initializers reject before output changes. Shared original
source48/work16,384/default16,384-nodes/32KiB/checked16/enum240/JVM240 budgets remain.

An independently validated default on an eligible physical enum-reference
collection child follows the separate [child annotation contract](collection-reference-child-defaults.md).
No recursive, tuple, union/hierarchy, const, provider or general composition graph
is admitted by this factory profile. Existing independent component annotations
are retained when accepted field/object walks consume their types; values are
never inherited from those annotations.

**Compatibility:** eligible formerly rejected factory inputs and independently
annotated enum sources now generate. Existing accepted APIs, wire behavior,
name-lock formats and output ownership formats retain their contracts. Compile
regenerated consumers when adopting the new factories. This extends D03/E03
with existing F04/F07/M03/M04/M07/M08/M12 consumers; wider default graphs,
complete platform matrices and final acceptance remain separate.

[Core contracts](../core/referenced_enum_collection_factories_test.go),
[bounded fuzz](../core/referenced_enum_collection_factories_fuzz_test.go),
[actual factory/source authority](../internal/schema/referenced_enum_collection_factories_test.go)
and [native preservation](../internal/cli/referenced_enum_collection_factories_process_test.go)
define the profile. The `string-enum-collection-bounds` fixture/consumer includes
String and all five native scalar factories, independent annotations, fresh
containers, four Json settings, Data/Plain profiles, optimization/name locks and
existing typed HTTP/SSE checks. A test entry or prepare-only generation is not
compilation/runtime evidence; verify the exact generated candidate.

The [platform consumer](platform-enum-field-bounds.md) separately exercises the
String and five native factory mappings on JVM, JS/Node and macOS ARM64 Native,
and compiles them for common metadata and Android AAR. That evidence covers its
exact fixture and options, not all factory contexts or Android device runtime.
