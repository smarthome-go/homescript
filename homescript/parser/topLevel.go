package parser

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Import item
//

func (self *Parser) importItem() (ast.ImportStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `import`
	if err := self.next(); err != nil {
		return ast.ImportStatement{}, err
	}

	toImport := make([]ast.ImportStatementCandidate, 0)

	switch self.CurrentToken.Kind {
	case Type, Identifier:
		startLoc := self.CurrentToken.Span.Start
		isTypeImport := false

		if self.CurrentToken.Kind == Type {
			isTypeImport = true
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		toImport = append(toImport, ast.ImportStatementCandidate{
			Ident:        self.CurrentToken.Value,
			IsTypeImport: isTypeImport,
			Span:         startLoc.Until(self.CurrentToken.Span.End, self.Filename),
		})

		if err := self.next(); err != nil {
			return ast.ImportStatement{}, err
		}
	case LCurly:
		// skip the `{`
		if err := self.next(); err != nil {
			return ast.ImportStatement{}, err
		}

		// make initial import

		isTypeImport := false
		startLoc := self.CurrentToken.Span.Start

		if self.CurrentToken.Kind == Type {
			isTypeImport = true
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		if err := self.expect(Identifier); err != nil {
			return ast.ImportStatement{}, err
		}

		toImport = append(toImport, ast.ImportStatementCandidate{
			Ident:        self.PreviousToken.Value,
			IsTypeImport: isTypeImport,
			Span:         startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		})

		// make remaining imports
		for self.CurrentToken.Kind == Comma {
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}

			startLoc := self.CurrentToken.Span.Start

			if self.CurrentToken.Kind == RCurly || self.CurrentToken.Kind == EOF {
				break
			}

			isTypeImport = false
			if self.CurrentToken.Kind == Type {
				isTypeImport = true
				if err := self.next(); err != nil {
					return ast.ImportStatement{}, err
				}
			}

			if err := self.expect(Identifier); err != nil {
				return ast.ImportStatement{}, err
			}

			toImport = append(toImport, ast.ImportStatementCandidate{
				Ident:        self.PreviousToken.Value,
				IsTypeImport: isTypeImport,
				Span:         startLoc.Until(self.PreviousToken.Span.End, self.Filename),
			})
		}

		if err := self.expectRecoverable(RCurly); err != nil {
			return ast.ImportStatement{}, err
		}
	default:
		return ast.ImportStatement{}, self.expectedOneOfErr([]TokenKind{Identifier, LCurly})
	}

	if err := self.expect(From); err != nil {
		return ast.ImportStatement{}, err
	}

	if err := self.expect(Identifier); err != nil {
		return ast.ImportStatement{}, err
	}
	fromModule := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expectRecoverable(Semicolon); err != nil {
		return ast.ImportStatement{}, err
	}

	return ast.ImportStatement{
		ToImport:   toImport,
		FromModule: fromModule,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

///
/// Function Definition
///

func (self *Parser) functionDefinition(isPub bool) (ast.FunctionDefinition, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip `fn`
	if err := self.next(); err != nil {
		return ast.FunctionDefinition{}, err
	}

	if err := self.expect(Identifier); err != nil {
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

	if self.CurrentToken.Kind == Arrow {
		if err := self.next(); err != nil {
			return ast.FunctionDefinition{}, err
		}

		resType, err := self.hmsType()
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
		IsPub:      isPub,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) parameterList() ([]ast.FnParam, *errors.Error) {
	if err := self.expect(LParen); err != nil {
		return nil, err
	}

	params := make([]ast.FnParam, 0)
	if self.CurrentToken.Kind != RParen && self.CurrentToken.Kind != EOF {
		// make initial parameter
		param, err := self.parameter()
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

			param, err := self.parameter()
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

func (self *Parser) parameter() (ast.FnParam, *errors.Error) {
	ident := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.next(); err != nil {
		return ast.FnParam{}, err
	}

	if err := self.expect(Colon); err != nil {
		return ast.FnParam{}, err
	}

	paramType, err := self.hmsType()
	if err != nil {
		return ast.FnParam{}, err
	}

	return ast.FnParam{
		Ident: ident,
		Type:  paramType,
		Span:  ident.Span().Start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}
