# Enum member defaults in Cartesian object-union defaults

A Cartesian object union (a union inside `allOf`) can retain String and native
Boolean/Int/Long/Float/Double property-own defaults while an ordinary consuming
field supplies its own object default. The shared plan must retain an actual
selected constructor and the original enum property as its scalar or named
constructor field. Ordinary tag/presence object unions already reuse the ordinary
object walker; this contract supplies the Cartesian constructor role.

For example, a fixed `version:const "v1"` combined with two `kind` variants can
have an A member `mode:enum ["on","off"]` with `default:"off"`. An enclosing
field's default containing `version:"v1",kind:"a",mode:"on"` selects A and
supplies on, while constructing A directly still uses off. A default must carry
every pin needed by the existing conjunction selector; omitted own-default
properties are not inserted before selection. Required properties must still
appear in the outer literal. Unknown is never a permitted default selection.

Each own annotation and supplied outer member matches the original vocabulary
strictly, including before native Float/Double rounding. Explicit nullable
members, extrema, validation-gated bounds and native approximation warnings
reuse the field planners. String enum runtime unknown values remain open;
native membership enforcement remains opt-in. Source errors retain the original
variant property Pointer; bad literal values use the consuming `/default` path.
Invalid annotations on other variants, required fields or Patch fields remain
validated before initializer suppression. Patch keeps `Absent` and its existing
null/value behavior.

The planner borrows original schema nodes and merged field order, authorizes
only actual selected constructor properties, and preserves the final linked
serializer under deduplication, hoisting, merging and locked names. Constants,
raw projections, collection-element annotations, component factories, aliases to
union roots and other source roles retain their own admission rules. There is
no target/default inheritance. Source48, shared work16384, value16384,
canonical default32KiB, native own vocabulary240, checked16 and JVMslots240
remain unchanged.

## Numeric selection and migration

Cartesian dispatch compares primitive JSON pin spellings. Canonical native
number formatting can change a spelling (`1.0` to `1`, Float rounding or exponent
formatting), previously turning an accepted known default into Unknown or a
different known variant. The generated decoding literal now retains original
root numeric pin tokens when necessary to preserve the original selected index.
Other members remain canonical. When canonical text selects the same index, the
existing bytes are retained. Both original and final literals obey the existing
budgets; selection rules and runtime serializers are unchanged.

This also fixes previously accepted inputs without member-own annotations.
Regenerate and rebuild affected consumers. A default can now have its intended
known subtype instead of Unknown or another subtype; equality, pattern matching,
subtype descriptors and encoded member order/numeric spelling can therefore
change. With `encodeDefaults=false`, a manually supplied known value equal to
the corrected default can now be omitted. Review generated/locked output changes
through the normal ownership workflow. Manifest and name-lock formats do not
change.

Native scalar mapping still rounds lossy floating values. A later serialization
of that typed value can produce a numeric spelling outside the original lexical
pin vocabulary and decode as Unknown, under the existing approximation contract.
Preserving the initialization selection does not add exact floating runtime
membership or round-trip dispatch guarantees.

## Verification

`bash scripts/test.sh feature composed-member-defaults` freshly generates
Data/Plain × validation-off/on projects in baseline and combined optimized/locked
packages. The synthetic [fixture](../tests/fixtures/http/composed-member-defaults.json)
and consumer check four Json configurations, actual known subtype creation,
independent constructor defaults, strict validation, fresh values, native extrema,
238 data fields plus two constant pins, vocabulary240, Patch, controlled HTTP and
byte-fragmented UTF-8 live SSE failure/closure/reuse. The HTTP registry uses the
same consumer. `OPENAPI_KT_PREPARE_ONLY=1` prepares sources without claiming JVM
execution. Go tests cover original constructor/source authority, cumulative work,
original AST immutability, numeric pin selection and exact final32KiB limits;
native tests cover core byte parity, no-op/readonly and failure preservation.
This is a bounded D03/M01–M04/M07/M08 increment; general defaults and additional
platforms remain separate.
