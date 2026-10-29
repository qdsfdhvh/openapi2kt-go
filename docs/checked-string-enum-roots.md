# Checked lexical String enum components

With `--validation`, a direct lexical String enum component may retain
`minLength`, `maxLength` and portable `pattern`. Named aliases, existing typed
references and finite List/Map consumers share its original String check object
and actual enum declaration. Supported format/native metadata remains lexical;
it creates no date, UUID or provider dependency for this model enum.

```json
{"Word":{"type":["string","null"],"format":"date",
 "enum":["one","two","x"],"minLength":2,"maxLength":6,
 "pattern":"^[a-z]+$","default":"one"},
 "Alias":{"$ref":"#/components/schemas/Word","nullable":false,"default":"two"},
 "Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},
 "Use":{"properties":{"word":{"$ref":"#/components/schemas/Alias",
 "default":"one"},"rows":{"type":"array",
 "items":{"$ref":"#/components/schemas/Alias"},"default":["two"]}}}}
```

The canonical Word classifier keeps its open lookup and serializer. Known `X`
and `Unrecognized("future")` remain serializable independently; checked typed
uses reject the short `x` and accept a matching unknown word. Enum assignment,
standalone alias decoding and caller collection mutation do not become intrinsic
validation boundaries. Ordinary DTO construction, Data copy, automatic decoding,
Patch values and admitted typed collection/HTTP/SSE consumers enforce checks on
the actual `serialName`. Existing serialName uniqueness remains unchanged.

Each root, alias factory, consuming field and object/default-tree annotation is
independent. Non-null annotations must belong to the original vocabulary and
satisfy its checks; a valid unknown runtime word is still an invalid default.
Factories remain opt-in and decode their own canonical JSON through the actual
classifier/alias. Unannotated consumers inherit no value. Required/Patch fields
validate annotations before suppressing constructor initializers.

Source null grammar remains separate from each consuming view. The nearest
explicit alias/field nullable intent wins; null skips only that layer's checks.
The original null-vocabulary and inference policies remain: bounds grant no new
null inference. A nullable component's own null factory returns null without
making the physical enum class nullable. Each malformed schema reports its
original keyword Pointer, including when another owner has an empty/null default.
Value failures report the annotating owner and exact value Pointer.

A root requires its unique actual enum declaration even with no consumers.
The analyzer retains its original component/node/bytes/vocabulary/check identity
and enforcing TypeRef. The original retained DTO getters keep their actual
type/check addresses too.
Forward named collection slots may fill their originally allocated zero-value
backing before the existing link checkpoint; the original root's recorded
deferred nominal may resolve its actual name there too. Other populated nodes
cannot be replaced or rewritten. Full getter snapshots become final there, retaining
the shared checks through actual collection consumers. Root and alias factories
retain their original annotation, canonical initializer, actual export, binding,
nullable intent, type and cached
alias height. Dropping a check, replacing a declaration/binding, duplicating an
export, changing an initializer or losing a required factory record rejects
generation before output changes. Existing original source/work/value budgets,
48-hop traversal, checked depth16, default32KiB, enum240 and JVM240 caps remain.

This profile adds no const/value-class/composition/intersection grammar, child
annotation role or general default graph. Validation-off still rejects these
constrained sources. Portable pattern and existing runtime timeout/heap policies
are unchanged. Typed HTTP scalar/collection bodies and responses preserve
preflight/postflight checks; known fragmented SSE data retains failure-close and
client/fresh-stream reuse contracts. Standalone codecs remain open.

**Compatibility:** eligible formerly rejected roots, aliases and their existing
typed consumer/default roles now generate. Existing accepted APIs, wire behavior
and persisted ownership/name-lock formats retain their contracts. Regenerate and
compile new consumers. This extends D01/D03/D05/F04/F07/M03/M04/M07/M08/M12;
general graphs, complete platform matrices and final acceptance remain open.
Adding or changing checks changes locked occurrence identity: an old
unconstrained lock reports `E_NAME_LOCK_CHANGED`, while a fresh lock may accept
the checked source. A failed locked candidate preserves its original output.

[Core/source/budget contracts](../core/checked_string_enum_roots_test.go),
[bounded fuzz](../core/checked_string_enum_roots_fuzz_test.go),
[actual declaration/check/factory authority](../internal/schema/checked_string_enum_roots_test.go)
and [native failure preservation](../internal/cli/checked_string_enum_roots_process_test.go)
define this profile. The existing `string-enum-collection-bounds` feature adds
fresh Data/Plain validation-off controls and checked consumers, four Json
configurations, default/null factories, required/Patch, optimizations/locks,
independent 240-root-getter compilation and direct HTTP/SSE assertions.
Compilation/runtime evidence applies only to the exact generated consumer;
prepare-only and old platform outputs do not prove new-feature runtime support.

[Enum-reference collection component factories](referenced-enum-collection-factories.md) retain independent root/alias annotations and existing enum-component annotations through accepted finite List/typed Map paths. Actual serializers, strict original defaults, nearest nullable, required/Patch and original source/export/backing/check/height proof remain; no inheritance or child role is added. General graphs, complete platforms and final acceptance remain open.
