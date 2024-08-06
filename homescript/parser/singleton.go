package parser

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Singleton (type definition)
//

func (self *Parser) singleton() (ast.SingletonTypeDefinition, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	ident, err := self.singletonIdent()
	if err != nil {
		return ast.SingletonTypeDefinition{}, err
	}

	if err := self.expect(lexer.Assign); err != nil {
		return ast.SingletonTypeDefinition{}, err
	}

	// For singleton types, additional information in the object fields
	// will be useful for some host applications
	rhsType, err := self.hmsType(true)
	if err != nil {
		return ast.SingletonTypeDefinition{}, err
	}

	if err := self.expectRecoverable(lexer.Semicolon); err != nil {
		return ast.SingletonTypeDefinition{}, err
	}

	return ast.SingletonTypeDefinition{
		Ident: ident,
		Type:  rhsType,
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
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
	// if self.CurrentToken.Kind == AtSymbol {
	// 	singleton, err := self.singletonIdent()
	// 	if err != nil {
	// 		return ast.ImplBlock{}, err
	// 	}
	//
	// 	// Handle impl block body
	// 	methods, err := self.implBlockBody()
	// 	if err != nil {
	// 		return ast.ImplBlock{}, err
	// 	}
	//
	// 	return ast.ImplBlock{
	// 		SingletonIdent: singleton,
	// 		UsingTemplate:  nil,
	// 		Methods:        methods,
	// 		Span:           startLoc.Until(self.CurrentToken.Span.End, self.Filename),
	// 	}, nil
	// }
	// NOTE: this is deprecated as an impl without a template does not make a lot of sense
	// considering that you can just extract values directly in `normal` functions.

	// In this case, we except a template
	templateIdent := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.expect(lexer.Identifier); err != nil {
		return ast.ImplBlock{}, err
	}

	usingTemplate := ast.ImplBlockTemplate{
		Template: templateIdent,
		UserDefinedCapabilities: ast.ImplBlockCapabilities{
			Defined: false,
			List:    make([]ast.SpannedIdent, 0),
			Span:    errors.Span{},
		},
	}

	capabilitiesStartLoc := self.CurrentToken.Span.Start

	// If there is the `with` token, there are optional capabilities for this implementation
	if self.CurrentToken.Kind == lexer.With {
		if err := self.next(); err != nil {
			return ast.ImplBlock{}, err
		}

		// Save start location
		capabilitiesStartLoc = self.CurrentToken.Span.Start
		usingTemplate.UserDefinedCapabilities.Defined = true

		if err := self.expect(lexer.LCurly); err != nil {
			return ast.ImplBlock{}, err
		}

		// Make initial capability
		usingTemplate.UserDefinedCapabilities.List = append(usingTemplate.UserDefinedCapabilities.List, ast.NewSpannedIdent(
			self.CurrentToken.Value,
			self.CurrentToken.Span,
		))

		if err := self.expect(lexer.Identifier); err != nil {
			return ast.ImplBlock{}, err
		}

		// As long as there is an `,` make additional capabilities
		for self.CurrentToken.Kind == lexer.Comma {
			// Skip the `,`
			if err := self.next(); err != nil {
				return ast.ImplBlock{}, err
			}

			// If there is a `}`, this was a trailing comma
			if self.CurrentToken.Kind == lexer.RCurly {
				if err := self.next(); err != nil {
					return ast.ImplBlock{}, err
				}

				break
			}

			// Make current capability
			usingTemplate.UserDefinedCapabilities.List = append(usingTemplate.UserDefinedCapabilities.List, ast.NewSpannedIdent(
				self.CurrentToken.Value,
				self.CurrentToken.Span,
			))

			if err := self.expect(lexer.Identifier); err != nil {
				return ast.ImplBlock{}, err
			}
		}

		// Expect a closing `}`
		if err := self.expectRecoverable(lexer.RCurly); err != nil {
			return ast.ImplBlock{}, err
		}

		usingTemplate.UserDefinedCapabilities.Span = capabilitiesStartLoc.Until(self.PreviousToken.Span.End, self.Filename)
	}

	// Expect a `for`
	if err := self.expect(lexer.For); err != nil {
		if self.CurrentToken.Kind == lexer.For {
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
		UsingTemplate:  usingTemplate,
		Methods:        methods,
		Span:           startLoc.Until(self.CurrentToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) implBlockBody() ([]ast.FunctionDefinition, *errors.Error) {
	// Expect `{`
	if err := self.expect(lexer.LCurly); err != nil {
		return nil, err
	}

	// Loop over function definitions until the end (`}`) is reached
	methods := make([]ast.FunctionDefinition, 0)
	for self.CurrentToken.Kind != lexer.EOF && self.CurrentToken.Kind != lexer.RCurly {
		modifier := ast.FN_MODIFIER_NONE

		if self.CurrentToken.Kind == lexer.Pub {
			if err := self.next(); err != nil {
				return nil, err
			}
			modifier = ast.FN_MODIFIER_PUB
		} else if self.CurrentToken.Kind == lexer.Event {
			if err := self.next(); err != nil {
				return nil, err
			}
			modifier = ast.FN_MODIFIER_EVENT
		}

		fn, err := self.functionDefinition(modifier)
		if err != nil {
			return nil, err
		}
		methods = append(methods, fn)
	}

	// Expect closing `}`
	if err := self.expectRecoverable(lexer.RCurly); err != nil {
		return nil, err
	}

	return methods, nil
}
