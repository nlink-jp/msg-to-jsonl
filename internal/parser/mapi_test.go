package parser

import (
	"encoding/binary"
	"testing"
	"time"
	"unicode/utf16"
)

// encodeUTF16LE encodes a Go string to UTF-16LE bytes for test fixtures.
func encodeUTF16LE(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	b := make([]byte, len(u16)*2+2) // +2 for null terminator
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return b
}

func TestDecodeUTF16LE(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"Hello World"},
		{"日本語テスト"},
		{"Subject: Re: Meeting"},
		{""},
	}
	for _, tt := range tests {
		encoded := encodeUTF16LE(tt.input)
		got := decodeUTF16LE(encoded)
		if got != tt.input {
			t.Errorf("decodeUTF16LE(%q) = %q, want %q", tt.input, got, tt.input)
		}
	}
}

func TestMapiProps_getString_Unicode(t *testing.T) {
	want := "テストメール"
	m := mapiProps{
		propSubject: mapiStream{
			propID:   propSubject,
			propType: typeUnicode,
			data:     encodeUTF16LE(want),
		},
	}
	got := m.getString(propSubject, 0)
	if got != want {
		t.Errorf("getString(propSubject) = %q, want %q", got, want)
	}
}

func TestMapiProps_getString_String8_ShiftJIS(t *testing.T) {
	// "テスト" in Shift_JIS
	shiftjis := []byte{0x83, 0x65, 0x83, 0x58, 0x83, 0x67}
	m := mapiProps{
		propSubject: mapiStream{
			propID:   propSubject,
			propType: typeString8,
			data:     shiftjis,
		},
	}
	got := m.getString(propSubject, 932)
	if got != "テスト" {
		t.Errorf("getString(String8/ShiftJIS) = %q, want テスト", got)
	}
}

func TestMapiProps_getTime(t *testing.T) {
	// 2026-03-27 10:00:00 UTC as Windows FILETIME
	// Unix timestamp: 1774526400
	// FILETIME = (unixSeconds + 11644473600) * 10000000
	unixSec := int64(1774526400)
	ft := uint64((unixSec+11644473600)*10000000)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, ft)

	m := mapiProps{
		propDeliveryTime: mapiStream{
			propID:   propDeliveryTime,
			propType: typeSystime,
			data:     b,
		},
	}
	got, ok := m.getTime(propDeliveryTime)
	if !ok {
		t.Fatal("getTime() returned false")
	}
	want := time.Unix(unixSec, 0).UTC()
	if !got.Equal(want) {
		t.Errorf("getTime() = %v, want %v", got, want)
	}
}

func TestMapiProps_getInt(t *testing.T) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, 1) // recipTo = 1
	m := mapiProps{
		propRecipType: mapiStream{
			propID:   propRecipType,
			propType: typeLong,
			data:     b,
		},
	}
	got, ok := m.getInt(propRecipType)
	if !ok {
		t.Fatal("getInt() returned false")
	}
	if got != 1 {
		t.Errorf("getInt() = %d, want 1", got)
	}
}

func TestFormatAddress(t *testing.T) {
	m := mapiProps{
		propDisplayName: mapiStream{propID: propDisplayName, propType: typeUnicode, data: encodeUTF16LE("Alice")},
		propSMTPAddr:    mapiStream{propID: propSMTPAddr, propType: typeUnicode, data: encodeUTF16LE("alice@example.com")},
	}
	got := m.formatAddress(0)
	want := "Alice <alice@example.com>"
	if got != want {
		t.Errorf("formatAddress() = %q, want %q", got, want)
	}
}

func TestFormatAddress_X400Ignored(t *testing.T) {
	m := mapiProps{
		propDisplayName: mapiStream{propID: propDisplayName, propType: typeUnicode, data: encodeUTF16LE("Bob")},
		propEmailAddr:   mapiStream{propID: propEmailAddr, propType: typeUnicode, data: encodeUTF16LE("/O=CONTOSO/OU=EXCHANGE/CN=Bob")},
	}
	got := m.formatAddress(0)
	// X.400 address should be ignored; only name returned.
	want := "Bob"
	if got != want {
		t.Errorf("formatAddress() = %q, want %q", got, want)
	}
}

func TestParseTransportHeaders(t *testing.T) {
	headers := "From: sender@example.com\r\nX-Mailer: Outlook 16.0\r\nTo: recv@example.com\r\n"
	got := parseTransportHeaders(headers)
	if got != "Outlook 16.0" {
		t.Errorf("parseTransportHeaders() = %q, want %q", got, "Outlook 16.0")
	}
}

func TestSplitAddressList(t *testing.T) {
	got := splitAddressList("Alice; Bob; Carol")
	want := []string{"Alice", "Bob", "Carol"}
	if len(got) != len(want) {
		t.Fatalf("splitAddressList() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseStreamName_Valid(t *testing.T) {
	stream, ok := parseStreamName("__substg1.0_0037001F", []byte("test"))
	if !ok {
		t.Fatal("parseStreamName returned false for valid name")
	}
	if stream.propID != 0x0037 {
		t.Errorf("propID = 0x%04X, want 0x0037", stream.propID)
	}
	if stream.propType != 0x001F {
		t.Errorf("propType = 0x%04X, want 0x001F", stream.propType)
	}
}

func TestParseStreamName_Invalid(t *testing.T) {
	cases := []string{
		"__substg1.0_", // too short
		"__attach_version1.0_#00000000",
		"Root Entry",
		"__substg1.0_XXXXXXXX", // invalid hex
	}
	for _, c := range cases {
		_, ok := parseStreamName(c, nil)
		if ok {
			t.Errorf("parseStreamName(%q) = true, want false", c)
		}
	}
}

func TestDecodeHTMLBody_UTF16LEBOM(t *testing.T) {
	content := "Hello <b>World</b>"
	// UTF-16LE BOM + content
	data := append([]byte{0xFF, 0xFE}, encodeUTF16LE(content)...)
	got := decodeHTMLBody(data, 0)
	if got != content {
		t.Errorf("decodeHTMLBody(UTF-16LE BOM) = %q, want %q", got, content)
	}
}

func TestCpidToEncodingLabel(t *testing.T) {
	if got := cpidToEncodingLabel(932); got != "Shift_JIS" {
		t.Errorf("cpidToEncodingLabel(932) = %q", got)
	}
	if got := cpidToEncodingLabel(65001); got != "" {
		t.Errorf("cpidToEncodingLabel(65001) = %q, want empty", got)
	}
}
