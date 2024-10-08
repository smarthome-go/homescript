package parser

import (
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Import item
//

func (self *Parser) importIdent() (ast.SpannedIdent, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start
	segments := make([]string, 0)

	if self.CurrentToken.Kind == lexer.AtSymbol {
		segments = append(segments, lexer.AtSymbol.String())
		self.next()
	}

	if self.CurrentToken.Kind != lexer.Identifier {
		return ast.SpannedIdent{}, self.expectedOneOfErr([]lexer.TokenKind{lexer.AtSymbol, lexer.Identifier})
	}

	self.next()

	segments = append(segments, self.PreviousToken.Value)

loop:
	for {
		switch self.CurrentToken.Kind {
		case lexer.Colon:
			segments = append(segments, self.CurrentToken.Kind.String())
			self.next()
			fallthrough
		case lexer.Identifier:
			self.expect(lexer.Identifier)
			segments = append(segments, self.PreviousToken.Value)
		default:
			break loop
		}
	}

	return ast.NewSpannedIdent(
		strings.Join(segments, ""),
		startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	), nil
}

func (self *Parser) importItem() (ast.ImportStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `import`
	if err := self.next(); err != nil {
		return ast.ImportStatement{}, err
	}

	toImport := make([]ast.ImportStatementCandidate, 0)

	switch self.CurrentToken.Kind {
	case lexer.Type, lexer.Templ, lexer.Trigger, lexer.Identifier, lexer.Underscore:
		startLoc := self.CurrentToken.Span.Start
		importKind := ast.IMPORT_KIND_NORMAL

		switch self.CurrentToken.Kind {
		case lexer.Type:
			importKind = ast.IMPORT_KIND_TYPE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		case lexer.Templ:
			importKind = ast.IMPORT_KIND_TEMPLATE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		case lexer.Trigger:
			importKind = ast.IMPORT_KIND_TRIGGER
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		default:
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
	case lexer.LCurly:
		// skip the `{`
		if err := self.next(); err != nil {
			return ast.ImportStatement{}, err
		}

		// make initial import
		importKind := ast.IMPORT_KIND_NORMAL
		startLoc := self.CurrentToken.Span.Start

		if self.CurrentToken.Kind == lexer.Type {
			importKind = ast.IMPORT_KIND_TYPE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		if self.CurrentToken.Kind == lexer.Templ {
			importKind = ast.IMPORT_KIND_TEMPLATE
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		if self.CurrentToken.Kind == lexer.Trigger {
			importKind = ast.IMPORT_KIND_TRIGGER
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}
		}

		if err := self.expectMultiple(lexer.Type, lexer.Templ, lexer.Identifier, lexer.Underscore); err != nil {
			return ast.ImportStatement{}, err
		}

		toImport = append(toImport, ast.ImportStatementCandidate{
			Ident: self.PreviousToken.Value,
			Kind:  importKind,
			Span:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		})

		// make remaining imports
		for self.CurrentToken.Kind == lexer.Comma {
			if err := self.next(); err != nil {
				return ast.ImportStatement{}, err
			}

			startLoc := self.CurrentToken.Span.Start

			if self.CurrentToken.Kind == lexer.RCurly || self.CurrentToken.Kind == lexer.EOF {
				break
			}

			importKind = ast.IMPORT_KIND_NORMAL
			if self.CurrentToken.Kind == lexer.Type {
				importKind = ast.IMPORT_KIND_TYPE
				if err := self.next(); err != nil {
					return ast.ImportStatement{}, err
				}
			}

			if self.CurrentToken.Kind == lexer.Templ {
				importKind = ast.IMPORT_KIND_TEMPLATE
				if err := self.next(); err != nil {
					return ast.ImportStatement{}, err
				}
			}

			if err := self.expectMultiple(lexer.Identifier, lexer.Underscore); err != nil {
				return ast.ImportStatement{}, err
			}

			toImport = append(toImport, ast.ImportStatementCandidate{
				Ident: self.PreviousToken.Value,
				Kind:  importKind,
				Span:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
			})
		}

		if err := self.expectRecoverable(lexer.RCurly); err != nil {
			return ast.ImportStatement{}, err
		}
	default:
		return ast.ImportStatement{}, self.expectedOneOfErr([]lexer.TokenKind{
			lexer.Type,
			lexer.Templ,
			lexer.Trigger,
			lexer.Identifier,
			lexer.LCurly,
		})
	}

	if err := self.expect(lexer.From); err != nil {
		return ast.ImportStatement{}, err
	}

	fromModule, err := self.importIdent()
	if err != nil {
		return ast.ImportStatement{}, err
	}

	if err := self.expectRecoverable(lexer.Semicolon); err != nil {
		return ast.ImportStatement{}, err
	}

	return ast.ImportStatement{
		ToImport:   toImport,
		FromModule: fromModule,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}
