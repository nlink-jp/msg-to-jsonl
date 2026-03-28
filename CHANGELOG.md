# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.2.0] - 2026-03-28

### Changed

- **Breaking:** renamed from `lite-msg` to `msg-to-jsonl`.
  - Repository: `github.com/nlink-jp/msg-to-jsonl`
  - Module path: `github.com/nlink-jp/msg-to-jsonl`
  - Binary name: `msg-to-jsonl`
  - Moved from lite-series to util-series.

## [0.1.1] - 2026-03-27

### Security

- Added per-stream memory cap (`maxStreamSize = 50 MiB`) using `io.LimitReader`
  when reading MAPI streams from OLE2 compound files, preventing memory exhaustion
  from maliciously oversized streams in crafted MSG files.


## [0.1.0] - 2026-03-27

### Added

- Initial release.
- `msg-to-jsonl`: reads Outlook MSG files from stdin, file arguments, or directories and outputs structured JSONL.
- Extracts headers: From, To, Cc, Bcc, Subject, Date, Message-Id, In-Reply-To, X-Mailer.
- Outputs the same JSON schema as eml-to-jsonl for pipeline compatibility.
- Handles Unicode (UTF-16LE) and String8 (codepage-encoded) MAPI properties.
- Supports Japanese charsets: Shift_JIS (CPID 932), EUC-KR, GBK, Big5, and common Windows codepages.
- Decodes HTML body with UTF-16LE BOM and codepage detection.
- Structured recipient parsing (To/CC/BCC) from MAPI recipient storages.
- Attachment metadata (filename, MIME type, size) without embedding binary content.
- X.400/Exchange internal addresses silently ignored; SMTP addresses preferred.
- `--pretty` flag for human-readable JSON output.


[0.1.1]: https://github.com/nlink-jp/msg-to-jsonl/releases/tag/v0.1.1
[0.1.0]: https://github.com/nlink-jp/msg-to-jsonl/releases/tag/v0.1.0
