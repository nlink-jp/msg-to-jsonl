package parser_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/nlink-jp/mailparse"
	"github.com/nlink-jp/mailparse/msgtest"
	"github.com/nlink-jp/msg-to-jsonl/internal/parser"
)

func TestParse_MSG(t *testing.T) {
	sent := time.Date(2026, 11, 7, 1, 2, 3, 0, time.UTC)
	b := msgtest.Message([]msgtest.Prop{
		msgtest.P(msgtest.PidTagSubject, "Hello"),
		msgtest.P(msgtest.PidTagSentRepresentingName, "Doe, Jane"),
		msgtest.P(msgtest.PidTagSentRepresentingSMTPAddress, "jane@example.com"),
		msgtest.P(msgtest.PidTagClientSubmitTime, sent),
		msgtest.P(msgtest.PidTagBody, msgtest.String8("plain")),
	},
		msgtest.Recipient(0, msgtest.P(msgtest.PidTagRecipientType, msgtest.RecipientTo), msgtest.P(msgtest.PidTagSMTPAddress, "bob@example.org")),
		msgtest.Recipient(1, msgtest.P(msgtest.PidTagRecipientType, msgtest.RecipientBcc), msgtest.P(msgtest.PidTagSMTPAddress, "hidden@example.net")),
		msgtest.Attachment(0, []msgtest.Prop{msgtest.P(msgtest.PidTagAttachMethod, uint32(msgtest.AttachEmbeddedMessage))},
			msgtest.Embedded([]msgtest.Prop{msgtest.P(msgtest.PidTagSubject, "inner")})),
	)
	email, err := parser.Parse(b, "x.msg")
	if err != nil {
		t.Fatal(err)
	}
	// A display name with a comma is quoted, so the From reads back as one mailbox.
	if email.From != `"Doe, Jane" <jane@example.com>` || email.Date != "2026-11-07T01:02:03Z" {
		t.Errorf("From %q Date %q", email.From, email.Date)
	}
	if len(email.To) != 1 || email.To[0] != "bob@example.org" || len(email.BCC) != 1 || email.BCC[0] != "hidden@example.net" {
		t.Errorf("To %q BCC %q", email.To, email.BCC)
	}
	if len(email.Attachments) != 1 || email.Attachments[0].Message == nil || email.Attachments[0].Message.Subject != "inner" {
		t.Errorf("Attachments %+v", email.Attachments)
	}
	if !hasNoTransportHeaders(email.Warnings) {
		t.Errorf("Warnings %+v", email.Warnings)
	}
}

func hasNoTransportHeaders(ws []mailparse.Warning) bool {
	for _, w := range ws {
		if w.Code == mailparse.WarnNoTransportHeaders {
			return true
		}
	}
	return false
}

// Anything but a .msg is refused, an .eml included.
func TestParse_RefusesEML(t *testing.T) {
	if _, err := parser.Parse([]byte("From: a@example.com\r\nTo: b@example.com\r\nSubject: s\r\nMessage-ID: <1@example.com>\r\n\r\nbody\r\n"), "x.msg"); err == nil {
		t.Fatal("an .eml was read as a .msg")
	}
}

// A winmail.dat (TNEF) is not a .msg either.
func TestParse_RefusesTNEF(t *testing.T) {
	b := msgtest.TNEF(msgtest.TNEFAttr(msgtest.LevelMessage, msgtest.AttSubject, []byte("s\x00")))
	if _, err := parser.Parse(b, "winmail.dat"); err == nil {
		t.Fatal("a TNEF stream was read as a .msg")
	}
}

func TestParse_TooLarge(t *testing.T) {
	big := bytes.Repeat([]byte("a"), mailparse.DefaultMaxSize+1)
	if _, err := parser.Parse(big, "big.msg"); !errors.Is(err, mailparse.ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}
