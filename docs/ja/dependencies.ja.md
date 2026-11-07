# 依存関係

## ランタイム

| パッケージ | バージョン | 用途 | 自前実装にしない理由 |
|---------|---------|---------|-----------------|
| `github.com/nlink-jp/mailparse` | v0.2.0 | `.msg` の読み取り: 複合ファイル（MS-CFB から書いた `cfb` パッケージ）、MAPI プロパティ（MS-OXMSG）、コードページ、圧縮 RTF、警告 | 自前である — 組織の共通のメールのパーサー（lib-series）。v0.5.0 より前に使っていたコミュニティ製の `github.com/richardlehane/mscfb` と `msoleps` を置き換える。 |
| `golang.org/x/text` | v0.35.0 | 文字セットの変換（mailparse を通した間接依存） | Go チームが保守する、Go 標準の文字セットライブラリである。 |
| `golang.org/x/net` | v0.52.0 | HTML の `<meta charset>` の判別（mailparse を通した間接依存） | Go チームが保守する。 |

## 利用している標準ライブラリのパッケージ

| パッケージ | 用途 |
|---------|---------|
| `encoding/binary`、`unicode/utf16` | mailparse の外で読む少数のルートのプロパティの UTF-16LE デコード |
| `encoding/json` | JSONL 出力 |
| `path/filepath` | ディレクトリの glob 展開 |

## 開発

| ツール | 用途 |
|------|---------|
| `golangci-lint` | 静的解析 |
| `govulncheck` | 依存の脆弱性スキャン |
