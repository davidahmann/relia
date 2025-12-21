package crypto

import (
	"encoding/json"
	"testing"
)

func TestCanonicalizeOrdersAndStripsNulls(t *testing.T) {
	input := map[string]any{
		"b": "value",
		"a": 1,
		"c": nil,
		"d": map[string]any{
			"z": nil,
			"y": true,
		},
	}

	got, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	want := `{"a":1,"b":"value","d":{"y":true}}`
	if string(got) != want {
		t.Fatalf("unexpected canonical json:\n%s\nwant:\n%s", got, want)
	}
}

func TestCanonicalizeRejectsFloat(t *testing.T) {
	_, err := Canonicalize(1.25)
	if err != ErrFloatNotAllowed {
		t.Fatalf("expected ErrFloatNotAllowed, got %v", err)
	}
}

func TestCanonicalizeJSONNumberIntegerOnly(t *testing.T) {
	_, err := Canonicalize(json.Number("1.25"))
	if err != ErrFloatNotAllowed {
		t.Fatalf("expected ErrFloatNotAllowed, got %v", err)
	}

	got, err := Canonicalize(json.Number("42"))
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	if string(got) != "42" {
		t.Fatalf("unexpected canonical json: %s", got)
	}
}

func TestCanonicalizeNormalizesNFC(t *testing.T) {
	input := map[string]any{
		"text": "e\u0301",
	}

	got, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	want := "{\"text\":\"\u00e9\"}"
	if string(got) != want {
		t.Fatalf("unexpected canonical json:\n%s\nwant:\n%s", got, want)
	}
}

func TestCanonicalizeMapKeyCollision(t *testing.T) {
	input := map[string]any{
		"e\u0301": 1,
		"\u00e9":  2,
	}

	_, err := Canonicalize(input)
	if err != ErrKeyCollision {
		t.Fatalf("expected ErrKeyCollision, got %v", err)
	}
}

func TestCanonicalizeNonStringMapKey(t *testing.T) {
	input := map[int]any{1: "a"}
	_, err := Canonicalize(input)
	if err != ErrNonStringMapKey {
		t.Fatalf("expected ErrNonStringMapKey, got %v", err)
	}
}

func TestCanonicalizeUnsupportedType(t *testing.T) {
	type payload struct{ A int }

	_, err := Canonicalize(payload{A: 1})
	if err != ErrUnsupportedType {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestCanonicalizeSlices(t *testing.T) {
	input := []any{1, nil, "a"}
	got, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	if string(got) != `[1,null,"a"]` {
		t.Fatalf("unexpected canonical json: %s", got)
	}

	var nilSlice []any
	got, err = Canonicalize(nilSlice)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	if string(got) != "null" {
		t.Fatalf("unexpected canonical json: %s", got)
	}
}
