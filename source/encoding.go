package source

import (
	"strings"
	"unicode/utf8"
)

// csvEncodings are the encodings tried in turn when none is specified. utf-8-sig
// comes first, so every valid utf-8 file is reported as utf-8-sig, even without a
// BOM. cp1252 comes last because it can decode almost any byte sequence.
var csvEncodings = []string{"utf-8-sig", "utf-8", "cp1252"}

// encodingAliases normalizes the encoding name the user passes to --encoding
// into a name decode understands. Only four encodings are accepted, enough for
// csv files exported from spreadsheets; other names are reported as unsupported.
var encodingAliases = map[string]string{
	"utf-8": "utf-8", "utf8": "utf-8", "u8": "utf-8",
	"utf-8-sig": "utf-8-sig", "utf8-sig": "utf-8-sig",
	"cp1252": "cp1252", "windows-1252": "cp1252", "1252": "cp1252",
	"latin-1": "latin-1", "latin1": "latin-1", "iso-8859-1": "latin-1", "iso8859-1": "latin-1", "l1": "latin-1",
}

func normalizeEncoding(name string) (string, bool) {
	n := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
	canon, ok := encodingAliases[n]
	return canon, ok
}

// decode decodes bytes with one encoding and never substitutes replacement
// characters: one undecodable byte makes the whole file undecodable. ok is false
// when decoding fails or the encoding is unknown.
func decode(raw []byte, enc string) (string, bool) {
	canon, ok := normalizeEncoding(enc)
	if !ok {
		return "", false
	}
	switch canon {
	case "utf-8-sig":
		if !utf8.Valid(raw) {
			return "", false
		}
		return strings.TrimPrefix(string(raw), "\ufeff"), true
	case "utf-8":
		if !utf8.Valid(raw) {
			return "", false
		}
		return string(raw), true
	case "latin-1":
		var b strings.Builder
		for _, c := range raw {
			b.WriteRune(rune(c))
		}
		return b.String(), true
	}
	var b strings.Builder
	for _, c := range raw {
		switch {
		case c < 0x80 || c >= 0xA0:
			b.WriteRune(rune(c))
		default:
			r, ok := cp1252High[c]
			if !ok {
				return "", false
			}
			b.WriteRune(r)
		}
	}
	return b.String(), true
}
