# msg-to-jsonl

シェルパイプライン向けの Outlook MSG パーサー。
`.msg` ファイルを読み、構造化された JSONL — 1 メッセージあたり 1 つの JSON オブジェクト — を標準出力へ書き出す。
出力スキーマは [eml-to-jsonl](https://github.com/nlink-jp/eml-to-jsonl) と同一なので、
両ツールは同じパイプラインに自然に並べられる。

## 機能

- `.msg` は組織のメールパーサー [mailparse](https://github.com/nlink-jp/mailparse)（攻撃者が作ったメールを前提に
  書かれている。MS-CFB の読み取りも自前で、コミュニティ製のパッケージは使わない）で読む
- eml-to-jsonl と同じ形を出力する: `from`、`sender`、`reply_to`、`return_path`、`to`、`cc`、`bcc`、`subject`、
  `date`、`message_id`、`in_reply_to`、`references`、`x_mailer`、`received`、`encoding`、`body`、`attachments`、`warnings`
- 受信したメールはインターネットヘッダーから、それ以外は MAPI プロパティから読む。To/Cc/Bcc は受信者の表から
- Unicode とコードページ（String8）のプロパティ、日本語のコードページ（932、50220〜50222、51932）を **UTF-8** に
- 圧縮 RTF の中にしか無い本文も取り出す
- 添付ファイルのメタデータ（ファイル名、MIME タイプ、サイズ）。添付されたメールは同じ形に解析する
- 入力: stdin、ファイル引数、ディレクトリ（ディレクトリ指定時は `*.msg` を一括処理）
- API キーもネットワークも不要。手元だけで解析する

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

各メッセージが1行の JSON として出力されます（eml-to-jsonl と msg-to-jsonl は同じ形）:

```json
{
  "source": "inbox/message.msg",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "references": "<first@example.com> <xyz@example.com>",
  "from": "山田太郎 <yamada@example.co.jp>",
  "sender": "list@example.org",
  "reply_to": ["\"営業, 東京\" <sales@example.co.jp>"],
  "return_path": "bounce@example.co.jp",
  "to": ["鈴木花子 <suzuki@example.co.jp>"],
  "subject": "テスト",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "received": [
    "from mx2.example.com by mx1.example.net; Mon, 30 Mar 2026 09:00:00 +0900",
    "from sender.example.org by mx2.example.com; Mon, 30 Mar 2026 08:59:55 +0900"
  ],
  "encoding": "ISO-2022-JP",
  "body": [
    {"type": "text/plain", "content": "本文テキスト..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400},
    {"filename": "", "mime_type": "message/rfc822", "size": 2048,
     "message": {"message_id": "<fwd@example.org>", "subject": "元のメール", "body": [], "attachments": [], "warnings": []}}
  ],
  "warnings": [
    {"code": "duplicate_field", "detail": "From appears 2 times; every occurrence is kept"}
  ]
}
```

**アドレス:** 欄のすべての出現の、すべての mailbox。`from` と `sender` は 1 本の文字列（`, ` で連ねる）、`to`・`cc`・
`bcc`・`reply_to` は配列。表示名に atom の文字と空白以外が含まれれば引用符で囲むので、復号すると
`Bank <security@bank.example>` になる表示名も表示名のまま。`return_path` はバウンスなら `<>`。

**本文:** `text/plain` と `text/html` のパートを、メールの中の順に。その他のテキストのパート（`text/calendar`、
`text/csv`）は添付になる。

**encoding フィールド:** UTF-8・US-ASCII 以外の最初の本文の元の文字コード（例: `ISO-2022-JP`）。それ以外は省略。

**attachments フィールド:** メタデータのみ（ファイル名・MIME タイプ・デコード後のバイトサイズ）。バイナリは
埋め込まない。メールに名前が無ければファイル名は空。添付されたメール（`message/rfc822`、または中身でメールと
分かるもの）は `message` に同じ形（`source` は無し）で入る。3 層まで。

**warnings:** mailparse がこのメールで失敗せずに処理したこと（重複したヘッダー、壊れたアドレス、壊れた符号化、
上限など）を `{code, detail, part}` で。コードは安定している（[mailparse の README](https://github.com/nlink-jp/mailparse)）。
常に配列。

**受け付けない入力:** Outlook の `.msg` 以外（`.eml` は eml-to-jsonl を使う。`winmail.dat` も不可）、64 MiB を
超えるファイル。エラーは stderr に出し、他のファイルの処理は続ける。

**値の出どころ:** 受信した `.msg` はインターネットヘッダーを持ち、差出人などの欄はそこから eml-to-jsonl と同じに
読む。MAPI の差出人や件名が食い違えば `mapi_header_mismatch`、受信者の表にしか無い Bcc はそこから読んで
`bcc_from_mapi` を付ける。インターネットヘッダーの無い `.msg`（下書き、送信済み）は MAPI プロパティから読み、
`no_transport_headers` の警告が付く。

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
