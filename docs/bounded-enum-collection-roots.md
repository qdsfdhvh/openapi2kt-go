# Lexical String enum bounds in collection roots

With `--validation`, component List/typed Map roots accept finite direct inline
paths ending in lexical String enums with `minLength`, `maxLength` and the
existing portable `pattern`. Single-layer and nested List/Map paths retain an
actual root typealias and an original enum peer, with existing nullable layers,
counts and structural uniqueness. Supported format/native metadata remains
lexical and creates no date, UUID, byte-array or provider dependency.

```json
{"Tree":{"type":"array","nullable":true,"items":{"additionalProperties":{
 "type":"string","enum":["one","two","x"],"minLength":2}},
 "default":[{"root":"one"}]},
 "Alias":{"$ref":"#/components/schemas/Tree","nullable":false,
 "default":[{"alias":"two"}]},
 "Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias",
 "default":[{"field":"one"}]}},"default":{"value":[{"outer":"two"}]}}}
```

Tree and Alias expose independent opt-in top-level default functions; Use's
field and component factory retain their own values. Consumers inherit no
factory annotation. A missing unannotated optional field remains null. Each
original field, component, intermediate alias and outer annotation validates
independently. Omitting a member of an object annotation leaves its actual DTO
initializer/required policy in control.

Every non-null default enum leaf must belong to the original vocabulary and
satisfy its String checks. Unknown runtime words remain invalid annotations.
Source schema/vocabulary/constraint errors retain their original keyword
Pointer, including when aliases own empty/null defaults. Value errors retain
the consuming `/default/...` Pointer and escaped map keys. Successful maps sort
keys after source-order validation. Child-owned defaults remain separate.

## Typed checks and open aliases

Shared DTO fields, required fields and Patch.Value retain recursive checks on
the actual lexical `serialName`. Known `X` is still an enum entry but fails a
checked consuming field's minLength; `Unrecognized("future")` passes. Required
annotations validate before initializer suppression; optional explicitly
nullable request fields retain Patch.Absent/Null/Value. Each container/leaf has
independent null intent, with the existing nearest-alias override rules.

Typed HTTP collection-root requests validate before sending; responses validate
after decoding. Known SSE collection data uses the same checks. An invalid
fragmented live SSE payload fails and closes the input; the caller client and
a new cold stream remain reusable. Counts and structural equality keep their
existing typed-tree semantics, including enum serialName and map order.

Transparent List/Map aliases and standalone enum codecs remain open: an
independent alias assignment or serializer decode is not a constructor check
boundary. Factory annotations validate during generation, and factories decode
fresh structured state using the final actual type. Caller mutation and request
builder replacement keep their existing limits. This matches existing primitive
collection-alias validation rather than introducing a nominal wrapper or a new
intrinsic serializer policy.

## Source ownership and scope

The original collection component owns the actual enum peer. Final validation
proves the original node, source bytes, unique alias/peer declaration, vocabulary,
linked backing/check addresses and final check values. Copying/removing peers or
mutating vocabulary/checks does not create a valid binding. Source/default trees
and effective checks participate in optimization and name-lock identities;
persisted formats remain unchanged.

Already admitted pure named aliases and ordinary named/internal collection
references consume the existing linked backing; this does not add checked enum
root/reference-leaf, tuple, union/hierarchy, recursive or child-default grammar.
Existing ordinary object-default collection graph walks also accept independent
original lexical enum collection/alias factories, including eligible unbounded
controls. Every referenced graph use retains its final actual-backing check.
Internal/unmarked/copied nodes gain no component annotation ownership.

Source48, checked collection16, source/value16384, default32KiB, enum240,
constructor240 and input8MiB budgets remain. Validation-off rejects the original
constrained input. Plain enum null vocabularies and metadata keep their existing
separate grammar. No compiler heap, pattern dialect or runtime timeout changes.

**Compatibility:** eligible formerly rejected collection roots/default graphs
now generate. Existing accepted APIs/wire and manifest/name-lock formats retain
their contracts. Regenerate and compile consumers of new inputs. This extends
D01/D03/D05/M03/M04/M07/M08/M12 and typed HTTP/SSE consumers; general graph,
platform and final candidate acceptance remain separate.

[Core roots](../core/bounded_enum_collection_roots_test.go),
[independent values/source guards](../core/bounded_enum_collection_roots_guards_test.go),
[bounded fuzz](../core/bounded_enum_collection_roots_fuzz_test.go),
[original backing/peer checks](../internal/schema/bounded_enum_collection_roots_test.go)
and [native preservation](../internal/cli/bounded_enum_collection_roots_process_test.go)
define the profile. The existing `string-enum-collection-bounds` feature generates
fresh Data/Plain checked and unbounded-off controls, actual factories and
consumers, four Json settings, Patch, optimization/locks, independent resource
probes and direct/shared HTTP/fragmented SSE. Prepare-only proves generation and
checks, not Kotlin compilation or execution.

[Checked lexical enum borrowing](borrowed-checked-string-enums.md) separately admits already retained ordinary field/collection-root sources through existing references, with the original enforcing checks and independent eligible defaults. Standalone checked enum components and other source roles keep their own eligibility.
