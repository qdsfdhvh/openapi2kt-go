# Component-root dictionary `allOf` wrappers

A component may use exactly one local `allOf` `$ref` to an admitted named
dictionary, including finite named aliases and the same wrapper form. Other
members may be empty objects or contain only string `title`, `description`
and `$comment` metadata; their order does not change the result. The resulting
Kotlin alias uses the original typed Map or open `JsonObject` backing and its
actual enum/DTO value peers.

Nullable precedence is wrapper, member, then nearest target/alias. Explicit
false uses the non-null backing while preserving the value's nullability.
The target's own default factory remains independent and is never inherited
by the wrapper or a consuming field. An eligible wrapper or named alias may
own its own validated `default` factory; each owner keeps its own value and
original error pointer. The terminal must already have an admitted component
default role; nullable dictionary terminals with property-count constraints
keep their separate rejection. Patch and shared DTO presence follow their
existing context rules.

With validation enabled, existing checks apply at supported DTO, Patch and
typed HTTP boundaries. Assignment or standalone generic dictionary
serialization does not enforce them. Each wrapper counts two levels against
the source depth of 48; checked collections retain the 16-edge limit.
Capturing all wrapper chains is additionally bounded to 16,384 binding/node
steps and 8 MiB, including neutral members and repeated targets.

Inline dictionary members, extra references, explicit nullable or other
non-neutral siblings, internal or remote terminal references and general
dictionary intersections are not admitted by this profile. Regenerate and
run same-options `--check`, and verify actual JSON/Patch/HTTP behavior in the
consumer when relevant.
