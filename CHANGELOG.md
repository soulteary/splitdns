# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-08-08

### Added

- Initial release of `splitdns`, a macOS-only CLI for managing `/etc/resolver`
  suffix-based Split DNS.
- Commands: `add`, `set`, `remove` (alias `rm`), `list` (alias `ls`), `show`,
  `check`, `test`, `flush`, `completion`, and `version`.
- **Atomic writes** (same-directory temp file → `fsync` → `chmod 0644` →
  `rename`) with automatic **backup** before any overwrite.
- **Symlink protection** and strict path containment: operations are refused if
  the target is a symlink, is not a regular file, or would escape
  `/etc/resolver`.
- **Config-preserving parser** that retains comments, blank lines, ordering,
  duplicate nameservers, and unknown-but-valid directives.
- Diagnostics (`check`) and an end-to-end resolution `test` using a hand-written
  minimal DNS query (UDP with timeout, TCP fallback).
- Machine-readable `--json` output for every command, plus global `--dry-run`,
  `--quiet`, and `--no-color` flags.
- macOS DNS cache flushing (`dscacheutil -flushcache` +
  `killall -HUP mDNSResponder`).

[Unreleased]: https://github.com/soulteary/splitdns/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/soulteary/splitdns/releases/tag/v0.1.0
