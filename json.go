package scalar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FromJSON decodes data into a scalar. Integers that fit in int64 become
// IVs and keep full precision; other numbers become NVs.
func FromJSON(data []byte) (*SV, error) {
	s := &SV{}
	if err := s.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return s, nil
}

// UnmarshalJSON implements json.Unmarshaler. Reading into a locked
// scalar, or an element that can't be written, is an error, with the
// message Set would panic with, and leaves s as it was.
func (s *SV) UnmarshalJSON(data []byte) error {
	if v, ok := readJSON(data); ok {
		return s.setRead(v)
	}
	return s.unmarshalJSONSlow(data)
}

// setRead is Set for UnmarshalJSON and UnmarshalYAML, whose callers expect
// an error rather than a panic: a write that Set would panic on, as one to
// a locked scalar does, is an error with the same message instead.
func (s *SV) setRead(v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			msg, ok := r.(string)
			if !ok || !strings.HasPrefix(msg, "scalar: ") {
				panic(r)
			}
			err = errors.New(msg)
		}
	}()
	s.Set(v)
	return nil
}

// unmarshalJSONSlow decodes with encoding/json, for whatever readJSON
// doesn't take, errors included.
func (s *SV) unmarshalJSONSlow(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var x any
	if err := dec.Decode(&x); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("scalar: trailing data after JSON value")
	}
	return s.setRead(x)
}

// MarshalJSON implements json.Marshaler. Undef encodes as null. A
// reference that leads back to itself is an error, as JSON can't represent
// it; so is a NaN or infinite number, as encoding/json can't either.
func (s *SV) MarshalJSON() ([]byte, error) {
	v, err := s.toAny(map[any]bool{}, true)
	if err != nil {
		return nil, fmt.Errorf("scalar: can't encode a %w as JSON", err)
	}
	return json.Marshal(v)
}

// MarshalJSON implements json.Marshaler, as SV.MarshalJSON does.
func (a *AV) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("null"), nil
	}
	return (&SV{v: a}).MarshalJSON()
}

// MarshalJSON implements json.Marshaler, as SV.MarshalJSON does.
func (h *HV) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	return (&SV{v: h}).MarshalJSON()
}

// jsonNumber is a number's text as a json.Number, for normalizeJSONNumber.
func jsonNumber(text []byte) json.Number { return json.Number(text) }

func normalizeJSONNumber(n json.Number) any {
	// A decimal or exponent skips ParseInt, whose failure allocates.
	if isIntText(string(n)) {
		if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
			return i
		}
	}
	f, _ := strconv.ParseFloat(string(n), 64)
	return keepNumber(string(n), f)
}
