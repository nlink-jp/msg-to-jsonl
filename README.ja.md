# msg-to-jsonl

シェルパイプライン向けの Outlook MSG パーサー。
`.msg` ファイルを読み、構造化された JSONL — 1 メッセージあたり 1 つの JSON オブジェクト — を標準出力へ書き出す。
出力スキーマは [eml-to-jsonl](https://github.com/nlink-jp/eml-to-jsonl) と同一なので、
両ツールは同じパイプラインに自然に並べられる。

## 機能

- `.msg` を [mailparse](https://github.com/nlink-jp/mailparse)（攻撃者が作ったメールを前提にした組織の
  メールのパーサー）で読む — コミュニティ製の依存はない
- ヘッダーを抽出: `from`、`to`、`cc`、`bcc`、`subject`、`date`、`message_id`、`in_reply_to`、`x_mailer`、`received`
- Outlook が残したインターネットヘッダー（`PR_TRANSPORT_MESSAGE_HEADERS`）があればそれを使う。無ければ
  差出人・宛先・件名・日付を MAPI プロパティから取る（`no_transport_headers` の警告つき）
- 固定長の MAPI プロパティも読むので、`date` は送信日時か配信日時から埋まる
- Unicode（UTF-16LE）とコードページ符号化（String8）の MAPI プロパティを扱う
- すべてのテキストを **UTF-8** にデコードし、関係がある場合は元の文字セットを `encoding` フィールドに記録する
- 日本語の文字セット: Shift_JIS（932）、ISO-2022-JP（50220〜50222）、EUC-JP（51932）とその他の Windows コードページ
- SMTP アドレスを優先する。SMTP アドレスの無い Exchange の `/O=...` アドレスは表示名だけを書く
- MAPI の受信者レコードから To/CC/BCC を構造化して分離（ヘッダーがあればヘッダーから。受信者の表にしか
  無い Bcc は表から取り、`bcc_from_mapi` の警告を付ける）
- アドレスに使う文字（`< > @ , ; : " \`）を含む表示名は、eml-to-jsonl と同じく引用符で囲む。符号化した
  表示名がアドレスに見えることはない
- 本文は `PR_BODY` と `PR_HTML`、さらに他のプロパティに無いときは圧縮 RTF の中の HTML（またはテキスト）
- 添付ファイルのメタデータ（ファイル名、MIME タイプ、サイズ）— バイナリの内容は埋め込まない。埋め込みメッセージも並べる
- `warnings` に、解析中に出会って回避したこと（壊れた構造、推定した文字セット、上限）を並べる
- 入力: 標準入力、ファイル引数、ディレクトリ（ディレクトリ内のすべての `*.msg` を処理）。1 ファイル 100 MiB まで
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
{ eml-to-jsonl inbox/eml/; msg-to-jsonl inbox/msg/; } | lagent -p "各メールを要約してください。"
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
  ],
  "warnings": [
    {"code": "no_transport_headers", "detail": "the message has no internet headers ...", "part": "msg"}
  ]
}
```

`warnings` は警告が無ければ省く。`code` は [mailparse の安定した警告コード](https://github.com/nlink-jp/mailparse/blob/main/warnings.go)、
`part` は当てはまる場所（`msg`、`msg/html`、`msg/attach/0` など）。インターネットヘッダーを持たずに
Outlook から保存された `.msg`（送信済みアイテム、下書き）には必ず `no_transport_headers` が付き、`received` は無い。

### 本文

`.msg` は同じ本文を最大 3 回持ちうる: `PR_BODY`（text/plain）、`PR_HTML`（text/html）、そして圧縮 RTF
（中に包まれた HTML かテキストを取り出す）。RTF 由来の本文は、他のプロパティが出していない種類のときだけ
出力する — 典型は HTML を RTF の中にだけ持つ HTML メール — ので、同じ本文は繰り返さない。text/plain を
text/html より前に置く。

### 上限

100 MiB を超えるファイルはエラーにする。デコード後のデータ（本文、添付のバイト列、展開した RTF）は
1 ファイルあたり 200 MiB まで。超えた分の後のストリームは読まず、`decoded_limit` の警告を付ける。

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
