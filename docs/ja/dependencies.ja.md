# 依存関係

## ランタイム

| パッケージ | バージョン | 用途 | 自前実装にしない理由 |
|---------|---------|---------|-----------------|
| `github.com/richardlehane/mscfb` | v1.0.6 | OLE2 Compound File Binary Format の解析（MSG コンテナ構造の読み取り） | CFBF の仕様は複雑である（セクタチェーン、FAT、mini-FAT、ディレクトリツリー）。正しい実装は数百行になる。mscfb はこの形式に絞った、テスト済みの Go ライブラリである。 |
| `golang.org/x/text` | v0.35.0 | String8 の MAPI プロパティの文字セット変換（Shift_JIS、GBK、Windows コードページ） | eml-to-jsonl と同じ理由。Go チームが保守する、Go 標準の文字セットライブラリである。 |

## 利用している標準ライブラリのパッケージ

| パッケージ | 用途 |
|---------|---------|
| `encoding/binary` | MAPI プロパティ値のリトルエンディアン整数デコード |
| `unicode/utf16` | PT_UNICODE の MAPI プロパティの UTF-16LE デコード |
| `encoding/hex` | ストリーム名の解析（`__substg1.0_PPPPTTTT`） |
| `encoding/json` | JSONL 出力 |
| `path/filepath` | ディレクトリの glob 展開 |

## 開発

| ツール | 用途 |
|------|---------|
| `golangci-lint` | 静的解析 |
| `govulncheck` | 依存の脆弱性スキャン |
