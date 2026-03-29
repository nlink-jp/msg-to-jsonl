# AGENTS.md — msg-to-jsonl

Outlook MSG parser for shell pipelines.
Reads `.msg` files and outputs structured JSONL — one JSON object per message — to stdout.
Uses the same output schema as [eml-to-jsonl](https://github.com/nlink-jp/eml-to-jsonl).
Part of [util-series](https://github.com/nlink-jp/util-series).

## Rules

- Project rules (security, testing, docs, release, etc.): → [RULES.md](RULES.md)
- Series-wide conventions: → [util-series CONVENTIONS.md](https://github.com/nlink-jp/util-series/blob/main/CONVENTIONS.md)

## Build & test

```sh
make build    # dist/msg-to-jsonl
make check    # vet → lint → test → build → govulncheck
go test ./... # tests only
```

## Key structure

```
main.go                        ← entry point, flag parsing, file reading
internal/parser/
  parse.go                     ← top-level Parse() entry point
  email.go                     ← Email struct, ToJSON() (shared schema with eml-to-jsonl)
  mapi.go                      ← MAPI property extraction (From, To, Subject, Body, etc.)
  mapi_test.go                 ← unit tests
  cfb.go                       ← Compound File Binary (OLE2) container reader
```

## Gotchas

- **CFB/OLE2 parsing**: MSG files are Compound File Binary containers; `cfb.go` reads the FAT chain directly.
- **MAPI properties**: email fields are extracted via MAPI property IDs, not MIME headers.
- **Same output schema as eml-to-jsonl**: designed to be interchangeable in pipelines.
- **Module path**: `github.com/nlink-jp/msg-to-jsonl`.
