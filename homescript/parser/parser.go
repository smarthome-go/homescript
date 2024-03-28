package parser

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type Parser struct {
	Lexer         lexer.Lexer
	Errors        []errors.Error
	PreviousToken lexer.Token
	CurrentToken  lexer.Token
	Filename      string
}

func NewParser(lex lexer.Lexer, filename string) Parser {
	return Parser{
		Lexer:         lex,
		PreviousToken: lexer.UnknownToken(errors.Location{}),
		CurrentToken:  lexer.UnknownToken(errors.Location{}),
		Errors:        make([]errors.Error, 0),
		Filename:      filename,
	}
}

func (self *Parser) next() *errors.Error {
	token, err := self.Lexer.NextToken()
	if err != nil {
		return err
	}

	self.PreviousToken = self.CurrentToken
	self.CurrentToken = token
	return nil
}

func (self *Parser) Parse() (program ast.Program, softErrors []errors.Error, hardError *errors.Error) {
	tree, err := self.program()
	if err != nil {
		return ast.Program{}, self.Errors, err
	}
	return tree, self.Errors, nil
}

func (self *Parser) program() (ast.Program, *errors.Error) {
	if err := self.next(); err != nil {
		return ast.Program{}, err
	}

	tree := ast.Program{
		Functions: make([]ast.FunctionDefinition, 0),
		Globals:   make([]ast.LetStatement, 0),
		Types:     make([]ast.TypeDefinition, 0),
		Imports:   make([]ast.ImportStatement, 0),
		Filename:  self.Filename,
	}

	for self.CurrentToken.Kind != lexer.EOF {
		switch self.CurrentToken.Kind {
		case lexer.Import:
			importStmt, err := self.importItem()
			if err != nil {
				return ast.Program{}, err
			}
			tree.Imports = append(tree.Imports, importStmt)
		case lexer.SINGLETON_TOKEN:
			// Handle singleton type definition
			singleton, err := self.singleton()
			if err != nil {
				return ast.Program{}, err
			}

			tree.Singletons = append(tree.Singletons, singleton)
		case lexer.Impl:
			implBlock, err := self.implBlockHead()
			if err != nil {
				return ast.Program{}, err
			}

			tree.ImplBlocks = append(tree.ImplBlocks, implBlock)
		case lexer.Event, lexer.Pub, lexer.Type, lexer.Let, lexer.Fn:
			isPub := self.CurrentToken.Kind == lexer.Pub
			isEvent := self.CurrentToken.Kind == lexer.Event

			if isPub || isEvent {
				if err := self.next(); err != nil {
					return ast.Program{}, err
				}
			}

			switch self.CurrentToken.Kind {
			case lexer.Type:
				typeDefinition, err := self.typeDefinition(isPub)
				if err != nil {
					return ast.Program{}, err
				}
				tree.Types = append(tree.Types, typeDefinition)
			case lexer.Let:
				letStmt, err := self.letStatement(isPub)
				if err != nil {
					return ast.Program{}, err
				}
				tree.Globals = append(tree.Globals, letStmt)
			case lexer.Fn:
				fnModifier := ast.FN_MODIFIER_NONE

				if isPub {
					fnModifier = ast.FN_MODIFIER_PUB
				} else if isEvent {
					fnModifier = ast.FN_MODIFIER_EVENT
				}

				fnDefinition, err := self.functionDefinition(fnModifier)
				if err != nil {
					return ast.Program{}, err
				}
				tree.Functions = append(tree.Functions, fnDefinition)
			default:
				return ast.Program{}, self.expectedOneOfErr([]lexer.TokenKind{lexer.Let, lexer.Fn})
			}
		default:
			return ast.Program{}, self.expectedOneOfErr([]lexer.TokenKind{
				lexer.Import,
				lexer.Type,
				lexer.Pub,
				lexer.Event,
				lexer.Let,
				lexer.Fn,
				lexer.SINGLETON_TOKEN,
			})
		}
	}

	return tree, nil
}
