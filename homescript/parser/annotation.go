package parser

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Function annotation
//

func (self *Parser) functionAnnotation() (ast.FunctionAnnotation, *errors.Error) {
	start := self.CurrentToken.Span.Start

	// Skip the #.
	if err := self.next(); err != nil {
		return ast.FunctionAnnotation{}, err
	}

	if err := self.expect(lexer.LBracket); err != nil {
		return ast.FunctionAnnotation{}, err
	}

	items := make([]ast.AnnotationItem, 1)

	initial, itemErr := self.annotationItem()
	if itemErr != nil {
		return ast.FunctionAnnotation{}, itemErr
	}
	items[0] = initial

	// Parse remaining items.
	for self.CurrentToken.Kind == lexer.Comma {
		if err := self.next(); err != nil {
			return ast.FunctionAnnotation{}, err
		}

		// Allow trailing `,`.
		if self.CurrentToken.Kind == lexer.RBracket {
			break
		}

		other, err := self.annotationItem()
		if err != nil {
			return ast.FunctionAnnotation{}, err
		}

		items = append(items, other)
	}

	if err := self.expectRecoverable(lexer.RBracket); err != nil {
		return ast.FunctionAnnotation{}, err
	}

	if err := self.expectMultipleInternal(false, lexer.Pub, lexer.Event, lexer.Fn); err != nil {
		return ast.FunctionAnnotation{}, err
	}

	var function ast.FunctionDefinition
	var err *errors.Error

	innerSpan := start.Until(self.PreviousToken.Span.End, self.Filename)

	switch self.CurrentToken.Kind {
	case lexer.Pub:
		if err := self.next(); err != nil {
			return ast.FunctionAnnotation{}, err
		}
		function, err = self.functionDefinition(ast.FN_MODIFIER_PUB)
	case lexer.Event:
		if err := self.next(); err != nil {
			return ast.FunctionAnnotation{}, err
		}
		function, err = self.functionDefinition(ast.FN_MODIFIER_EVENT)
	case lexer.Fn:
		function, err = self.functionDefinition(ast.FN_MODIFIER_NONE)
	default:
		panic(fmt.Sprintf("Unreachable: %s", self.CurrentToken.Kind))
	}

	if err != nil {
		return ast.FunctionAnnotation{}, err
	}

	// Set annotation of the function here.
	function.Annotation = &ast.FunctionAnnotationInner{
		Span:  innerSpan,
		Items: items,
	}

	return ast.FunctionAnnotation{
		Function: function,
		Span:     start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) annotationItem() (ast.AnnotationItem, *errors.Error) {
	switch self.CurrentToken.Kind {
	case lexer.Identifier:
		item := ast.AnnotationItemIdent{
			Ident: ast.NewSpannedIdent(
				self.CurrentToken.Value,
				self.CurrentToken.Span,
			),
		}

		if err := self.next(); err != nil {
			return nil, err
		}

		return item, nil
	case lexer.Trigger:
		return self.annotationItemTrigger()
	default:
		return nil, self.expectedOneOfErr([]lexer.TokenKind{lexer.Identifier, lexer.Trigger})
	}
}

func (self *Parser) annotationItemTrigger() (ast.AnnotationItemTrigger, *errors.Error) {
	start := self.CurrentToken.Span.Start

	// Skip `trigger`.
	if err := self.next(); err != nil {
		return ast.AnnotationItemTrigger{}, err
	}

	dispatchKeyword, err := self.triggerConnective()
	if err != nil {
		return ast.AnnotationItemTrigger{}, err
	}

	// Event identifier.

	eventIdent := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.expect(lexer.Identifier); err != nil {
		return ast.AnnotationItemTrigger{}, err
	}

	// Event args
	args, err := self.callArgs()
	if err != nil {
		return ast.AnnotationItemTrigger{}, err
	}

	return ast.AnnotationItemTrigger{
		TriggerConnective: dispatchKeyword,
		TriggerSource:     eventIdent,
		TriggerArgs:       args,
		Range:             start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}
