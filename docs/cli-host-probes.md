# Local CLI host and input probe

`python3 scripts/cli/host_probe.py --binary PATH --report REPORT.json` runs a
local executable against ten fixed synthetic inputs in Data/Plain and
validation-off/on profiles. It does not build a binary, fetch a specification,
install dependencies or choose a remote host. Use an already installed Python
and the offline Go toolchain required by [Contributing](../CONTRIBUTING.md).

```bash
env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  go build -trimpath -o build/host-probe/openapi-kt ./cmd/openapi-kt
python3 scripts/cli/host_probe.py --binary build/host-probe/openapi-kt \
  --report build/host-probe/native.json
# Run the same source/input inventory on another host, comparing its report:
python3 scripts/cli/host_probe.py --binary build/host-probe/openapi-kt-other \
  --report build/host-probe/other.json --compare build/host-probe/native.json
```

`--repo` can select a read-only checkout; temporary output uses the host's
standard temporary directory. The supplied report path must be writable. Only
pass executables authorized to run on that host. Reports contain binary format,
interpreter architecture and SHA-256 separately: an interpreter's architecture
does not establish the executable's architecture.

The inputs cover catalog, structural uniqueness, allOf tuples, independent
component defaults, scalar tuple defaults, primitive-union merging, form/raw
requests and a JSON/YAML pair. Successful profiles check identical generation
and check reports, deterministic bytes, no-op file modes/mtimes, file/stdin
parity, validation and preservation after invalid JSON or YAML. All profiles
use combined dedup/hoist/merge/lock options. The structural-default fixture
requires validation; its two validation-off profiles retain their exact original
unsupported diagnostic and verify file/stdin parity with no output creation.
They are explicit negative contracts.

[Recorded fixed-profile results](benchmarks/cli-host-probes.json) passed on a
native macOS ARM64 host and for a static Linux ARM64 executable in a local Linux
compatibility environment. The Linux test used an existing Python3.12 slim
**amd64 interpreter container** to execute the **ARM64 CLI**; it was not a native
ARM64 Python image or a full ARM64 host test suite. Its network and capabilities
were disabled and source/root mounts were read-only. Both hosts passed all 40
cases: 38 generated profiles and two original rejection profiles. Exact input
hashes, reports and output-file inventories matched across hosts, including
JSON/YAML equivalence. The report binds the generator source fingerprint and Go
1.27.2 builds; it is evidence for these inputs/options only.

These runs establish neither all CLI host/architecture combinations nor full Go
race/transaction suites on Linux. They do not compile Kotlin, execute HTTP
engines, establish minimum dependency versions or broaden the bounded
[YAML input profile](yaml-input.md). Complete candidate acceptance still requires
[the development plan](development-plan.md) and the final unified checks.

The separately bound [Linux Go/native contract suite](linux-go-contracts.md) executes all repository Go packages and native CLI tests with vet and race on native Linux ARM64. It has its own source/toolchain/image inventory; the fixed input probe above remains a separate byte/report comparison.
