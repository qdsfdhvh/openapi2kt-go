#!/usr/bin/env bash
set -euo pipefail
INTERNAL_DTO_FIXTURE=tests/fixtures/http/root-internal-dto-bridges.json INTERNAL_DTO_ALIAS_TERMINALS=1 INTERNAL_DTO_ROOT_BRIDGES=1 bash "$(dirname "${BASH_SOURCE[0]}")/internal-dto.sh"
INTERNAL_DTO_FIXTURE=tests/fixtures/http/root-named-dto-bridges.json INTERNAL_DTO_NAMED_TERMINALS=1 INTERNAL_DTO_ALIAS_TERMINALS=1 INTERNAL_DTO_ROOT_BRIDGES=1 bash "$(dirname "${BASH_SOURCE[0]}")/internal-dto.sh"
