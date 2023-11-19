package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type Value interface {
	TypeKind() ast.TypeKind
	String() string
}

type IntValue struct {
	Value int64
}

func (self IntValue) TypeKind() ast.TypeKind {
	return ast.IntTypeKind
}

func (self IntValue) String() string {
	return fmt.Sprint(self.Value)
}

type FloatValue struct {
	Value float64
}

func (self FloatValue) TypeKind() ast.TypeKind {
	return ast.FloatTypeKind
}

func (self FloatValue) String() string {
	return fmt.Sprint(self.Value)
}

type BoolValue struct {
	Value bool
}

func (self BoolValue) TypeKind() ast.TypeKind {
	return ast.BoolTypeKind
}

func (self BoolValue) String() string {
	return fmt.Sprint(self.Value)
}

type StringValue struct {
	Value string
}

func (self StringValue) TypeKind() ast.TypeKind {
	return ast.StringTypeKind
}

func (self StringValue) String() string {
	return fmt.Sprint(self.Value)
}

type NullValue struct {
}

func (self NullValue) TypeKind() ast.TypeKind {
	return ast.NullTypeKind
}

func (self NullValue) String() string {
	return "null"
}

type OptionValue struct {
	inner *Value
}

func (self OptionValue) TypeKind() ast.TypeKind {
	return ast.OptionTypeKind
}

func (self OptionValue) String() string {
	if self.inner == nil {
		return "None"
	} else {
		return (*self.inner).String()
	}
}
