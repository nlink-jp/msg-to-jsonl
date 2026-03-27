package parser

import (
	"fmt"
	"strings"
	"time"
)

// Parse reads raw MSG file bytes and returns a structured Email.
// source is recorded as-is in the Source field (e.g. a file path or "stdin").
func Parse(data []byte, source string) (*Email, error) {
	doc, err := loadDocument(data)
	if err != nil {
		return nil, fmt.Errorf("reading compound file: %w", err)
	}
	return buildEmail(doc, source), nil
}

// buildEmail assembles an Email from the parsed OLE2 document.
func buildEmail(doc *document, source string) *Email {
	m := doc.root
	cpid := m.cpid()

	email := &Email{
		Source:      source,
		Body:        []BodyPart{},
		Attachments: []Attachment{},
	}

	// --- Headers ---
	email.MessageID = strings.TrimSpace(m.getString(propMessageID, cpid))
	email.InReplyTo = strings.TrimSpace(m.getString(propInReplyTo, cpid))
	email.Subject = m.getString(propSubject, cpid)

	// From: prefer SMTP address, fall back to whatever address is stored.
	senderName := m.getString(propSenderName, cpid)
	senderEmail := m.getString(propSenderSMTP, cpid)
	if senderEmail == "" {
		senderEmail = m.getString(propSenderEmail, cpid)
	}
	if strings.HasPrefix(senderEmail, "/O=") || strings.HasPrefix(senderEmail, "/o=") {
		senderEmail = "" // discard X.400 internal Exchange addresses
	}
	email.From = formatName(senderName, senderEmail)

	// Date: prefer delivery time, fall back to client submit time.
	if t, ok := m.getTime(propDeliveryTime); ok {
		email.Date = t.Format(time.RFC3339)
	} else if t, ok := m.getTime(propClientSubmitTime); ok {
		email.Date = t.Format(time.RFC3339)
	}

	// X-Mailer from raw transport headers (only present for internet mail).
	if raw := m.getString(propTransportHeaders, cpid); raw != "" {
		email.XMailer = parseTransportHeaders(raw)
	}

	// --- Recipients ---
	// First try structured recipient storages for accurate To/CC/BCC split.
	if len(doc.recipients) > 0 {
		for _, r := range doc.recipients {
			recipCPID := r.cpid()
			if recipCPID == 0 {
				recipCPID = cpid
			}
			addr := r.formatAddress(recipCPID)
			if addr == "" {
				continue
			}
			rtype, _ := r.getInt(propRecipType)
			switch rtype {
			case recipCC:
				email.CC = append(email.CC, addr)
			case recipBCC:
				email.BCC = append(email.BCC, addr)
			default: // 1 = To, or unknown → treat as To
				email.To = append(email.To, addr)
			}
		}
	} else {
		// Fall back to the flat display strings (semicolon-separated).
		email.To = splitAddressList(m.getString(propDisplayTo, cpid))
		email.CC = splitAddressList(m.getString(propDisplayCC, cpid))
		email.BCC = splitAddressList(m.getString(propDisplayBCC, cpid))
	}

	// --- Body ---
	// text/plain first, then text/html (mirrors lite-eml ordering).
	if plain := m.getString(propBody, cpid); plain != "" {
		email.Body = append(email.Body, BodyPart{Type: "text/plain", Content: plain})
	}

	// HTML body is stored as PT_BINARY; it may be UTF-8, UTF-16LE, or codepage-encoded.
	if htmlBytes := m.getBinary(propHTMLBody); len(htmlBytes) > 0 {
		htmlContent := decodeHTMLBody(htmlBytes, cpid)
		email.Body = append(email.Body, BodyPart{Type: "text/html", Content: htmlContent})
	}

	// --- Encoding ---
	// Only set when String8 properties were used with a non-UTF-8 codepage.
	if hasString8Body(m) {
		label := cpidToEncodingLabel(cpid)
		if label != "" {
			email.Encoding = label
		}
	}

	// --- Attachments ---
	for _, a := range doc.attachments {
		att := buildAttachment(a, cpid)
		if att != nil {
			email.Attachments = append(email.Attachments, *att)
		}
	}

	return email
}

// buildAttachment extracts attachment metadata from a MAPI attachment storage.
func buildAttachment(a mapiProps, parentCPID int32) *Attachment {
	cpid := a.cpid()
	if cpid == 0 {
		cpid = parentCPID
	}

	// Long filename preferred over 8.3 filename.
	filename := a.getString(propAttachLongName, cpid)
	if filename == "" {
		filename = a.getString(propAttachFilename, cpid)
	}

	mimeType := a.getString(propAttachMIMETag, cpid)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Size: use PR_ATTACH_SIZE if present, otherwise measure the data.
	size := 0
	if v, ok := a.getInt(propAttachSize); ok && v > 0 {
		size = int(v)
	} else if b := a.getBinary(propAttachDataBin); b != nil {
		size = len(b)
	}

	// Skip embedded MSG sub-objects (no filename, no data).
	if filename == "" && size == 0 {
		return nil
	}

	return &Attachment{
		Filename: filename,
		MIMEType: mimeType,
		Size:     size,
	}
}

// hasString8Body reports whether the message uses String8 (PT_STRING8) for
// the body — indicating a legacy codepage-encoded message.
func hasString8Body(m mapiProps) bool {
	if s, ok := m[propBody]; ok && s.propType == typeString8 {
		return true
	}
	if s, ok := m[propHTMLBody]; ok && s.propType == typeString8 {
		return true
	}
	return false
}

// decodeHTMLBody converts raw HTML body bytes to a UTF-8 string.
// The HTML body in MSG files may begin with a BOM or charset declaration.
func decodeHTMLBody(b []byte, cpid int32) string {
	// Check for UTF-16LE BOM (FF FE).
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return decodeUTF16LE(b[2:])
	}
	// Check for UTF-8 BOM (EF BB BF).
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return string(b[3:])
	}
	// Use the message codepage for String8 data, or assume UTF-8.
	if cpid != 0 && cpid != 65001 {
		charset := cpidToCharset(cpid)
		if charset != "" {
			return decodeString8(b, cpid)
		}
	}
	return string(b)
}

// formatName builds "Name <email>" or just the available piece.
func formatName(name, email string) string {
	if name != "" && email != "" {
		return name + " <" + email + ">"
	}
	if email != "" {
		return email
	}
	return name
}

// splitAddressList splits a semicolon-separated display address list.
func splitAddressList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
