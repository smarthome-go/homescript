package parser

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Typing
//

func (self *Parser) hmsType() (ast.HmsType, *errors.Error) {
	switch self.CurrentToken.Kind {
	case AtSymbol:
		return self.singletonReferenceType()
	case Null, Identifier, Underscore:
		return self.nameReferenceType()
	case LBracket:
		return self.listType()
	case LCurly:
		return self.objectType()
	case QuestionMark:
		return self.optionType()
	case Fn:
		return self.functionType()
	default:
		return nil, errors.NewSyntaxError(
			self.CurrentToken.Span,
			fmt.Sprintf("Expected type, found '%s'", self.CurrentToken.Kind),
		)
	}
}

//
// Singleton reference type
//

func (self *Parser) singletonReferenceType() (ast.SingletonReferenceType, *errors.Error) {
	ident, err := self.singletonIdent()
	if err != nil {
		return ast.SingletonReferenceType{}, err
	}

	return ast.SingletonReferenceType{
		Ident: ident,
	}, nil
}

//
// Name reference type
//

func (self *Parser) nameReferenceType() (ast.NameReferenceType, *errors.Error) {
	ident := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)

	if err := self.next(); err != nil {
		return ast.NameReferenceType{}, err
	}

	return ast.NameReferenceType{
		Ident: ident,
	}, nil
}

//
// List type
//

func (self *Parser) listType() (ast.ListType, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.next(); err != nil {
		return ast.ListType{}, err
	}

	innerType, err := self.hmsType()
	if err != nil {
		return ast.ListType{}, err
	}

	if err = self.expectRecoverable(RBracket); err != nil {
		return ast.ListType{}, err
	}

	return ast.ListType{
		Inner: innerType,
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Object type
//

func (self *Parser) objectType() (ast.ObjectType, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.next(); err != nil {
		return ast.ObjectType{}, err
	}

	fields := make([]ast.ObjectTypeField, 0)

	// if this is an `any-object`, expect `?` and `}`
	if self.CurrentToken.Kind == QuestionMark {
		if err := self.next(); err != nil {
			return ast.ObjectType{}, err
		}

		if err := self.expectRecoverable(RCurly); err != nil {
			return ast.ObjectType{}, err
		}

		return ast.ObjectType{
			Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
			Fields: ast.ObjectTypeFieldTypeAny{},
		}, nil
	}

	// if there are no fields, return early
	if self.CurrentToken.Kind == RCurly {
		if err := self.next(); err != nil {
			return ast.ObjectType{}, err
		}

		return ast.ObjectType{
			Fields: ast.ObjectTypeFieldTypeFields{Fields: fields},
			Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		}, nil
	}

	// Make initial field
	field, err := self.objectTypeFieldComponent()
	if err != nil {
		return ast.ObjectType{}, err
	}
	fields = append(fields, field)

	// Make remaining fields
	for self.CurrentToken.Kind == Comma {
		if err := self.next(); err != nil {
			return ast.ObjectType{}, err
		}

		// Handle optional trailing comma
		if self.CurrentToken.Kind == RCurly {
			break
		}

		field, err := self.objectTypeFieldComponent()
		if err != nil {
			return ast.ObjectType{}, err
		}
		fields = append(fields, field)
	}

	// Expect a `}`
	if self.CurrentToken.Kind != RCurly {
		return ast.ObjectType{}, self.expectedOneOfErr([]TokenKind{Comma, RCurly})
	}
	if err := self.next(); err != nil {
		return ast.ObjectType{}, err
	}

	return ast.ObjectType{
		Fields: ast.ObjectTypeFieldTypeFields{Fields: fields},
		Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) objectTypeFieldComponent() (ast.ObjectTypeField, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.expectMultiple(Identifier, Underscore, String); err != nil {
		return ast.ObjectTypeField{}, err
	}

	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expectRecoverable(Colon); err != nil {
		return ast.ObjectTypeField{}, err
	}

	rhsType, err := self.hmsType()
	if err != nil {
		return ast.ObjectTypeField{}, err
	}

	return ast.ObjectTypeField{
		FieldName: ident,
		Type:      rhsType,
		Range:     startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Option type
//

func (self *Parser) optionType() (ast.OptionType, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start
	if err := self.next(); err != nil {
		return ast.OptionType{}, err
	}

	inner, err := self.hmsType()
	if err != nil {
		return ast.OptionType{}, err
	}
	return ast.OptionType{
		Inner: inner,
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Function type
//

func (self *Parser) functionType() (ast.FunctionType, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.next(); err != nil {
		return ast.FunctionType{}, err
	}

	paramStartLoc := self.CurrentToken.Span
	params, err := self.functionTypeParameterList()
	if err != nil {
		return ast.FunctionType{}, err
	}
	paramEndLoc := self.PreviousToken.Span

	// make optional return type
	returnType := ast.HmsType(ast.NameReferenceType{
		Ident: ast.NewSpannedIdent("null", self.PreviousToken.Span.End.Until(self.CurrentToken.Span.End, self.Filename)),
	})

	if self.CurrentToken.Kind == Arrow {
		if err := self.next(); err != nil {
			return ast.FunctionType{}, err
		}

		returnTypeTemp, err := self.hmsType()
		if err != nil {
			return ast.FunctionType{}, err
		}
		returnType = returnTypeTemp
	}

	return ast.FunctionType{
		Params:     params,
		ParamsSpan: paramStartLoc.Start.Until(paramEndLoc.End, self.Filename),
		ReturnType: returnType,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) functionTypeParameterList() ([]ast.FunctionTypeParam, *errors.Error) {
	if err := self.expect(LParen); err != nil {
		return nil, err
	}

	params := make([]ast.FunctionTypeParam, 0)
	if self.CurrentToken.Kind != RParen && self.CurrentToken.Kind != EOF {
		// make initial parameter
		param, err := self.functionTypeParameter()
		if err != nil {
			return nil, err
		}
		params = append(params, param)

		// make remaining parameters
		for self.CurrentToken.Kind == Comma {
			if err := self.next(); err != nil {
				return nil, err
			}

			if self.CurrentToken.Kind == RParen || self.CurrentToken.Kind == EOF {
				break
			}

			param, err := self.functionTypeParameter()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
		}
	}

	if err := self.expectRecoverable(RParen); err != nil {
		return nil, err
	}

	return params, nil
}

func (self *Parser) functionTypeParameter() (ast.FunctionTypeParam, *errors.Error) {
	if err := self.expectMultiple(Identifier, Underscore); err != nil {
		return ast.FunctionTypeParam{}, err
	}
	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expect(Colon); err != nil {
		return ast.FunctionTypeParam{}, err
	}

	paramType, err := self.hmsType()
	if err != nil {
		return ast.FunctionTypeParam{}, err
	}

	return ast.FunctionTypeParam{
		Name: ident,
		Type: paramType,
	}, nil
}
