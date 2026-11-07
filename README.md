# msg-to-jsonl

Outlook MSG parser for shell pipelines.
Reads `.msg` files and outputs structured JSONL — one JSON object per message — to stdout.
Uses the same output schema as [eml-to-jsonl](https://github.com/nlink-jp/eml-to-jsonl),
so both tools compose naturally in the same pipeline.

## Features

- Reads the `.msg` with [mailparse](https://github.com/nlink-jp/mailparse), the organization's mail
  parser written for messages composed by an attacker (its own MS-CFB reader; no community package)
- Prints the same schema as eml-to-jsonl: `from`, `sender`, `reply_to`, `return_path`, `to`, `cc`,
  `bcc`, `subject`, `date`, `message_id`, `in_reply_to`, `references`, `x_mailer`, `received`,
  `encoding`, `body`, `attachments`, `warnings`
- Internet headers of a received message, MAPI properties otherwise; To/Cc/Bcc from the recipient table
- Unicode and code-page (String8) properties, Japanese code pages (932, 50220–50222, 51932) decoded to **UTF-8**
- A body that exists only in the compressed RTF is recovered
- Attachment metadata (filename, MIME type, size); an attached message is parsed into the same shape
- Input: stdin, file arguments, or directory (processes all `*.msg` in the directory)
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

Each message produces one JSON line (msg-to-jsonl and eml-to-jsonl print the same schema):

```json
{
  "source": "inbox/message.msg",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "references": "<first@example.com> <xyz@example.com>",
  "from": "Alice <alice@example.com>",
  "sender": "list@example.org",
  "reply_to": ["\"Sales, Tokyo\" <sales@example.com>"],
  "return_path": "bounce@example.com",
  "to": ["Bob <bob@example.com>"],
  "cc": ["Carol <carol@example.com>"],
  "subject": "Hello World",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "received": [
    "from mx2.example.com by mx1.example.net; Mon, 30 Mar 2026 09:00:00 +0900",
    "from sender.example.org by mx2.example.com; Mon, 30 Mar 2026 08:59:55 +0900"
  ],
  "encoding": "ISO-2022-JP",
  "body": [
    {"type": "text/plain", "content": "Hello..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400},
    {"filename": "", "mime_type": "message/rfc822", "size": 2048,
     "message": {"message_id": "<fwd@example.org>", "subject": "Original", "body": [], "attachments": [], "warnings": []}}
  ],
  "warnings": [
    {"code": "duplicate_field", "detail": "From appears 2 times; every occurrence is kept"}
  ]
}
```

**Addresses:** every mailbox of every occurrence of the field. `from` and `sender` are one string
(mailboxes joined by `, `); `to`, `cc`, `bcc` and `reply_to` are arrays. A display name holding
anything but atom characters and spaces is written as a quoted string, so a name that decodes to
`Bank <security@bank.example>` stays a name. `return_path` is `<>` for a bounce.

**Body:** the `text/plain` and `text/html` parts, in message order. Other text parts
(`text/calendar`, `text/csv`) are attachments.

**Encoding field:** the original charset of the first body that is not UTF-8 or US-ASCII
(e.g. `ISO-2022-JP`); omitted otherwise.

**Attachments:** metadata only (filename, MIME type, decoded byte size); binary content is not
embedded. A filename is empty when the message gives none. An attached message (`message/rfc822`,
or a message recognized by its content) is parsed into `message`, in the same shape (without
`source`), up to three levels deep.

**Warnings:** what mailparse met and handled in this message instead of failing — a duplicated
header, a malformed address, a broken encoding, a limit — as `{code, detail, part}`. The codes are
stable; see the [mailparse README](https://github.com/nlink-jp/mailparse). Always an array.

**Refused input:** anything that is not an Outlook `.msg` (an `.eml`: use eml-to-jsonl; a
`winmail.dat`), and files larger than 64 MiB. The error goes to stderr and the other files are still
processed.

**Where the values come from:** a received `.msg` carries the internet headers, and the identity
fields are read from them as eml-to-jsonl reads them; a disagreeing MAPI sender or subject is
reported as `mapi_header_mismatch`, and a Bcc known only to the recipient table is taken from there
(`bcc_from_mapi`). A `.msg` without internet headers (a draft, a sent item) is read from its MAPI
properties, with a `no_transport_headers` warning.

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
