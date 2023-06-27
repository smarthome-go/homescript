package value

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
