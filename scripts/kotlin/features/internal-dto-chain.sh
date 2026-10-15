#!/usr/bin/env bash
set -euo pipefail
export INTERNAL_DTO_FIXTURE=tests/fixtures/http/schema-internal-dto-chains.json
exec bash "$(dirname "${BASH_SOURCE[0]}")/internal-dto.sh"
