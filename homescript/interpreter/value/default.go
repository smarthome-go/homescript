package value

import "github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"

func CreateDefault(typ ast.Type) *Value {
	switch typ.Kind() {
	case ast.UnknownTypeKind, ast.NeverTypeKind, ast.AnyTypeKind, ast.IdentTypeKind, ast.FnTypeKind:
		panic("Unsupported type")
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
		return NewValueRange(*NewValueInt(0), *NewValueInt(1))
	case ast.ListTypeKind:
		return NewValueList(make([]*Value, 0))
	case ast.AnyObjectTypeKind:
		return NewValueAnyObject(make(map[string]*Value))
	case ast.ObjectTypeKind:
		return createDefaultObject(typ.(ast.ObjectType))
	case ast.OptionTypeKind:
		return NewNoneOption()
	default:
		panic("A new type kind was introduced without updating this code")
	}
}

func createDefaultObject(typ ast.ObjectType) *Value {
	objFields := make(map[string]*Value, 0)

	for _, field := range typ.ObjFields {
		objFields[field.FieldName.Ident()] = CreateDefault(field.Type)
	}

	return NewValueObject(objFields)
}
