// Package pyjson renders JSON the way CPython's json module does.
package pyjson

import (
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// String renders s as CPython's json.dumps(s, ensure_ascii=True) does: non-ASCII
// escaped to \uXXXX, and <, > and & left literal. encoding/json does neither.
func String(s string) string {
	return encodeString(s, true)
}

// StringRaw is the ensure_ascii=False form: non-ASCII passes through as UTF-8.
// This is what the state emitter uses.
func StringRaw(s string) string {
	return encodeString(s, false)
}

func encodeString(s string, ascii bool) string {
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
			case !ascii:
				// Everything above ASCII goes out as UTF-8, including U+007F,
				// which the ASCII path escapes.
				b.WriteRune(r)
			case r == utf8.RuneError:
				// A lone surrogate or invalid byte: emit the replacement character
				// rather than fail a bar click.
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

// Payloads are ordered slices rather than maps because encoding/json sorts map
// keys. Field is one key/value pair.
type Field struct {
	Key   string
	Value string
}

// F is shorthand for a Field.
func F(key, value string) Field { return Field{Key: key, Value: value} }

// Payload is an ordered set of fields.
type Payload []Field

// JSON renders the payload with compact separators, keys in insertion order.
func (p Payload) JSON() string {
	var b strings.Builder
	b.WriteByte('{')
	for i, f := range p {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(String(f.Key))
		b.WriteByte(':')
		b.WriteString(String(f.Value))
	}
	b.WriteByte('}')
	return b.String()
}
