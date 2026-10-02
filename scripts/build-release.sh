#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
release_version=$(python3 scripts/version.py "$(cat VERSION)")
release_commit=${DEVDOCK_COMMIT:-$(git rev-parse HEAD)}
release_date=${DEVDOCK_BUILD_DATE:-$(date -u -d "@$(git log -1 --format=%ct)" +%Y-%m-%dT%H:%M:%SZ)}
client_id=${DEVDOCK_GITHUB_CLIENT_ID:-}
output_dir=${DEVDOCK_DIST_DIR:-dist}
# Public identifiers and metadata must be single linker tokens.
[[ "$release_commit" =~ ^[0-9a-f]{40}$ ]] || { echo 'invalid commit SHA' >&2; exit 1; }
[[ "$release_date" =~ ^[0-9TZ:+-]+$ ]] || { echo 'invalid build date' >&2; exit 1; }
[[ "$client_id" =~ ^[a-zA-Z0-9._-]*$ ]] || { echo 'invalid public OAuth Client ID' >&2; exit 1; }
python3 scripts/release_output.py "$output_dir"
flags="-s -w -X main.version=$release_version -X main.commit=$release_commit -X main.buildDate=$release_date -X github.com/richardnascimento18/devdock/internal/github.ClientID=$client_id"
for architecture in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$architecture" go build -trimpath -buildvcs=false \
    -ldflags "$flags" -o "$output_dir/devdock_${release_version}_linux_${architecture}" .
done
(cd "$output_dir" && sha256sum "devdock_${release_version}_linux_amd64" "devdock_${release_version}_linux_arm64" > SHA256SUMS)
printf 'Built version %s at commit %s\n' "$release_version" "$release_commit"
