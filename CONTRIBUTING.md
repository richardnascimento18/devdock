# Contributing to DevDock

## Scope and package ownership

Pass 1 stabilizes the current model and interface. Recursive hierarchy redesign, bulk selection, new responsive panes, visual themes, and major features require later approved passes.

- The root package owns Bubble Tea presentation and typed workflows. External work returns commands/messages with operation identities; Update must not perform network/process work or rescan the workspace synchronously.
- `internal/core` owns workspace validation, discovery, creation, deletion, and collision-safe moves.
- `internal/detect` classifies projects using a scan-local cache.
- `internal/config`, `preset`, `template`, and `state` own validation and persistence of their data. `internal/fileutil` supplies atomic writes.
- `internal/github`, `tmux`, and `pty` own external integrations. GitHub HTTP/auth and tmux execution have small injectable boundaries for testing.
- Editor files divide collection orchestration, preset/window editing, panes, and template steps. Proposed collections are cloned and fully validated before disk persistence; active state changes afterward.

Keep changes incremental and protected by behavioral tests, especially for destructive operations, persistence, async cancellation, and failure propagation. Avoid introducing interfaces without an external boundary or concrete testing need.

## Local workflow

Use Linux, Go 1.26+, Bash, Python 3, Git, and a C compiler for race testing. Dependency metadata and rooted filesystem APIs have a technical floor of Go 1.25; Go 1.26 is the deliberately supported minimum because the maintained toolchain pair is 1.26/1.27 and current analysis tools require 1.26. There is no claim of supported 1.25 builds. Use tmux to verify workspace attachment. Install pinned tools with `make tools` and put `$(go env GOPATH)/bin` on PATH.

```sh
make fmt
make check
```

`make check` runs gofmt verification, `go mod tidy` consistency, `go mod verify`, version-script tests, actionlint, version validation, `go test ./...`, `go test -race ./...`, `go vet ./...`, `staticcheck ./...`, `govulncheck ./...`, and `go build ./...`. Vulnerability checking requires network access. CI runs the suite on Linux with the latest Go 1.26 and 1.27 patches and cross-builds both release architectures.

Tests must use `t.TempDir()` and isolated configuration. Use fake HTTP/auth clients, tmux runners, and Git command runners rather than real services or your own filesystem. PTY tests intentionally run short local child commands. Do not disable checks, loosen validation, or discard tests to make a change pass.

## Branches and commits

Permanent branches:

- `staging`: integration branch and next release candidate.
- `production`: officially released code.

After Pass 1, do not develop directly on either branch. Create a purpose-specific branch from staging: `feat/…`, `fix/…`, `refactor/…`, `test/…`, `ci/…`, `docs/…`, `build/…`, `perf/…`, or `chore/…`. Use Conventional Commits, for example `fix(pty): preserve child exit errors`.

1. Commit coherent changes, run relevant checks, and push your development branch.
2. Open a PR targeting `staging`. Require review and green **Required validation**.
3. Merge through GitHub after protections pass. Staging push CI runs again.
4. Prepare a separate development-branch version change in `VERSION` when ready to release, and merge it into staging through the same PR process.
5. Open a promotion PR **from the same repository's `staging` into `production`**. It must carry a strictly newer explicit SemVer than production.
6. The full suite reruns against the PR candidate; earlier staging results do not substitute for it.
7. Merge only after review and checks. A production push runs the full suite again, builds official binaries, and publishes the exact production commit.

Use merge commits for staging→production promotions so the branches retain their ancestry. After a production promotion, reconcile the production merge back into staging through a reviewed `chore/…` branch/PR, preserving ancestry. Review any conflict resolution through CI. Do not force-push or rewrite published history. Auto-merge is optional and must respect required checks/reviews.

For an emergency hotfix, branch `fix/…` from production, merge the correction and a new reviewed VERSION into staging through PRs, then promote staging normally. This policy deliberately does not permit direct hotfix pushes to production. If staging contains unsafe unreleased work, first make a reviewed corrective PR to remove/defer that work, then promote the safe candidate. Any owner-authorized emergency bypass must be explicitly recorded and followed by reconciliation/reverification.

## Required GitHub settings (owner actions)

Committed workflows cannot configure repository rules or OAuth registration. **These settings have not been applied by Pass 1.**

Bootstrap `staging` and `production` from the reviewed Pass 1 baseline; this gives production a valid VERSION before the first promotion. Establishing the baseline does not publish it as a stable release. If importing an older production branch without VERSION, the owner must establish a reviewed baseline VERSION before normal promotion policy can pass.

For **both** staging and production, configure GitHub branch protection or rulesets:

- Require a pull request before merging; require at least one approving review.
- Dismiss stale approvals when new commits are pushed and require approval of the most recent reviewable push.
- Require status check **Required validation** from the CI workflow. Let the initial CI run register the check before selecting it.
- Require the branch to be up to date before merging and require conversation resolution.
- Block force pushes and branch deletion; apply rules to administrators and bypass roles. Do not allow routine bypass actors.

The production PR policy additionally enforces same-repository staging as the source. GitHub branch rules alone do not encode that source restriction; the required CI check does. Do not enable a merge queue without adding/test-validating `merge_group` support (not configured here).

Configure the repository variable **DEVDOCK_GITHUB_CLIENT_ID** with the official public OAuth App Client ID, with device flow enabled on that app. This is a public identifier, not a secret. Never add a client secret to source, variables, flags, artifacts, or logs.

Create the GitHub environment **production-release**, limit deployment branches to `production`, and optionally require a release-owner reviewer. Publication waits for this environment's configured approval. Ensure repository Actions policies allow the SHA-pinned official checkout/setup-go/upload/download actions and allow `contents: write` for the publication job. Do not give PR workflows write permissions. No workflow uses `pull_request_target`.

## Version and release lifecycle

`VERSION` contains one reviewed SemVer without `v` or build metadata, such as `0.1.0-dev.1`, `0.1.0-rc.1`, `0.1.0`, or `1.0.0`. Versions are explicit; Conventional Commits describe changes but do not drive automatic increments. The Python validator rejects malformed versions and compares prerelease ordering. A production candidate must be greater than the production base version. Choose pre-1.0 prereleases until later passes and release-candidate hardening justify a stable version.

A production push runs reusable verification on Go 1.26/1.27. A read-only build job compiles CGO-free Linux amd64/arm64 binaries with trimpath, version/commit/commit-date metadata, and the public repository Client ID. It uploads:

- `devdock_<version>_linux_amd64`
- `devdock_<version>_linux_arm64`
- `SHA256SUMS`

A separate publication job has only `contents: write` and the production-release environment. It downloads/checks artifacts without checking out or executing repository code, creates lightweight tag `v<VERSION>` at the exact push SHA, and creates a GitHub release with generated notes. Versions containing prerelease identifiers create GitHub prereleases. Tags already pointing elsewhere are refused. A failed asset upload can be retried without moving tags; an existing matching release receives replacement assets.

CI uses read-only tokens, SHA-pinned actions, Go module/build caches, Linux job timeouts, and cancellation of superseded PR runs. Production releases serialize and do not cancel a running release. Complete a release before promoting another candidate; GitHub concurrency retains only one pending run, not an unlimited queue.

Local reproduction:

```sh
DEVDOCK_GITHUB_CLIENT_ID=YOUR_PUBLIC_CLIENT_ID make release
(cd dist && sha256sum --check SHA256SUMS)
./dist/devdock_$(cat VERSION)_linux_amd64 --version
```

The build script uses the commit timestamp by default. Optional `DEVDOCK_COMMIT`, `DEVDOCK_BUILD_DATE`, and `DEVDOCK_DIST_DIR` overrides support CI/reproducible builds. The public Client ID must be a single safe linker token. For identical artifacts use the same commit, Go toolchain, VERSION, timestamp, and Client ID. Cross-compilation proves buildability; native arm64 smoke testing remains a release-hardening step.

Do not push branches, merge, tag, or publish releases through an agent without explicit owner authorization.

## Safety and failure semantics

Filesystem moves use no-replace rename. Cross-filesystem moves copy into a separate staging directory, preserve symlinks as symlinks, reject unsupported special files, and leave the source intact on copy failure. If destination commit succeeds but source removal fails, the error identifies the partial move; both copies may remain. Inspect and reconcile deliberately rather than repeating blindly.

Builtin template paths use rooted filesystem handles and must be strict descendants without symlink components or `..`. Arbitrary template/preset commands are trusted user configuration and can perform arbitrary actions; they are not sandboxed.

Persistence uses a same-directory temporary file, fsync, close, and atomic rename. Validation/persistence failures leave active editor/config preferences unchanged. State-save failures remain visible after TUI shutdown. Directory fsync is not promised; atomicity does not guarantee survival of the latest rename after sudden power loss. Workspace mutations and metadata persistence cannot be one filesystem transaction; failures are reported and later scans reconcile missing paths.

Root permission or scan failures return partial results with errors. Cancelled OAuth/PTY flows use contexts and operation identities; stale messages cannot mutate the new screen. GitHub error bodies are not echoed, protecting tokens from accidental logging.
