# Deterministic terminal review fixtures

Each `.ansi` file is the real foreground-styled render; `.txt` is the same
render with ANSI removed. No timestamp, filesystem, network or compositor data
is required. Review with `cat testdata/ui/main-wide.ansi` in a transparent terminal.
Use `less -R testdata/ui/help.ansi` for larger fixtures. A text editor can inspect
the matching plain text.

The 40 pairs include wide/medium/narrow dashboard, 200-level tree, duplicate
names, search/collections, inspector, palette/help, bulk confirmation, editors,
loading/error/OAuth, template terminal, static progress and tiny-terminal states.
Motion frames and terminal profiles are explicit, independent of runner env.
Run `python3 scripts/smoke.py PATH_TO_BINARY --oauth-configured
--configuration-editor --reduced-motion --capture /tmp/devdock.ansi` for an
isolated real-terminal capture; omit `--reduced-motion` for animated progress.

Regenerate deliberately after inspecting a presentation change:

```sh
DEVDOCK_UPDATE_GOLDEN=1 go test -run TestRenderGoldens .
go test ./...
```

`TestRenderTransparencyProfiles` checks all fixtures with light/dark terminal
defaults and truecolor/256/basic/no-color profiles. `TestPresentationStyleInvariant`
rejects first-party filled styles; the PTY smoke checks actual emitted bytes.
Codes 0 and 49 are allowed terminal resets. Inverse video is prohibited because
it effectively fills cells even without a literal background setter.

These fixtures establish output behavior. Wallpaper/compositor visibility must
still be inspected by the user; an ANSI capture cannot establish that observation.
