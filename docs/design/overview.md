# Design Overview

## Purpose

lite-msg parses Outlook MSG files (OLE2/MAPI format) and outputs structured JSONL to stdout.
It uses the same output schema as lite-eml, enabling both tools to feed the same downstream
pipeline without format conversion.

## MSG file format

MSG files are OLE2 Compound File Binary Format (CFBF) containers.
Inside the container, email data is stored as MAPI properties — streams named
`__substg1.0_PPPPTTTT` where `PPPP` is the property ID (hex) and `TTTT` is the type (hex).

Recipients are stored in sub-storages named `__recip_version1.0_#XXXXXXXX`.
Attachments are in `__attach_version1.0_#XXXXXXXX`.

## Parsing pipeline

```
[]byte (file content)
  └─ mscfb.New()               — OLE2 compound file parsing
       └─ loadDocument()        — stream collection by scope
            ├─ root mapiProps   — message-level properties
            ├─ []mapiProps      — per-recipient properties
            └─ []mapiProps      — per-attachment properties
                 └─ buildEmail() — assembles Email struct
                      ├─ headers from root props
                      ├─ To/CC/BCC from recipient storages
                      ├─ body from PR_BODY / PR_HTML
                      └─ attachments from attachment storages
```

## MAPI property types

| Type code | Name | Go handling |
|-----------|------|-------------|
| `001F` | PT_UNICODE | UTF-16LE → UTF-8 via `unicode/utf16` |
| `001E` | PT_STRING8 | codepage → UTF-8 via `golang.org/x/text` |
| `0040` | PT_SYSTIME | Windows FILETIME → `time.Time` |
| `0003` | PT_LONG | `binary.LittleEndian.Uint32` |
| `0102` | PT_BINARY | raw `[]byte` |

## Address handling

Outlook can store addresses in two formats:
- **SMTP** (`PR_SENDER_SMTP_ADDRESS`, `PR_SMTP_ADDRESS`): standard email address — always preferred.
- **X.400** (`PR_SENDER_EMAIL_ADDRESS`, `PR_EMAIL_ADDRESS` with addr-type `EX`): internal Exchange
  routing address like `/O=CONTOSO/OU=...`. These are discarded silently.

`PR_TRANSPORT_MESSAGE_HEADERS` (the original SMTP headers, when present) is parsed for `X-Mailer`.

## Encoding field

Set only when String8 (PT_STRING8) body properties are present with a non-UTF-8 code page.
For modern Outlook (2007+) which stores strings as Unicode, the field is omitted.
Code page is read from `PR_INTERNET_CPID` (0x3FDE).

## Schema compatibility with lite-eml

The output JSON schema is intentionally identical to lite-eml. This enables:

```sh
{ lite-eml dir/eml/; lite-msg dir/msg/; } | lite-llm -p "Summarise each email."
```
