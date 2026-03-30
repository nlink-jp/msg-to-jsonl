package parser

import (
	"encoding/binary"
	"strings"
	"time"
	"unicode/utf16"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// MAPI property IDs (PR_* constants).
const (
	propSubject          uint16 = 0x0037
	propSenderName       uint16 = 0x0C1A
	propSenderEmail      uint16 = 0x0C1F
	propSenderAddrType   uint16 = 0x0C1E
	propSenderSMTP       uint16 = 0x5D01
	propDisplayTo        uint16 = 0x0E04
	propDisplayCC        uint16 = 0x0E03
	propDisplayBCC       uint16 = 0x0E02
	propBody             uint16 = 0x1000
	propHTMLBody         uint16 = 0x1013
	propDeliveryTime     uint16 = 0x0E06
	propClientSubmitTime uint16 = 0x0039
	propMessageID        uint16 = 0x1035
	propInReplyTo        uint16 = 0x1042
	propTransportHeaders uint16 = 0x007D
	propInternetCPID     uint16 = 0x3FDE

	// Recipient-level properties.
	propRecipType   uint16 = 0x0C15
	propEmailAddr   uint16 = 0x3003
	propDisplayName uint16 = 0x3001
	propSMTPAddr    uint16 = 0x39FE

	// Attachment-level properties.
	propAttachFilename uint16 = 0x3704
	propAttachLongName uint16 = 0x3707
	propAttachMIMETag  uint16 = 0x370E
	propAttachDataBin  uint16 = 0x3701
	propAttachSize     uint16 = 0x0E20

	// Property types.
	typeUnicode uint16 = 0x001F
	typeString8 uint16 = 0x001E
	typeBinary  uint16 = 0x0102
	typeSystime uint16 = 0x0040
	typeLong    uint16 = 0x0003
)

// mapiStream holds a single MAPI property's ID, type, and raw bytes.
type mapiStream struct {
	propID   uint16
	propType uint16
	data     []byte
}

// mapiProps is a map from property ID to its stream.
type mapiProps map[uint16]mapiStream

// getString returns the string value of a MAPI property.
// It handles both Unicode (UTF-16LE) and String8 (codepage-encoded) types.
// cpid is the Windows code page for String8 fallback decoding.
func (m mapiProps) getString(propID uint16, cpid int32) string {
	s, ok := m[propID]
	if !ok {
		return ""
	}
	switch s.propType {
	case typeUnicode:
		return decodeUTF16LE(s.data)
	case typeString8:
		return decodeString8(s.data, cpid)
	}
	return ""
}

// getInt returns the int32 value of a PT_LONG property.
func (m mapiProps) getInt(propID uint16) (int32, bool) {
	s, ok := m[propID]
	if !ok || s.propType != typeLong || len(s.data) < 4 {
		return 0, false
	}
	return int32(binary.LittleEndian.Uint32(s.data[:4])), true
}

// getTime returns the time.Time value of a PT_SYSTIME property.
// MAPI SYSTIME is a Windows FILETIME: 100-nanosecond intervals since 1601-01-01.
func (m mapiProps) getTime(propID uint16) (time.Time, bool) {
	s, ok := m[propID]
	if !ok || s.propType != typeSystime || len(s.data) < 8 {
		return time.Time{}, false
	}
	ft := binary.LittleEndian.Uint64(s.data[:8])
	if ft == 0 {
		return time.Time{}, false
	}
	const filetimeEpoch int64 = 116444736000000000 // 100-ns intervals 1601→1970
	unixNano := (int64(ft) - filetimeEpoch) * 100
	return time.Unix(0, unixNano).UTC(), true
}

// getBinary returns the raw bytes of a PT_BINARY property.
func (m mapiProps) getBinary(propID uint16) []byte {
	s, ok := m[propID]
	if !ok || s.propType != typeBinary {
		return nil
	}
	return s.data
}

// cpid returns the code page ID stored in PR_INTERNET_CPID, or 0 if absent.
func (m mapiProps) cpid() int32 {
	v, _ := m.getInt(propInternetCPID)
	return v
}

// decodeUTF16LE converts a UTF-16LE byte slice to a UTF-8 string.
// Strips a trailing null terminator if present.
func decodeUTF16LE(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	// Strip trailing null terminator (U+0000).
	for len(b) >= 2 && b[len(b)-2] == 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-2]
	}
	if len(b) == 0 {
		return ""
	}
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u16))
}

// decodeString8 decodes a String8 (PT_STRING8) MAPI property using the given
// Windows code page. Falls back to raw UTF-8 interpretation when the code page
// is unknown or zero.
func decodeString8(b []byte, cpid int32) string {
	// Strip trailing null.
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return ""
	}
	charset := cpidToCharset(cpid)
	if charset == "" {
		return string(b)
	}
	enc, err := htmlindex.Get(charset)
	if err != nil {
		return string(b)
	}
	result, _, err := transform.Bytes(enc.NewDecoder(), b)
	if err != nil {
		return string(b)
	}
	return string(result)
}

// cpidToCharset maps a Windows code page identifier to an IANA charset name
// understood by golang.org/x/text/encoding/htmlindex.
func cpidToCharset(cpid int32) string {
	switch cpid {
	case 932:
		return "windows-31j" // Shift_JIS superset
	case 936:
		return "gbk"
	case 949:
		return "euc-kr"
	case 950:
		return "big5"
	case 1250:
		return "windows-1250"
	case 1251:
		return "windows-1251"
	case 1252:
		return "windows-1252"
	case 1253:
		return "windows-1253"
	case 1254:
		return "windows-1254"
	case 1255:
		return "windows-1255"
	case 1256:
		return "windows-1256"
	case 65001:
		return "utf-8"
	default:
		return ""
	}
}

// cpidToEncodingLabel returns a human-readable encoding label for the output
// `encoding` field. Returns "" (omit) for UTF-8 / unknown.
func cpidToEncodingLabel(cpid int32) string {
	switch cpid {
	case 932:
		return "Shift_JIS"
	case 936:
		return "GBK"
	case 949:
		return "EUC-KR"
	case 950:
		return "Big5"
	case 1250:
		return "Windows-1250"
	case 1251:
		return "Windows-1251"
	case 1252:
		return "Windows-1252"
	default:
		return ""
	}
}

// recipientType constants (PR_RECIPIENT_TYPE).
const (
	recipTo  int32 = 1
	recipCC  int32 = 2
	recipBCC int32 = 3
)

// formatAddress builds a "Display Name <email>" string from MAPI recipient props.
func (m mapiProps) formatAddress(cpid int32) string {
	name := m.getString(propDisplayName, cpid)

	// Prefer the SMTP address over the raw email address (which may be X.400 for Exchange).
	email := m.getString(propSMTPAddr, cpid)
	if email == "" {
		email = m.getString(propEmailAddr, cpid)
	}
	// Ignore X.400/Exchange internal addresses.
	if strings.HasPrefix(email, "/O=") || strings.HasPrefix(email, "/o=") {
		email = ""
	}

	if name != "" && email != "" {
		return name + " <" + email + ">"
	}
	if email != "" {
		return email
	}
	return name
}

// parseTransportHeaders parses a raw internet header block for X-Mailer.
func parseTransportHeaders(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "x-mailer:") {
			return strings.TrimSpace(line[len("x-mailer:"):])
		}
	}
	return ""
}

// parseReceivedHeaders extracts all Received header values from a raw header block.
// Handles multi-line (folded) Received headers per RFC 2822.
func parseReceivedHeaders(raw string) []string {
	var received []string
	var current string
	inReceived := false

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if strings.HasPrefix(strings.ToLower(trimmed), "received:") {
			if inReceived && current != "" {
				received = append(received, strings.TrimSpace(current))
			}
			current = strings.TrimSpace(trimmed[len("received:"):])
			inReceived = true
		} else if inReceived && len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t') {
			// Continuation line (folded header)
			current += " " + strings.TrimSpace(trimmed)
		} else {
			if inReceived && current != "" {
				received = append(received, strings.TrimSpace(current))
				current = ""
			}
			inReceived = false
		}
	}
	if inReceived && current != "" {
		received = append(received, strings.TrimSpace(current))
	}
	return received
}
