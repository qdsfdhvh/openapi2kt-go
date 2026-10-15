#!/usr/bin/env bash
set -euo pipefail
export INTERNAL_DTO_FIXTURE=tests/fixtures/http/root-internal-dto-aliases.json
export INTERNAL_DTO_ROOT_ALIASES=1
exec bash "$(dirname "${BASH_SOURCE[0]}")/internal-dto.sh"
