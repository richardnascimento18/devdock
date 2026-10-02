# Pass 2B pre-UI cleanup validation

This is distinct from the historical configuration-editor pass-2b.md.
Scope: history/branch hygiene, configured-root ownership, and release/source
output cleanliness. No Pass 3 UI or Pass 4 general-hardening work.

## Starting topology and reconciliation

Starting staging: acf22572e4a71f54739757e7491d51297b31f1da.
Starting production: 5c33b17103d3184c3aade18f95a5784055e466e8.
Latest prerelease: v1.1.0-beta.4. Production is the default; effective rulesets
24336971 and 24337571 retain strict required checks, PR-only merges, deletion
and non-fast-forward protection with no bypass actors. Historical tags matched
the prior independent-review snapshot; no open PRs or unexpected versions.

Production -> staging PR #15 changed zero files. Required CI 37014199990 and
staging push CI 37014393808 passed. Merge baaaac2ef459a7d437c865f76bf58926fd5ec796
includes production ancestry with an identical tree. The code branch
fix/root-identity-validation starts from this reconciled staging commit.

Verified reachability from both staging and production and absence of open PRs
before deleting feat/configuration-editor, fix/deep-location-disambiguation,
fix/default-preset-rename and refactor/recursive-workspace-model. Automatic
head-branch deletion was enabled; permanent-branch deletion rules remain active.

## Root ownership and compatibility

config.ValidateRoots centralizes path-aware lexical checks (filepath.Rel) and
physical checks (filepath.EvalSymlinks). Exact duplicates and parent/child pairs
in either order are rejected. Prefix-only siblings such as code/code2 are valid.
Resolved aliases and physical parent/child pairs are rejected with typed
RootConflictError values identifying the candidate and existing root.

RootIdentity records canonicalization failures explicitly as unknown identity,
including missing paths, broken/cyclic symlinks and permission failures. These
failures do not invalidate existing config solely for unavailability; lexical
checks still apply. No root or favorite/recent is automatically removed.
Conflicting legacy configs fail startup with both roots and an actionable config
file/restart error. The original disk file is preserved; interactive startup
repair was intentionally not added. Correct the roots list manually and restart.

Config validation, proposal saves and AddRoot use the same abstraction. Normal
and Settings root additions share the existing handler, reject conflicts before
persistence, and preserve drafts/live config/state on failure. Relative UI input
is normalized to an absolute clean path. Existing root removal persistence,
state pruning and directory-preservation behavior is unchanged.

## Release and source output

build-release.sh invokes release_output.py before compilation. Previous generated
DevDock Linux binaries and SHA256SUMS are removed. Unexpected files/directories
cause an explicit preparation failure and are preserved rather than deleted.
A successful build therefore starts with empty, dedicated output. The existing
release workflow calls this same path; dist remains ignored and uncommitted.
make clean-release reuses preparation. make source-archive uses git archive of
committed HEAD and reports the source commit, excluding dirty edits and dist.

## Tests and actual local results

New roots_test.go covers lexical pairs, prefix/siblings, normalized-path rules,
physical aliases and both overlap orders, missing/broken/cyclic roots, permission
uncertainty, legacy-file preservation and mutation-free rejected proposals.
root_identity_flows_test.go covers both add-root entry points, persisted config
and state preservation, relative-input normalization, and missing-root state.
Existing editor/root-removal tests remain green; duplicate addition now rejects
rather than acting as a no-op. Python tests prove stale output removal,
unexpected-file refusal and committed-source-only archives.

All passed: gofmt, git diff --check, tidy consistency/module verification,
go test ./..., race, vet, staticcheck, govulncheck (none found), build, actionlint,
Python tests (6), exact minimum Go 1.26.0, Linux amd64/arm64 release builds,
both local checksums, isolated TUI smoke and persisted Settings edit smoke.
Go 1.27.1 local test totals: 102 top-level / 230 including subtests; diagnostic
aggregate statement coverage 51.8%. Actual pre/post dist listings confirm old
versions were removed and only beta.5 binaries/checksums remained.

Version: 1.1.0-beta.5, next unused beta after beta.4. Hosted PR, promotion,
release, final reconciliation and exact final SHAs are recorded in the finalized
validation evidence/report accompanying the review source archive. This source
document captures the validated implementation before release without adding
post-release content changes to the permanent branches.
