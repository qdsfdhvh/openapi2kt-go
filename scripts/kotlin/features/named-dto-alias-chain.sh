#!/usr/bin/env bash
set -euo pipefail
export INTERNAL_DTO_ALIAS_TERMINALS=1
export INTERNAL_DTO_ROOT_ALIASES=0
for kind in internal named; do
  export INTERNAL_DTO_FIXTURE="tests/fixtures/http/schema-$kind-dto-alias-chains.json"
  export INTERNAL_DTO_NAMED_TERMINALS=0
  if [[ "$kind" == named ]]; then export INTERNAL_DTO_NAMED_TERMINALS=1; fi
  bash "$(dirname "${BASH_SOURCE[0]}")/internal-dto.sh"
done
