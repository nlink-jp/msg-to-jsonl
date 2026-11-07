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
make verify-release  # gate: notarized, fresh, runs at this version, clean linux archives (run before upload)
go test ./... # tests only
```

## Key structure

```
main.go                        ← entry point, flag parsing, file reading
internal/parser/
  parse.go                     ← Parse(): mailparse.ParseMSG → eml-to-jsonl schema (bodies, encoding, addresses)
  email.go                     ← Email struct (shared schema with eml-to-jsonl, plus warnings)
  parse_test.go                ← tests on .msg files built in code with mailparse/msgtest
```

## Gotchas

- **mailparse reads the container** (`github.com/nlink-jp/mailparse` v0.2.0); this tool only maps
  fields. No community dependencies: mscfb/msoleps were removed in v0.5.0.
- **Fixed-length MAPI properties** (dates, attach method, code pages) live in `__properties_version1.0`;
  mailparse reads them. The old reader looked only at `__substg1.0_` streams, so `date` was always empty.
- **Headers vs MAPI**: with `PR_TRANSPORT_MESSAGE_HEADERS`, identity and recipients come from the
  headers, and mailparse adds a Bcc found only in the recipient table (`bcc_from_mapi`); without,
  from MAPI with a `no_transport_headers` warning. In-Reply-To, the display-list fallback and the
  UTF-16 HTML byte order mark are all mailparse's (v0.2.0); this tool reads no MAPI property itself.
- **Address strings use `formatMailbox`**, the quoting rule eml-to-jsonl uses — not
  `mailparse.Mailbox.String()`, which quotes more ("Alice B.") and would make the two tools differ.
- **RTF bodies** are output only for a media type no other property gave (`selectBodies`).
- **Test fixtures are built in code** (`github.com/nlink-jp/mailparse/msgtest`); never commit a real `.msg`.
- **Same output schema as eml-to-jsonl**: designed to be interchangeable in pipelines.
- **`make build` signs on darwin**: the codesign timestamp needs network access.
- **Module path**: `github.com/nlink-jp/msg-to-jsonl`.
