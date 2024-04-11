package parser

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

///
/// Function Definition
///

func (self *Parser) functionDefinition(fnModifier ast.FunctionModifier) (ast.FunctionDefinition, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip `fn`
	if err := self.next(); err != nil {
		return ast.FunctionDefinition{}, err
	}

	if err := self.expectMultiple(lexer.Identifier, lexer.Underscore); err != nil {
		return ast.FunctionDefinition{}, err
	}
	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	paramStartLoc := self.CurrentToken.Span.Start
	params, err := self.parameterList()
	if err != nil {
		return ast.FunctionDefinition{}, err
	}
	paramEndLoc := self.PreviousToken.Span.End

	returnType := ast.HmsType(ast.NameReferenceType{
		Ident: ast.NewSpannedIdent("null", self.PreviousToken.Span.End.Until(self.CurrentToken.Span.End, self.Filename)),
	})

	if self.CurrentToken.Kind == lexer.Arrow {
		if err := self.next(); err != nil {
			return ast.FunctionDefinition{}, err
		}

		resType, err := self.hmsType(false)
		if err != nil {
			return ast.FunctionDefinition{}, err
		}

		returnType = resType
	}

	body, err := self.block()
	if err != nil {
		return ast.FunctionDefinition{}, err
	}

	return ast.FunctionDefinition{
		Ident:      ident,
		Parameters: params,
		ParamSpan:  paramStartLoc.Until(paramEndLoc, self.Filename),
		ReturnType: returnType,
		Body:       body,
		Modifier:   fnModifier,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) parameterList() ([]ast.FnParam, *errors.Error) {
	if err := self.expect(lexer.LParen); err != nil {
		return nil, err
	}

	params := make([]ast.FnParam, 0)
	if self.CurrentToken.Kind != lexer.RParen && self.CurrentToken.Kind != lexer.EOF {
		// make initial parameter
		param, err := self.parameter()
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

			param, err := self.parameter()
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

func (self *Parser) parameter() (ast.FnParam, *errors.Error) {
	// TODO: does this also work with singletons?
	ident := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.next(); err != nil {
		return ast.FnParam{}, err
	}

	if err := self.expect(lexer.Colon); err != nil {
		return ast.FnParam{}, err
	}

	paramType, err := self.hmsType(false)
	if err != nil {
		return ast.FnParam{}, err
	}

	return ast.FnParam{
		Ident: ident,
		Type:  paramType,
		Span:  ident.Span().Start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}
