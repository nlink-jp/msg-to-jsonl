# msg-to-jsonl

Outlook MSG parser for shell pipelines.
Reads `.msg` files and outputs structured JSONL — one JSON object per message — to stdout.
Uses the same output schema as [eml-to-jsonl](https://github.com/nlink-jp/eml-to-jsonl),
so both tools compose naturally in the same pipeline.

## Features

- Reads `.msg` files with [mailparse](https://github.com/nlink-jp/mailparse), the organization's
  mail parser written for attacker-composed input — no community dependencies
- Extracts headers: `from`, `to`, `cc`, `bcc`, `subject`, `date`, `message_id`, `in_reply_to`, `x_mailer`, `received`
- Uses the internet headers Outlook kept (`PR_TRANSPORT_MESSAGE_HEADERS`) when present; otherwise the
  sender, recipients, subject and date come from MAPI properties (with a `no_transport_headers` warning)
- Reads fixed-length MAPI properties too, so `date` is filled from the submit or delivery time
- Handles Unicode (UTF-16LE) and codepage-encoded (String8) MAPI properties
- Decodes all text to **UTF-8**; records the original charset in the `encoding` field when relevant
- Japanese charsets: Shift_JIS (932), ISO-2022-JP (50220–50222), EUC-JP (51932) and other Windows code pages
- Prefers SMTP addresses; an Exchange `/O=...` address with no SMTP address is written as the display name alone
- Structured To/CC/BCC split from MAPI recipient records (or from the headers when present; a Bcc
  only in the recipient table is still taken from it, with a `bcc_from_mapi` warning)
- A display name holding address characters (`< > @ , ; : " \`) is quoted, as eml-to-jsonl writes it,
  so an encoded name cannot pass for an address
- Bodies from `PR_BODY` and `PR_HTML`, plus the HTML (or text) inside the compressed RTF when no other property holds it
- Attachment metadata (filename, MIME type, size) — binary content not embedded; embedded messages are listed too
- `warnings` lists what the parse met and worked around (damaged structure, guessed charset, limits)
- Input: stdin, file arguments, or directory (processes all `*.msg` in the directory); at most 100 MiB per file
- No API keys or network access required — pure local parser

## Installation

```sh
git clone https://github.com/nlink-jp/msg-to-jsonl.git
cd msg-to-jsonl
make build
# Add dist/ to PATH or copy dist/msg-to-jsonl to a directory on PATH
```

## Usage

```sh
# Single file
msg-to-jsonl message.msg

# Multiple files
msg-to-jsonl mail1.msg mail2.msg

# Directory batch (all *.msg)
msg-to-jsonl ~/exported-mail/

# Stdin
cat message.msg | msg-to-jsonl

# Pretty-print for inspection
msg-to-jsonl --pretty message.msg

# Combine with eml-to-jsonl in the same pipeline
{ eml-to-jsonl inbox/eml/; msg-to-jsonl inbox/msg/; } | lagent -p "Summarise each email."
```

## Output format

Each message produces one JSON line (identical schema to eml-to-jsonl):

```json
{
  "source": "inbox/message.msg",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "from": "Alice <alice@example.com>",
  "to": ["Bob <bob@example.com>"],
  "cc": [],
  "bcc": [],
  "subject": "Hello World",
  "date": "2026-03-27T10:00:00Z",
  "x_mailer": "Microsoft Outlook 16.0",
  "encoding": "Shift_JIS",
  "body": [
    {"type": "text/plain", "content": "Hello..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400}
  ],
  "warnings": [
    {"code": "no_transport_headers", "detail": "the message has no internet headers ...", "part": "msg"}
  ]
}
```

`warnings` is omitted when there are none. `code` is [mailparse's stable warning
code](https://github.com/nlink-jp/mailparse/blob/main/warnings.go); `part` says where it applies
(`msg`, `msg/html`, `msg/attach/0`, ...). A `.msg` saved from Outlook without its internet headers
(a sent item, a draft) always carries `no_transport_headers` and has no `received`.

### Bodies

A `.msg` can hold its text up to three times: `PR_BODY` (text/plain), `PR_HTML` (text/html) and
the compressed RTF, from which the encapsulated HTML or text is taken. The RTF body is output only
for a media type no other property supplied — typically an HTML message whose HTML exists only
inside the RTF — so the same text is not repeated. text/plain comes before text/html.

### Limits

A file larger than 100 MiB is refused with an error. Decoded data (bodies, attachment bytes,
expanded RTF) is bounded at 200 MiB per file; beyond it, later streams are skipped with a
`decoded_limit` warning.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-pretty` | false | Pretty-print JSON instead of JSONL |
| `-version` | — | Print version and exit |

## Building

```sh
make build       # current platform
make build-all   # all release platforms → dist/
make test        # run tests
make check       # vet + lint + test + build + govulncheck
```

## Documentation

- [docs/en/design/overview.md](docs/en/design/overview.md) — architecture and design decisions
- [docs/en/dependencies.md](docs/en/dependencies.md) — third-party dependencies

## Part of util-series

msg-to-jsonl is part of the [util-series](https://github.com/nlink-jp/util-series) —
a collection of lightweight CLI tools for working with local and cloud LLMs.
