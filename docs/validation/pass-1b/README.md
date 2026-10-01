# Pass 1B candidate validation

All required local checks, minimum-supported Go tests, both release builds, checksums, and the isolated terminal smoke test passed. The deterministic GOBIN was `/tmp/devdock-pass1b-tools`; ShellCheck v0.11.0 was present for actionlint. No real authorization was completed.

Validated source-content fingerprint: `e4f7fab02a1d52ee2d1d49ff887dc21d9e9bae60d5b1ccafaf750bdcfb962047`.

The fingerprint is SHA-256 over sorted tracked filenames and their SHA-256 content hashes, excluding `docs/validation/` and historical `docs/pass-1*` audit/report files. It identifies the complete source/workflow/script/module/README/CONTRIBUTING state without the self-reference introduced by committing logs. Validation is repeated on the final candidate HEAD before push, and its hosted PR run records the exact commit SHA. Build logs assert checkout-matching metadata rather than retaining an earlier commit SHA. Original sibling logs are historical Pass 1 evidence.

Initial staging bootstrap CI run [36932002634](https://github.com/richardnascimento18/devdock/actions/runs/36932002634) failed because hosted ShellCheck found an unquoted version-derived executable path. It was corrected on the development branch; no direct staging correction or check bypass was used.

Pre-flight found existing v1.1.0-beta in addition to v1.0.0-beta. The owner explicitly chose 1.1.0-beta.1 for the rehearsal, preserving both tags/releases. Publication remains disabled for the production bootstrap.
