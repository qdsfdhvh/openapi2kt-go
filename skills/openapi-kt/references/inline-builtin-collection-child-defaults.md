# Builtin scalar defaults on direct collection elements

An ordinary named DTO without `$ref` or composition may put an independent
`default` on the original direct array `items` or typed Map
`additionalProperties` of one of its fields. The inline child must be a
builtin String, Int, Long, Float, Double, Boolean, UUID, LocalDate or Instant schema. The original
collection and any borrowed or uniqueness consumer must already be admitted.
An already admitted owner-root default factory validates independently and
never inherits the child annotation.

Validate the annotation using its own type, nullable intent and supported
checks. It never initializes or populates the enclosing collection and is not
inherited. Validate it even when an outer default is empty or null; failures
point to the original child `/default`. Existing source and default budgets
apply, as do the normal validation gates. Eligible annotations change no
generated API or wire behavior. UUID checks use canonical UUID text. Dates
must be valid ISO dates; Instants allow up to nine fractional digits and
offsets up to 18:00. Temporal length and pattern constraints remain unsupported.

Root collections, nested children, tuples, fixed extras, unadmitted owner
factories, ordinary `$ref` leaves, bytes, const and
native/value-class roles, and hierarchy-owned DTOs are outside this profile. Inline enum and
enum-reference children use their [separate contract](collection-reference-child-defaults.md).
Validate and generate the exact input, run `generate --check`, and compile
fresh Kotlin for the intended target. Check missing, null, empty and supplied
collection values independently.
