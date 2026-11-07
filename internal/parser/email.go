// Package parser implements MSG file parsing.
// It extracts headers and body from Outlook MSG files (OLE2/MAPI format),
// outputting the same JSON schema as eml-to-jsonl for pipeline compatibility.
package parser

// Email is the structured representation of a parsed MSG message.
// The schema is intentionally identical to eml-to-jsonl's output.
type Email struct {
	Source      string       `json:"source"`
	MessageID   string       `json:"message_id,omitempty"`
	InReplyTo   string       `json:"in_reply_to,omitempty"`
	From        string       `json:"from,omitempty"`
	To          []string     `json:"to,omitempty"`
	CC          []string     `json:"cc,omitempty"`
	BCC         []string     `json:"bcc,omitempty"`
	Subject     string       `json:"subject,omitempty"`
	Date        string       `json:"date,omitempty"`
	XMailer     string       `json:"x_mailer,omitempty"`
	Received    []string     `json:"received,omitempty"`
	Encoding    string       `json:"encoding,omitempty"`
	Body        []BodyPart   `json:"body"`
	Attachments []Attachment `json:"attachments"`
}

// BodyPart represents a single decoded body section.
type BodyPart struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Attachment holds metadata for an attached file.
type Attachment struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mime_type"`
	Size     int    `json:"size"`
}
