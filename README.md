# splitdns

[![CI](https://github.com/soulteary/splitdns/actions/workflows/ci.yml/badge.svg)](https://github.com/soulteary/splitdns/actions/workflows/ci.yml) [![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Go](https://img.shields.io/github/go-mod/go-version/soulteary/splitdns)](https://github.com/soulteary/splitdns/blob/main/go.mod)

![](.github/assets/splitdns-banner.png)

`splitdns` is a small, safe command-line tool for managing macOS
`/etc/resolver` configuration to enable **suffix-based Split DNS**: routing
queries for specific domain suffixes (e.g. `lab.dev`, `corp.example.com`) to a
dedicated DNS server, while everything else uses your normal resolvers.

It is macOS-only, ships as a single static binary, and never shells out through
`sh -c` — every external command is invoked with separated arguments.

> Read this in another language: [简体中文](./README.zh-CN.md)

## Features

- Create, update, remove, list and inspect `/etc/resolver` entries safely.
- **Atomic writes** (same-directory temp file → `fsync` → `chmod 0644` →
  `rename`) with automatic **backup** before any overwrite.
- **Symlink protection** and strict path containment: operations are refused if
  the target is a symlink, is not a regular file, or would escape
  `/etc/resolver`.
- **Config-preserving parser**: comments, blank lines, ordering, duplicate
  nameservers and unknown-but-valid directives are all retained.
- **Diagnostics** (`check`) and an end-to-end **resolution test** (`test`) that
  probes DNS directly using a hand-written minimal DNS query (UDP with timeout,
  TCP fallback) — never relying on a bare TCP connect.
- Clear separation between "config written" and "cache flushed", so partial
  success is always visible.
- Machine-readable `--json` output for every command.

## Requirements

- macOS (Apple Silicon or Intel).
- `scutil`, `dscacheutil`, `killall` (part of macOS).
- Write operations (`add`, `set`, `remove`, `flush`) modify `/etc/resolver` and
  therefore require `sudo`. `splitdns` **never** auto-elevates; it prints the
  exact `sudo splitdns …` command to run.

## Install

### Homebrew

```bash
brew tap soulteary/tap
brew install soulteary/tap/splitdns
```

Verify:

```bash
splitdns version
```

### From source

```bash
go install github.com/soulteary/splitdns@latest
```

Or build locally:

```bash
git clone https://github.com/soulteary/splitdns.git
cd splitdns
make build      # produces ./splitdns
```

### Release archives

Prebuilt `darwin/amd64` and `darwin/arm64` archives are published via
GoReleaser on the releases page.

## Usage

```
splitdns <command> [flags]

Commands:
  add         Add a new resolver entry for a domain suffix
  set         Update fields of an existing resolver entry
  remove      Remove a resolver entry (alias: rm)
  list        List resolver entries (alias: ls)
  show        Show a resolver entry
  check       Run configuration and environment diagnostics
  test        Test resolution of a hostname through the split-DNS layers
  flush       Flush macOS DNS caches
  completion  Generate shell completion script
  version     Print version information

Global flags:
  --dry-run   Show planned changes without modifying the system
  --json      Emit machine-readable JSON
  --quiet     Suppress non-essential output
  --no-color  Disable colored output
```

### Examples

Route `lab.dev` to a local DNS server on port 53:

```bash
sudo splitdns add lab.dev --nameserver 127.0.0.1 --port 53
```

Point multiple nameservers and a custom port:

```bash
sudo splitdns add corp.example.com \
  --nameserver 10.0.0.53 --nameserver 10.0.1.53 --port 5353
```

Preview changes without writing anything:

```bash
splitdns add lab.dev --dry-run
```

Update only the port of an existing entry (other fields and comments preserved):

```bash
sudo splitdns set lab.dev --port 5353
```

List and inspect:

```bash
splitdns list
splitdns show lab.dev
splitdns show lab.dev --raw       # original file contents
```

Remove (with confirmation, or `--yes` for automation):

```bash
sudo splitdns remove lab.dev
sudo splitdns remove lab.dev --yes
```

Diagnose and test:

```bash
splitdns check                    # validate all entries + environment
splitdns check lab.dev            # focus on one suffix
splitdns test host.lab.dev        # three-layer resolution test
```

Flush caches manually:

```bash
sudo splitdns flush
```

## Command reference

### `add <domain>`

Creates a new resolver file. Fails if one already exists unless `--force`
(which backs up the existing file first).

| Flag | Default | Description |
| --- | --- | --- |
| `--nameserver` | `127.0.0.1` | Nameserver IP (repeatable) |
| `--port` | `53` | DNS server port |
| `--search-order` | | `search_order` value |
| `--timeout` | | `timeout` value (seconds) |
| `--force` | | Allow overwrite and relax `.local`/single-label warnings |
| `--no-flush` | | Do not flush DNS caches after writing |
| `--backup-dir` | system temp | Directory for pre-overwrite backups |

The global `--dry-run` flag previews planned changes (target path, planned
content and cache-flush commands) without writing anything.

### `set <domain>`

Updates an existing entry. Fails if the file is absent. Only the fields you
specify are changed; all other directives and comments are preserved. A backup
is taken before the update.

### `remove <domain>`

Shows the target file, then deletes it. Prompts for confirmation on a TTY; use
`--yes` for automation. In a non-interactive session it refuses to delete
without `--yes`. Honors the global `--dry-run` flag and supports `--no-flush`.

### `list`

Prints a table of entries (name, domain, nameservers, port, managed flag).
`--json` emits a stable array.

### `show <domain>`

Prints a structured view; `--raw` prints the original file verbatim.

### `check [domain]`

Runs diagnostics: platform, resolver directory, filename validity, regular-file
/ symlink checks, permissions, syntax, nameserver/port validity, `.local`
usage, overlapping suffixes, identical configs, DNS reachability (via a real DNS
query), `scutil --dns` load status, and `/etc/hosts` entries affecting the
domain. Any `ERROR` status yields a non-zero exit code.

### `test <hostname>`

Three layers: (1) `dscacheutil -q host`, (2) `scutil --dns` load status,
(3) a direct DNS query against the configured nameserver(s). Reports the
longest matched suffix, resolver file, nameservers, whether the rule is loaded,
system vs. direct addresses, consistency, and troubleshooting hints.

### `flush`

Runs `dscacheutil -flushcache` then `killall -HUP mDNSResponder`. A missing
`mDNSResponder` process is treated as a benign no-op. With the global
`--dry-run` flag it prints the planned commands without executing them.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `2` | Argument / usage error |
| `3` | Permission error (needs `sudo`) |
| `4` | Configuration error |
| `5` | Runtime check failure (e.g. a `check` ERROR, or write OK but flush failed) |

## Security notes

- Domains are normalized (lowercased, trailing dot stripped) and rejected if
  they contain `/`, `\`, `..`, NUL, whitespace or control characters.
- Target paths are cleaned and must stay strictly inside `/etc/resolver`.
- Files are verified to be regular files (via `Lstat`) before any read, write
  or delete; symlinks are refused.
- No `sh -c` and no shell string concatenation; commands use `os/exec` with
  separated arguments.
- `splitdns` never auto-elevates privileges.

## Known limitations

- macOS only. There is no Linux/Windows support.
- `splitdns` manages `/etc/resolver` files only. It does not run a DNS server,
  edit `/etc/hosts`, change global network/DNS settings, or install
  dnsmasq/CoreDNS.
- Entries created by other tools are readable and can be adopted (a
  `# Managed by splitdns` marker is added on write), but `splitdns` will not
  reformat unrelated files.

## Development

```bash
make fmt-check   # gofmt
make vet         # go vet
make test        # unit tests (hermetic; no real system commands)
make lint        # golangci-lint
make build       # build ./splitdns

# Read-only integration tests (macOS; observes real DNS state):
make integration
```

Unit tests are hermetic: all filesystem access uses temp directories and all
external commands go through an injectable fake runner. Integration tests use
the `//go:build integration` tag and never modify `/etc/resolver`.

## License

[Apache-2.0](./LICENSE) © soulteary
