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

Ordinary scalar, DTO and collection targets, tuple members/tails and discarded
extra-property annotations retain their own restrictions. Direct inline builtin
scalar children have a [separate limited contract](inline-builtin-collection-child-defaults.md). Validate and generate
the exact input, then use `generate --check` and compile the fresh Kotlin for
the relevant target. Test missing, null, empty and supplied values independently.
Do not infer general default graphs or complete platform coverage from this role.
