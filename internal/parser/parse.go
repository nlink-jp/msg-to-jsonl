package parser

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/nlink-jp/mailparse"
)

// Limits for one parse. mailparse's defaults (64 MiB input, 120 MiB decoded)
// suit a mail analyser; a pipeline tool reading saved Outlook messages meets
// larger ones, so the input limit is raised and the decoded budget follows it
// (attachment bytes, bodies and the expanded compressed RTF all count).
const (
	maxInputSize = 100 << 20 // 100 MiB
	maxDecoded   = 2 * maxInputSize
)

var options = mailparse.Options{MaxSize: maxInputSize, MaxDecoded: maxDecoded}

// Parse reads raw MSG file bytes and returns a structured Email.
// source is recorded as-is in the Source field (e.g. a file path or "stdin").
func Parse(data []byte, source string) (*Email, error) {
	m, err := mailparse.ParseMSG(bytes.NewReader(data), options)
	if err != nil {
		return nil, fmt.Errorf("reading .msg: %w", err)
	}
	return buildEmail(m, source), nil
}

// buildEmail maps a parsed message onto the eml-to-jsonl schema.
func buildEmail(m *mailparse.Message, source string) *Email {
	email := &Email{
		Source:      source,
		MessageID:   strings.TrimSpace(m.MessageID),
		InReplyTo:   foldCompat(m.InReplyTo),
		Subject:     m.Subject,
		Body:        []BodyPart{},
		Attachments: []Attachment{},
	}

	// --- Headers ---
	// With internet headers, mailparse reads identity from them; without, from
	// MAPI properties (In-Reply-To from PidTagInReplyToId).
	if f, ok := m.Header.First("X-Mailer"); ok {
		email.XMailer = f.Collapsed()
	}
	for _, f := range m.Header.All("Received") {
		if v := foldCompat(f.Raw); v != "" {
			email.Received = append(email.Received, v)
		}
	}

	email.From = strings.Join(mailboxes(m.From), ", ")
	email.To = mailboxes(m.To)
	email.CC = mailboxes(m.Cc)
	// With headers, Bcc that is only in the recipient table comes from there
	// (warned as bcc_from_mapi). With neither recipient objects nor headers,
	// mailparse fills To/Cc/Bcc from Outlook's display lists, names only
	// (warned as recipient_type).
	email.BCC = mailboxes(m.Bcc)

	switch {
	case !m.Date.IsZero():
		email.Date = m.Date.Format(time.RFC3339)
	case m.DateRaw != "":
		email.Date = m.DateRaw // unparsable Date header: kept as written, as eml-to-jsonl does
	}

	// --- Body ---
	email.Body, email.Encoding = selectBodies(m.Bodies)

	// --- Attachments ---
	for _, a := range m.Attachments {
		mt := a.MediaType
		if mt == "" {
			mt = "application/octet-stream"
		}
		email.Attachments = append(email.Attachments, Attachment{Filename: a.Filename, MIMEType: mt, Size: a.Size})
	}

	for _, w := range m.Warnings {
		email.Warnings = append(email.Warnings, Warning(w))
	}
	return email
}

// selectBodies returns the bodies to output and the original charset label.
//
// A .msg may hold the same text up to three times: PidTagBody (text/plain),
// PidTagHtml (text/html) and the compressed RTF, from which mailparse takes
// the encapsulated HTML or text. The RTF-derived body is added only for a
// media type no other property supplied — an HTML message whose HTML exists
// only inside the RTF — so a message that has both does not repeat its text.
// text/plain comes before text/html, as in eml-to-jsonl.
func selectBodies(bodies []mailparse.Body) ([]BodyPart, string) {
	direct := map[string]bool{}
	for _, b := range bodies {
		if !isRTF(b) {
			direct[b.MediaType] = true
		}
	}
	var plain, html []BodyPart
	encoding := ""
	for _, b := range bodies {
		if isRTF(b) && direct[b.MediaType] {
			continue
		}
		if encoding == "" {
			encoding = charsetLabel(b.Charset)
		}
		p := BodyPart{Type: b.MediaType, Content: b.Content}
		if b.MediaType == "text/html" {
			html = append(html, p)
		} else {
			plain = append(plain, p)
		}
	}
	return append(append([]BodyPart{}, plain...), html...), encoding
}

func isRTF(b mailparse.Body) bool { return b.Origin == "rtf" }

// charsetLabel returns the label for the `encoding` field: the original
// charset of a body that was not Unicode, or "" (omit) for Unicode and ASCII.
func charsetLabel(charset string) string {
	c := strings.ToLower(charset)
	switch {
	case c == "", c == "utf-8", c == "us-ascii", strings.HasPrefix(c, "utf-16"):
		return ""
	}
	if l, ok := charsetLabels[c]; ok {
		return l
	}
	return strings.ToUpper(c)
}

// charsetLabels keeps the labels earlier versions wrote for the code pages
// they knew, and spells the common others as they are usually written.
var charsetLabels = map[string]string{
	"shift_jis":    "Shift_JIS",
	"windows-31j":  "Shift_JIS",
	"gbk":          "GBK",
	"gb18030":      "GB18030",
	"euc-kr":       "EUC-KR",
	"big5":         "Big5",
	"euc-jp":       "EUC-JP",
	"iso-2022-jp":  "ISO-2022-JP",
	"windows-874":  "Windows-874",
	"windows-1250": "Windows-1250",
	"windows-1251": "Windows-1251",
	"windows-1252": "Windows-1252",
	"windows-1253": "Windows-1253",
	"windows-1254": "Windows-1254",
	"windows-1255": "Windows-1255",
	"windows-1256": "Windows-1256",
	"windows-1257": "Windows-1257",
	"windows-1258": "Windows-1258",
}

// mailboxes formats every mailbox of every occurrence as "Name <address>",
// or the part that is present. A mailbox with only an Exchange "/O=..."
// address has no SMTP address in mailparse's Address, so it is written as
// its display name alone, as earlier versions did.
func mailboxes(lists []mailparse.AddressList) []string {
	var out []string
	for _, l := range lists {
		if len(l.Mailboxes) == 0 && !l.Parsed && strings.TrimSpace(l.Raw) != "" {
			out = append(out, foldCompat(l.Raw)) // unreadable field: kept as written
			continue
		}
		for _, mb := range l.Mailboxes {
			if s := formatMailbox(mb); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// formatMailbox writes "Name <addr>", or the part that is present, with the
// rule eml-to-jsonl uses. A display name is attacker text: one that decodes
// to "Bank <security@bank.example>" would otherwise read as a second address,
// so a name holding any character that means something in an address is
// written as an RFC 5322 quoted-string. (mailparse's Mailbox.String quotes
// more — any name beyond atom characters and spaces, "Alice B." among them —
// so the two tools' output would differ.)
func formatMailbox(mb mailparse.Mailbox) string {
	name, addr := strings.TrimSpace(mb.Name), strings.TrimSpace(mb.Address)
	if name != "" && strings.ContainsAny(name, `<>@,;:"\`) {
		name = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name) + `"`
	}
	switch {
	case name != "" && addr != "":
		return name + " <" + addr + ">"
	case addr != "":
		return addr
	}
	return name
}

// foldCompat unfolds a header value the way earlier versions did: every line
// trimmed and the lines joined by one space.
func foldCompat(raw string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var parts []string
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			parts = append(parts, l)
		}
	}
	return strings.Join(parts, " ")
}
