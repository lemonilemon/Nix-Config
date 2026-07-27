package main

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// jsonString renders s the way CPython's
// json.dumps(s, ensure_ascii=True, separators=(",", ":")) does.
//
// encoding/json is deliberately not used for this. It differs from CPython in
// two ways that both reach the wire: it escapes <, > and & as </>/
// & (CPython leaves them literal), and it emits non-ASCII as raw UTF-8
// (CPython escapes it to \uXXXX). Both are decode-identical, but matching
// CPython byte-for-byte means the equivalence gate against the Python client is
// a plain diff rather than a decode-and-compare, which is worth ~30 lines.
func jsonString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			switch {
			case r < 0x20:
				b.WriteString(`\u`)
				b.WriteString(hex4(uint16(r)))
			case r < 0x7f:
				b.WriteRune(r)
			case r == utf8.RuneError:
				// A lone surrogate or invalid byte. CPython's encoder would
				// raise; the client's callers are argv, so emit the
				// replacement character rather than fail a bar click.
				b.WriteString(`�`)
			case r > 0xffff:
				hi, lo := utf16.EncodeRune(r)
				b.WriteString(`\u`)
				b.WriteString(hex4(uint16(hi)))
				b.WriteString(`\u`)
				b.WriteString(hex4(uint16(lo)))
			default:
				b.WriteString(`\u`)
				b.WriteString(hex4(uint16(r)))
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func hex4(v uint16) string {
	const digits = "0123456789abcdef"
	return string([]byte{
		digits[(v>>12)&0xf],
		digits[(v>>8)&0xf],
		digits[(v>>4)&0xf],
		digits[v&0xf],
	})
}

// field is one key/value pair. Payloads are ordered slices rather than maps
// because encoding/json sorts map keys, and the Python client emits them in
// insertion order -- another difference that would show up in a byte diff.
type field struct {
	Key   string
	Value string
}

type payload []field

func (p payload) JSON() string {
	var b strings.Builder
	b.WriteByte('{')
	for i, f := range p {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(jsonString(f.Key))
		b.WriteByte(':')
		b.WriteString(jsonString(f.Value))
	}
	b.WriteByte('}')
	return b.String()
}
