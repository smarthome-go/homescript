package value

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type ValueKind uint8

const (
	NullValueKind ValueKind = iota
	IntValueKind
	FloatValueKind
	BoolValueKind
	StringValueKind
	AnyObjectValueKind
	ObjectValueKind
	OptionValueKind
	ListValueKind
	RangeValueKind
	FunctionValueKind
	ClosureValueKind
	BuiltinFunctionValueKind
	PointerValueKind
	IteratorValueKind
)

func (self ValueKind) String() string {
	switch self {
	case NullValueKind:
		return "null"
	case IntValueKind:
		return "int"
	case FloatValueKind:
		return "float"
	case BoolValueKind:
		return "bool"
	case StringValueKind:
		return "string"
	case AnyObjectValueKind:
		return "any-object"
	case ObjectValueKind:
		return "object"
	case OptionValueKind:
		return "option"
	case ListValueKind:
		return "list"
	case RangeValueKind:
		return "range"
	case FunctionValueKind:
		return "function"
	case ClosureValueKind:
		return "closure"
	case BuiltinFunctionValueKind:
		return "builtin-function"
	case PointerValueKind:
		return "pointer"
	case IteratorValueKind:
		return "iterator"
	default:
		panic("A new ValueKind was introduced without updating this code")
	}
}

type Value interface {
	Kind() ValueKind
	Display() (string, *Interrupt)
	IsEqual(other Value) (bool, *Interrupt)
	Fields() (map[string]*Value, *Interrupt)
	IntoIter() func() (Value, bool)
}

func ZeroValue(typ ast.Type) *Value {
	switch typ.Kind() {
	case ast.NullTypeKind:
		return NewValueNull()
	case ast.IntTypeKind:
		return NewValueInt(0)
	case ast.FloatTypeKind:
		return NewValueFloat(0.0)
	case ast.BoolTypeKind:
		return NewValueBool(false)
	case ast.StringTypeKind:
		return NewValueString("")
	case ast.RangeTypeKind:
		return NewValueRange(*NewValueInt(0), *NewValueInt(0))
	case ast.ListTypeKind:
		return NewValueList(make([]*Value, 0))
	case ast.AnyObjectTypeKind:
		return NewValueAnyObject(make(map[string]*Value))
	case ast.ObjectTypeKind:
		return NewValueObject(make(map[string]*Value))
	case ast.OptionTypeKind:
		return NewNoneOption()
	case ast.FnTypeKind:
		return NewValueFunction("__init__", ast.AnalyzedBlock{})
	case ast.UnknownTypeKind:
		fallthrough
	case ast.NeverTypeKind:
		fallthrough
	case ast.IdentTypeKind:
		fallthrough
	case ast.AnyTypeKind:
		fallthrough
	default:
	}
	panic(fmt.Sprintf("Invalid type: %s", typ))
}
