package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/util"
)

var HMS_BUILTIN_TYPES = []string{"null", "int", "float", "range", "bool", "str"}

type ParserTypeKind uint8

const (
	NameReferenceParserTypeKind ParserTypeKind = iota
	SingletonReferenceParserTypeKind
	OptionParserTypeKind
	ObjectFieldsParserTypeKind
	ListTypeKind
	FunctionTypeKind
)

type HmsType interface {
	Kind() ParserTypeKind
	Span() errors.Span
	String() string
}

//
// Singleton reference type
//

type SingletonReferenceType struct {
	Ident SpannedIdent
}

func (self SingletonReferenceType) Span() errors.Span    { return self.Ident.span }
func (self SingletonReferenceType) Kind() ParserTypeKind { return SingletonReferenceParserTypeKind }
func (self SingletonReferenceType) String() string       { return self.Ident.String() }

//
// Name reference type
//

type NameReferenceType struct {
	Ident SpannedIdent
}

func (self NameReferenceType) Span() errors.Span    { return self.Ident.span }
func (self NameReferenceType) Kind() ParserTypeKind { return NameReferenceParserTypeKind }
func (self NameReferenceType) String() string       { return self.Ident.String() }

//
// Object type
//

type ObjectType struct {
	Range  errors.Span
	Fields ObjectTypeFieldType
}

func (self ObjectType) Span() errors.Span    { return self.Range }
func (self ObjectType) Kind() ParserTypeKind { return ObjectFieldsParserTypeKind }
func (self ObjectType) String() string {
	switch self.Fields.Kind() {
	case AnyObjectTypeFieldTypeKind:
		return "{ ? }"
	case NormalObjectTypeFieldTypeKind:
		normalFields := self.Fields.(ObjectTypeFieldTypeFields).Fields
		fields := make([]string, 0)
		for _, field := range normalFields {
			fields = append(fields, strings.ReplaceAll(field.String(), "\n", "\n    "))
		}
		return fmt.Sprintf("{\n    %s\n}", strings.Join(fields, ",\n    "))
	default:
		panic("A new field kind was added without updating this code")
	}
}

type ObjectTypeFieldTypeKind uint8

const (
	NormalObjectTypeFieldTypeKind ObjectTypeFieldTypeKind = iota
	AnyObjectTypeFieldTypeKind
)

type ObjectTypeFieldType interface {
	Kind() ObjectTypeFieldTypeKind
}

type ObjectTypeFieldTypeAny struct {
}

func (self ObjectTypeFieldTypeAny) Kind() ObjectTypeFieldTypeKind { return AnyObjectTypeFieldTypeKind }

type ObjectTypeFieldTypeFields struct {
	Fields []ObjectTypeField
}

func (self ObjectTypeFieldTypeFields) Kind() ObjectTypeFieldTypeKind {
	return NormalObjectTypeFieldTypeKind
}

type ObjectTypeField struct {
	Range     errors.Span
	FieldName SpannedIdent
	Type      HmsType
}

func (self ObjectTypeField) String() string {
	var key string
	if !util.IsIdent(self.FieldName.ident) {
		key = fmt.Sprintf("\"%s\"", self.FieldName.ident)
	} else {
		key = self.FieldName.ident
	}
	return fmt.Sprintf("%s: %s", key, self.Type)
}

//
// Option type
//

type OptionType struct {
	Inner HmsType
	Range errors.Span
}

func (self OptionType) Span() errors.Span    { return self.Range }
func (self OptionType) Kind() ParserTypeKind { return OptionParserTypeKind }
func (self OptionType) String() string {
	return fmt.Sprintf("?%s", self.Inner)
}

//
// List type
//

type ListType struct {
	Inner HmsType
	Range errors.Span
}

func (self ListType) Span() errors.Span    { return self.Range }
func (self ListType) Kind() ParserTypeKind { return ListTypeKind }
func (self ListType) String() string {
	return fmt.Sprintf("[%s]", self.Inner)
}

//
// Range type
//

type RangeType struct {
	Start HmsType
	End   HmsType
	Range errors.Span
}

func (self RangeType) String() string { return fmt.Sprintf("%s:%s", self.Start, self.End) }

//
// Function type
//

type FunctionType struct {
	Params     []FunctionTypeParam
	ParamsSpan errors.Span
	ReturnType HmsType
	Range      errors.Span
}

func (self FunctionType) Span() errors.Span    { return self.Range }
func (self FunctionType) Kind() ParserTypeKind { return FunctionTypeKind }
func (self FunctionType) String() string {
	params := make([]string, 0)
	for _, param := range self.Params {
		params = append(params, param.String())
	}
	return fmt.Sprintf("fn(%s) -> %s", strings.Join(params, ", "), self.ReturnType)
}

type FunctionTypeParam struct {
	Name SpannedIdent
	Type HmsType
}

func (self FunctionTypeParam) String() string { return fmt.Sprintf("%s:%s", self.Name, self.Type) }
