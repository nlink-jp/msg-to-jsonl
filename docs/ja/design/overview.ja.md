# 設計概要

## 目的

msg-to-jsonl は Outlook の MSG ファイル（OLE2/MAPI 形式）を解析し、構造化された JSONL を標準出力へ書き出す。
出力スキーマは eml-to-jsonl と同一なので、両ツールは形式変換なしで同じ下流のパイプラインへ
流し込める。

## MSG ファイルの形式

MSG ファイルは OLE2 Compound File Binary Format (CFBF) のコンテナである。
コンテナの中では、メールのデータは MAPI プロパティとして格納されている — `__substg1.0_PPPPTTTT`
という名前のストリームで、`PPPP` がプロパティ ID（16 進）、`TTTT` が型（16 進）である。

受信者は `__recip_version1.0_#XXXXXXXX` という名前のサブストレージに格納される。
添付ファイルは `__attach_version1.0_#XXXXXXXX` にある。

## 解析パイプライン

```
[]byte（ファイルの内容）
  └─ mscfb.New()               — OLE2 複合ファイルの解析
       └─ loadDocument()        — スコープ別のストリーム収集
            ├─ root mapiProps   — メッセージ単位のプロパティ
            ├─ []mapiProps      — 受信者ごとのプロパティ
            └─ []mapiProps      — 添付ごとのプロパティ
                 └─ buildEmail() — Email 構造体を組み立てる
                      ├─ headers は root のプロパティから
                      ├─ To/CC/BCC は受信者ストレージから
                      ├─ body は PR_BODY / PR_HTML から
                      └─ attachments は添付ストレージから
```

## MAPI のプロパティ型

| 型コード | 名称 | Go 側の扱い |
|-----------|------|-------------|
| `001F` | PT_UNICODE | UTF-16LE → UTF-8（`unicode/utf16` を使う） |
| `001E` | PT_STRING8 | コードページ → UTF-8（`golang.org/x/text` を使う） |
| `0040` | PT_SYSTIME | Windows FILETIME → `time.Time` |
| `0003` | PT_LONG | `binary.LittleEndian.Uint32` |
| `0102` | PT_BINARY | 生の `[]byte` |

## アドレスの扱い

Outlook はアドレスを 2 つの形式で格納しうる:
- **SMTP**（`PR_SENDER_SMTP_ADDRESS`、`PR_SMTP_ADDRESS`）: 標準的なメールアドレス — 常にこちらを優先する。
- **X.400**（`PR_SENDER_EMAIL_ADDRESS`、addr-type が `EX` の `PR_EMAIL_ADDRESS`）: `/O=CONTOSO/OU=...`
  のような Exchange 内部のルーティングアドレス。これらは黙って捨てる。

`PR_TRANSPORT_MESSAGE_HEADERS`（元の SMTP ヘッダー。存在する場合）は `X-Mailer` のために解析する。

## encoding フィールド

本文のプロパティが String8 (PT_STRING8) で、かつコードページが UTF-8 以外のときだけ設定する。
文字列を Unicode で格納する近年の Outlook（2007 以降）では、このフィールドを省く。
コードページは `PR_INTERNET_CPID` (0x3FDE) から読む。

## eml-to-jsonl とのスキーマ互換性

出力の JSON スキーマは、意図的に eml-to-jsonl と同一にしている。これにより次のことができる:

```sh
{ eml-to-jsonl dir/eml/; msg-to-jsonl dir/msg/; } | llm-cli -s "Summarise each email."
```
