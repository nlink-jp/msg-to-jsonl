# Design Overview

## Purpose

msg-to-jsonl reads Outlook .msg files and outputs structured JSONL to stdout. It prints the same
schema as eml-to-jsonl, so both tools feed the same downstream pipeline without format conversion.

## Parsing

All parsing is done by [mailparse](https://github.com/nlink-jp/mailparse), which reads the
compound file (its own implementation of MS-CFB), the MAPI properties, the internet headers a
received message carries (`PR_TRANSPORT_MESSAGE_HEADERS`), compressed RTF and attached messages.
This tool holds no parsing of its own.

```
file / stdin
  └─ read up to 64 MiB + 1 byte   — mailparse.DefaultMaxSize; more is ErrTooLarge
       └─ mailparse.ParseMSG()    — anything but a .msg (an .eml, a winmail.dat) is an error
            └─ parser.FromMessage() — the shared mapping below
```

When the .msg carries internet headers, the identity fields come from them, as eml-to-jsonl reads
them; mailparse reports where the MAPI properties Outlook displays disagree
(`mapi_header_mismatch`) and takes a Bcc known only to the recipient table from there
(`bcc_from_mapi`). A .msg without internet headers (a draft, a sent item) is read from its MAPI
properties with a `no_transport_headers` warning.

## Mapping (identical in eml-to-jsonl and msg-to-jsonl)

`internal/parser/email.go` maps a `mailparse.Message` onto the JSONL record. The file is the same in
both tools, byte for byte, so the two print one schema; only the entry (`parse.go`) differs.

| Field | From mailparse |
|---|---|
| `from`, `sender` | every mailbox of every occurrence, `Mailbox.String()`, joined by `, ` |
| `to`, `cc`, `bcc`, `reply_to` | every mailbox of every occurrence, `Mailbox.String()`, one element each |
| `return_path` | `Mailbox.String()`; `<>` for a null reverse path |
| `date` | `Date` in RFC 3339, else `DateRaw` |
| `subject`, `message_id`, `in_reply_to`, `references` | as mailparse gives them |
| `x_mailer` | the first X-Mailer, `Field.Collapsed()` |
| `received` | every Received, `Field.Collapsed()`, in order |
| `encoding` | the first body charset other than UTF-8 / US-ASCII, upper-cased; a .msg Unicode property (UTF-16LE) is the container's storage and does not count |
| `body` | the message's bodies; a body recovered from compressed RTF (`Origin == "rtf"`) only when the message has no other body (in a .msg the RTF normally copies another body) |
| `attachments` | every attachment: `Filename`, `MediaType`, `Size`; an attached message becomes `message`, mapped the same way, without `source` |
| `warnings` | the message's own `Warnings` (a nested message carries its own); always an array |

`Mailbox.String()` is the only way an address becomes text: it quotes a display name that is not
plain atoms, so a name that decodes to an address form stays a name. No address is formatted here.
