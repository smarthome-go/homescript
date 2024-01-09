package parser

import (
	"fmt"

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
	case Type, Templ, Identifier, Underscore:
		startLoc := self.CurrentToken.Span.Start
		importKind := ast.IMPORT_KIND_NORMAL

		switch self.CurrentToken.Kind {
		case Type:
			importKind = ast.IMPORT_KIND_TYPE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		case Templ:
			importKind = ast.IMPORT_KIND_TEMPLATE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		// if self.CurrentToken.Kind == Type {
		// 	importKind = ast.IMPORT_KIND_TYPE
		// 	if err := self.next(); err != nil {
		// 		return ast.ImportStatement{}, err
		// 	}
		// }

		// if self.CurrentToken.Kind == Templ {
		//
		// }

		toImport = append(toImport, ast.ImportStatementCandidate{
			Ident: self.CurrentToken.Value,
			Kind:  importKind,
			Span:  startLoc.Until(self.CurrentToken.Span.End, self.Filename),
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
		importKind := ast.IMPORT_KIND_NORMAL
		startLoc := self.CurrentToken.Span.Start

		if self.CurrentToken.Kind == Type {
			importKind = ast.IMPORT_KIND_TYPE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		if self.CurrentToken.Kind == Templ {
			importKind = ast.IMPORT_KIND_TEMPLATE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		if err := self.expectMultiple(Type, Templ, Identifier, Underscore); err != nil {
			return ast.ImportStatement{}, err
		}

		toImport = append(toImport, ast.ImportStatementCandidate{
			Ident: self.PreviousToken.Value,
			Kind:  importKind,
			Span:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
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

			importKind = ast.IMPORT_KIND_NORMAL
			if self.CurrentToken.Kind == Type {
				importKind = ast.IMPORT_KIND_TYPE
				if err := self.next(); err != nil {
					return ast.ImportStatement{}, err
				}
			}

			if self.CurrentToken.Kind == Templ {
				importKind = ast.IMPORT_KIND_TEMPLATE
				if err := self.next(); err != nil {
					return ast.ImportStatement{}, err
				}
			}

			if err := self.expectMultiple(Identifier, Underscore); err != nil {
				return ast.ImportStatement{}, err
			}

			toImport = append(toImport, ast.ImportStatementCandidate{
				Ident: self.PreviousToken.Value,
				Kind:  importKind,
				Span:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
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

	if err := self.expectMultiple(Identifier, Underscore); err != nil {
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

func (self *Parser) functionDefinition(fnModifier ast.FunctionModifier) (ast.FunctionDefinition, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip `fn`
	if err := self.next(); err != nil {
		return ast.FunctionDefinition{}, err
	}

	if err := self.expectMultiple(Identifier, Underscore); err != nil {
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
		Modifier:   fnModifier,
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

//
// Singleton (type definition)
//

func (self *Parser) singleton() (ast.SingletonTypeDefinition, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	ident, err := self.singletonIdent()
	if err != nil {
		return ast.SingletonTypeDefinition{}, err
	}

	typedef, err := self.typeDefinition(false)
	if err != nil {
		return ast.SingletonTypeDefinition{}, err
	}

	return ast.SingletonTypeDefinition{
		Ident:   ident,
		TypeDef: typedef,
		Range:   startLoc.Until(self.CurrentToken.Span.End, self.Filename),
	}, nil
}

//
// Impl blocks
//

func (self *Parser) implBlockHead() (ast.ImplBlock, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start
	// Skip the `impl`
	if err := self.next(); err != nil {
		return ast.ImplBlock{}, err
	}

	// If there is an `@`, there is no template
	if self.CurrentToken.Kind == AtSymbol {
		singleton, err := self.singletonIdent()
		if err != nil {
			return ast.ImplBlock{}, err
		}

		// Handle impl block body
		methods, err := self.implBlockBody()
		if err != nil {
			return ast.ImplBlock{}, err
		}

		return ast.ImplBlock{
			SingletonIdent: singleton,
			UsingTemplate:  nil,
			Methods:        methods,
			Span:           startLoc.Until(self.CurrentToken.Span.End, self.Filename),
		}, nil
	}

	// In this case, we except a template
	templateIdent := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.expect(Identifier); err != nil {
		return ast.ImplBlock{}, err
	}

	usingTemplate := ast.ImplBlockTemplate{
		Template: templateIdent,
		Capabilities: ast.ImplBlockCapabilities{
			Defined: false,
			List:    make([]ast.SpannedIdent, 0),
			Span:    errors.Span{},
		},
	}

	capabilitiesStartLoc := self.CurrentToken.Span.Start

	// If there is the `with` token, there are optional capabilities for this implementation
	if self.CurrentToken.Kind == With {
		if err := self.next(); err != nil {
			return ast.ImplBlock{}, err
		}

		// Save start location
		capabilitiesStartLoc = self.CurrentToken.Span.Start
		usingTemplate.Capabilities.Defined = true

		if err := self.expect(LCurly); err != nil {
			return ast.ImplBlock{}, err
		}

		// Make initial capability
		usingTemplate.Capabilities.List = append(usingTemplate.Capabilities.List, ast.NewSpannedIdent(
			self.CurrentToken.Value,
			self.CurrentToken.Span,
		))

		if err := self.expect(Identifier); err != nil {
			return ast.ImplBlock{}, err
		}

		// As long as there is an `,` make additional capabilities
		for self.CurrentToken.Kind == Comma {
			// Skip the `,`
			if err := self.next(); err != nil {
				return ast.ImplBlock{}, err
			}

			// If there is a `}`, this was a trailing comma
			if self.CurrentToken.Kind == RCurly {
				if err := self.next(); err != nil {
					return ast.ImplBlock{}, err
				}

				break
			}

			// Make current capability
			usingTemplate.Capabilities.List = append(usingTemplate.Capabilities.List, ast.NewSpannedIdent(
				self.CurrentToken.Value,
				self.CurrentToken.Span,
			))

			if err := self.expect(Identifier); err != nil {
				return ast.ImplBlock{}, err
			}
		}

		// Expect a closing `}`
		if err := self.expectRecoverable(RCurly); err != nil {
			return ast.ImplBlock{}, err
		}

		usingTemplate.Capabilities.Span = capabilitiesStartLoc.Until(self.PreviousToken.Span.End, self.Filename)
	}

	// Expect a `for`
	if err := self.expect(For); err != nil {
		if self.CurrentToken.Kind == For {
			self.Errors = append(self.Errors, *err)
		} else {
			return ast.ImplBlock{}, err
		}
	}

	// Expect singleton identifier
	singleton, err := self.singletonIdent()
	if err != nil {
		return ast.ImplBlock{}, err
	}

	// Handle impl block body
	methods, err := self.implBlockBody()
	if err != nil {
		return ast.ImplBlock{}, err
	}

	return ast.ImplBlock{
		SingletonIdent: singleton,
		UsingTemplate:  &usingTemplate,
		Methods:        methods,
		Span:           startLoc.Until(self.CurrentToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) implBlockBody() ([]ast.FunctionDefinition, *errors.Error) {
	// Expect `{`
	if err := self.expect(LCurly); err != nil {
		return nil, err
	}

	// Loop over function definitions until the end (`}`) is reached
	methods := make([]ast.FunctionDefinition, 0)
	for self.CurrentToken.Kind != EOF && self.CurrentToken.Kind != RCurly {
		if self.CurrentToken.Kind == Pub {
			self.Errors = append(self.Errors, *errors.NewSyntaxError(
				self.CurrentToken.Span,
				fmt.Sprintf("Illegal token `%s` in impl block (public functions not allowed)", self.CurrentToken.Value),
			))
			if err := self.next(); err != nil {
				return nil, err
			}
		}

		fn, err := self.functionDefinition(ast.FN_MODIFIER_NONE)
		if err != nil {
			return nil, err
		}
		methods = append(methods, fn)
	}

	// Expect closing `}`
	if err := self.expectRecoverable(RCurly); err != nil {
		return nil, err
	}

	return methods, nil
}
