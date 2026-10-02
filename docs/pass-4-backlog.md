# Pass 4 backlog (not implemented in Pass 3)

- Node.js GitHub Action deprecation warnings.
- Native arm64 runtime and terminal smoke coverage.
- Crash journals, power-loss operation recovery, durable directory writes.
- Configuration/state multi-file recovery and compensation failure rehearsal.
- Real OAuth authorization, rejection, expiry, interruption and credential flows.
- Real tmux integration, attachment, nested layouts and failure/interruption tests.
- Real scaffold command/package integration and long-lived descendant processes.
- Signal/interruption and shutdown behavior under active external work.
- Immutable release publication hardening (existing rerun asset clobber behavior).
- Exact toolchain/reproducibility and reproducible build infrastructure.
- Signing/provenance and release trust policy.
- Clean-machine installation rehearsal.
- Dependency/security/license audit.
- Final release rehearsal and release-candidate/stable readiness criteria.

Pass 3 must preserve existing safety and only fix a direct regression in these
areas. Cross-build verification does not imply native arm64 runtime verification.
