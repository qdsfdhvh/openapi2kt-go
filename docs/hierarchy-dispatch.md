# Compound and reused hierarchy tags

This profile extends [typed hierarchies](model-semantics.md#typed-discriminator-hierarchies) and [fixed/inherited fields](hierarchy-fields.md) with several inferred finite string tag fields and a discriminator wire name reused by nested explicit levels.

## Several inferred fields

An intermediate level without an explicit discriminator may promote several common varying string const/enum fields. Every promoted field gets its typed open peer enum and remains part of the parent interface. Const fields retain body constants; enum fields stay required non-null constructor arguments. Unknown implements all promoted and inherited properties with typed empty Unrecognized values while preserving raw JSON.

Dispatch requires a finite match for every promoted field. Values within one enum field use OR; fields combine with AND. This does not multiply source schema branches. Source field order controls condition order. All-const branches claim before branches containing enum fields; within each group the original branch order wins. Identical compound rules select the first claim. Overlapping enum/const rules follow this order, with no exact schema-match counting.

A known selected rule whose required payload is invalid fails immediately. It does not try a later matching rule. Missing, null or unrecognized primitive tag combinations select the current Unknown. Non-object payloads and nonprimitive tag fields retain existing active-dispatch failures.

## Reused wire names

An explicit nested level may reuse an ancestor discriminator wire name. It reuses that ancestor's existing enum type and emits one abstract override, rather than a second enum type or duplicate property. A missing leaf property uses the nearest matching explicit edge to choose its generated typed body. Actual source enum constructor and const rules remain distinct.

Each level keeps its own mapping routes. For example, an outer route `one -> Group` and inner routes `one -> First`, `two -> Second` allow direct inner decoding of `two`, while the outer decoder returns Unknown unless its own mapping also recognizes `two`. Encoding does not add an outer alias or rewrite the tag. Unknown below a reused level preserves raw JSON and implements that shared tag property once.

## Sources and limits

Rules borrow original source fields and catalog nodes. They retain canonical parent/leaf names, suffix-independent mapping wire values, source diagnostics and finite enum/constructor limits. Compound planning spends the bounded hierarchy work budget for each inspected finite tag set. Graph levels remain at most 48, each union and enum at most 240 members, and serialized constructors at most 240 slots.

Previously accepted one-tag hierarchy and object union/allOf output remains unchanged. Multiple parents, inline/primitive/nullable hierarchy members, nested inferred union descendants, general nominal ownership and exact discriminator schema validation still require separate contracts. A reused field does not grant multiple inheritance or nullable graph support.

The [original fixture](../tests/fixtures/http/hierarchy-dispatch.json), [core assertions](../core/hierarchy_dispatch_test.go) and [native preservation checks](../internal/cli/hierarchy_dispatch_process_test.go) cover compound tag combinations, const/enum overlap, duplicate first claims, canonical reused enums and nearest generated mappings. Fresh Kotlin profiles verify the applicable four Json, constructor, Unknown, HTTP/Patch/SSE and independent boundary behavior; this does not establish Android/KMP or every maximum-size combination.
