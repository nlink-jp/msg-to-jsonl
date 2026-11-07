// Package parser implements MSG file parsing.
// It reads Outlook MSG files with github.com/nlink-jp/mailparse and maps the
// result onto the same JSON schema as eml-to-jsonl for pipeline compatibility.
package parser

// Email is the structured representation of a parsed MSG message.
// The schema is intentionally identical to eml-to-jsonl's output, plus the
// warnings mailparse reported (omitted when there are none).
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
	Warnings    []Warning    `json:"warnings,omitempty"`
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

// Warning is something the parse met and worked around instead of failing:
// a damaged structure, a missing header block, a guessed charset, a limit.
// Code is mailparse's stable warning code; Part says where (e.g. "msg",
// "msg/attach/0").
type Warning struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Part   string `json:"part,omitempty"`
}
