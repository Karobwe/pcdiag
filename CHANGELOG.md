# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- The standalone Windows executable is published next to the archive in each release.

## [0.1.0] - 2026-10-24

### Added

- Fyne desktop application with Overview, System, Network, Connectivity and Printers tabs.
- Automatic diagnosis: uptime, memory, CPU and disk usage, APIPA addresses, missing gateway, failed or
  degraded connectivity checks, print spooler state, offline printers, missing default printer.
- Connectivity checks: gateway ping, per-server DNS queries, name resolution, Internet TCP and HTTP with
  captive portal / proxy detection, and custom checks from `pcdiag.json`.
- Plain-text and JSON reports: copy to clipboard, save to file, or `-report` command-line mode.
- Optional `pcdiag.json` configuration: support contact, checks and thresholds.
- English and French translations.
- CI on Windows, macOS and Linux; releases packaged with the fyne CLI, started by a tag or manually from GitHub Actions.

[Unreleased]: https://github.com/Karobwe/pcdiag/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Karobwe/pcdiag/releases/tag/v0.1.0
