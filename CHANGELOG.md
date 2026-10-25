# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- MSI installer (`pcdiag_<version>_windows_amd64.msi`) for deployment with Group Policy, Intune or
  Configuration Manager, with `CONFIGFILE` and `DESKTOPSHORTCUT` properties. It is built and tested by
  a silent install / uninstall in CI.

## [0.1.1] - 2026-10-25

### Added

- The standalone Windows executable (`pcdiag_<version>_windows_amd64.exe`) is published next to the
  archive, for a direct download.

### Changed

- Releases can be published from GitHub Actions (*Run workflow* with a version) as well as by pushing a tag.

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

[Unreleased]: https://github.com/Karobwe/pcdiag/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/Karobwe/pcdiag/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Karobwe/pcdiag/releases/tag/v0.1.0
