package parser

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nlink-jp/mailparse"
	mt "github.com/nlink-jp/mailparse/msgtest"
)

func parse(t *testing.T, data []byte) *Email {
	t.Helper()
	e, err := Parse(data, "test.msg")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return e
}

func hasWarning(e *Email, code string) bool {
	for _, w := range e.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestSubject_Unicode(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagSubject, "テストメール")}))
	if e.Subject != "テストメール" {
		t.Errorf("Subject = %q", e.Subject)
	}
	if e.Source != "test.msg" {
		t.Errorf("Source = %q", e.Source)
	}
}

func TestString8_ShiftJIS(t *testing.T) {
	sjis := mt.String8{0x83, 0x65, 0x83, 0x58, 0x83, 0x67} // "テスト"
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagInternetCodepage, uint32(932)), mt.P(mt.PidTagSubject, sjis), mt.P(mt.PidTagBody, sjis)}))
	if e.Subject != "テスト" {
		t.Errorf("Subject = %q, want テスト", e.Subject)
	}
	if len(e.Body) != 1 || e.Body[0].Content != "テスト" {
		t.Errorf("Body = %+v", e.Body)
	}
	if e.Encoding != "Shift_JIS" {
		t.Errorf("Encoding = %q, want Shift_JIS", e.Encoding)
	}
}

// Fixed-length properties live in __properties_version1.0; earlier versions
// read only __substg1.0_ streams, so a real .msg never had a date.
func TestDate_FromFixedProperties(t *testing.T) {
	submit := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagClientSubmitTime, submit), mt.P(mt.PidTagMessageDeliveryTime, submit.Add(time.Minute))}))
	if e.Date != "2026-03-27T10:00:00Z" {
		t.Errorf("Date = %q", e.Date)
	}
	e = parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagMessageDeliveryTime, submit)}))
	if e.Date != "2026-03-27T10:00:00Z" {
		t.Errorf("Date (delivery time only) = %q", e.Date)
	}
}

func TestRecipients_TypeSplit(t *testing.T) {
	data := mt.Message(nil,
		mt.Recipient(0, mt.P(mt.PidTagRecipientType, mt.RecipientTo), mt.P(mt.PidTagDisplayName, "Alice"), mt.P(mt.PidTagSMTPAddress, "alice@example.com")),
		mt.Recipient(1, mt.P(mt.PidTagRecipientType, mt.RecipientCc), mt.P(mt.PidTagDisplayName, "Carol"), mt.P(mt.PidTagEmailAddress, "carol@example.com")),
		mt.Recipient(2, mt.P(mt.PidTagRecipientType, mt.RecipientBcc), mt.P(mt.PidTagSMTPAddress, "dave@example.com")),
	)
	e := parse(t, data)
	if !reflect.DeepEqual(e.To, []string{"Alice <alice@example.com>"}) {
		t.Errorf("To = %q", e.To)
	}
	if !reflect.DeepEqual(e.CC, []string{"Carol <carol@example.com>"}) {
		t.Errorf("CC = %q", e.CC)
	}
	if !reflect.DeepEqual(e.BCC, []string{"dave@example.com"}) {
		t.Errorf("BCC = %q", e.BCC)
	}
}

func TestFrom_PrefersSMTP(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagSenderName, "Alice"), mt.P(mt.PidTagSenderEmailAddress, "/O=CONTOSO/OU=EXCHANGE/CN=ALICE"), mt.P(mt.PidTagSenderSMTPAddress, "alice@example.com")}))
	if e.From != "Alice <alice@example.com>" {
		t.Errorf("From = %q", e.From)
	}
}

// An Exchange "/O=..." address with no SMTP address is written as the display
// name alone, as earlier versions did.
func TestAddress_X400Ignored(t *testing.T) {
	data := mt.Message([]mt.Prop{mt.P(mt.PidTagSentRepresentingName, "Bob"), mt.P(mt.PidTagSentRepresentingEmailAddress, "/O=CONTOSO/OU=EXCHANGE/CN=Bob")},
		mt.Recipient(0, mt.P(mt.PidTagRecipientType, mt.RecipientTo), mt.P(mt.PidTagDisplayName, "Bob"), mt.P(mt.PidTagEmailAddress, "/O=CONTOSO/OU=EXCHANGE/CN=Bob")))
	e := parse(t, data)
	if e.From != "Bob" {
		t.Errorf("From = %q, want Bob", e.From)
	}
	if !reflect.DeepEqual(e.To, []string{"Bob"}) {
		t.Errorf("To = %q, want [Bob]", e.To)
	}
}

func TestTransportHeaders(t *testing.T) {
	headers := "Received: from mx.example.com\r\n\tby mail.example.net; Fri, 27 Mar 2026 10:00:00 +0900\r\n" +
		"Received: from client.example.com by mx.example.com\r\n" +
		"From: \"Alice, Sales\" <alice@example.com>\r\n" +
		"To: bob@example.net, \"Carol, Ops\" <carol@example.net>\r\n" +
		"Subject: =?UTF-8?B?44OG44K544OI?=\r\n" +
		"Date: Fri, 27 Mar 2026 10:00:00 +0900\r\n" +
		"Message-ID: <abc@example.com>\r\n" +
		"In-Reply-To: <xyz@example.com>\r\n" +
		"X-Mailer: Microsoft Outlook\r\n\t16.0\r\n"
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagTransportMessageHeaders, headers), mt.P(mt.PidTagSubject, "テスト")}))
	if e.XMailer != "Microsoft Outlook 16.0" {
		t.Errorf("XMailer = %q", e.XMailer)
	}
	want := []string{
		"from mx.example.com by mail.example.net; Fri, 27 Mar 2026 10:00:00 +0900",
		"from client.example.com by mx.example.com",
	}
	if !reflect.DeepEqual(e.Received, want) {
		t.Errorf("Received = %q", e.Received)
	}
	// A display name with a comma is quoted, as eml-to-jsonl writes it.
	if e.From != `"Alice, Sales" <alice@example.com>` {
		t.Errorf("From = %q", e.From)
	}
	// A comma inside a quoted display name does not split the address.
	if !reflect.DeepEqual(e.To, []string{"bob@example.net", `"Carol, Ops" <carol@example.net>`}) {
		t.Errorf("To = %q", e.To)
	}
	if e.Subject != "テスト" || e.MessageID != "<abc@example.com>" || e.InReplyTo != "<xyz@example.com>" {
		t.Errorf("Subject/MessageID/InReplyTo = %q %q %q", e.Subject, e.MessageID, e.InReplyTo)
	}
	if e.Date != "2026-03-27T10:00:00+09:00" {
		t.Errorf("Date = %q", e.Date)
	}
	if hasWarning(e, "no_transport_headers") {
		t.Error("unexpected no_transport_headers warning")
	}
}

func TestNoTransportHeaders_Warned(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagSubject, "x"), mt.P(mt.PidTagInternetMessageID, "<id@example.com>"), mt.P(mt.PidTagInReplyToID, "<parent@example.com>")}))
	if !hasWarning(e, "no_transport_headers") {
		t.Errorf("Warnings = %+v, want no_transport_headers", e.Warnings)
	}
	if e.MessageID != "<id@example.com>" || e.InReplyTo != "<parent@example.com>" {
		t.Errorf("MessageID/InReplyTo = %q %q", e.MessageID, e.InReplyTo)
	}
}

func TestDisplayListFallback(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagDisplayTo, "Alice; Bob"), mt.P(mt.PidTagDisplayCc, "Carol")}))
	if !reflect.DeepEqual(e.To, []string{"Alice", "Bob"}) || !reflect.DeepEqual(e.CC, []string{"Carol"}) {
		t.Errorf("To/CC = %q %q", e.To, e.CC)
	}
	if !hasWarning(e, "recipient_type") {
		t.Errorf("Warnings = %+v, want recipient_type", e.Warnings)
	}
}

// With internet headers, a Bcc recipient exists only in the recipient table
// (the sender's copy). Earlier versions read it from there, and so does this
// one, now that mailparse does.
func TestHeaders_BccFromRecipientTable(t *testing.T) {
	headers := "From: alice@example.com\r\nTo: bob@example.net\r\nSubject: x\r\n"
	data := mt.Message([]mt.Prop{mt.P(mt.PidTagTransportMessageHeaders, headers)},
		mt.Recipient(0, mt.P(mt.PidTagRecipientType, mt.RecipientTo), mt.P(mt.PidTagSMTPAddress, "bob@example.net")),
		mt.Recipient(1, mt.P(mt.PidTagRecipientType, mt.RecipientBcc), mt.P(mt.PidTagDisplayName, "Dave"), mt.P(mt.PidTagSMTPAddress, "dave@example.org")),
	)
	e := parse(t, data)
	if !reflect.DeepEqual(e.To, []string{"bob@example.net"}) {
		t.Errorf("To = %q", e.To)
	}
	if !reflect.DeepEqual(e.BCC, []string{"Dave <dave@example.org>"}) {
		t.Errorf("BCC = %q", e.BCC)
	}
	if !hasWarning(e, "bcc_from_mapi") {
		t.Errorf("Warnings = %+v, want bcc_from_mapi", e.Warnings)
	}
}

// A display name that decodes to an address must not read as one in the
// output: "Bank <security@bank.example>" <attacker@example.net> would
// otherwise come out as two addresses.
func TestFormatMailbox_SpoofedName(t *testing.T) {
	headers := "From: =?UTF-8?Q?Bank_<security@bank.example>?= <attacker@example.net>\r\nSubject: x\r\n"
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagTransportMessageHeaders, headers)}))
	if e.From != `"Bank <security@bank.example>" <attacker@example.net>` {
		t.Errorf("From = %q", e.From)
	}
	for in, want := range map[mailparse.Mailbox]string{
		{Name: "山田 太郎", Address: "taro@example.jp"}:    "山田 太郎 <taro@example.jp>",
		{Name: `a "b" \ c,`, Address: "x@example.com"}: `"a \"b\" \\ c," <x@example.com>`,
		{Address: "x@example.com"}:                     "x@example.com",
		{Name: "Exchange User"}:                        "Exchange User",
		{Name: "Ops; Sales"}:                           `"Ops; Sales"`,
	} {
		if got := formatMailbox(in); got != want {
			t.Errorf("formatMailbox(%+v) = %q, want %q", in, got, want)
		}
	}
}

func TestHTMLBody_UTF16LEBOM(t *testing.T) {
	content := "Hello <b>World</b>"
	html := append([]byte{0xFF, 0xFE}, mt.UTF16(content)...)
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagHtml, html)}))
	if len(e.Body) != 1 || e.Body[0].Type != "text/html" || e.Body[0].Content != content {
		t.Errorf("Body = %+v", e.Body)
	}
	if e.Encoding != "" {
		t.Errorf("Encoding = %q, want empty for Unicode", e.Encoding)
	}
	for _, w := range e.Warnings {
		if w.Part == "msg/html" {
			t.Errorf("warning about the replaced HTML decoding: %+v", w)
		}
	}
}

func TestHTMLBody_UTF8BOM(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagHtml, []byte("\xEF\xBB\xBF<p>hi</p>"))}))
	if len(e.Body) != 1 || e.Body[0].Content != "<p>hi</p>" {
		t.Errorf("Body = %+v", e.Body)
	}
}

func TestBody_PlainBeforeHTML(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagHtml, []byte("<p>hi</p>")), mt.P(mt.PidTagBody, "hi")}))
	if len(e.Body) != 2 || e.Body[0].Type != "text/plain" || e.Body[1].Type != "text/html" {
		t.Errorf("Body = %+v", e.Body)
	}
}

// The code pages earlier versions lacked: ISO-2022-JP (50220) and EUC-JP (51932).
func TestCodePages_ISO2022JP_EUCJP(t *testing.T) {
	jis := mt.String8("\x1b$B%F%9%H\x1b(B") // "テスト"
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagMessageCodepage, uint32(50220)), mt.P(mt.PidTagInternetCodepage, uint32(50220)), mt.P(mt.PidTagSubject, jis), mt.P(mt.PidTagBody, jis)}))
	if e.Subject != "テスト" || len(e.Body) != 1 || e.Body[0].Content != "テスト" {
		t.Errorf("ISO-2022-JP: Subject %q Body %+v", e.Subject, e.Body)
	}
	if e.Encoding != "ISO-2022-JP" {
		t.Errorf("ISO-2022-JP: Encoding = %q", e.Encoding)
	}
	euc := mt.String8{0xA5, 0xC6, 0xA5, 0xB9, 0xA5, 0xC8} // "テスト"
	e = parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagMessageCodepage, uint32(51932)), mt.P(mt.PidTagInternetCodepage, uint32(51932)), mt.P(mt.PidTagSubject, euc), mt.P(mt.PidTagBody, euc)}))
	if e.Subject != "テスト" || len(e.Body) != 1 || e.Body[0].Content != "テスト" {
		t.Errorf("EUC-JP: Subject %q Body %+v", e.Subject, e.Body)
	}
	if e.Encoding != "EUC-JP" {
		t.Errorf("EUC-JP: Encoding = %q", e.Encoding)
	}
}

const encapsulatedHTML = `{\rtf1\ansi\ansicpg1252\fromhtml1 {\*\htmltag <html><body>}{\*\htmltag <a href="https://example.com/">}link{\*\htmltag </a>}{\*\htmltag </body></html>}}`

// An HTML message whose HTML exists only inside the compressed RTF.
func TestBody_HTMLOnlyInRTF(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagBody, "link"), mt.P(mt.PidTagRtfCompressed, mt.UncompressedRTF([]byte(encapsulatedHTML)))}))
	if len(e.Body) != 2 || e.Body[0].Type != "text/plain" || e.Body[1].Type != "text/html" {
		t.Fatalf("Body = %+v", e.Body)
	}
	if !strings.Contains(e.Body[1].Content, `href="https://example.com/"`) {
		t.Errorf("RTF HTML = %q", e.Body[1].Content)
	}
}

// When PR_HTML exists, the RTF copy of the same HTML is not repeated.
func TestBody_RTFNotDuplicated(t *testing.T) {
	e := parse(t, mt.Message([]mt.Prop{mt.P(mt.PidTagHtml, []byte("<p>hi</p>")), mt.P(mt.PidTagRtfCompressed, mt.UncompressedRTF([]byte(encapsulatedHTML)))}))
	if len(e.Body) != 1 || e.Body[0].Content != "<p>hi</p>" {
		t.Errorf("Body = %+v", e.Body)
	}
}

func TestAttachments(t *testing.T) {
	data := mt.Message(nil,
		mt.Attachment(0, []mt.Prop{mt.P(mt.PidTagAttachMethod, mt.AttachByValue), mt.P(mt.PidTagAttachLongFilename, "report.pdf"), mt.P(mt.PidTagAttachMimeTag, "application/pdf"), mt.P(mt.PidTagAttachDataBinary, []byte("%PDF-1.4 test"))}),
		mt.Attachment(1, []mt.Prop{mt.P(mt.PidTagAttachMethod, mt.AttachByValue), mt.P(mt.PidTagAttachLongFilename, "blob.bin"), mt.P(mt.PidTagAttachDataBinary, []byte{1, 2, 3})}),
	)
	e := parse(t, data)
	want := []Attachment{
		{Filename: "report.pdf", MIMEType: "application/pdf", Size: 13},
		{Filename: "blob.bin", MIMEType: "application/octet-stream", Size: 3},
	}
	if !reflect.DeepEqual(e.Attachments, want) {
		t.Errorf("Attachments = %+v", e.Attachments)
	}
}

// Earlier versions dropped an embedded message ("no name, no data") and could
// mix its own attachments into the outer message's.
func TestAttachments_EmbeddedMessage(t *testing.T) {
	inner := mt.Embedded([]mt.Prop{mt.P(mt.PidTagSubject, "inner")},
		mt.Attachment(0, []mt.Prop{mt.P(mt.PidTagAttachMethod, mt.AttachByValue), mt.P(mt.PidTagAttachLongFilename, "inner.txt"), mt.P(mt.PidTagAttachDataBinary, []byte("inner"))}))
	data := mt.Message(nil,
		mt.Attachment(0, []mt.Prop{mt.P(mt.PidTagAttachMethod, mt.AttachByValue), mt.P(mt.PidTagAttachLongFilename, "outer.txt"), mt.P(mt.PidTagAttachDataBinary, []byte("outer!"))}),
		mt.Attachment(1, []mt.Prop{mt.P(mt.PidTagAttachMethod, mt.AttachEmbeddedMessage), mt.P(mt.PidTagDisplayName, "Forwarded")}, inner),
	)
	e := parse(t, data)
	want := []Attachment{
		{Filename: "outer.txt", MIMEType: "application/octet-stream", Size: 6},
		{Filename: "Forwarded", MIMEType: "application/vnd.ms-outlook", Size: 0},
	}
	if !reflect.DeepEqual(e.Attachments, want) {
		t.Errorf("Attachments = %+v", e.Attachments)
	}
}

func TestNotAMessage(t *testing.T) {
	for name, data := range map[string][]byte{
		"text":  []byte("From: a@example.com\r\n\r\nhi"),
		"empty": {},
		"cfb without MSG streams": func() []byte {
			b, _ := mt.Build(3, mt.Stream("WordDocument", []byte("x")))
			return b
		}(),
	} {
		if _, err := Parse(data, "x"); err == nil {
			t.Errorf("%s: Parse succeeded, want an error", name)
		}
	}
}

func TestCharsetLabel(t *testing.T) {
	for in, want := range map[string]string{
		"shift_jis": "Shift_JIS", "windows-1252": "Windows-1252", "iso-2022-jp": "ISO-2022-JP",
		"euc-jp": "EUC-JP", "koi8-r": "KOI8-R", "utf-8": "", "us-ascii": "", "utf-16le": "", "": "",
	} {
		if got := charsetLabel(in); got != want {
			t.Errorf("charsetLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldCompat(t *testing.T) {
	if got := foldCompat("  a\r\n\t b\r\n   c "); got != "a b c" {
		t.Errorf("foldCompat = %q", got)
	}
}

// The JSON keys stay those of eml-to-jsonl; warnings is added only when there
// are some.
func TestJSONSchema(t *testing.T) {
	b, err := json.Marshal(&Email{Source: "s", Body: []BodyPart{}, Attachments: []Attachment{}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != `{"source":"s","body":[],"attachments":[]}` {
		t.Errorf("JSON = %s", got)
	}
}
