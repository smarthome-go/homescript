package parser

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

func (self *Parser) nonCriticalErr(span errors.Span, message string) {
	self.Errors = append(self.Errors, *errors.NewError(
		span,
		message,
		errors.SyntaxError,
	))
}

func (self *Parser) expect(expected TokenKind) *errors.Error {
	if self.CurrentToken.Kind != expected {
		return errors.NewSyntaxError(
			self.CurrentToken.Span,
			fmt.Sprintf("Expected '%s', found '%s'", expected, self.CurrentToken.Kind),
		)
	}

	if err := self.next(); err != nil {
		return err
	}

	return nil
}

func (self *Parser) expectRecoverable(expected TokenKind) *errors.Error {
	if self.CurrentToken.Kind != expected {
		self.nonCriticalErr(
			self.CurrentToken.Span,
			fmt.Sprintf("Expected '%s', found '%s'", expected, self.CurrentToken.Kind),
		)
		return nil
	}

	if err := self.next(); err != nil {
		return err
	}

	return nil
}

func (self *Parser) expectMultiple(expected ...TokenKind) *errors.Error {
	for _, test := range expected {
		if self.CurrentToken.Kind == test {
			if err := self.next(); err != nil {
				return err
			}
			return nil
		}
	}

	return self.expectedOneOfErr(expected)
}

func (self Parser) expectedOneOfErr(expected []TokenKind) *errors.Error {
	message := ""

	if len(expected) == 2 {
		message = fmt.Sprintf("either '%s' or '%s'", expected[0], expected[1])
	} else {
		for idx, expectedItem := range expected {
			if idx == len(expected)-1 {
				message += ", or "
			} else if message != "" {
				message += ", "
			}
			message += fmt.Sprintf("'%s'", expectedItem)
		}
	}

	return errors.NewSyntaxError(
		self.CurrentToken.Span,
		fmt.Sprintf("Expected %s, found '%s'", message, self.CurrentToken.Kind),
	)
}
