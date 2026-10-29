# Scalar reference defaults on collection elements

An already admitted physical List `items` or typed Map
`additionalProperties` may put its own `default` on a local `$ref` to an
accepted builtin String, Boolean, Int, Long, Float, Double or UUID, standard
date/date-time, or byte/binary scalar. Byte defaults use the existing signed
integer array representation. Existing reference and alias admission remains
authoritative; the annotation does not make a new path valid.

Validate the child value independently using its actual effective nullable
view, scalar grammar and retained checks. A child does not initialize the
container, add an element or inherit a target value. Empty/null outer
defaults and required/Patch suppression do not hide an invalid child.

An admitted scalar target and admitted named aliases may retain their own
independent component-default factories. Each factory and child annotation is
validated at its own source, with no default inheritance. These factories
may coexist with already admitted collection field and object defaults.
Named finite List/typed Map paths ending at an admitted scalar reference,
and their admitted aliases, can own independent collection factories. Paths
may cross existing named collection roots and their finite named aliases.
Each root, alias, field and object annotation is independent. The scalar
child annotation is optional and does not populate an outer value. Internal
inter-container references and other factory roles retain separate
restrictions.

A physical List element or typed Map value may also put its own `default` on
a reference to an admitted named finite List/typed Map or named aliases when
the collection path ends at an admitted physical scalar reference or a direct
inline builtin, UUID, date/date-time or byte/binary leaf. Validate that
collection value independently against its actual type and nearest nullable
view. Target and alias factories, nested child annotations and any separately
admitted outer value remain independent. This does not admit internal
inter-container references, DTO leaves or recursive paths. Inline enum leaves
have a separate [enum child contract](collection-reference-child-defaults.md).

DTO, const, native or value-class targets, tuple positions/tails and discarded
fixed-object extras retain their own restrictions. Inline builtin and enum elements have separate
[inline](inline-builtin-collection-child-defaults.md) and
[enum](collection-reference-child-defaults.md) guidance.

Keep the original source/target/element/check identities, actual intermediate
collection declarations, nullable bindings and cached heights, and the existing
source48/work16,384/value16,384/default32KiB/checked16/JVM240 limits. Validate
and generate the exact input, run `generate --check` with matching options,
and compile and exercise the fresh consumer on the relevant target. Separate
source preparation from actual runtime evidence.
