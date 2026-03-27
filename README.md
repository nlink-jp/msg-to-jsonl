# lite-msg

Outlook MSG parser for shell pipelines.
Reads `.msg` files and outputs structured JSONL — one JSON object per message — to stdout.
Uses the same output schema as [lite-eml](https://github.com/nlink-jp/lite-eml),
so both tools compose naturally in the same pipeline.

## Features

- Extracts headers: `from`, `to`, `cc`, `bcc`, `subject`, `date`, `message_id`, `in_reply_to`, `x_mailer`
- Handles Unicode (UTF-16LE) and codepage-encoded (String8) MAPI properties
- Decodes all text to **UTF-8**; records original charset in the `encoding` field when relevant
- Supports Japanese charsets: Shift_JIS (CPID 932) and other Windows codepages
- Prefers SMTP addresses; silently discards X.400/Exchange internal addresses
- Structured To/CC/BCC split from MAPI recipient records
- HTML body decoding with UTF-16LE BOM and codepage detection
- Attachment metadata (filename, MIME type, size) — binary content not embedded
- Input: stdin, file arguments, or directory (processes all `*.msg` in the directory)
- No API keys or network access required — pure local parser

## Installation

```sh
git clone https://github.com/nlink-jp/lite-msg.git
cd lite-msg
make build
# Add bin/ to PATH or copy bin/lite-msg to a directory on PATH
```

## Usage

```sh
# Single file
lite-msg message.msg

# Multiple files
lite-msg mail1.msg mail2.msg

# Directory batch (all *.msg)
lite-msg ~/exported-mail/

# Stdin
cat message.msg | lite-msg

# Pretty-print for inspection
lite-msg --pretty message.msg

# Combine with lite-eml in the same pipeline
{ lite-eml inbox/eml/; lite-msg inbox/msg/; } | lite-llm -p "Summarise each email."
```

## Output format

Each message produces one JSON line (identical schema to lite-eml):

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
  ]
}
```

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

- [docs/design/overview.md](docs/design/overview.md) — architecture and design decisions
- [docs/dependencies.md](docs/dependencies.md) — third-party dependencies

## Part of lite-series

lite-msg is part of the [lite-series](https://github.com/nlink-jp/lite-series) —
a collection of lightweight CLI tools for working with local and cloud LLMs.
