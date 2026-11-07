# Dependencies

## Runtime

| Package | Version | Purpose | Why not in-house |
|---------|---------|---------|-----------------|
| `github.com/richardlehane/mscfb` | v1.0.6 | OLE2 Compound File Binary Format parsing (reading MSG container structure) | The CFBF specification is complex (sector chains, FAT, mini-FAT, directory trees). A correct implementation is several hundred lines; mscfb is a focused, tested Go library for this format. |
| `golang.org/x/text` | v0.35.0 | Charset conversion for String8 MAPI properties (Shift_JIS, GBK, Windows codepages) | Same rationale as eml-to-jsonl: canonical Go charset library maintained by the Go team. |

## Standard library packages used

| Package | Purpose |
|---------|---------|
| `encoding/binary` | Little-endian integer decoding for MAPI property values |
| `unicode/utf16` | UTF-16LE decoding for PT_UNICODE MAPI properties |
| `encoding/hex` | Stream name parsing (`__substg1.0_PPPPTTTT`) |
| `encoding/json` | JSONL output |
| `path/filepath` | Directory globbing |

## Development

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis |
| `govulncheck` | Dependency vulnerability scanning |
