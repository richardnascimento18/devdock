#!/usr/bin/env bash
set -euo pipefail
if [[ "${BASE_REF:-}" != production ]]; then
  exit 0
fi
if [[ "${HEAD_REF:-}" != staging || -z "${REPOSITORY:-}" || "${HEAD_REPO:-}" != "$REPOSITORY" ]]; then
  echo 'Production PRs must come from this repository staging branch.' >&2
  exit 1
fi
printf 'Verified same-repository staging -> production promotion.\n'
