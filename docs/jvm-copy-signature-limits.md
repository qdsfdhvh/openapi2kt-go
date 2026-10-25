# Retained Data copy signature limits

After hoisting, deduplication, merging and final Data/Plain style selection,
generation checks a one-sided lower bound for each retained Data model's copy
method signature. It combines the existing known parameter syntax/inline List
variance with only the validated ASCII package prefix of the return type.
Classifier, nested owner, models/api suffix and uncertain alias spelling are
omitted from that return-type bound. A bound over the JVM 65,535-byte UTF-8
constant limit yields `E_MODEL_LIMIT` before artifacts are returned or written.
Native generate/check/validate preserve existing output on failure.

Constructor feasibility and Data copy feasibility are separate. With 203
parameters of List nested 16 levels around Int, the demonstrated constructor
signature is 65,369 bytes. The same parameters plus a 316-byte package prefix
already require at least 65,686 bytes for Data copy, before even counting its
class name. A fresh Data control fails Kotlin/JVM compilation while the same
Plain source compiles and runs. Plain classes do not own the Data copy API;
they are not rejected by this copy proof. Final style/retention must therefore
be checked after transformations, rather than assuming Data during analysis.

A private Plan diagnostic map retains original component, nested object and
HTTP schema Pointers, including projected model locations. It is separate from
declaration equality, shape/name-lock serialization and source/annotation
permissions. Renaming or retaining an existing physical model through an
optimization keeps its diagnostic origin. Discarded aliases do not own methods
that need this check. No model member, public Kotlin API, wire rule or persisted
format is added for diagnostic tracking.

`bash scripts/test.sh feature jvm-copy-signatures` freshly compiles a feasible
short-package Data model and a long-package Plain model, with baseline and
combined dedup/hoist/merge/locked options. It tests four Json configurations,
Data copy, the final field's retained validation and early rejection of the
otherwise equivalent long-package Data model. Native regressions compare Plain
core/CLI bytes and preserve owned output for failing Data generate/check/validate.
Core regressions exercise Data/Plain across all eight optimization combinations,
original escaped/nested/HTTP Pointers and determinism. Existing identity
regressions ensure diagnostic provenance does not change model sharing.

This is not a complete signature encoder. Exact nominal/alias/classifier names,
uncertain Map/mixed/nullable-element variance, other methods/metadata and other
platforms remain separate. A passing lower bound does not guarantee backend
representability. Original source/work/value/default/checked/JVM-slot caps and
public types remain; the guard does not erase, wrap or silently truncate types.

**Migration:** additional Data inputs already unable to compile now fail early
with a structured resource diagnostic. Proven feasible Data and Plain inputs,
JSON/HTTP/SSE wire behavior, output manifests and name-lock formats retain their
contracts. Reducing the package prefix or splitting the model can resolve the
proven resource excess, subject to the other existing limits.
