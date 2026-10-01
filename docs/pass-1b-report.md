# Pass 1B — corrective hardening and live CI/CD report

Evidence recorded on 2026-10-01 UTC. The live release rehearsal succeeded. No Pass 2 features or domain/UI redesign were introduced. The release remains a prerelease, not a stable-production readiness claim.

## A. Pre-flight

The clean authoritative local `main` HEAD was `344f2932a9dc5b24551d1c3d27accb4ab41be58b`, containing 15 unpushed Conventional Commits from Pass 1. Git fetch confirmed only remote `main`, at `76879fb7fa55fc6b1525ed7eaffe77f78c47bd14`; GitHub default was main. Authenticated repository permission was ADMIN. The initial status, branch/log/remote/fetch, gh authentication and repository checks were performed before changes. Existing workflows, scripts, Makefile, VERSION, README and CONTRIBUTING were inspected.

Historical tags and published prereleases already existed: `v1.0.0-beta` at `ea978d947b7de04a251a9fa2bbc379b03af39b7f` and `v1.1.0-beta` at `76879fb7fa55fc6b1525ed7eaffe77f78c47bd14`. Both remain unchanged. The latter was an unexpected newer release lineage, so the owner explicitly selected **1.1.0-beta.1** instead of moving backward to 1.0.0-beta.1 or 0.1.0-dev.1. No remote history conflict required a force update. [Pre-flight evidence](validation/pass-1b/preflight.json).

## B–D. Corrective changes and tool PATH

| Root cause | Change | Verification | Commit |
| --- | --- | --- | --- |
| Go-installed tools were invoked by name without a deterministic PATH. | Reusable workflow creates `$RUNNER_TEMP/devdock-tools`, exports GOBIN via GITHUB_ENV and directory via GITHUB_PATH before `make tools`; subsequent step verifies all three tools with command -v. scripts/check.sh honors configured GOBIN, Makefile documents caller ownership. | Dedicated /tmp GOBIN locally and both hosted Go jobs, without relying on developer GOPATH/bin. | afb3f30 |
| Explicit configured-root removal left hidden favorites/recents occupying the five-favorite limit. | Clone proposed configuration and UI state, remove contained paths with existing filepath.Rel-based state.RemovePath, persist config/state before replacing active snapshots. State-write failure compensates configuration; rollback failures are returned with context. | TestExplicitRootRemovalPrunesFavoritesAndRecents; TestFailedRootRemovalKeepsStateAndConfiguration; expanded TestSnapshotKeepsStateForUnavailableRoots. | cdf735a |
| Arbitrary branches could promote directly to production; initial production creation could publish. | Same-repository staging source predicate is a required check, tested with six subprocess cases. Publish requires vars.DEVDOCK_RELEASES_ENABLED == 'true'; build requires the public ID when publication is enabled. | Local Python tests, actual rejected PRs #3/#5, accepted PR #6, skipped bootstrap publication. | ab97d19 |
| Manual startup evidence needed repeatability. | Isolated PTY smoke harness supports local or downloaded binaries, strips runtime Client ID overrides, exercises rendering/navigation/editor/auth cancellation and shutdown. | Local release and GitHub-downloaded amd64 binary. | a68ca5c |
| Hosted ShellCheck caught an unquoted executable path in the bootstrap workflow. | Quote the version-derived dist executable path. ShellCheck installed locally for hosted parity. | actionlint locally and all corrected hosted runs. | afb3f30 |
| GitHub coalesced identically named checks for two PRs sharing a HEAD, making a legitimate staging PR inherit a diagnostic production failure. | Separate Integration source validation / Required validation from Production source validation / Production validation; update production ruleset to require both production-specific contexts. | Green corrective PR #4; independent-HEAD rejection PR #5 blocked while integration remained green; promotion #6 passed. | 8dc7932 |

Pinned tools: staticcheck v0.8.1, govulncheck v1.8.0, actionlint v1.7.12. Local ShellCheck v0.11.0 provided parity with Ubuntu Actions. No linter was disabled and no broad suppression was added.

Root removal means removing configuration and its associated metadata, not deleting workspace directories. Tests cover all five stale favorites, four removed/one retained, sibling code/code2 paths, recents, persisted snapshots, preserved files, adding a new favorite after removal, configuration-save failure and state-save failure. Path containment uses cleaned filepath.Rel segments rather than naive string prefixes. A still-configured unavailable root retains both favorites and recents after a failed scan. No project IDs or favorites product redesign were added.

## E. CI/CD hardening and security

The ordinary flow is development branch → PR staging → staging push verification → PR same-repository staging to production → full candidate verification → production verification/build/publication. A production source check rejects any other branch and forks named staging. Its check is independently required, alongside the production aggregate. Integration and production context names are distinct.

All Actions are SHA pinned. PR/test workflows use contents:read, persist-credentials:false, no secrets and no pull_request_target. Only publication has contents:write. The publication job downloads/checks artifacts without checking out or executing repository code. Validation has timeouts, Go caches and cancellation of superseded PR runs. Releases serialize without cancellation. The production-release environment restricts deployment to production and requires no second-human approval.

The repository publication gate is exactly **DEVDOCK_RELEASES_ENABLED=true**. Unset/false disables publication but permits verification/build. The gate was explicitly false for production bootstrap and true only after rulesets, environment and public OAuth variable were verified. This is also the documented emergency publication kill switch; no source edit is required.

## F. Local validation

All checks below actually ran successfully. The released e61a26f checkout was freshly validated again after publication, with logs in [live evidence](validation/pass-1b/live). Earlier candidate logs remain historical, explicitly identified as such; hosted run records bind each final PR candidate to its actual SHA.

| Check | Outcome |
| --- | --- |
| gofmt verification | PASS; no unformatted Go files |
| go mod tidy consistency / go mod verify | PASS; go.mod/go.sum unchanged, all modules verified |
| go test ./... | PASS across root package and ten internal packages |
| go test -race ./... | PASS on Linux with CGO/C compiler |
| go vet ./... | PASS |
| staticcheck ./... | PASS |
| govulncheck ./... | PASS, no vulnerabilities found; final invocation completed against the live advisory service |
| go build ./... | PASS |
| actionlint with ShellCheck | PASS |
| Python/version/source-policy tests | PASS; three test methods including table cases |
| Release scripts | PASS, Linux amd64 and arm64 built; hosted jobs also verified their checksums |
| Isolated TUI smoke | PASS for local build and the downloaded official amd64 binary |
| Minimum supported Go | PASS, local Go 1.26.8 go test ./...; hosted full suite on Go 1.26.x and 1.27.x |

The supported floor remains Go 1.26. The host/release toolchain was Go 1.27.1. The full suite has 66 named Go test functions across 14 files; no new coverage percentage was measured in Pass 1B. PTY tests use short child processes; GitHub/tmux boundaries use fakes. Some repeated Go results came from Go's content-based cache; changed state tests ran successfully and hosted runners separately validated the candidates.

## G–J. Live bootstrap, protections and pull requests

Staging was created at the exact reviewed Pass 1 HEAD 344f293 before any corrective changes. Local main was never pushed to remote main. Production was created only after corrected staging was green, at exact staging merge `7923c4248841fea5c837a7143fb7564e36e64520` with releases disabled. [Bootstrap run 36935113664](https://github.com/richardnascimento18/devdock/actions/runs/36935113664) passed verification/build and skipped publication. API inspection confirmed only the two historical tags/releases afterward.

| Operation | Hosted run | Result / merge |
| --- | --- | --- |
| Initial staging bootstrap | [36932002634](https://github.com/richardnascimento18/devdock/actions/runs/36932002634) | Failed ShellCheck quoting; fixed on development branch, no bypass |
| Hardening PR [#1](https://github.com/richardnascimento18/devdock/pull/1), fix/pass-1b-hardening → staging | [36934010228](https://github.com/richardnascimento18/devdock/actions/runs/36934010228) | Green; merge 7923c42 |
| Staging after #1 | [36934507739](https://github.com/richardnascimento18/devdock/actions/runs/36934507739) | Green |
| Initial smoke PR [#2](https://github.com/richardnascimento18/devdock/pull/2), chore/release-pipeline-smoke → staging | [36936095081](https://github.com/richardnascimento18/devdock/actions/runs/36936095081) | Green before shared-context diagnostic; waited and corrected, never bypassed |
| Draft diagnostic [#3](https://github.com/richardnascimento18/devdock/pull/3) → production | [36936098862](https://github.com/richardnascimento18/devdock/actions/runs/36936098862) | Source rejected; closed without merge; exposed shared check names |
| Context correction [#4](https://github.com/richardnascimento18/devdock/pull/4), fix/pass-1b-hardening → staging | [36937079734](https://github.com/richardnascimento18/devdock/actions/runs/36937079734) | Green; merge 59cd595 |
| Staging after #4 | [36937400556](https://github.com/richardnascimento18/devdock/actions/runs/36937400556) | Green |
| Independent-HEAD diagnostic [#5](https://github.com/richardnascimento18/devdock/pull/5), test/production-source-rule → production | [36937929289](https://github.com/richardnascimento18/devdock/actions/runs/36937929289) | Full Go/version checks passed; both required production checks failed; merge BLOCKED, closed without merge |
| Final smoke PR #2 candidate b3e62cf | [36937775429](https://github.com/richardnascimento18/devdock/actions/runs/36937775429) | Green; merge 395d46e |
| Staging after smoke merge | [36938199493](https://github.com/richardnascimento18/devdock/actions/runs/36938199493) | Green |
| Promotion [#6](https://github.com/richardnascimento18/devdock/pull/6), staging → production | [36938741296](https://github.com/richardnascimento18/devdock/actions/runs/36938741296) | Source accepted; full Go 1.26/1.27 suite reran and passed; merge e61a26f |
| Production release | [36939047360](https://github.com/richardnascimento18/devdock/actions/runs/36939047360) | Verification, build and publication all green |

Repository rulesets were configured and queried, including effective branch rules: staging **24336971**, production **24337571**, both active with no bypass actors. Both require PRs, conversation resolution, zero approving reviews for solo maintenance, merge commits, strict/up-to-date checks bound to GitHub Actions app 15368, and block force pushes/deletion. Staging requires **Required validation**; production requires **Production validation** and **Production source validation**. No GitHub plan/API limitation prevented these settings. No direct staging correction, forced merge or failing-check bypass was used.

## K–L. Release and OAuth

The workflow created [v1.1.0-beta.1](https://github.com/richardnascimento18/devdock/releases/tag/v1.1.0-beta.1), marked prerelease, published 2026-10-01T23:08:34Z, at exact production commit **e61a26f5b4ad69b0d7c0891e28532d7577f64e1b**. Build date is 2026-10-01T23:07:20Z (commit timestamp). Assets were downloaded from GitHub into a fresh temporary directory, not taken from local dist.

| GitHub asset | Bytes | SHA-256 |
| --- | --- | --- |
| devdock_1.1.0-beta.1_linux_amd64 | 10883232 | 99454b86788d89f0d0e49dcecfe4057f8f6c6469887bd5c6fc42e69e55aa154c |
| devdock_1.1.0-beta.1_linux_arm64 | 10289312 | a4f2e933a6e0eb653a0c95f95f22badbd3eb1123036043541187dfc829b2b445 |
| SHA256SUMS | 198 | afd1c8a0679b5a7ac8dad1e45d668d08d67249af08b578a700e3ef8d616323ff |

`sha256sum --check SHA256SUMS` passed for both downloaded binaries. Native amd64 output was:

```text
DevDock 1.1.0-beta.1 (commit e61a26f5b4ad69b0d7c0891e28532d7577f64e1b, built 2026-10-01T23:07:20Z)
```

The downloaded amd64 TUI started, rendered the temporary project and editor, accepted navigation/favorite/root keys, entered and cancelled the GitHub screen without a missing Client ID error, and quit successfully. This did not complete real OAuth authorization. Arm64 passed checksums, ELF machine/static linkage, Go OS/architecture/CGO metadata, and embedded version/commit/date checks; native execution was unavailable and is not claimed.

The supplied official **public Client ID** was configured as repository variable DEVDOCK_GITHUB_CLIENT_ID and verified nonempty, without printing its value. Both downloaded binaries were checked for embedded ID presence and smoke testing removed runtime overrides. No Client Secret was requested, used, committed, configured or compiled. Custom public-ID self-build instructions remain in README. Application network auth behavior remains covered by fakes; this pass does not claim a live OAuth completion or independent verification of app registration settings.

An initial verification harness expected linker flags in `go version -m`, but Go deliberately omits them with -trimpath (documented in the installed primary Go source, load/pkg.go, issue 52372). The verifier was corrected to check runtime/embedded metadata and buildinfo platform fields. This was a verification assumption, not a release binary defect.

## M. Final branch model

GitHub default is **production**, protected at the released e61a26f commit. Staging is protected and receives next-candidate work. Remote main was deleted only after release success, downloaded checksums/execution, default verification, and git merge-base ancestry checks proving both old origin/main and Pass 1 HEAD reachable from production. No old-main commit was unique to the retired branch. Historical tag API objects were rechecked and unchanged. Temporary development/diagnostic remote branches were removed after use.

The final report/documentation branch starts from staging and fast-forwards to the production merge before its changes, reconciling production ancestry into staging through an ordinary protected PR. Its full hosted checks are required before merging. Production remains the exact released commit; final report-only changes are staged for the next release. Retained local development branches are historical references, not permanent remote branches.

## N. Commits created

In creation order (branch topology is preserved, not rewritten):

1. afb3f30 — fix(ci): expose deterministic Go tool installation on PATH.
2. cdf735a — fix(state): prune favorites and recents on explicit root removal.
3. ab97d19 — ci(release): guard bootstrap publication and require staging promotions.
4. a68ca5c — test(smoke): automate isolated binary startup and cancellation checks.
5. e9b889c — docs: describe guarded solo-maintainer release bootstrap.
6. d82d094 — test: record Pass 1B candidate validation and bootstrap evidence.
7. 7923c42 — GitHub merge of hardening PR #1 into staging.
8. c9c40ef — build(release): prepare v1.1.0-beta.1 pipeline rehearsal.
9. 8dc7932 — fix(ci): isolate production verification check contexts.
10. 59cd595 — GitHub merge of corrective PR #4 into staging.
11. b3e62cf — chore(release): synchronize rehearsal with production CI hardening; merge current staging into smoke branch.
12. da8ad03 — test(ci): probe rejected production promotion on a separate HEAD; diagnostic-only, not merged into staging/production.
13. 395d46e — build(release): merge v1.1.0-beta.1 pipeline rehearsal (#2).
14. e61a26f — build(release): promote v1.1.0-beta.1 from staging (#6).
15. docs: record verified Pass 1B release and branch migration — this final report/evidence commit; its hash and protected PR merge are recorded in the final user report and Git history (a file cannot include its own commit SHA).

The full 15-commit Pass 1 history is now reachable remotely through both permanent branches. No history was rewritten or force-pushed.

## O–P. Limits and readiness

Pass 1B's required remote lifecycle and corrective engineering work have been proven live. The foundation is ready for an independently approved Pass 2; no Pass 2 work has begun.

Remaining limitations: native arm64 execution awaits an arm64 runner; OAuth authorization was intentionally not completed; external editor, live tmux attach and real scaffold packages were not manually exercised here. These retain Pass 1 automated coverage. Configuration/state are separate atomic files, so compensation handles ordinary root-removal save failures but cannot guarantee cross-file power-loss atomicity; a failed compensation is surfaced. Directory fsync/durable power-loss guarantees remain deferred. Public OAuth app registration ownership/settings are controlled by the owner. Release assets are checksummed, without signing/provenance enhancements; those fit later release hardening. Rulesets have no bypass actor, though an administrator can still edit repository settings. GitHub check association is SHA-based; production/integration context isolation and the independent-HEAD negative probe address the observed collision, without claiming exhaustive adversarial workflow testing.

The final documentation PR preserves exact hosted evidence separately from earlier bootstrap logs. No release was fabricated manually to conceal workflow failure. All source/configuration/workflow corrections used purpose-specific branches and green protected merges. Stop here; future hierarchy, identities, favorites product behavior and visual redesign remain outside this pass.
