# Design Overview

## Purpose

msg-to-jsonl parses Outlook MSG files (OLE2/MAPI format) and outputs structured JSONL to stdout.
It uses the same output schema as eml-to-jsonl, enabling both tools to feed the same downstream
pipeline without format conversion.

## MSG file format

MSG files are Compound File Binary (MS-CFB) containers. Inside, the message is a set of MAPI
properties (MS-OXMSG): variable-length values in streams named `__substg1.0_PPPPTTTT` (`PPPP` the
property ID, `TTTT` the type, in hex), and fixed-length values (times, integers, booleans) in the
`__properties_version1.0` stream of each object. Recipients are sub-storages
`__recip_version1.0_#XXXXXXXX`, attachments `__attach_version1.0_#XXXXXXXX`; an embedded message
is a storage inside its attachment.

## Parsing pipeline

```
[]byte (file content)
  ├─ mailparse.ParseMSG()       — compound file, MAPI properties, headers, bodies,
  │                               attachments, embedded messages, warnings
  └─ buildEmail()               — maps mailparse.Message onto the eml-to-jsonl schema
```

Reading the container is mailparse's job (github.com/nlink-jp/mailparse, lib-series); this tool
only chooses what goes into each JSON field. It reads no MAPI property itself.

## Headers and identity

When the message holds the internet headers Outlook kept (`PR_TRANSPORT_MESSAGE_HEADERS`), mailparse
reads `from`, `to`, `cc`, `bcc`, `subject`, `message_id`, `date` and `in_reply_to` from them, and
this tool takes `x_mailer` and `received` from the same header block. A Bcc that only the recipient
table holds (the sender's copy) is added from the table with a `bcc_from_mapi` warning, and a table
recipient the headers do not name is reported as `mapi_header_mismatch`. Without headers, mailparse
fills the identity fields from MAPI properties (`in_reply_to` from `PidTagInReplyToId`) and warns
`no_transport_headers`; with no recipient objects either, `to`/`cc`/`bcc` come from Outlook's display
lists, names only, with a `recipient_type` warning.

`date` is RFC 3339: the Date header's own offset, or UTC for a MAPI time (submit time first, then
delivery time). A Date header that does not parse is written as it stands, as eml-to-jsonl does.

`received` lists every Received field in order, each line trimmed and the lines joined by one space.

## Address handling

Every mailbox is written as `Name <address>`, or whichever part is present. A display name holding
`<`, `>`, `@`, `,`, `;`, `:`, `"` or `\` is written as an RFC 5322 quoted-string, the rule eml-to-jsonl
uses, so a name that decodes to `Bank <security@bank.example>` stays a name. mailparse parses header
addresses before decoding display names (so an encoded display name cannot inject an address), and
for MAPI mailboxes puts the SMTP address in `Address` — never an Exchange `/O=...` distinguished
name. A mailbox with only such a name is therefore written as its display name alone.

## Bodies

`PR_BODY` becomes text/plain and `PR_HTML` text/html. mailparse also returns the body inside the
compressed RTF (`PR_RTF_COMPRESSED`): the encapsulated HTML or text, or the visible text of plain
RTF. That body is output only when no other property supplied its media type, so an HTML message
whose HTML exists only in the RTF gets its HTML, and a message holding both does not repeat it.

## Encoding field

The original charset of the first body that was not Unicode (`Shift_JIS`, `ISO-2022-JP`, `EUC-JP`,
`Windows-1252`, ...). Omitted for Unicode properties and UTF-8 or ASCII text. mailparse decodes 8-bit
strings with `PidTagMessageCodepage` and bodies with `PidTagInternetCodepage` (MS-OXCMSG).

## Warnings

`warnings` carries mailparse's warnings for the top message (code, detail, part) and is omitted
when empty. Nested messages' own warnings are not included.

## Limits

100 MiB of input per file (larger files are an error), 200 MiB of decoded data per file. mailparse's
other defaults apply (nesting depth 3, 1,000 parts).

## Schema compatibility with eml-to-jsonl

The output JSON schema is intentionally identical to eml-to-jsonl. This enables:

```sh
{ eml-to-jsonl dir/eml/; msg-to-jsonl dir/msg/; } | lagent -p "Summarise each email."
```
