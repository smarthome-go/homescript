package value

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

// TODO: set maximum recursion here

type fieldURI struct {
	Parts []fieldURIComponent
}

func NewFieldURI(initial ...fieldURIComponent) fieldURI {
	return fieldURI{
		Parts: initial,
	}
}

func (f fieldURI) String() string {
	components := make([]string, 0)
	for _, c := range f.Parts {
		components = append(components, c.String())
	}
	return strings.Join(components, "")
}

func (f fieldURI) isEmpty() bool {
	return len(f.Parts) == 0
}

func (f *fieldURI) push(kind fieldURIComponentKind, field string, index uint64) {
	f.Parts = append(f.Parts, fieldURIComponent{
		kind:      0,
		fieldName: field,
		index:     index,
	})
}

func (f fieldURI) clone() fieldURI {
	cloned := make([]fieldURIComponent, len(f.Parts))
	copy(cloned, f.Parts)
	return fieldURI{
		Parts: cloned,
	}
}

type fieldURIComponentKind uint8

const (
	componentKindField fieldURIComponentKind = iota
	componentKindIndex
	componentKindOptionInner
)

type fieldURIComponent struct {
	kind      fieldURIComponentKind
	fieldName string
	index     uint64
}

func (f fieldURIComponent) String() string {
	switch f.kind {
	case componentKindField:
		return fmt.Sprintf(".%s", f.fieldName)
	case componentKindIndex:
		return fmt.Sprintf("[%d]", f.index)
	case componentKindOptionInner:
		return "<option-inner>"
	default:
		panic("A new fieldURIComponent was added without updating this code")
	}
}

type CastError struct {
	typeErr   string
	Span      errors.Span
	FieldPath fieldURI
}

func (e CastError) Message() string {
	uriLocation := ""
	if !e.FieldPath.isEmpty() {
		uriLocation = fmt.Sprintf(" at `%s`", e.FieldPath.String())
	}
	return fmt.Sprintf("Cast error%s: %s", uriLocation, e.typeErr)
}

func DeepCast(val Value, typ ast.Type, span errors.Span, allowCasts bool) (*Value, *CastError) {
	return deepCastRecursive(val, typ, span, allowCasts, NewFieldURI())
}

// `fieldURI` is used to describe values in nested structures so that the error message will be clearer.
func deepCastRecursive(val Value, typ ast.Type, span errors.Span, allowCasts bool, fieldURI fieldURI) (*Value, *CastError) {
	// This does nothing as casting to an `any` does not validate anything.
	if typ.Kind() == ast.AnyTypeKind {
		return &val, nil
	}

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

			newUri := fieldURI.clone()
			newUri.push(componentKindOptionInner, "", 0)
			innerCast, i := deepCastRecursive(valInner, typInner, span, allowCasts, fieldURI)
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
			// TODO: should cast be a normal exception? is this sane?
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
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
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
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
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
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
		if !allowCasts && (typ.Kind() != ast.ObjectTypeKind && typ.Kind() != ast.AnyObjectTypeKind) {
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
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
						newUri := fieldURI.clone()
						newUri.push(componentKindField, key, 0)

						newField, i := deepCastRecursive(*field, otherField.Type, span, allowCasts, newUri)
						if i != nil {
							return nil, i
						}
						outputFields[key] = newField
						found = true
						break
					}
				}
				if !found {
					return nil, &CastError{
						typeErr:   fmt.Sprintf("Incompatible values: found unexpected field '%s'", key),
						Span:      span,
						FieldPath: fieldURI,
					}
				}
			}

			for _, field := range objType.ObjFields {
				_, found := objVal.FieldsInternal[field.FieldName.Ident()]
				if !found {
					return nil, &CastError{
						typeErr:   fmt.Sprintf("Incompatible values: field '%s' was expected but not found", field.FieldName.Ident()),
						Span:      span,
						FieldPath: fieldURI,
					}
				}
			}

			return NewValueObject(outputFields), nil
		default:
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
		}
	case ListValueKind:
		listVal := val.(ValueList)
		switch typ.Kind() {
		case ast.ListTypeKind:
			asType := typ.(ast.ListType)

			outputList := make([]*Value, 0)
			for index, item := range *listVal.Values {
				newUri := fieldURI.clone()
				newUri.push(componentKindIndex, "", uint64(index))

				newVal, i := deepCastRecursive(*item, asType.Inner, span, allowCasts, newUri)
				if i != nil {
					return nil, i
				}
				outputList = append(outputList, newVal)
			}

			return NewValueList(outputList), nil
		}
	case AnyObjectValueKind:
		if typ.Kind() != ast.AnyObjectTypeKind {
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
		}
	case OptionValueKind:
		if typ.Kind() != ast.OptionTypeKind {
			return nil, &CastError{
				typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				Span:      span,
				FieldPath: fieldURI,
			}
		}

		opt := val.(ValueOption)
		optType := typ.(ast.OptionType)

		// allow conversions like `none as ?str`
		if !opt.IsSome() {
			return &val, nil
		}

		// otherwise, the inner type must also match
		newUri := fieldURI.clone()
		newUri.Parts = append(fieldURI.Parts, fieldURIComponent{
			kind:      componentKindOptionInner,
			fieldName: "",
			index:     0,
		})
		return deepCastRecursive(*opt.Inner, optType, span, allowCasts, newUri)
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
	return nil, &CastError{
		typeErr:   fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
		Span:      span,
		FieldPath: fieldURI,
	}
}
