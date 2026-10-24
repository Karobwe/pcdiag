# Contributing to PC Diag

Thank you for your interest! Bug reports, new checks and translations are all welcome.

## Development setup

1. Install Go 1.24+ and the [Fyne prerequisites](README.md#building-from-source) (a C compiler, plus the
   OpenGL / X11 / Wayland headers on Linux).
2. Clone the repository and run `go run .`.
3. Install [golangci-lint](https://golangci-lint.run/welcome/install/) v2 for `make lint`.

Useful commands (`make help` lists them all):

| Command | Purpose |
| --- | --- |
| `make test` / `make race` | Unit tests, optionally with the race detector |
| `make vet` | `go vet` for Windows, macOS and Linux at once |
| `make lint` | golangci-lint with the repository configuration |
| `make screenshots` | Regenerate `docs/screenshots` from the sample data in `internal/ui/ui_test.go` |

## Guidelines

- **Code and documentation are written in English.** User-facing strings go through `lang.L("English text")`
  and must be added to every file in `internal/i18n/translations/` (a test enforces it).
- **Keep collectors best effort.** A failing command must never block or crash the application: use
  `runCommand` (it applies a timeout and hides console windows on Windows) and return partial data.
- **Separate collection from parsing.** OS-specific code lives in files with a build constraint
  (`windows.go`, `linux.go`, `darwin.go`); the parsing of command output lives in files without a
  constraint (`winparse.go`, `unixparse.go`) so that it is unit-tested on every platform. Add a test with a
  real output sample, ideally from a non-English system.
- **No elevation, no side effects.** PC Diag must run as a standard user and must not modify the system.
- **Do not leak secrets.** Mask credentials (see `redactURL`) and keep reports free of passwords or tokens.
- Run `gofmt`, `go vet` and the tests before opening a pull request, and add an entry to `CHANGELOG.md`.

## Adding a diagnosis rule

Rules live in `internal/health/health.go`. Add a message constant (it doubles as the translation key),
emit it from `Analyze`, translate it, and cover it in `health_test.go`.

## Adding a language

Copy `internal/i18n/translations/fr.json` to `<language>.json` (for example `de.json` or `pt_BR.json`) and
translate the values; keep the `{{.Placeholders}}` unchanged. Note that go-i18n reserves a few keys
(`Other`, `One`, `Few`, `Many`, `Zero`, `Two`, `ID`, `Description`, `Hash`…), which therefore cannot be
used as source strings.

## Commit messages

Use short imperative messages, optionally prefixed with a scope (`collector: detect APIPA addresses`).
