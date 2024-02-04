package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/parser/util"
)

type TypeKind uint8

const (
	UnknownTypeKind = iota
	NeverTypeKind
	AnyTypeKind
	NullTypeKind
	// UnitTypeKind
	IntTypeKind
	FloatTypeKind
	BoolTypeKind
	StringTypeKind
	IdentTypeKind
	RangeTypeKind
	ListTypeKind
	AnyObjectTypeKind
	ObjectTypeKind
	OptionTypeKind
	FnTypeKind
)

func (self TypeKind) String() string {
	switch self {
	case UnknownTypeKind:
		return "unknown"
	case NeverTypeKind:
		return "never"
	case AnyTypeKind:
		return "any"
	case NullTypeKind:
		return "null"
	// case UnitTypeKind:
	// 	return "()"
	case IntTypeKind:
		return "int"
	case FloatTypeKind:
		return "float"
	case BoolTypeKind:
		return "bool"
	case StringTypeKind:
		return "str"
	case IdentTypeKind:
		panic("Cannot display ident type")
	case RangeTypeKind:
		return "range"
	case ListTypeKind:
		return "list"
	case AnyObjectTypeKind:
		return "any-object"
	case ObjectTypeKind:
		return "object"
	case OptionTypeKind:
		return "Option"
	case FnTypeKind:
		return "function"
	default:
		panic("A new type kind was introduced without updating this code")
	}
}

type Type interface {
	Kind() TypeKind
	String() string
	Span() errors.Span
	SetSpan(span errors.Span) Type
	Fields(fieldSpan errors.Span) map[string]Type
}

//
// Unknown type: this is only used internally
//

type UnknownType struct{}

func (self UnknownType) Kind() TypeKind                       { return UnknownTypeKind }
func (self UnknownType) String() string                       { return "unknown" }
func (self UnknownType) Span() errors.Span                    { return errors.Span{} }
func (self UnknownType) SetSpan(_ errors.Span) Type           { return NewUnknownType() }
func (self UnknownType) Fields(_ errors.Span) map[string]Type { return make(map[string]Type) }
func NewUnknownType() Type                                    { return Type(UnknownType{}) }

//
// Never type: this is only used internally
//

type NeverType struct{}

func (self NeverType) Kind() TypeKind                       { return NeverTypeKind }
func (self NeverType) String() string                       { return "never" }
func (self NeverType) Span() errors.Span                    { return errors.Span{} }
func (self NeverType) SetSpan(_ errors.Span) Type           { return self }
func (self NeverType) Fields(_ errors.Span) map[string]Type { return make(map[string]Type) }
func NewNeverType() Type                                    { return Type(NeverType{}) }

//
// Any type
//

type AnyType struct {
	Range errors.Span
}

func (self AnyType) Kind() TypeKind                       { return AnyTypeKind }
func (self AnyType) String() string                       { return "any" }
func (self AnyType) Span() errors.Span                    { return self.Range }
func (self AnyType) SetSpan(span errors.Span) Type        { return Type(NewAnyType(span)) }
func (self AnyType) Fields(_ errors.Span) map[string]Type { return make(map[string]Type) }
func NewAnyType(span errors.Span) Type                    { return Type(AnyType{Range: span}) }

//
// Unit type
//

// type UnitType struct {
// 	Range errors.Span
// }
//
// func (self UnitType) Kind() TypeKind                       { return UnitTypeKind }
// func (self UnitType) String() string                       { return "()" }
// func (self UnitType) Span() errors.Span                    { return self.Range }
// func (self UnitType) SetSpan(span errors.Span) Type        { return NewUnitType(span) }
// func (self UnitType) Fields(_ errors.Span) map[string]Type { return make(map[string]Type) }
// func NewUnitType(span errors.Span) Type                    { return Type(UnitType{Range: span}) }

//
// Null type
//

type NullType struct {
	Range errors.Span
}

func (self NullType) Kind() TypeKind                       { return NullTypeKind }
func (self NullType) String() string                       { return "null" }
func (self NullType) Span() errors.Span                    { return self.Range }
func (self NullType) SetSpan(span errors.Span) Type        { return NewNullType(span) }
func (self NullType) Fields(_ errors.Span) map[string]Type { return make(map[string]Type) }
func NewNullType(span errors.Span) Type                    { return Type(NullType{Range: span}) }

//
// Int type
//

type IntType struct {
	Range errors.Span
}

func (self IntType) Kind() TypeKind                { return IntTypeKind }
func (self IntType) String() string                { return "int" }
func (self IntType) Span() errors.Span             { return self.Range }
func (self IntType) SetSpan(span errors.Span) Type { return NewIntType(span) }
func (self IntType) Fields(fieldSpan errors.Span) map[string]Type {
	return map[string]Type{
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"to_range": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewRangeType(fieldSpan),
			fieldSpan,
		),
	}
}
func NewIntType(span errors.Span) Type { return Type(IntType{Range: span}) }

//
// Float type
//

type FloatType struct {
	Range errors.Span
}

func (self FloatType) Kind() TypeKind                { return FloatTypeKind }
func (self FloatType) String() string                { return "float" }
func (self FloatType) Span() errors.Span             { return self.Range }
func (self FloatType) SetSpan(span errors.Span) Type { return NewFloatType(span) }
func (self FloatType) Fields(fieldSpan errors.Span) map[string]Type {
	return map[string]Type{
		"is_int": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewBoolType(fieldSpan),
			fieldSpan,
		),
		"trunc": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewIntType(fieldSpan),
			fieldSpan,
		),
		"round": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewIntType(fieldSpan),
			fieldSpan,
		),
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
	}
}
func NewFloatType(span errors.Span) Type { return Type(FloatType{Range: span}) }

//
// Bool type
//

type BoolType struct {
	Range errors.Span
}

func (self BoolType) Kind() TypeKind                { return BoolTypeKind }
func (self BoolType) String() string                { return "bool" }
func (self BoolType) Span() errors.Span             { return self.Range }
func (self BoolType) SetSpan(span errors.Span) Type { return NewBoolType(span) }
func (self BoolType) Fields(fieldSpan errors.Span) map[string]Type {
	return map[string]Type{
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
	}
}
func NewBoolType(span errors.Span) Type { return Type(BoolType{Range: span}) }

//
// String type
//

type StringType struct {
	Range errors.Span
}

func (self StringType) Kind() TypeKind                { return StringTypeKind }
func (self StringType) String() string                { return "str" }
func (self StringType) Span() errors.Span             { return self.Range }
func (self StringType) SetSpan(span errors.Span) Type { return NewStringType(span) }
func (self StringType) Fields(fieldSpan errors.Span) map[string]Type {
	return map[string]Type{
		"len": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewIntType(fieldSpan),
			fieldSpan,
		),
		"replace": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("old", fieldSpan), NewStringType(fieldSpan), nil),
				NewFunctionTypeParam(ast.NewSpannedIdent("new", fieldSpan), NewStringType(fieldSpan), nil),
			}),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"repeat": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("count", fieldSpan), NewIntType(fieldSpan), nil),
			}),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"contains": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(ast.NewSpannedIdent("substring", fieldSpan), NewStringType(fieldSpan), nil)}),
			fieldSpan,
			NewBoolType(fieldSpan),
			fieldSpan,
		),
		"split": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(ast.NewSpannedIdent("separator", fieldSpan), NewStringType(fieldSpan), nil)}),
			fieldSpan,
			NewListType(
				NewStringType(fieldSpan),
				fieldSpan,
			),
			fieldSpan,
		),
		"parse_int": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewIntType(fieldSpan),
			fieldSpan,
		),
		"parse_float": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewFloatType(fieldSpan),
			fieldSpan,
		),
		"parse_bool": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewBoolType(fieldSpan),
			fieldSpan,
		),
		"to_lower": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"to_upper": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"parse_json": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewAnyType(fieldSpan),
			fieldSpan,
		),
	}
}
func NewStringType(span errors.Span) Type { return Type(StringType{Range: span}) }

//
// Range type
//

type RangeType struct {
	SpanRange errors.Span
}

func (self RangeType) Kind() TypeKind                { return RangeTypeKind }
func (self RangeType) String() string                { return "int..int" }
func (self RangeType) Span() errors.Span             { return self.SpanRange }
func (self RangeType) SetSpan(span errors.Span) Type { return NewRangeType(span) }
func (self RangeType) Fields(fieldSpan errors.Span) map[string]Type {
	return map[string]Type{
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"start": NewIntType(fieldSpan),
		"end":   NewIntType(fieldSpan),
		"rev": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewRangeType(fieldSpan),
			fieldSpan,
		),
		"diff": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewIntType(fieldSpan),
			fieldSpan,
		),
	}
}
func NewRangeType(span errors.Span) Type { return Type(RangeType{SpanRange: span}) }

//
// List type
//

type ListType struct {
	Inner Type
	Range errors.Span
}

func (self ListType) Kind() TypeKind { return ListTypeKind }
func (self ListType) String() string {
	return fmt.Sprintf("[%s]", self.Inner)
}
func (self ListType) Span() errors.Span { return self.Range }
func (self ListType) SetSpan(span errors.Span) Type {
	return NewListType(self.Inner.SetSpan(span), span)
}
func (self ListType) Fields(fieldSpan errors.Span) map[string]Type {
	fields := map[string]Type{
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"len": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewIntType(fieldSpan),
			fieldSpan,
		),
		"contains": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(ast.NewSpannedIdent("element", fieldSpan), self.Inner, nil)}),
			fieldSpan,
			NewBoolType(fieldSpan),
			fieldSpan,
		),
		"concat": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(
				ast.NewSpannedIdent("other", fieldSpan),
				NewListType(self.Inner.SetSpan(fieldSpan), fieldSpan),
				nil,
			)}),
			fieldSpan,
			NewNullType(fieldSpan),
			fieldSpan,
		),
		"join": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(
				ast.NewSpannedIdent("sep", fieldSpan),
				NewStringType(fieldSpan),
				nil,
			)}),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"push": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(ast.NewSpannedIdent("element", fieldSpan), self.Inner, nil)}),
			fieldSpan,
			NewNullType(fieldSpan),
			fieldSpan,
		),
		"pop": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewOptionType(self.Inner.SetSpan(fieldSpan), fieldSpan),
			fieldSpan,
		),
		"push_front": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{NewFunctionTypeParam(ast.NewSpannedIdent("element", fieldSpan), self.Inner, nil)}),
			fieldSpan,
			NewNullType(fieldSpan),
			fieldSpan,
		),
		"pop_front": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewOptionType(self.Inner.SetSpan(fieldSpan), fieldSpan),
			fieldSpan,
		),
		"insert": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("index", fieldSpan), NewIntType(fieldSpan), nil),
				NewFunctionTypeParam(ast.NewSpannedIdent("element", fieldSpan), self.Inner.SetSpan(fieldSpan), nil),
			}),
			fieldSpan,
			NewNullType(fieldSpan),
			fieldSpan,
		),
		"remove": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("index", fieldSpan), NewIntType(fieldSpan), nil),
			}),
			fieldSpan,
			NewNullType(fieldSpan),
			fieldSpan,
		),
		"last": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewOptionType(self.Inner.SetSpan(fieldSpan), fieldSpan),
			fieldSpan,
		),
		"to_json": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"to_json_indent": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
	}

	// only include `sort` if the inner value allows it
	switch self.Inner.Kind() {
	case IntTypeKind, FloatTypeKind, StringTypeKind:
		fields["sort"] = NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewNullType(fieldSpan),
			fieldSpan,
		)
	}

	return fields
}
func NewListType(inner Type, span errors.Span) Type { return Type(ListType{Inner: inner, Range: span}) }

//
// Any object type
//

type AnyObjectType struct {
	Range errors.Span
}

func (self AnyObjectType) Kind() TypeKind                { return AnyObjectTypeKind }
func (self AnyObjectType) String() string                { return "{ ? }" }
func (self AnyObjectType) Span() errors.Span             { return self.Range }
func (self AnyObjectType) SetSpan(span errors.Span) Type { return NewAnyObjectType(span) }
func (self AnyObjectType) Fields(span errors.Span) map[string]Type {
	return map[string]Type{
		"set": NewFunctionType(NewNormalFunctionTypeParamKind(
			[]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("key", span), NewStringType(span), nil),
				NewFunctionTypeParam(ast.NewSpannedIdent("value", span), NewUnknownType(), nil),
			}),
			span,
			NewNullType(span),
			span,
		),
		"get": NewFunctionType(NewNormalFunctionTypeParamKind(
			[]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("key", span), NewStringType(span), nil),
			}),
			span,
			NewOptionType(NewAnyType(span), span),
			span,
		),
		"keys": NewFunctionType(
			NewNormalFunctionTypeParamKind(
				make([]FunctionTypeParam, 0),
			),
			span,
			NewListType(NewStringType(span), span),
			span,
		),
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			NewStringType(span),
			span,
		),
		"to_json": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			NewStringType(span),
			span,
		),
		"to_json_indent": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			NewStringType(span),
			span,
		),
	}
}

func NewAnyObjectType(span errors.Span) Type {
	return Type(AnyObjectType{Range: span})
}

//
// Object type
//

type ObjectType struct {
	ObjFields []ObjectTypeField
	Range     errors.Span
}

func (self ObjectType) Kind() TypeKind { return ObjectTypeKind }
func (self ObjectType) String() string {
	fields := make([]string, 0)
	for _, field := range self.ObjFields {
		fields = append(fields, strings.ReplaceAll(field.String(), "\n", "\n    "))
	}
	return fmt.Sprintf("{\n    %s\n}", strings.Join(fields, ",\n    "))
}
func (self ObjectType) Span() errors.Span { return self.Range }
func (self ObjectType) SetSpan(span errors.Span) Type {
	newFields := make([]ObjectTypeField, 0)
	for _, field := range self.ObjFields {
		newFields = append(newFields, NewObjectTypeFieldWithAnnotation(
			field.Annotation,
			ast.NewSpannedIdent(field.FieldName.Ident(), span),
			field.Type.SetSpan(span),
			span,
		))
	}
	return NewObjectType(newFields, span)
}
func (self ObjectType) Fields(fieldSpan errors.Span) map[string]Type {
	// include all builtin methods
	fields := map[string]Type{
		"keys": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewListType(NewStringType(fieldSpan), fieldSpan),
			fieldSpan,
		),
		"to_json": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
		"to_json_indent": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			fieldSpan,
			NewStringType(fieldSpan),
			fieldSpan,
		),
	}

	// also include fields on this specific object
	for _, internal := range self.ObjFields {
		fields[internal.FieldName.Ident()] = internal.Type
	}

	return fields
}
func NewObjectType(fields []ObjectTypeField, span errors.Span) Type {
	return Type(ObjectType{ObjFields: fields, Range: span})
}

type ObjectTypeField struct {
	Annotation *ast.SpannedIdent
	FieldName  ast.SpannedIdent
	Type       Type
	Span       errors.Span
}

func (self ObjectTypeField) String() string {
	var key string
	if util.IsIdent(self.FieldName.Ident()) {
		key = fmt.Sprintf("\"%s\"", self.FieldName.Ident())
	} else {
		key = self.FieldName.Ident()
	}

	annotationStr := ""
	if self.Annotation != nil {
		annotationStr = self.Annotation.Ident()
	}

	return fmt.Sprintf("%s%s: %s", annotationStr, key, self.Type)
}

// For external use only: this function does not allow the caller to specify an annotation.
// If an annotation is used, the internal function should be used
func NewObjectTypeField(name ast.SpannedIdent, typ Type, span errors.Span) ObjectTypeField {
	return ObjectTypeField{
		Annotation: nil,
		FieldName:  name,
		Type:       typ,
		Span:       span,
	}
}

func NewObjectTypeFieldWithAnnotation(annotation *ast.SpannedIdent, name ast.SpannedIdent, typ Type, span errors.Span) ObjectTypeField {
	return ObjectTypeField{
		Annotation: annotation,
		FieldName:  name,
		Type:       typ,
		Span:       span,
	}
}

//
// Option type
//

type OptionType struct {
	Inner Type
	Range errors.Span
}

func (self OptionType) Kind() TypeKind                { return OptionTypeKind }
func (self OptionType) String() string                { return fmt.Sprintf("?%s", self.Inner) }
func (self OptionType) Span() errors.Span             { return self.Range }
func (self OptionType) SetSpan(span errors.Span) Type { return NewOptionType(self.Inner, span) }
func (self OptionType) Fields(span errors.Span) map[string]Type {
	return map[string]Type{
		"is_some": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			NewBoolType(span),
			span,
		),
		"is_none": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			NewBoolType(span),
			span,
		),
		"unwrap": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			self.Inner.SetSpan(span),
			span,
		),
		"unwrap_or": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("fallback", span), self.Inner, nil),
			}),
			span,
			self.Inner.SetSpan(span),
			span,
		),
		"expect": NewFunctionType(
			NewNormalFunctionTypeParamKind([]FunctionTypeParam{
				NewFunctionTypeParam(ast.NewSpannedIdent("message", span), NewStringType(span), nil),
			}),
			span,
			self.Inner.SetSpan(span),
			span,
		),
		"to_string": NewFunctionType(
			NewNormalFunctionTypeParamKind(make([]FunctionTypeParam, 0)),
			span,
			NewStringType(span),
			span,
		),
	}
}
func NewOptionType(inner Type, span errors.Span) Type {
	return Type(OptionType{
		Inner: inner,
		Range: span,
	})
}

//
// Function type
//

type FunctionType struct {
	Params     FunctionTypeParamKind
	ParamsSpan errors.Span
	ReturnType Type
	Range      errors.Span
}

func (self FunctionType) Kind() TypeKind { return FnTypeKind }
func (self FunctionType) String() string {
	paramStr := self.Params.String()
	return fmt.Sprintf("fn(%s) -> %s", paramStr, self.ReturnType)
}
func (self FunctionType) Span() errors.Span { return self.Range }
func (self FunctionType) SetSpan(span errors.Span) Type {
	return NewFunctionType(self.Params, span, self.ReturnType.SetSpan(span), span)
}
func (self FunctionType) Fields(_ errors.Span) map[string]Type { return make(map[string]Type) }
func NewFunctionType(
	params FunctionTypeParamKind,
	paramsSpan errors.Span,
	returnType Type,
	span errors.Span,
) Type {
	if params == nil {
		panic("Param type cannot be <nil>")
	}
	return Type(FunctionType{
		Params:     params,
		ParamsSpan: paramsSpan,
		ReturnType: returnType,
		Range:      span,
	})
}

type FunctionTypeParamKindIdentifier uint8

const (
	NormalFunctionTypeParamKindIdentifierKind FunctionTypeParamKindIdentifier = iota
	VarArgsFunctionTypeParamKindIdentifierKind
)

func (self FunctionTypeParamKindIdentifier) String() string {
	switch self {
	case NormalFunctionTypeParamKindIdentifierKind:
		return "finite"
	case VarArgsFunctionTypeParamKindIdentifierKind:
		return "variable"
	default:
		panic("A new function parameter type kind was introduced without updating this code")
	}
}

type FunctionTypeParamKind interface {
	Kind() FunctionTypeParamKindIdentifier
	String() string
}

type NormalFunctionTypeParamKindIdentifier struct {
	Params []FunctionTypeParam
}

func (self NormalFunctionTypeParamKindIdentifier) Kind() FunctionTypeParamKindIdentifier {
	return NormalFunctionTypeParamKindIdentifierKind
}
func (self NormalFunctionTypeParamKindIdentifier) String() string {
	paramStr := ""

	params := make([]string, 0)
	for _, param := range self.Params {
		params = append(params, param.String())
	}
	paramStr = strings.Join(params, ", ")
	return paramStr
}
func NewNormalFunctionTypeParamKind(params []FunctionTypeParam) FunctionTypeParamKind {
	return FunctionTypeParamKind(NormalFunctionTypeParamKindIdentifier{Params: params})
}

type VarArgsFunctionTypeParamKindIdentifier struct {
	ParamTypes    []Type
	RemainingType Type
}

func (self VarArgsFunctionTypeParamKindIdentifier) Kind() FunctionTypeParamKindIdentifier {
	return VarArgsFunctionTypeParamKindIdentifierKind
}
func (self VarArgsFunctionTypeParamKindIdentifier) String() string {
	paramStr := ""
	initial := make([]string, 0)
	for _, typ := range self.ParamTypes {
		initial = append(initial, typ.String())
	}
	paramStr = fmt.Sprintf("%s...%s", strings.Join(initial, ", "), self.RemainingType)
	return paramStr
}
func NewVarArgsFunctionTypeParamKind(paramTypes []Type, remainingType Type) FunctionTypeParamKind {
	return FunctionTypeParamKind(VarArgsFunctionTypeParamKindIdentifier{ParamTypes: paramTypes, RemainingType: remainingType})
}

type FunctionTypeParam struct {
	Name                 ast.SpannedIdent
	Type                 Type
	IsSingletonExtractor bool
	SingletonIdent       string
}

func (self FunctionTypeParam) String() string { return fmt.Sprintf("%s: %s", self.Name, self.Type) }
func NewFunctionTypeParam(name ast.SpannedIdent, typ Type, singletonIdent *string) FunctionTypeParam {
	singletonIdentNormal := ""
	hasSingletonExtractor := false
	if singletonIdent != nil {
		hasSingletonExtractor = true
		singletonIdentNormal = *singletonIdent
	}

	return FunctionTypeParam{
		Name:                 name,
		Type:                 typ,
		IsSingletonExtractor: hasSingletonExtractor,
		SingletonIdent:       singletonIdentNormal,
	}
}
