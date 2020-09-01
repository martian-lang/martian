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
		binding.Type.TypeId().jsonSizeEstimate() +
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
		binding.Type.TypeId().jsonSizeEstimate()
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
	if err := binding.Type.TypeId().EncodeJSON(buf); err != nil {
		return err
	}
	_, err := buf.WriteRune('}')
	return err
}

// MarshalJSON encodes the map as json with sorted keys.
func (m ResolvedBindingMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	keys := make([]string, 0, len(m))
	kt := 1 + 4*len(m)
	for key, v := range m {
		kt += len(key) + jsonSizeEstimate(v)
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.Grow(kt)
	err := m.encodeJSON(&buf, keys)
	return buf.Bytes(), err
}

func (m ResolvedBindingMap) jsonSizeEstimate() int {
	if m == nil {
		return 4
	}
	if len(m) == 0 {
		return 2
	}
	s := 1 + 4*len(m)
	for k, v := range m {
		s += len(k) + jsonSizeEstimate(v)
	}
	return s
}

func (m ResolvedBindingMap) EncodeJSON(buf *bytes.Buffer) error {
	if m == nil {
		_, err := buf.WriteString("null")
		return err
	}
	if len(m) == 0 {
		_, err := buf.WriteString("{}")
		return err
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return m.encodeJSON(buf, keys)
}

func (m ResolvedBindingMap) encodeJSON(buf *bytes.Buffer, keys []string) error {
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
		if err := m[key].EncodeJSON(buf); err != nil {
			return err
		}
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
		binding.Type.TypeId().jsonSizeEstimate()
	buf.Grow(bidLen + len(`{"ref":".","type":}`))
	err := binding.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (binding *BoundReference) jsonSizeEstimate() int {
	if binding == nil {
		return 4
	}
	return jsonSizeEstimate(binding.Exp) +
		binding.Type.TypeId().jsonSizeEstimate() + len(`{"ref":".","type":}`)
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
	if _, err := buf.WriteString(`","type":`); err != nil {
		return err
	}
	if err := binding.Type.TypeId().EncodeJSON(buf); err != nil {
		return err
	}
	_, err := buf.WriteString(`}`)
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

func (e *SplitExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	if e.Value == nil {
		return []byte(`{"split":null}`), nil
	}
	var buf bytes.Buffer
	buf.Grow(e.jsonSizeEstimate())
	err := e.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (e *SplitExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil || e.Value == nil {
		_, err := buf.WriteString("null")
		return err
	}
	return e.encodeJSON(buf)
}

func (e *SplitExp) encodeJSON(buf *bytes.Buffer) error {
	if e.Call != nil {
		if _, err := buf.WriteString(`{"call":`); err != nil {
			return err
		}
		quoteString(buf, e.Call.Id)
		if e.Source.CallMode() != ModeUnknownMapCall {
			if _, err := buf.WriteString(`,"mode":"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(e.Source.CallMode().String()); err != nil {
				return err
			}
			if err := buf.WriteByte('"'); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`,"split":`); err != nil {
			return err
		}
	} else if _, err := buf.WriteString(`{"split":`); err != nil {
		return err
	}
	if err := e.Value.EncodeJSON(buf); err != nil {
		return err
	}
	if e.Source != nil && !e.IsEmpty() {
		if v, ok := e.Value.(MapCallSource); !ok || v != e.Source {
			if s, ok := e.Source.(JsonWriter); ok {
				if _, err := buf.WriteString(`,"source":`); err != nil {
					return err
				}
				if err := s.EncodeJSON(buf); err != nil {
					return err
				}
			}
		}
	}
	_, err := buf.WriteRune('}')
	return err
}

func (e *SplitExp) jsonSizeEstimate() int {
	if e == nil {
		return 4
	}
	s := len(`{"split":}`) +
		len(`,"mode":"`) + 1 + len(KindArray) +
		jsonSizeEstimate(e.Value)
	if e.Source != nil {
		s += len(`,"source":`) + estimateMapSourceJsonSize(e.Source)
	}
	if e.Call != nil {
		s += 10 + len(e.Call.Id)
	}
	return s
}

func (e *MergeExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	if e.Value == nil {
		return []byte(`{"merge":null}`), nil
	}
	var buf bytes.Buffer
	buf.Grow(e.jsonSizeEstimate())
	err := e.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (e *MergeExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil || e.Value == nil {
		_, err := buf.WriteString("null")
		return err
	}
	return e.encodeJSON(buf)
}

func (e *MergeExp) encodeJSON(buf *bytes.Buffer) error {
	if e.Call != nil {
		if _, err := buf.WriteString(`{"call":`); err != nil {
			return err
		}
		quoteString(buf, e.Call.GetFqid())
		if e.CallMode() != ModeUnknownMapCall {
			if _, err := buf.WriteString(`,"mode":"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(e.CallMode().String()); err != nil {
				return err
			}
			if err := buf.WriteByte('"'); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`,"merge_value":`); err != nil {
			return err
		}
	} else if _, err := buf.WriteString(`{"merge_value":`); err != nil {
		return err
	}
	if err := e.Value.EncodeJSON(buf); err != nil {
		return err
	}
	if e.MergeOver != nil {
		if v, ok := e.Value.(MapCallSource); !ok || v != e.MergeOver {
			if _, err := buf.WriteString(`,"merge_over":`); err != nil {
				return err
			}
			if err := encodeMapSourceJson(buf, e.MergeOver); err != nil {
				return err
			}
		}
	}
	if e.ForkNode != nil {
		if _, err := buf.WriteString(`,"fork_node":`); err != nil {
			return err
		}
		quoteString(buf, e.ForkNode.Id)
	}
	_, err := buf.WriteRune('}')
	return err
}

func (e *MergeExp) jsonSizeEstimate() int {
	if e == nil {
		return 4
	}
	s := len(`{"call":,"mode":,"merge_value":,}`) +
		len(KindArray) +
		jsonSizeEstimate(e.Value)
	if e.MergeOver != nil {
		s += len(`,"merge_over":`) + estimateMapSourceJsonSize(e.MergeOver)
	}
	if e.Call != nil {
		s += 15 + len(e.Call.GetFqid())
	}
	if e.ForkNode != nil {
		s += len(`,"fork_node":`) + 2 + len(e.ForkNode.Id)
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
	return encodeInt(e.Value, buf)
}

func encodeInt(v int64, buf *bytes.Buffer) error {
	if v >= 0 && v < 10 {
		_, err := buf.WriteRune('0' + rune(v))
		return err
	}
	_, err := buf.WriteString(strconv.FormatInt(v, 10))
	return err
}

func (e *IntExp) jsonSizeEstimate() int {
	return 19
}

func (e *FloatExp) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	var buf [68]byte
	return strconv.AppendFloat(buf[:0], e.Value, 'g', -1, 64), nil
}

func (e *FloatExp) jsonSizeEstimate() int {
	return 24
}

func (e *FloatExp) EncodeJSON(buf *bytes.Buffer) error {
	if e == nil {
		_, err := buf.WriteString("null")
		return err
	}
	var b [68]byte
	_, err := buf.Write(strconv.AppendFloat(b[:0], e.Value, 'g', -1, 64))
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
	if err := buf.WriteByte('"'); err != nil {
		return err
	}
	if len(self.Forks) > 0 {
		if _, err := buf.WriteString(`,"fork":{`); err != nil {
			return err
		}
		first := true
		dims := make([]*CallStm, 0, len(self.Forks))
		for k := range self.Forks {
			dims = append(dims, k)
		}
		sort.Slice(dims, func(i, j int) bool {
			return dims[i].Id < dims[j].Id
		})
		for _, s := range dims {
			i := self.Forks[s]
			if first {
				first = false
			} else {
				if err := buf.WriteByte(','); err != nil {
					return err
				}
			}
			quoteString(buf, s.Id)
			if err := buf.WriteByte(':'); err != nil {
				return err
			}
			if err := encodeIndex(i, buf); err != nil {
				return err
			}
		}
		if err := buf.WriteByte('}'); err != nil {
			return err
		}
	}
	return buf.WriteByte('}')
}

func (ref *RefExp) jsonSizeEstimate() int {
	bufLen := len(`{"__reference__":""}`) + len(ref.Id) + len(ref.OutputId)
	if ref.Kind == KindSelf {
		bufLen += len("self.")
	}
	if ref.OutputId != "" {
		bufLen++
	}
	if len(ref.Forks) > 0 {
		bufLen += len(`,"dim":[]`)
		for k, v := range ref.Forks {
			bufLen += len(`,{"dim":"","index":}`) + len(k.Id) + estimateIndexJsonSize(v)
		}
	}
	return bufLen
}

func estimateIndexJsonSize(i CollectionIndex) int {
	if i == nil {
		return 4
	}
	if i.IndexSource() != nil {
		return len("unknown")
	}
	switch i.Mode() {
	case ModeArrayCall:
		w := 1
		for a := i.ArrayIndex(); a > 0; a /= 10 {
			w++
		}
		return w
	case ModeMapCall:
		return 2 + len(i.MapKey())
	}
	panic("invalid index mode " + i.Mode().String())
}

func encodeIndex(i CollectionIndex, buf *bytes.Buffer) error {
	if i == nil {
		_, err := buf.WriteString(KindNull)
		return err
	}
	if i.IndexSource() != nil {
		_, err := buf.WriteString(`"unknown"`)
		return err
	}
	switch i.Mode() {
	case ModeArrayCall:
		return encodeInt(int64(i.ArrayIndex()), buf)
	case ModeMapCall:
		quoteString(buf, i.MapKey())
		return nil
	}
	panic("invalid index mode " + i.Mode().String())
}

func (s ForkRootList) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	if len(s) == 0 {
		return []byte("[]"), nil
	}
	var buf bytes.Buffer
	buf.Grow(s.jsonSizeEstimate())
	err := s.encodeJSON(&buf)
	return buf.Bytes(), err
}

func (s ForkRootList) EncodeJSON(buf *bytes.Buffer) error {
	if s == nil {
		_, err := buf.WriteString("null")
		return err
	}
	if len(s) == 0 {
		_, err := buf.WriteString("[]")
		return err
	}
	return s.encodeJSON(buf)
}

func encodeMapSourceJson(buf *bytes.Buffer, src MapCallSource) error {
	switch v := src.(type) {
	case CallGraphNode:
		quoteString(buf, v.GetFqid())
	case *CallStm:
		quoteString(buf, v.Id)
	case *MapCallSet:
		return encodeMapSourceJson(buf, v.Master)
	case *ArrayExp:
		if _, err := buf.WriteString(`{"type":"array","len":`); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.Itoa(len(v.Value))); err != nil {
			return err
		}
		return buf.WriteByte('}')
	case *MapExp:
		if _, err := buf.WriteString(`{"type":"map","keys":[`); err != nil {
			return err
		}
		if len(v.Value) > 0 {
			keys := make([]string, 0, len(v.Value))
			for k := range v.Value {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for i, k := range keys {
				if i != 0 {
					if err := buf.WriteByte(','); err != nil {
						return err
					}
				}
				quoteString(buf, k)
			}
		}
		if _, err := buf.WriteString(`]}`); err != nil {
			return err
		}
	case *MergeExp:
		return encodeMapSourceJson(buf, v.MergeOver)
	case *placeholderArrayMapSource:
		_, err := buf.WriteString(`"placeholder array"`)
		return err
	case *placeholderMapMapSource:
		_, err := buf.WriteString(`"placeholder map"`)
		return err
	case *placeholderMapSource:
		_, err := buf.WriteString(`"placeholder"`)
		return err
	case JsonWriter:
		return v.EncodeJSON(buf)
	default:
		quoteString(buf, src.CallMode().String())
	}
	return nil
}

func estimateMapSourceJsonSize(src MapCallSource) int {
	switch v := src.(type) {
	case CallGraphNode:
		return len(v.GetFqid()) + 2
	case *CallStm:
		return len(v.Id) + 2
	case *MapCallSet:
		return estimateMapSourceJsonSize(v.Master)
	case Exp:
		return v.jsonSizeEstimate()
	}
	return 19
}

func (s ForkRootList) encodeJSON(buf *bytes.Buffer) error {
	buf.WriteRune('[')
	for i, v := range s {
		if i != 0 {
			if _, err := buf.WriteRune(','); err != nil {
				return err
			}
		}
		quoteString(buf, v.Fqid)
	}
	_, err := buf.WriteRune(']')
	return err
}

func (s ForkRootList) jsonSizeEstimate() int {
	if s == nil {
		return 4
	} else if len(s) == 0 {
		return 2
	}
	t := 1 + 3*len(s)
	for _, v := range s {
		t += len(v.GetFqid())
	}
	return t
}
