# 設計概要

## 目的

msg-to-jsonl は Outlook の MSG ファイル（OLE2/MAPI 形式）を解析し、構造化された JSONL を標準出力へ書き出す。
出力スキーマは eml-to-jsonl と同一なので、両ツールは形式変換なしで同じ下流のパイプラインへ
流し込める。

## MSG ファイルの形式

MSG ファイルは Compound File Binary（MS-CFB）のコンテナである。中のメッセージは MAPI プロパティの集まり
（MS-OXMSG）で、可変長の値は `__substg1.0_PPPPTTTT`（`PPPP` がプロパティ ID、`TTTT` が型、16 進）という
名前のストリームに、固定長の値（日時・整数・真偽値）は各オブジェクトの `__properties_version1.0`
ストリームに入る。受信者は `__recip_version1.0_#XXXXXXXX`、添付は `__attach_version1.0_#XXXXXXXX` の
サブストレージで、埋め込みメッセージは添付の中のストレージである。

## 解析パイプライン

```
[]byte（ファイルの内容）
  ├─ mailparse.ParseMSG()       — 複合ファイル、MAPI プロパティ、ヘッダー、本文、
  │                               添付、埋め込みメッセージ、警告
  └─ buildEmail()               — mailparse.Message を eml-to-jsonl のスキーマに写す
```

入れ物の読み取りは mailparse（github.com/nlink-jp/mailparse、lib-series）が受け持ち、このツールは
JSON の各フィールドに何を入れるかだけを決める。MAPI プロパティを自分では読まない。

## ヘッダーと差出人・宛先

Outlook が残したインターネットヘッダー（`PR_TRANSPORT_MESSAGE_HEADERS`）があれば、mailparse がそこから
`from`、`to`、`cc`、`bcc`、`subject`、`message_id`、`date`、`in_reply_to` を読み、このツールは同じヘッダーから
`x_mailer`、`received` を取る。受信者の表にしか無い Bcc（送信者の控え）は表から補い、`bcc_from_mapi` の
警告を付ける。ヘッダーに無い表の受信者は `mapi_header_mismatch` で知らせる。ヘッダーが無ければ mailparse が
MAPI プロパティから埋め（`in_reply_to` は `PidTagInReplyToId` から）、`no_transport_headers` の警告を付ける。
受信者オブジェクトも無ければ、`to`/`cc`/`bcc` は Outlook の表示用の一覧から名前だけで埋め、
`recipient_type` の警告を付ける。

`date` は RFC 3339。Date ヘッダーならそのオフセットのまま、MAPI の日時なら UTC（送信日時、無ければ
配信日時）。解析できない Date ヘッダーは、eml-to-jsonl と同じく書かれたまま出す。

`received` はすべての Received を順に並べ、各行を trim して 1 つの空白でつなぐ。

## アドレスの扱い

メールボックスは `Name <address>`、または有る方だけを書く。`<`、`>`、`@`、`,`、`;`、`:`、`"`、`\` を含む
表示名は、eml-to-jsonl と同じく RFC 5322 の quoted-string にする。`Bank <security@bank.example>` に復号される
名前も名前のままである。mailparse はヘッダーのアドレスを解析してから
表示名を復号する（符号化した表示名でアドレスを仕込めない）。MAPI のメールボックスでは SMTP アドレスを
`Address` に入れ、Exchange の `/O=...` 形式の識別名は入れない。そのため、その形式しか無いメールボックスは
表示名だけを書く。

## 本文

`PR_BODY` は text/plain、`PR_HTML` は text/html になる。mailparse は圧縮 RTF（`PR_RTF_COMPRESSED`）の
本文 — 包まれた HTML かテキスト、または素の RTF の見える文字 — も返す。それは他のプロパティがその種類を
出していないときだけ出力する。HTML を RTF の中にだけ持つ HTML メールには HTML が出て、両方を持つ
メールでは繰り返さない。

## encoding フィールド

Unicode でなかった最初の本文の元の文字セット（`Shift_JIS`、`ISO-2022-JP`、`EUC-JP`、`Windows-1252`
など）。Unicode のプロパティ、UTF-8・ASCII のテキストでは省く。mailparse は 8 ビットの文字列を
`PidTagMessageCodepage`、本文を `PidTagInternetCodepage` で復号する（MS-OXCMSG）。

## 警告

`warnings` は最上位のメッセージに対する mailparse の警告（code、detail、part）で、空なら省く。入れ子の
メッセージ自身の警告は含めない。

## 上限

1 ファイルあたり入力 100 MiB（超えればエラー）、デコード後のデータ 200 MiB。それ以外は mailparse の
既定値（入れ子の深さ 3、パーツ 1,000）に従う。

## eml-to-jsonl とのスキーマ互換性

出力の JSON スキーマは、意図的に eml-to-jsonl と同一にしている。これにより次のことができる:

```sh
{ eml-to-jsonl dir/eml/; msg-to-jsonl dir/msg/; } | lagent -p "各メールを要約してください。"
```
