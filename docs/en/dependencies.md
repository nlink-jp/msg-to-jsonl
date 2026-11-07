# Dependencies

## Runtime

| Package | Version | Purpose | Why not in-house |
|---------|---------|---------|-----------------|
| `github.com/nlink-jp/mailparse` | v0.2.0 | Reads the `.msg` file: the compound file (its `cfb` package, written from MS-CFB), MAPI properties (MS-OXMSG), code pages, compressed RTF, warnings | It is in-house: the organization's shared mail parser (lib-series). Replaces the community packages `github.com/richardlehane/mscfb` and `msoleps` used before v0.5.0. |
| `golang.org/x/text` | v0.35.0 | Charset conversion (indirect, through mailparse) | Canonical Go charset library maintained by the Go team. |
| `golang.org/x/net` | v0.52.0 | HTML `<meta charset>` detection (indirect, through mailparse) | Maintained by the Go team. |

## Standard library packages used

| Package | Purpose |
|---------|---------|
| `encoding/binary`, `unicode/utf16` | UTF-16LE decoding of the few root properties read beside mailparse |
| `encoding/json` | JSONL output |
| `path/filepath` | Directory globbing |

## Development

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis |
| `govulncheck` | Dependency vulnerability scanning |
