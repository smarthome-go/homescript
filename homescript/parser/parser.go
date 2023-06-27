package parser

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type Parser struct {
	Lexer         Lexer
	Errors        []errors.Error
	PreviousToken Token
	CurrentToken  Token
	Filename      string
}

func NewParser(lexer Lexer, filename string) Parser {
	return Parser{
		Lexer:         lexer,
		PreviousToken: unknownToken(errors.Location{}),
		CurrentToken:  unknownToken(errors.Location{}),
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

	for self.CurrentToken.Kind != EOF {
		switch self.CurrentToken.Kind {
		case Import:
			importStmt, err := self.importItem()
			if err != nil {
				return ast.Program{}, err
			}
			tree.Imports = append(tree.Imports, importStmt)
		case Pub, Type, Let, Fn:
			isPub := self.CurrentToken.Kind == Pub
			if isPub {
				if err := self.next(); err != nil {
					return ast.Program{}, err
				}
			}

			switch self.CurrentToken.Kind {
			case Type:
				typeDefinition, err := self.typeDefinition(isPub)
				if err != nil {
					return ast.Program{}, err
				}
				tree.Types = append(tree.Types, typeDefinition)
			case Let:
				letStmt, err := self.letStatement(isPub)
				if err != nil {
					return ast.Program{}, err
				}
				tree.Globals = append(tree.Globals, letStmt)
			case Fn:
				fnDefinition, err := self.functionDefinition(isPub)
				if err != nil {
					return ast.Program{}, err
				}
				tree.Functions = append(tree.Functions, fnDefinition)
			default:
				return ast.Program{}, self.expectedOneOfErr([]TokenKind{Let, Fn})
			}
		default:
			return ast.Program{}, self.expectedOneOfErr([]TokenKind{Import, Type, Pub, Let, Fn})
		}
	}

	return tree, nil
}
