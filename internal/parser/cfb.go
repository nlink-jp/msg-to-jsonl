package parser

import (
	"bytes"
	"encoding/hex"
	"io"
	"strings"

	"github.com/richardlehane/mscfb"
)

// maxStreamSize is the maximum number of bytes read from a single MAPI stream.
// Prevents memory exhaustion from maliciously oversized streams in crafted MSG files.
const maxStreamSize = 50 * 1024 * 1024 // 50 MiB

// document holds all MAPI properties extracted from an MSG OLE2 compound file,
// organized by scope: root (message), recipients, and attachments.
type document struct {
	root        mapiProps
	recipients  []mapiProps
	attachments []mapiProps
}

// loadDocument reads an MSG file from data and organises its MAPI streams.
func loadDocument(data []byte) (*document, error) {
	r, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	d := &document{root: make(mapiProps)}

	recipIdx := map[string]int{}
	attachIdx := map[string]int{}

	for {
		entry, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Tolerate read errors on individual entries.
			break
		}

		if !strings.HasPrefix(entry.Name, "__substg1.0_") {
			continue
		}

		raw, err := io.ReadAll(io.LimitReader(entry, maxStreamSize))
		if err != nil {
			continue
		}

		stream, ok := parseStreamName(entry.Name, raw)
		if !ok {
			continue
		}

		// Determine the scope by the entry's parent path.
		scope := parentScope(entry.Path)
		switch {
		case scope == "":
			// Root-level message property.
			d.root[stream.propID] = stream

		case strings.HasPrefix(scope, "__recip_version1.0_"):
			idx, exists := recipIdx[scope]
			if !exists {
				idx = len(d.recipients)
				d.recipients = append(d.recipients, make(mapiProps))
				recipIdx[scope] = idx
			}
			d.recipients[idx][stream.propID] = stream

		case strings.HasPrefix(scope, "__attach_version1.0_"):
			idx, exists := attachIdx[scope]
			if !exists {
				idx = len(d.attachments)
				d.attachments = append(d.attachments, make(mapiProps))
				attachIdx[scope] = idx
			}
			d.attachments[idx][stream.propID] = stream
		}
	}

	return d, nil
}

// parentScope returns the immediate parent storage name from a Path slice,
// or "" when the entry is at the root level.
func parentScope(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[len(path)-1]
}

// parseStreamName parses an MSG MAPI stream name of the form
// "__substg1.0_PPPPTTTT" and returns the property ID, type, and raw data.
// The hex digits are uppercase (e.g. "0037001F" → propID=0x0037, type=0x001F).
func parseStreamName(name string, data []byte) (mapiStream, bool) {
	const prefix = "__substg1.0_"
	if !strings.HasPrefix(name, prefix) {
		return mapiStream{}, false
	}
	suffix := name[len(prefix):]
	if len(suffix) != 8 {
		return mapiStream{}, false
	}
	b, err := hex.DecodeString(suffix)
	if err != nil || len(b) != 4 {
		return mapiStream{}, false
	}
	propID := uint16(b[0])<<8 | uint16(b[1])
	propType := uint16(b[2])<<8 | uint16(b[3])
	return mapiStream{propID: propID, propType: propType, data: data}, true
}
