package syntax

import (
	"bytes"
)

// DisabledExp is an expression which evaluates to either the Inner
// expression or null, depending on whether it is disabled.
type DisabledExp struct {
	Value    Exp
	Disabled *RefExp
}

func (s *DisabledExp) getNode() *AstNode {
	return s.Value.getNode()
}
func (s *DisabledExp) File() *SourceFile {
	return s.Value.File()
}
func (s *DisabledExp) Line() int {
	return s.Value.Line()
}
func (s *DisabledExp) inheritComments() bool {
	return s.Value.inheritComments()
}
func (s *DisabledExp) getSubnodes() []AstNodable {
	return s.Value.getSubnodes()
}
func (s *DisabledExp) HasRef() bool {
	return true
}
func (s *DisabledExp) HasSplit() bool {
	return s.Value.HasSplit()
}
func (s *DisabledExp) FindRefs() []*RefExp {
	return append(s.Value.FindRefs(), s.Disabled)
}
func (s *DisabledExp) getKind() ExpKind {
	return s.Value.getKind()
}

func (s *DisabledExp) BindingPath(bindPath string,
	fork map[MapCallSource]CollectionIndex,
	index []CollectionIndex) (Exp, error) {
	inner, err := s.Value.BindingPath(bindPath, fork, index)
	if err != nil {
		return s, err
	}
	disable, err := s.Disabled.BindingPath("", fork, index)
	if err != nil {
		return s, err
	}
	return s.makeDisabledExp(disable, inner)
}

func (s *DisabledExp) EncodeJSON(buf *bytes.Buffer) error {
	if _, err := buf.WriteString(`{"__disabled__":`); err != nil {
		return err
	}
	if err := s.Disabled.EncodeJSON(buf); err != nil {
		return err
	}
	if _, err := buf.WriteString(`,"value":`); err != nil {
		return err
	}
	if err := s.Value.EncodeJSON(buf); err != nil {
		return err
	}
	return buf.WriteByte('}')
}

func (s *DisabledExp) jsonSizeEstimate() int {
	return s.Value.jsonSizeEstimate() +
		s.Disabled.jsonSizeEstimate() +
		len(`{"__disabled__":,"value":}`)
}

func (s *DisabledExp) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.Grow(s.jsonSizeEstimate())
	err := s.EncodeJSON(&buf)
	return buf.Bytes(), err
}

func (s *DisabledExp) GoString() string {
	if s == nil {
		return KindNull
	}
	return s.Value.GoString() + " unless " + s.Disabled.GoString()
}

func (s *DisabledExp) String() string {
	return s.GoString()
}

func (s *DisabledExp) format(w stringWriter, prefix string) {
	s.Value.format(w, prefix)
}

func (s *DisabledExp) equal(other Exp) bool {
	o, ok := other.(*DisabledExp)
	return ok && s.Value.equal(o.Value) && s.Disabled.equal(o.Disabled)
}

func (s *DisabledExp) filter(t Type, lookup *TypeLookup) (Exp, error) {
	inner, err := s.Value.filter(t, lookup)
	if err != nil || inner == s.Value {
		return s, err
	}
	return s.makeDisabledExp(s.Disabled, inner)
}

func (s *DisabledExp) makeDisabledExp(disable, inner Exp) (Exp, error) {
	if n, ok := inner.(*NullExp); ok {
		return n, nil
	}
	switch disable := disable.(type) {
	case *BoolExp:
		if disable.Value {
			return &NullExp{
				valExp: valExp{Node: *inner.getNode()},
			}, nil
		} else {
			return inner, nil
		}
	case *RefExp:
		if s != nil && inner == s.Value && disable == s.Disabled {
			return s, nil
		}
		if id, ok := inner.(*DisabledExp); ok && id.Disabled == disable {
			// Already disabled on the same control, no need to nest.
			return id, nil
		}
		return &DisabledExp{
			Value:    inner,
			Disabled: disable,
		}, nil
	case *DisabledExp:
		return s, &IncompatibleTypeError{
			Message: "disabled modifier cannot be bound to a value that may be null",
		}
	case *SplitExp:
		if disable.IsEmpty() {
			return &NullExp{
				valExp: valExp{Node: *s.Value.getNode()},
			}, nil
		}
		switch dv := disable.Value.(type) {
		case *ArrayExp:
			if len(dv.Value) == 0 {
				panic("should not be possible")
			}
			allSame := true
			for i := 1; i < len(dv.Value); i++ {
				if !dv.Value[i-1].equal(dv.Value[i]) {
					allSame = false
					break
				}
			}
			if allSame {
				return s.makeDisabledExp(dv.Value[0], inner)
			}
			arr := make([]Exp, len(dv.Value))
			for i, dvi := range dv.Value {
				var err error
				arr[i], err = s.makeDisabledExp(dvi, inner)
				if err != nil {
					return s, err
				}
			}
			return &SplitExp{
				valExp: disable.valExp,
				Value: &ArrayExp{
					valExp: dv.valExp,
					Value:  arr,
				},
				Call:   disable.Call,
				Source: disable.Source,
			}, nil
		case *MapExp:
			if len(dv.Value) == 0 {
				panic("should not be possible")
			}
			allSame := true
			var val Exp
			for _, mv := range dv.Value {
				if val == nil {
					val = mv
				} else if !dv.equal(mv) {
					allSame = false
					break
				}
			}
			if allSame {
				return s.makeDisabledExp(val, inner)
			}
			m := make(map[string]Exp, len(dv.Value))
			for k, dvi := range dv.Value {
				var err error
				m[k], err = s.makeDisabledExp(dvi, inner)
				if err != nil {
					return s, err
				}
			}
			return &SplitExp{
				valExp: disable.valExp,
				Value: &MapExp{
					valExp: dv.valExp,
					Value:  m,
				},
				Call:   disable.Call,
				Source: disable.Source,
			}, nil
		}
	}
	return nil, &IncompatibleTypeError{
		Message: "disabled modifier cannot be bound to an expression of type " + string(disable.getKind()),
	}
}

func (s *DisabledExp) resolveRefs(self, siblings map[string]*ResolvedBinding,
	lookup *TypeLookup, keepSplit bool) (Exp, error) {
	disable, err := s.Disabled.resolveRefs(self, siblings, lookup, keepSplit)
	if err != nil {
		return s, err
	}
	// Early outs to avoid resolving the inner exp if we don't have to.
	switch disable := disable.(type) {
	case *BoolExp:
		if disable.Value {
			return &NullExp{
				valExp: valExp{Node: *s.Value.getNode()},
			}, nil
		} else {
			return s.Value.resolveRefs(self, siblings, lookup, keepSplit)
		}
	case *SplitExp:
		if disable.IsEmpty() {
			return &NullExp{
				valExp: valExp{Node: *s.Value.getNode()},
			}, nil
		}
	}
	inner, err := s.Value.resolveRefs(self, siblings, lookup, keepSplit)
	if err != nil {
		return s, err
	}
	return s.makeDisabledExp(disable, inner)
}
