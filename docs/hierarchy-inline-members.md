# Hierarchy inline members

A `oneOf`/`anyOf` hierarchy branch is normally a `$ref` to a named component. An **inline** branch has no component name, so this contract derives one from the branch's own discriminator constant.

```json
{"Shape": {
  "oneOf": [
    {"type": "object", "properties": {"kind": {"type": "string", "const": "circle"}, "r": {"type": "integer"}}, "required": ["kind", "r"]}
  ],
  "discriminator": {"propertyName": "kind"}
}}
```

```kotlin
public sealed class ShapeDto { public abstract val kind: ShapeDtoKind }

public data class CircleDto(public val r: kotlin.Int) : ShapeDto() {
    @EncodeDefault(ALWAYS) @SerialName("kind")
    public override val kind: ShapeDtoKind = ShapeDtoKind.Serializer.fromSerialName("circle")
}
```

The result is **identical to the `$ref` spelling** of the same leaf, so a call site cannot tell the two apart.

## Why the constant names the class

The alternative — naming by `oneOf` position (`Member0`, `Member1`) — is machine-simple but **reorders rename the class**, which breaks Kotlin call sites. Deriving the name from the discriminator constant makes the name stable under reordering, and reuses the same vocabulary the `$ref` form already uses (`circle` → `CircleDto`).

A test asserts that reordering the branches produces byte-identical leaf classes.

## Name sources

| Branch tag | Class name | Tag on the leaf |
|---|---|---|
| `{"type":"string","const":"circle"}` | `CircleDto` | fixed `fromSerialName("circle")`, always written |
| `{"type":"string","enum":["only"]}` (single member) | `OnlyDto` | required typed constructor argument |

The second row matters: a vocabulary leaf keeps the **required** constructor argument, exactly as the `$ref` spelling does. Only a `const` inlines a fixed value. Two branches resolving to the same class name fail with `E_NAME_COLLISION` instead of overwriting one another.

## What still requires a separate contract

An inline branch with no single constant keeps **its original rejection at its original Pointer** (`.../oneOf/N/type`), because there is no name source:

- an empty or member-less inline object;
- a missing or non-String discriminator property;
- a multi-member `enum` (no single value to name the leaf);
- a primitive, array or nullable inline branch.

Multiple parents remain a separate contract: the planner still holds one parent per leaf, and combining two hierarchies would need a decision on shared serializer identity and conflicting tag values that is not implied by this change.

## How it is implemented

The planner keys everything — `graph`, `shapes`, `parents`, `heights`, `components` — by component wire name. Rather than teaching each of those about nameless branches, an eligible inline branch is registered as a **synthetic component** carrying its derived class name before the first consumer runs. The rest of the hierarchy pipeline is unchanged, which is why the emitted bytes match the `$ref` form exactly.

Names are generated in sorted-parent, source-index order and appended to the declaration order, so output stays deterministic.

## Verification

- [Fixture](../tests/fixtures/http/hierarchy-inline-members.json), [core contract](../core/hierarchy_inline_members_test.go) (same shape as references, order stability, vocabulary leaves, namingless rejections and Pointers, duplicate-name failure), and the [consumer contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyInlineMembersContracts.kt) (dispatch, fixed tag round-trip, Unknown fallback, known-payload failure).
- `bash scripts/test.sh feature hierarchy-inline-members` runs fresh Data/Plain × validation-off/on projects and four `Json { encodeDefaults, explicitNulls }` configurations.
- Output for the five existing hierarchy fixtures is byte-identical to the base commit.
