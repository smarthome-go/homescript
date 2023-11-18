package compiler

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type Value interface {
	TypeKind() ast.TypeKind
}

type IntValue struct {
	Value int64
}

func (self IntValue) TypeKind() ast.TypeKind {
	return ast.IntTypeKind
}

type FloatValue struct {
	Value float64
}

func (self FloatValue) TypeKind() ast.TypeKind {
	return ast.FloatTypeKind
}

type BoolValue struct {
	Value bool
}

func (self BoolValue) TypeKind() ast.TypeKind {
	return ast.BoolTypeKind
}

type StringValue struct {
	Value string
}

func (self StringValue) TypeKind() ast.TypeKind {
	return ast.StringTypeKind
}

type NullValue struct {
}

func (self NullValue) TypeKind() ast.TypeKind {
	return ast.NullTypeKind
}

type OptionValue struct {
	inner *Value
}

func (self OptionValue) TypeKind() ast.TypeKind {
	return ast.OptionTypeKind
}
