package scalar

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
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

// UnmarshalJSON implements json.Unmarshaler.
func (s *SV) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var x any
	if err := dec.Decode(&x); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("scalar: trailing data after JSON value")
	}
	s.Set(x)
	return nil
}

// MarshalJSON implements json.Marshaler. Undef encodes as null.
func (s *SV) MarshalJSON() ([]byte, error) {
	switch v := s.val().(type) {
	case nil:
		return []byte("null"), nil
	case *AV:
		return v.MarshalJSON()
	case *HV:
		return v.MarshalJSON()
	default:
		return json.Marshal(v)
	}
}

// MarshalJSON implements json.Marshaler.
func (a *AV) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("null"), nil
	}
	if a.items == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(a.items)
}

// MarshalJSON implements json.Marshaler.
func (h *HV) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	if h.m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(h.m)
}

func normalizeJSONNumber(n json.Number) any {
	if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
		return i
	}
	f, _ := strconv.ParseFloat(string(n), 64)
	return f
}
