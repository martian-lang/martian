// Copyright (c) 2020 10X Genomics, Inc. All rights reserved.

package syntax

import "fmt"

// TypeLookup is used to cache a type lookup.
type TypeLookup struct {
	baseTypes map[TypeId]Type
	frozen    bool
}

// NewTypeLookup creates a TypeLookup object populated with the builtin types.
func NewTypeLookup() *TypeLookup {
	var lookup TypeLookup
	lookup.init(0)
	return &lookup
}

func (lookup *TypeLookup) init(count int) {
	lookup.baseTypes = make(map[TypeId]Type, len(builtinTypes)+1+count*2)
	for _, t := range builtinTypes {
		lookup.baseTypes[t.TypeId()] = t
	}
	lookup.baseTypes[builtinNull.TypeId()] = builtinNull
}

var (
	duplicateOfStructTypeError = IncompatibleTypeError{
		Message: "type name conflicts with previously declared struct type",
	}
	duplicateOfUserTypeError = IncompatibleTypeError{
		Message: "type name conflicts with previously declared struct type",
	}
	userBaseTypeNameError = IncompatibleTypeError{
		Message: "type name conflicts with a base type name",
	}
)

func (lookup *TypeLookup) AddUserType(t *UserType) error {
	if existing, ok := lookup.baseTypes[t.TypeId()]; !ok {
		lookup.baseTypes[t.TypeId()] = t
		return nil
	} else {
		switch existing := existing.(type) {
		case *UserType:
			return nil
		case *BuiltinType:
			// The parser should prevent this from ever happening
			return &userBaseTypeNameError
		case AstNodable:
			return &wrapError{
				innerError: &duplicateOfStructTypeError,
				loc:        existing.getNode().Loc,
			}
		default:
			panic(fmt.Sprintf("Unexpected type %T", existing))
		}
	}
}

func (lookup *TypeLookup) AddStructType(t *StructType) error {
	if existing, ok := lookup.baseTypes[t.TypeId()]; !ok {
		lookup.baseTypes[t.TypeId()] = t
		return nil
	} else {
		switch existing := existing.(type) {
		case *UserType:
			return &wrapError{
				innerError: &duplicateOfUserTypeError,
				loc:        existing.getNode().Loc,
			}
		case *StructType:
			if err := t.CheckEqual(existing); err != nil {
				return &wrapError{
					innerError: &IncompatibleTypeError{
						Message: "name conflicts with previously declared struct type",
						Reason:  err,
					},
					loc: existing.Node.Loc,
				}
			} else {
				return nil
			}
		case *BuiltinType:
			// The parser should prevent this from ever happening
			return fmt.Errorf("type name conflicts with a base type")
		case AstNodable:
			return &wrapError{
				innerError: &duplicateOfStructTypeError,
				loc:        existing.getNode().Loc,
			}
		default:
			panic(fmt.Sprintf("Unexpected type %T", existing))
		}
	}
}

// Freeze the type lookup so that it will no longer cache constructed types,
// making it safe for concurrent access.
func (lookup *TypeLookup) Freeze() {
	lookup.frozen = true
}

// Gets a type object by id.
func (lookup *TypeLookup) Get(id TypeId) Type {
	elem := lookup.baseTypes[id]
	if elem != nil {
		return elem
	}
	if id.ArrayDim != 0 {
		elem := lookup.Get(TypeId{
			Tname:  id.Tname,
			MapDim: id.MapDim,
		})
		if elem == nil {
			return nil
		}
		elem = &ArrayType{
			Elem: elem,
			Dim:  id.ArrayDim,
		}
		if !lookup.frozen {
			lookup.baseTypes[id] = elem
		}
		return elem
	} else if id.MapDim != 0 {
		elem := lookup.Get(TypeId{
			Tname:    id.Tname,
			ArrayDim: id.MapDim - 1,
		})
		if elem == nil {
			return nil
		}
		elem = &TypedMapType{
			Elem: elem,
		}
		if !lookup.frozen {
			lookup.baseTypes[id] = elem
		}
		return elem
	} else {
		return nil
	}
}

func (lookup *TypeLookup) GetMap(t Type) *TypedMapType {
	id := t.TypeId()
	if id.MapDim != 0 {
		panic("map<map> is not allowed!")
	}
	id.MapDim = id.ArrayDim + 1
	id.ArrayDim = 0
	elem := lookup.baseTypes[id]
	if elem == nil {
		elem = &TypedMapType{
			Elem: t,
		}
		if !lookup.frozen {
			lookup.baseTypes[id] = elem
		}
	}
	return elem.(*TypedMapType)
}

// Gets the array or map form of a type.
func (lookup *TypeLookup) GetArray(t Type, dim int16) Type {
	if dim == 0 {
		return t
	}
	id := t.TypeId()
	id.ArrayDim += dim
	return lookup.Get(id)
}

func (lookup *TypeLookup) AddDim(t Type, mode CallMode) (Type, error) {
	switch mode {
	case ModeArrayCall:
		return lookup.GetArray(t, 1), nil
	case ModeMapCall:
		if t.TypeId().MapDim > 0 {
			return t, &bindingError{
				Msg: "map call generates a nested map of " + t.TypeId().str(),
			}
		}
		return lookup.GetMap(t), nil
	case ModeNullMapCall:
		return nullType{}, nil
	default:
		return t, nil
	}
}
