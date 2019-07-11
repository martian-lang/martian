package syntax

import (
	"bytes"
	"sort"
	"strconv"
)

// JsonWriter describes types that can marshal json directly to a write buffer.
// In many cases when building up compound objects like maps or arrays, this is
// much more efficient, both in terms of performance and memory consumption,
// than just implementing MarshalJSON(), since it doesn't make unnessessary
// copies.
type JsonWriter interface {
	// EncodeJSON writes a json representation of the object to a buffer.
	EncodeJSON(buf *bytes.Buffer) error
}

type jsonSizeEstimator interface {
	// Gets an estimate of the number of bytes needed to encode
	// this object as json.
	jsonSizeEstimate() int
}

func jsonSizeEstimate(v jsonSizeEstimator) int {
	if v == nil {
		return 4
	}
	return v.jsonSizeEstimate()
}

// MarshalJSON encodes a resolved binding as json, though dropping the resolved
// type information and keeping only type ID strings.
func (binding *ResolvedBinding) MarshalJSON() ([]byte, error) {
	if binding == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.Grow(jsonSizeEstimate(binding.Exp) +
		binding.Type.GetId().jsonSizeEstimate() +
		len(`{"expression":,"type":}`))
	err := binding.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (binding *ResolvedBinding) jsonSizeEstimate() int {
	if binding == nil {
		return 4
	}
	return jsonSizeEstimate(binding.Exp) +
		len(`{"expression":,"type":}`) +
		binding.Type.GetId().jsonSizeEstimate()
}

// EncodeJSON writes a json representation of the object to a buffer.
func (binding *ResolvedBinding) EncodeJSON(buf *bytes.Buffer) error {
	if binding == nil {
		_, err := buf.WriteString("null")
		return err
	}
	return binding.encodeJSON(buf)
}

func (binding *ResolvedBinding) encodeJSON(buf *bytes.Buffer) error {
	if _, err := buf.WriteString(`{"expression":`); err != nil {
		return err
	}
	if err := binding.Exp.EncodeJSON(buf); err != nil {
		return err
	}
	if _, err := buf.WriteString(`,"type":`); err != nil {
		return err
	}
	if err := binding.Type.GetId().EncodeJSON(buf); err != nil {
		return err
	}
	_, err := buf.WriteRune('}')
	return err
}

// MarshalJSON encodes the resolved reference path and type ID as json.
func (binding *BoundReference) MarshalJSON() ([]byte, error) {
	if binding == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	bidLen := len(binding.Exp.Id) +
		len(binding.Exp.OutputId) +
		binding.Type.GetId().jsonSizeEstimate()
	buf.Grow(bidLen + len(`{"ref":".","type":}`))
	err := binding.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (binding *BoundReference) jsonSizeEstimate() int {
	if binding == nil {
		return 4
	}
	return jsonSizeEstimate(binding.Exp) +
		binding.Type.GetId().jsonSizeEstimate() + len(`{"ref":".","type":}`)
}

// EncodeJSON writes a json representation of the object to a buffer.
func (binding *BoundReference) EncodeJSON(buf *bytes.Buffer) error {
	if binding == nil {
		_, err := buf.WriteString("null")
		return err
	}
	return binding.encodeJSON(buf)
}

func (binding *BoundReference) encodeJSON(buf *bytes.Buffer) error {
	if _, err := buf.WriteString(`{"ref":"`); err != nil {
		return err
	}
	// The grammar guarantees that IDs do not to need encoding.
	if _, err := buf.WriteString(binding.Exp.Id); err != nil {
		return err
	}
	if len(binding.Exp.OutputId) > 0 {
		if _, err := buf.WriteRune('.'); err != nil {
			return err
		}
		if _, err := buf.WriteString(binding.Exp.OutputId); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(`","type":"`); err != nil {
		return err
	}
	if err := binding.Type.GetId().EncodeJSON(buf); err != nil {
		return err
	}
	_, err := buf.WriteString(`"}`)
	return err
}

func (e *ArrayExp) MarshalJSON() ([]byte, error) {
	if e == nil || e.Value == nil {
		return []byte("null"), nil
	}
	if len(e.Value) == 0 {
		return []byte("[]"), nil
	}
	var buf bytes.Buffer
	buf.Grow(e.jsonSizeEstimate())
	err := e.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (e *ArrayExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil || e.Value == nil {
		_, err := buf.WriteString("null")
		return err
	}
	if len(e.Value) == 0 {
		_, err := buf.WriteString("[]")
		return err
	}
	return e.encodeJSON(buf)
}

func (e *ArrayExp) encodeJSON(buf *bytes.Buffer) error {
	buf.WriteRune('[')
	for i, v := range e.Value {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		if err := v.EncodeJSON(buf); err != nil {
			return err
		}
	}
	_, err := buf.WriteRune(']')
	return err
}

func (e *ArrayExp) jsonSizeEstimate() int {
	if e == nil {
		return 4
	} else if len(e.Value) == 0 {
		return 2
	}
	s := 1
	for _, v := range e.Value {
		s += jsonSizeEstimate(v) + 1
	}
	return s
}

func (e *SweepExp) MarshalJSON() ([]byte, error) {
	if e == nil || e.Value == nil {
		return []byte("null"), nil
	}
	if len(e.Value) == 0 {
		return []byte(`{"sweep":[]}`), nil
	}
	var buf bytes.Buffer
	buf.Grow(e.jsonSizeEstimate())
	err := e.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (e *SweepExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil || e.Value == nil {
		_, err := buf.WriteString("null")
		return err
	}
	return e.encodeJSON(buf)
}

func (e *SweepExp) encodeJSON(buf *bytes.Buffer) error {
	buf.WriteString(`{"sweep":[`)
	for i, v := range e.Value {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		if err := v.EncodeJSON(buf); err != nil {
			return err
		}
	}
	_, err := buf.WriteString("]}")
	return err
}

func (e *SweepExp) jsonSizeEstimate() int {
	if e == nil {
		return 4
	}
	s := len(`{"sweep":[`) + 1
	for _, v := range e.Value {
		s += jsonSizeEstimate(v) + 1
	}
	return s
}

func (e *MapExp) MarshalJSON() ([]byte, error) {
	if e == nil || e.Value == nil {
		return []byte("null"), nil
	}
	if len(e.Value) == 0 {
		return []byte("{}"), nil
	}
	keys := make([]string, 0, len(e.Value))
	kt := 1 + 4*len(e.Value)
	for key, v := range e.Value {
		kt += len(key) + jsonSizeEstimate(v)
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.Grow(kt)
	err := e.encodeJSON(&buf, keys)
	return buf.Bytes(), err
}

func (e *MapExp) jsonSizeEstimate() int {
	if e == nil || e.Value == nil {
		return 4
	}
	if len(e.Value) == 0 {
		return 2
	}
	s := 1 + 4*len(e.Value)
	for k, v := range e.Value {
		s += len(k) + jsonSizeEstimate(v)
	}
	return s
}

func (e *MapExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil || e.Value == nil {
		_, err := buf.WriteString("null")
		return err
	}
	if len(e.Value) == 0 {
		_, err := buf.WriteString("{}")
		return err
	}
	keys := make([]string, 0, len(e.Value))
	for key := range e.Value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return e.encodeJSON(buf, keys)
}

func (e *MapExp) encodeJSON(buf *bytes.Buffer, keys []string) error {
	buf.WriteRune('{')
	for i, key := range keys {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		quoteString(buf, key)
		if _, err := buf.WriteRune(':'); err != nil {
			return err
		}
		if err := e.Value[key].EncodeJSON(buf); err != nil {
			return err
		}
	}
	_, err := buf.WriteRune('}')
	return err
}

func (e *IntExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(e.Value, 10)), nil
}

func (e *IntExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil {
		_, err := buf.WriteString("null")
		return err
	}
	if e.Value >= 0 && e.Value < 10 {
		_, err := buf.WriteRune('0' + rune(e.Value))
		return err
	}
	_, err := buf.WriteString(strconv.FormatInt(e.Value, 10))
	return err
}

func (e *IntExp) jsonSizeEstimate() int {
	return 19
}

func (e *FloatExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatFloat(e.Value, 'g', -1, 64)), nil
}

func (e *FloatExp) jsonSizeEstimate() int {
	return 24
}

func (e *FloatExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil {
		_, err := buf.WriteString("null")
		return err
	}
	_, err := buf.WriteString(strconv.FormatFloat(e.Value, 'g', -1, 64))
	return err
}

func (e *StringExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	} else if len(e.Value) == 0 {
		return []byte(`""`), nil
	}
	var buf bytes.Buffer
	buf.Grow(2 + len(e.Value))
	quoteString(&buf, e.Value)
	return buf.Bytes(), nil
}

func (e *StringExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil {
		_, err := buf.WriteString("null")
		return err
	}
	quoteString(buf, e.Value)
	return nil
}

func (e *StringExp) jsonSizeEstimate() int {
	if e == nil {
		return 4
	}
	return 2 + len(e.Value)
}

func (e *BoolExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatBool(e.Value)), nil
}

func (e *BoolExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil {
		_, err := buf.WriteString("null")
		return err
	}
	_, err := buf.WriteString(strconv.FormatBool(e.Value))
	return err
}

func (e *BoolExp) jsonSizeEstimate() int {
	if e == nil || e.Value {
		return 4
	}
	return 5
}

func (e *NullExp) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

func (e *NullExp) EncodeJSON(buf *bytes.Buffer) error {
	_, err := buf.WriteString("null")
	return err
}

func (e *NullExp) jsonSizeEstimate() int {
	return 4
}

func (self *RefExp) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(self.jsonSizeEstimate())
	err := self.EncodeJSON(&buf)
	return buf.Bytes(), err
}

func (self *RefExp) EncodeJSON(buf *bytes.Buffer) error {
	if _, err := buf.WriteString(`{"__reference__":"`); err != nil {
		return err
	}
	if self.Kind == KindSelf {
		if _, err := buf.WriteString("self."); err != nil {
			return err
		}
		if _, err := buf.WriteString(self.Id); err != nil {
			return err
		}
	} else if _, err := buf.WriteString(self.Id); err != nil {
		return err
	}
	if self.OutputId != "" {
		if _, err := buf.WriteRune('.'); err != nil {
			return err
		}
		if _, err := buf.WriteString(self.OutputId); err != nil {
			return err
		}
	}
	_, err := buf.WriteString(`"}`)
	return err
}

func (ref *RefExp) jsonSizeEstimate() int {
	bufLen := len(`{"__reference__":""}`) + len(ref.Id) + len(ref.OutputId)
	if ref.Kind == KindSelf {
		bufLen += len("self.")
	}
	if ref.OutputId != "" {
		bufLen++
	}
	return bufLen
}
