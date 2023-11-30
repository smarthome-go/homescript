package value

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

// TODO: set maximum recursion here
func DeepCast(val Value, typ ast.Type, span errors.Span, allowCasts bool) (*Value, *Interrupt) {
	// TODO: is this OK?
	if typ.Kind() == ast.OptionTypeKind {
		if val.Kind() == OptionValueKind {
			valOption := val.(ValueOption)
			typOption := typ.(ast.OptionType)
			if !valOption.IsSome() {
				return NewNoneOption(), nil
			}

			valInner := *valOption.Inner
			typInner := typOption.Inner

			innerCast, i := DeepCast(valInner, typInner, span, allowCasts)
			if i != nil {
				return nil, i
			}
			return NewValueOption(innerCast), nil
		}
		return NewValueOption(&val), nil
	}

	switch val.Kind() {
	case BoolValueKind:
		if !allowCasts && typ.Kind() != ast.BoolTypeKind {
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}

		baseBool := val.(ValueBool).Inner
		switch typ.Kind() {
		case ast.IntTypeKind:
			var intRes int64
			if baseBool {
				intRes = 1
			} else {
				intRes = 0
			}
			return NewValueInt(intRes), nil
		case ast.FloatTypeKind:
			var floatRes float64
			if baseBool {
				floatRes = 1.0
			} else {
				floatRes = 0.0
			}
			return NewValueFloat(floatRes), nil
		case ast.BoolTypeKind:
			return &val, nil
		}
	case IntValueKind:
		if !allowCasts && typ.Kind() != ast.IntTypeKind {
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}

		baseInt := val.(ValueInt).Inner
		switch typ.Kind() {
		case ast.BoolTypeKind:
			var boolRes bool
			if baseInt == 0 {
				boolRes = false
			} else {
				boolRes = true
			}
			return NewValueBool(boolRes), nil
		case ast.FloatTypeKind:
			baseInt := val.(ValueInt).Inner
			return NewValueFloat(float64(baseInt)), nil
		case ast.IntTypeKind:
			return &val, nil
		}
	case FloatValueKind:
		if !allowCasts && typ.Kind() != ast.FloatTypeKind {
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}

		baseFloat := val.(ValueFloat).Inner
		switch typ.Kind() {
		case ast.BoolTypeKind:
			var boolRes bool
			if baseFloat == 0 {
				boolRes = false
			} else {
				boolRes = true
			}
			return NewValueBool(boolRes), nil
		case ast.IntTypeKind:
			baseFloat := val.(ValueFloat).Inner
			return NewValueInt(int64(baseFloat)), nil
		case ast.FloatTypeKind:
			return &val, nil
		}
	case ObjectValueKind:
		if !allowCasts && typ.Kind() != ast.ObjectTypeKind {
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}

		objVal := val.(ValueObject)
		switch typ.Kind() {
		case ast.AnyObjectTypeKind:
			return objVal.IntoAnyObject(), nil
		case ast.ObjectTypeKind:
			objVal := val.(ValueObject)
			objType := typ.(ast.ObjectType)

			outputFields := make(map[string]*Value)

			for key, field := range objVal.FieldsInternal {
				found := false
				for _, otherField := range objType.ObjFields {
					if key == otherField.FieldName.Ident() {
						newField, i := DeepCast(*field, otherField.Type, span, allowCasts)
						if i != nil {
							return nil, i
						}
						fmt.Printf("=== DBG: PREV %s (%v) | CURR: %s (%v)\n", (*field).Kind(), field, (*newField).Kind(), newField)
						outputFields[key] = newField
						found = true
						break
					}
				}
				if !found {
					return nil, NewRuntimeErr(
						fmt.Sprintf("Incompatible values: found unexpected field '%s'", key),
						CastErrorKind,
						span,
					)
				}
			}

			for _, field := range objType.ObjFields {
				_, found := objVal.FieldsInternal[field.FieldName.Ident()]
				if !found {
					return nil, NewRuntimeErr(
						fmt.Sprintf("Incompatible values: field '%s' was expected but not found", field.FieldName.Ident()),
						CastErrorKind,
						span,
					)
				}
			}

			return NewValueObject(outputFields), nil
		default:
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}
	case ListValueKind:
		listVal := val.(ValueList)
		switch typ.Kind() {
		case ast.ListTypeKind:
			asType := typ.(ast.ListType)

			outputList := make([]*Value, 0)
			for _, item := range *listVal.Values {
				newVal, i := DeepCast(*item, asType.Inner, span, allowCasts)
				if i != nil {
					return nil, i
				}
				outputList = append(outputList, newVal)
			}

			return NewValueList(outputList), nil
		}
	case AnyObjectValueKind:
		if typ.Kind() != ast.AnyObjectTypeKind {
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}
	case OptionValueKind:
		if typ.Kind() != ast.OptionTypeKind {
			return nil, NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				CastErrorKind,
				span,
			)
		}

		opt := val.(ValueOption)
		optType := typ.(ast.OptionType)

		// allow conversions like `none as ?str`
		if !opt.IsSome() {
			return &val, nil
		}

		// otherwise, the inner type must also match
		return DeepCast(*opt.Inner, optType, span, allowCasts)
	case ClosureValueKind, FunctionValueKind, BuiltinFunctionValueKind:
		panic("Unreachable, the analyzer prevents this")
	case NullValueKind:
		switch typ.Kind() {
		case ast.NullTypeKind:
			return &val, nil
		case ast.OptionTypeKind:
			return NewNoneOption(), nil
		}
	case StringValueKind:
		switch typ.Kind() {
		case ast.StringTypeKind:
			return &val, nil
		}
	case RangeValueKind:
		switch typ.Kind() {
		case ast.RangeTypeKind:
			return &val, nil
		}
	}
	return nil, NewRuntimeErr(
		fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
		CastErrorKind,
		span,
	)
}
