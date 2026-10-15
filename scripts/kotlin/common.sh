#!/usr/bin/env bash
# Shared repository/bootstrap rules for Kotlin verification, not CLI runtime.
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

openapi_kt_build_cli() {
  env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
    "${GO:-go}" build -trimpath -o "$1" ./cmd/openapi-kt
}
