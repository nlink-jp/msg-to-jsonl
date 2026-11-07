# 設計概要

## 目的

msg-to-jsonl は Outlook の .msg ファイルを読み、構造化した JSONL を stdout に出力する。eml-to-jsonl と同じ形を
出力するので、2 つのツールは形式の変換なしに同じ下流のパイプラインへつながる。

## 解析

解析はすべて [mailparse](https://github.com/nlink-jp/mailparse) が行う。複合ファイル（MS-CFB の自前実装）、MAPI
プロパティ、受信したメールが持つインターネットヘッダー（`PR_TRANSPORT_MESSAGE_HEADERS`）、圧縮 RTF、添付された
メールを読む。このツールは自前の解析を持たない。

```
ファイル / stdin
  └─ 64 MiB + 1 バイトまで読む      — mailparse.DefaultMaxSize。超えれば ErrTooLarge
       └─ mailparse.ParseMSG()    — .msg 以外（.eml、winmail.dat）はエラー
            └─ parser.FromMessage() — 下の共通の写し方
```

.msg がインターネットヘッダーを持てば、差出人などの欄はそこから読む（eml-to-jsonl と同じ）。Outlook が表示する MAPI
プロパティと食い違えば mailparse が `mapi_header_mismatch` で知らせ、受信者の表にしか無い Bcc はそこから読む
（`bcc_from_mapi`）。インターネットヘッダーの無い .msg（下書き、送信済み）は MAPI プロパティから読み、
`no_transport_headers` の警告が付く。

## 写し方（eml-to-jsonl と msg-to-jsonl で同一）

`internal/parser/email.go` が `mailparse.Message` を JSONL の 1 行に写す。このファイルは 2 つのツールでバイト単位で
同じなので、2 つは同じ形を出力する。違うのは入口（`parse.go`）だけ。

| フィールド | mailparse から |
|---|---|
| `from`・`sender` | すべての出現のすべての mailbox を `Mailbox.String()` にして `, ` で連ねる |
| `to`・`cc`・`bcc`・`reply_to` | すべての出現のすべての mailbox を `Mailbox.String()` で 1 要素ずつ |
| `return_path` | `Mailbox.String()`。空の逆経路は `<>` |
| `date` | `Date` を RFC 3339 で。無ければ `DateRaw` |
| `subject`・`message_id`・`in_reply_to`・`references` | mailparse の値のまま |
| `x_mailer` | 1 つ目の X-Mailer の `Field.Collapsed()` |
| `received` | すべての Received の `Field.Collapsed()`、順のまま |
| `encoding` | UTF-8・US-ASCII 以外の最初の本文の文字コードを大文字で。.msg の Unicode プロパティ（UTF-16LE）は入れ物の保存形式なので数えない |
| `body` | そのメールの本文。圧縮 RTF から復元した本文（`Origin == "rtf"`）は、他の本文が無いときだけ（.msg の RTF はふつう他の本文の写し） |
| `attachments` | すべての添付の `Filename`・`MediaType`・`Size`。添付されたメールは同じ写し方で `message` に（`source` は無し） |
| `warnings` | そのメール自身の `Warnings`（入れ子のメールは自分の分を持つ）。常に配列 |

アドレスを文字にするのは `Mailbox.String()` だけ。atom だけでない表示名を引用符で囲むので、復号するとアドレスの
形になる表示名も表示名のまま。ここでアドレスを整形するコードは持たない。
