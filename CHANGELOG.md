# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).


## [Unreleased]

### Fixed

- **`make verify-release` now fails closed.** Its last block chained unzip, the
  packaged binary's `--version` and `spctl` with `&&` and ended the whole chain
  in `|| true`, so a zip that did not unpack or a binary that did not run exited
  0 and the upload proceeded. Each step is now judged on its own, the packaged
  binary's `--version` must contain the tag being released, and only the
  informational `spctl` line may be ignored. Matches the org template
  (CONVENTIONS.md §Code Signing → Verifying a release).
- **The Linux archives no longer carry macOS file metadata.** macOS `tar` wrote
  each bundled file's extended attributes (`com.apple.provenance`, and a Dropbox
  attribute where the tree is synced) into the `.tar.gz` twice: as AppleDouble
  `._` members, which GNU tar extracts as stray `._<name>` files beside the real
  ones, and as `LIBARCHIVE.xattr.*` / `SCHILY.xattr.*` pax headers, which it
  reports as unknown keywords. `make package` now archives with
  `COPYFILE_DISABLE=1 tar --no-xattrs`; each setting stops one of the two.
  Archives already published still carry them; the files themselves are
  unaffected.

### Documentation

- The pipe examples and comparisons named lite-llm, which is archived; they name
  its successor, llm-cli, in the form its README documents for piped data
  (`… | llm-cli -s "<instruction>"`).

### Internal

- `make verify-release` also judges each Linux archive: no AppleDouble or other
  macOS metadata members — listed with `--options 'tar:!mac-ext'`, because a
  plain macOS listing folds `._` members away — no extended attributes as pax
  headers, and exactly the canonical binary, `README.md` and `LICENSE`, compared
  in the C locale.

## [0.4.0] - 2026-07-12

### Added

- **`LICENSE` file (MIT).** The repository previously had no license file;
  it is now MIT-licensed and the license is bundled in every release archive.

### Removed

- **darwin/amd64 (Intel) pre-built binary.** macOS releases now ship
  **arm64 only**, per the org-wide policy (darwin is Apple-Silicon only; no
  universal binaries). Intel Mac users can build from source.

### Changed

- **Linux release archives are now `.tar.gz`** (darwin/windows remain `.zip`),
  per `nlink-jp/.github` CONVENTIONS.md §Release Archive Standard. Archives
  now bundle `README.md` + `LICENSE` alongside the canonical binary.
- **darwin code-signature identifier** is now the canonical `msg-to-jsonl`
  (was `msg-to-jsonl-darwin-arm64`), set via `codesign -i` so it stays stable
  after the archived binary is renamed to its canonical name.

No change to the binary's behaviour — a packaging / build-config release.

## [0.3.1] - 2026-05-23

### Added

- **`package` Makefile target.** Builds all 5 platforms, signs darwin
  binaries with Developer ID, zips each with README.md using
  versioned naming (`msg-to-jsonl-vX.Y.Z-<os>-<arch>.zip`), and
  notarizes the darwin zips.

### Changed

- **Darwin releases are now Developer ID signed and Apple-notarized.**
  `msg-to-jsonl-v0.3.1-darwin-{amd64,arm64}.zip` carry full Apple
  Developer ID Application signatures and notarization tickets from
  Apple. End users on macOS no longer need to bypass Gatekeeper
  with right-click → Open or `xattr -d com.apple.quarantine` on
  first launch; local users who place `msg-to-jsonl` under
  Dropbox-synced (or any other FileProvider-managed) paths are no
  longer killed by macOS's ad-hoc + provenance distrust policy.
  Pipeline: `scripts/codesign-darwin.sh` +
  `scripts/notarize-darwin.sh`, driven by `make package`. Adopts
  the org-wide convention in `nlink-jp/.github` CONVENTIONS.md
  §Code Signing.
- **Release zip filenames now embed the version**
  (`msg-to-jsonl-vX.Y.Z-<os>-<arch>.zip`), aligning with sibling
  util-series tools. v0.3.0 assets used version-less names.

No behaviour change to the binary itself — feature-wise this is
identical to v0.3.0.

## [0.3.0] - 2026-03-30

### Added

- **`received` field** — All Received headers extracted from MAPI transport headers (property 0x007D) and included as a string array. Handles RFC 2822 folded headers.

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
