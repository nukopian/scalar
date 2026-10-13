package scalar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

// MarshalJSON implements json.Marshaler. Undef encodes as null. A
// reference that leads back to itself is an error, as JSON can't represent
// it; so is a NaN or infinite number, as encoding/json can't either.
func (s *SV) MarshalJSON() ([]byte, error) {
	v, err := s.toAny(map[any]bool{})
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

func normalizeJSONNumber(n json.Number) any {
	if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
		return i
	}
	f, _ := strconv.ParseFloat(string(n), 64)
	return f
}
