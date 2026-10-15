#!/usr/bin/env bash
set -euo pipefail
export INTERNAL_DTO_FIXTURE=tests/fixtures/http/schema-named-dto-chains.json
export INTERNAL_DTO_NAMED_TERMINALS=1
exec bash "$(dirname "${BASH_SOURCE[0]}")/internal-dto.sh"
