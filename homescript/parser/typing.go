package parser

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Typing
//

func (self *Parser) hmsType(allowAnnotations bool) (ast.HmsType, *errors.Error) {
	switch self.CurrentToken.Kind {
	case lexer.SINGLETON_TOKEN:
		return self.singletonReferenceType()
	case lexer.Null, lexer.Identifier, lexer.Underscore:
		return self.nameReferenceType()
	case lexer.LBracket:
		return self.listType()
	case lexer.LCurly:
		return self.objectType(allowAnnotations)
	case lexer.QuestionMark:
		return self.optionType()
	case lexer.Fn:
		// Here, annotations are always illegal
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

	innerType, err := self.hmsType(false)
	if err != nil {
		return ast.ListType{}, err
	}

	if err = self.expectRecoverable(lexer.RBracket); err != nil {
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

func (self *Parser) objectType(allowAnnotations bool) (ast.ObjectType, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.next(); err != nil {
		return ast.ObjectType{}, err
	}

	fields := make([]ast.ObjectTypeField, 0)

	// if this is an `any-object`, expect `?` and `}`
	if self.CurrentToken.Kind == lexer.QuestionMark {
		if err := self.next(); err != nil {
			return ast.ObjectType{}, err
		}

		if err := self.expectRecoverable(lexer.RCurly); err != nil {
			return ast.ObjectType{}, err
		}

		return ast.ObjectType{
			Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
			Fields: ast.ObjectTypeFieldTypeAny{},
		}, nil
	}

	// if there are no fields, return early
	if self.CurrentToken.Kind == lexer.RCurly {
		if err := self.next(); err != nil {
			return ast.ObjectType{}, err
		}

		return ast.ObjectType{
			Fields: ast.ObjectTypeFieldTypeFields{Fields: fields},
			Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		}, nil
	}

	// Make initial field
	field, err := self.objectTypeFieldComponent(allowAnnotations)
	if err != nil {
		return ast.ObjectType{}, err
	}
	fields = append(fields, field)

	// Make remaining fields
	for self.CurrentToken.Kind == lexer.Comma {
		if err := self.next(); err != nil {
			return ast.ObjectType{}, err
		}

		// Handle optional trailing comma
		if self.CurrentToken.Kind == lexer.RCurly {
			break
		}

		field, err := self.objectTypeFieldComponent(allowAnnotations)
		if err != nil {
			return ast.ObjectType{}, err
		}
		fields = append(fields, field)
	}

	// Expect a `}`
	if self.CurrentToken.Kind != lexer.RCurly {
		return ast.ObjectType{}, self.expectedOneOfErr([]lexer.TokenKind{lexer.Comma, lexer.RCurly})
	}
	if err := self.next(); err != nil {
		return ast.ObjectType{}, err
	}

	return ast.ObjectType{
		Fields: ast.ObjectTypeFieldTypeFields{Fields: fields},
		Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) objectTypeFieldComponent(allowAnnotations bool) (ast.ObjectTypeField, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// If there is a `@` token, add a annotation
	var annotation *ast.SpannedIdent = nil
	if self.CurrentToken.Kind == lexer.TYPE_ANNOTATION_TOKEN {
		// If annotations are not allowed, create an error
		if !allowAnnotations {
			return ast.ObjectTypeField{}, errors.NewSyntaxError(
				self.CurrentToken.Span,
				"Object field annotations are not legal here",
			)
		}

		if err := self.next(); err != nil {
			return ast.ObjectTypeField{}, err
		}

		ident := ast.NewSpannedIdent(fmt.Sprintf("@%s", self.CurrentToken.Value), self.CurrentToken.Span)
		annotation = &ident
		if err := self.expect(lexer.Identifier); err != nil {
			return ast.ObjectTypeField{}, err
		}
	}

	if err := self.expectMultiple(lexer.Identifier, lexer.Underscore, lexer.String); err != nil {
		return ast.ObjectTypeField{}, err
	}

	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expectRecoverable(lexer.Colon); err != nil {
		return ast.ObjectTypeField{}, err
	}

	rhsType, err := self.hmsType(false)
	if err != nil {
		return ast.ObjectTypeField{}, err
	}

	return ast.ObjectTypeField{
		FieldName:  ident,
		Type:       rhsType,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		Annotation: annotation,
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

	inner, err := self.hmsType(false)
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

	if self.CurrentToken.Kind == lexer.Arrow {
		if err := self.next(); err != nil {
			return ast.FunctionType{}, err
		}

		returnTypeTemp, err := self.hmsType(false)
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
	if err := self.expect(lexer.LParen); err != nil {
		return nil, err
	}

	params := make([]ast.FunctionTypeParam, 0)
	if self.CurrentToken.Kind != lexer.RParen && self.CurrentToken.Kind != lexer.EOF {
		// make initial parameter
		param, err := self.functionTypeParameter()
		if err != nil {
			return nil, err
		}
		params = append(params, param)

		// make remaining parameters
		for self.CurrentToken.Kind == lexer.Comma {
			if err := self.next(); err != nil {
				return nil, err
			}

			if self.CurrentToken.Kind == lexer.RParen || self.CurrentToken.Kind == lexer.EOF {
				break
			}

			param, err := self.functionTypeParameter()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
		}
	}

	if err := self.expectRecoverable(lexer.RParen); err != nil {
		return nil, err
	}

	return params, nil
}

func (self *Parser) functionTypeParameter() (ast.FunctionTypeParam, *errors.Error) {
	if err := self.expectMultiple(lexer.Identifier, lexer.Underscore); err != nil {
		return ast.FunctionTypeParam{}, err
	}
	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expect(lexer.Colon); err != nil {
		return ast.FunctionTypeParam{}, err
	}

	paramType, err := self.hmsType(false)
	if err != nil {
		return ast.FunctionTypeParam{}, err
	}

	return ast.FunctionTypeParam{
		Name: ident,
		Type: paramType,
	}, nil
}
