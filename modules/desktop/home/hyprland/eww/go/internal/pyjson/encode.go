package pyjson

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Encode renders value as CPython's json.dumps does with separators=(",", ":").
//
// ensureASCII is false for the state emitter and true for the control socket.
// Swapping them changes every byte of the snapshot without breaking anything
// visibly. encoding/json is not used: it escapes <, > and &, sorts map keys,
// formats floats by its own rule, and renders a nil slice as null.
func Encode(value any, ensureASCII bool) (string, error) {
	var b strings.Builder
	if err := encodeValue(&b, reflect.ValueOf(value), ensureASCII); err != nil {
		return "", err
	}
	return b.String(), nil
}

func encodeValue(b *strings.Builder, v reflect.Value, ascii bool) error {
	if !v.IsValid() {
		b.WriteString("null")
		return nil
	}

	// json.Number before the generic string case: it carries its own literal.
	if v.Type() == reflect.TypeOf(json.Number("")) {
		literal := v.String()
		if literal == "" {
			b.WriteString("null")
			return nil
		}
		b.WriteString(literal)
		return nil
	}

	if ordered, ok := v.Interface().(Ordered); ok {
		return encodeOrdered(b, ordered, ascii)
	}

	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			b.WriteString("null")
			return nil
		}
		return encodeValue(b, v.Elem(), ascii)

	case reflect.Bool:
		if v.Bool() {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString(strconv.FormatInt(v.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		b.WriteString(strconv.FormatUint(v.Uint(), 10))

	case reflect.Float32, reflect.Float64:
		return encodeFloat(b, v.Float())

	case reflect.String:
		b.WriteString(encodeString(v.String(), ascii))

	case reflect.Slice, reflect.Array:
		// A nil slice renders as [], not null: eww.yuck has nine (for ...) loops
		// and eight arraylength() calls over these fields, and the . index
		// operator hard-errors on null.
		b.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := encodeValue(b, v.Index(i), ascii); err != nil {
				return err
			}
		}
		b.WriteByte(']')

	case reflect.Map:
		return encodeMap(b, v, ascii)

	case reflect.Struct:
		return encodeStruct(b, v, ascii)

	default:
		return fmt.Errorf("pyjson: cannot encode %s", v.Kind())
	}
	return nil
}

// encodeStruct writes fields in DECLARATION order, which is why the bar state is
// a struct rather than a map.
func encodeStruct(b *strings.Builder, v reflect.Value, ascii bool) error {
	b.WriteByte('{')
	first := true
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue // unexported
		}
		name := field.Name
		if tag, ok := field.Tag.Lookup("json"); ok {
			tagName, _, _ := strings.Cut(tag, ",")
			if tagName == "-" {
				continue
			}
			if tagName != "" {
				name = tagName
			}
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(encodeString(name, ascii))
		b.WriteByte(':')
		if err := encodeValue(b, v.Field(i), ascii); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	return nil
}

// encodeMap sorts its keys, which CPython does NOT. Nothing in the emitted state
// is a map; this exists so a future nested map still produces stable output.
func encodeMap(b *strings.Builder, v reflect.Value, ascii bool) error {
	if v.Kind() == reflect.Map && v.IsNil() {
		b.WriteString("null")
		return nil
	}
	keys := make([]string, 0, v.Len())
	byKey := map[string]reflect.Value{}
	for _, key := range v.MapKeys() {
		if key.Kind() != reflect.String {
			return fmt.Errorf("pyjson: map keys must be strings, got %s", key.Kind())
		}
		keys = append(keys, key.String())
		byKey[key.String()] = v.MapIndex(key)
	}
	sort.Strings(keys)

	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(encodeString(key, ascii))
		b.WriteByte(':')
		if err := encodeValue(b, byKey[key], ascii); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	return nil
}

// encodeFloat matches CPython's repr(), which json.dumps uses verbatim. strconv's
// 'g' is not it: repr(1e15) is "1000000000000000.0" where 'g' gives "1e+15", and
// repr(100.0) is "100.0" where 'g' gives "100".
func encodeFloat(b *strings.Builder, f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		// CPython emits NaN/Infinity, which is not valid JSON and which eww
		// would reject. Refuse rather than break the bar.
		return fmt.Errorf("pyjson: cannot encode non-finite float %v", f)
	}

	// Shortest round-tripping digits, then Python's presentation rule:
	// exponent notation when the decimal exponent is < -4 or >= 16.
	shortest := strconv.FormatFloat(f, 'e', -1, 64)
	mantissa, expPart, _ := strings.Cut(shortest, "e")
	exp, err := strconv.Atoi(expPart)
	if err != nil {
		return fmt.Errorf("pyjson: unparsable float %v", f)
	}

	if exp < -4 || exp >= 16 {
		// Python writes the exponent with a sign and at least two digits.
		sign := "+"
		if exp < 0 {
			sign = "-"
			exp = -exp
		}
		// The mantissa keeps whatever shortest form it has: repr(1e+16) is
		// "1e+16" with no ".0", repr(1.5e+20) is "1.5e+20".
		b.WriteString(fmt.Sprintf("%se%s%02d", mantissa, sign, exp))
		return nil
	}

	plain := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(plain, ".") {
		plain += ".0" // repr(100.0) is "100.0", not "100"
	}
	b.WriteString(plain)
	return nil
}

// Pair is one key/value of an Ordered object.
type Pair struct {
	Key   string
	Value any
}

// Ordered is a JSON object that remembers its insertion order, so the
// differential gate can byte-compare nested objects against json.dumps.
type Ordered []Pair

func encodeOrdered(b *strings.Builder, o Ordered, ascii bool) error {
	b.WriteByte('{')
	for i, pair := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(encodeString(pair.Key, ascii))
		b.WriteByte(':')
		if err := encodeValue(b, reflect.ValueOf(pair.Value), ascii); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	return nil
}

// DecodeOrdered parses JSON while preserving object key order.
func DecodeOrdered(data []byte) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	value, err := decodeOrderedValue(decoder)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func decodeOrderedValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch typed := token.(type) {
	case json.Delim:
		switch typed {
		case '{':
			object := Ordered{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("pyjson: object key is not a string")
				}
				value, err := decodeOrderedValue(decoder)
				if err != nil {
					return nil, err
				}
				object = append(object, Pair{Key: key, Value: value})
			}
			if _, err := decoder.Token(); err != nil { // closing }
				return nil, err
			}
			return object, nil
		case '[':
			items := []any{}
			for decoder.More() {
				value, err := decodeOrderedValue(decoder)
				if err != nil {
					return nil, err
				}
				items = append(items, value)
			}
			if _, err := decoder.Token(); err != nil { // closing ]
				return nil, err
			}
			return items, nil
		}
		return nil, fmt.Errorf("pyjson: unexpected delimiter %v", typed)
	}
	return token, nil
}
