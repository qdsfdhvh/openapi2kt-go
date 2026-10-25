# Linux Go and native CLI contracts

The [recorded Linux ARM64 profile](benchmarks/linux-go-contracts.json) passed
`gofmt`, `go vet ./...`, `go test ./...` and `go test -race ./...` for the complete
repository Go package inventory. CLI tests built and executed the local Go
command, including output ownership, locks, paths, permissions, rollback,
configuration, JSON/YAML input and failure-preservation contracts.

An unprivileged user ran in a native ARM64 Ubuntu24.04 local container.
Go1.27.2 command/compiler/tools were built offline from matching installed Go
source; standard-library sources and tool inputs were read-only. Ordinary tests
used CGO=0; race tests used CGO=1 and GCC13.3.0/libc6-dev2.39 in a separately
prepared local prerequisite image. Exact images, resolved versions and source
fingerprint are in the report. Preparation was separate from network-disabled
test runs and did not install host global tools. No source patches or version
bypasses were applied.

On an already provisioned Linux host, select matching Go/gofmt executables and
an available native C compiler:

```bash
gofmt -l core internal cmd
env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 \
  go vet -p=2 ./...
env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 \
  go test -count=1 -p=2 -timeout=10m ./...
env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=1 \
  go test -race -count=1 -p=2 -timeout=10m ./...
```

Require empty formatting output and terminal success for every command. A
non-root user preserves the meaning of permission tests. Containers need
executable temporary storage and writable ignored `.pi` process-test scratch
and Go cache; these mounts can be separate from read-only sources and tools.
Loopback sockets must be available; the tests need no external network. The
recorded runs used two CPUs, 4GiB memory and uncached test execution.

This is separate from the [fixed CLI input/host parity probe](cli-host-probes.md).
It does not compile/run Kotlin or HTTP engines on Linux, establish minimum
versions, run Python tooling or establish other hosts and architectures. Follow
[Contributing](../CONTRIBUTING.md) and the [development plan](development-plan.md)
for the remaining unified acceptance.
