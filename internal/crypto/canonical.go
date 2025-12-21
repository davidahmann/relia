package crypto

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"

	"golang.org/x/text/unicode/norm"
)

// Canonicalize encodes v as canonical JSON bytes.
func Canonicalize(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeValue(&buf, v, true); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type mapEntry struct {
	key   string
	value any
}

func writeValue(buf *bytes.Buffer, v any, stripNulls bool) error {
	if v == nil {
		buf.WriteString("null")
		return nil
	}

	switch value := v.(type) {
	case json.Number:
		return writeJSONNumber(buf, value)
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.String:
		return writeString(buf, rv.String())
	case reflect.Bool:
		if rv.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.WriteString(strconv.FormatInt(rv.Int(), 10))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		buf.WriteString(strconv.FormatUint(rv.Uint(), 10))
		return nil
	case reflect.Float32, reflect.Float64:
		return ErrFloatNotAllowed
	case reflect.Map:
		return writeMap(buf, rv, stripNulls)
	case reflect.Slice, reflect.Array:
		return writeSlice(buf, rv, stripNulls)
	case reflect.Invalid:
		buf.WriteString("null")
		return nil
	default:
		return ErrUnsupportedType
	}
}

func writeString(buf *bytes.Buffer, s string) error {
	normalized := norm.NFC.String(s)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	buf.Write(encoded)
	return nil
}

func writeJSONNumber(buf *bytes.Buffer, n json.Number) error {
	if stringsHasFloat(n.String()) {
		return ErrFloatNotAllowed
	}
	value, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil {
		return ErrFloatNotAllowed
	}
	buf.WriteString(strconv.FormatInt(value, 10))
	return nil
}

func writeMap(buf *bytes.Buffer, rv reflect.Value, stripNulls bool) error {
	if rv.Type().Key().Kind() != reflect.String {
		return ErrNonStringMapKey
	}

	entries := make([]mapEntry, 0, rv.Len())
	seen := map[string]struct{}{}

	for _, key := range rv.MapKeys() {
		keyStr := norm.NFC.String(key.String())
		if _, ok := seen[keyStr]; ok {
			return ErrKeyCollision
		}
		seen[keyStr] = struct{}{}

		val := rv.MapIndex(key).Interface()
		if stripNulls && isNilValue(val) {
			continue
		}
		entries = append(entries, mapEntry{key: keyStr, value: val})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	buf.WriteByte('{')
	for i, entry := range entries {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeString(buf, entry.key); err != nil {
			return err
		}
		buf.WriteByte(':')
		if err := writeValue(buf, entry.value, stripNulls); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

func writeSlice(buf *bytes.Buffer, rv reflect.Value, stripNulls bool) error {
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		buf.WriteString("null")
		return nil
	}

	buf.WriteByte('[')
	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeValue(buf, rv.Index(i).Interface(), stripNulls); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func stringsHasFloat(s string) bool {
	for _, r := range s {
		if r == '.' || r == 'e' || r == 'E' {
			return true
		}
	}
	return false
}
