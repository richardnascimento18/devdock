#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
archive_path=${1:?usage: package-source.sh OUTPUT.zip [COMMIT_OR_REF]}
source_ref=${2:-HEAD}
source_commit=$(git rev-parse --verify --end-of-options "$source_ref^{commit}")
git archive --format=zip --prefix=devdock/ --output="$archive_path" "$source_commit"
printf 'Archived committed source at %s to %s\n' "$source_commit" "$archive_path"
