# Defaults on enum collection elements

An existing finite List or typed Map may put its own `default` on the original
`items` or `additionalProperties` inline lexical String or native
numeric/Boolean enum, or on a local `$ref` resolving to an admitted enum of
those kinds. The annotation must match the original vocabulary, the effective
nullable view and supported validation checks; native numeric membership is
checked before floating-point mapping.

Treat this annotation independently from the enum component, its aliases,
collection factories and consuming fields. It does not add an element or
initialize the container; empty or null outer defaults do not hide an invalid
child. Runtime String enum codecs remain open, and existing native codecs keep
their scalar representation. The existing shared planner still determines which
reference and collection paths are admitted.

The child may also reference an admitted named finite List/typed Map or named
alias whose finite inline tree ends in an eligible direct enum. Paths ending in
a physical enum reference may pass through admitted named collections and
aliases. Validate its collection value separately from target, enum and alias
factories. The String codec remains open at runtime; annotations must use the
original vocabulary. Internal collection references and internal enum leaves
remain outside this collection-valued role, as do other outer-default roles.

DTO targets, tuple members/tails and discarded extra-property
annotations retain their own restrictions. Eligible ordinary scalar references
and direct inline builtin scalar children have separate
[reference](scalar-reference-collection-child-defaults.md) and
[inline](inline-builtin-collection-child-defaults.md) contracts. Validate and generate
the exact input, then use `generate --check` and compile the fresh Kotlin for
the relevant target. Test missing, null, empty and supplied values independently.
Do not infer general default graphs or complete platform coverage from this role.
