# msg-to-jsonl

シェルパイプライン向けの Outlook MSG パーサー。
`.msg` ファイルを読み、構造化された JSONL — 1 メッセージあたり 1 つの JSON オブジェクト — を標準出力へ書き出す。
出力スキーマは [eml-to-jsonl](https://github.com/nlink-jp/eml-to-jsonl) と同一なので、
両ツールは同じパイプラインに自然に並べられる。

## 機能

- ヘッダーを抽出: `from`、`to`、`cc`、`bcc`、`subject`、`date`、`message_id`、`in_reply_to`、`x_mailer`
- Unicode（UTF-16LE）とコードページ符号化（String8）の MAPI プロパティを扱う
- すべてのテキストを **UTF-8** にデコードし、関係がある場合は元の文字セットを `encoding` フィールドに記録する
- 日本語の文字セットに対応: Shift_JIS（CPID 932）とその他の Windows コードページ
- SMTP アドレスを優先し、X.400/Exchange の内部アドレスは黙って捨てる
- MAPI の受信者レコードから To/CC/BCC を構造化して分離
- UTF-16LE の BOM 判定とコードページ判定を伴う HTML 本文のデコード
- 添付ファイルのメタデータ（ファイル名、MIME タイプ、サイズ）— バイナリの内容は埋め込まない
- 入力: 標準入力、ファイル引数、ディレクトリ（ディレクトリ内のすべての `*.msg` を処理）
- API キーもネットワークアクセスも不要 — 純粋にローカルで動くパーサー

## インストール

```sh
git clone https://github.com/nlink-jp/msg-to-jsonl.git
cd msg-to-jsonl
make build
# dist/ を PATH に追加するか、dist/msg-to-jsonl を PATH 上のディレクトリにコピーする
```

## 使い方

```sh
# ファイル 1 つ
msg-to-jsonl message.msg

# ファイル複数
msg-to-jsonl mail1.msg mail2.msg

# ディレクトリ一括（すべての *.msg）
msg-to-jsonl ~/exported-mail/

# 標準入力
cat message.msg | msg-to-jsonl

# 確認用に整形して表示
msg-to-jsonl --pretty message.msg

# 同じパイプラインで eml-to-jsonl と組み合わせる
{ eml-to-jsonl inbox/eml/; msg-to-jsonl inbox/msg/; } | lite-llm -p "Summarise each email."
```

## 出力形式

メッセージ 1 通につき JSON を 1 行出力する（スキーマは eml-to-jsonl と同一）:

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

## フラグ

| フラグ | 既定値 | 説明 |
|------|---------|-------------|
| `-pretty` | false | JSONL ではなく、整形した JSON を出力する |
| `-version` | — | バージョンを表示して終了する |

## ビルド

```sh
make build       # 現在のプラットフォーム向け
make build-all   # リリース対象の全プラットフォーム → dist/
make test        # テストを実行
make check       # vet + lint + test + build + govulncheck
```

## ドキュメント

- [docs/ja/design/overview.ja.md](docs/ja/design/overview.ja.md) — アーキテクチャと設計判断
- [docs/ja/dependencies.ja.md](docs/ja/dependencies.ja.md) — サードパーティ依存

## util-series の一部

msg-to-jsonl は [util-series](https://github.com/nlink-jp/util-series) の一部 —
ローカル LLM とクラウド LLM を扱うための軽量な CLI ツール群。
