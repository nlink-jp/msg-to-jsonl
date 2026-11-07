package parser

import (
	"bytes"

	"github.com/nlink-jp/mailparse"
)

// Parse reads the bytes of an Outlook .msg file and returns a structured Email.
// source is recorded as-is in the Source field (e.g. a file path or "stdin").
func Parse(data []byte, source string) (*Email, error) {
	m, err := mailparse.ParseMSG(bytes.NewReader(data), mailparse.Options{})
	if err != nil {
		return nil, err
	}
	return FromMessage(m, source), nil
}
