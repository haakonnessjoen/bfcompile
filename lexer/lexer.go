package lexer

import (
	"bufio"
	"fmt"
	"io"
)

type Token struct {
	Tok       TokenId
	TokenName string
	Character string
}

type TokenId int

const (
	EOF = iota

	ADD  // +
	SUB  // -
	INCP // >
	DECP // <
	OUT  // .
	IN   // ,
	JMPF // [
	JMPB // ]

	// Non-BF operations
	MUL
	DIV
	BZ
	LBL
	NOP
	SCANL
	SCANR
)

var tokens = []string{
	EOF: "EOF",

	ADD:  "ADD",
	SUB:  "SUB",
	INCP: "INCP",
	DECP: "DECP",
	OUT:  "OUT",
	IN:   "IN",
	JMPF: "JMPF",
	JMPB: "JMPB",

	MUL:   "MUL",
	DIV:   "DIV",
	BZ:    "BZ",
	LBL:   "LBL",
	NOP:   "NOP",
	SCANL: "SCANL",
	SCANR: "SCANR",
}

func (t Token) String() string {
	return fmt.Sprintf("%s (%s)", t.Character, t.TokenName)
}

type Position struct {
	Line   int
	Column int
}

type Lexer struct {
	Pos    Position
	Reader *bufio.Reader
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{
		Pos:    Position{Line: 1, Column: 0},
		Reader: bufio.NewReader(reader),
	}
}

func (l *Lexer) Lex() (Position, Token) {
	for {
		r, _, err := l.Reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return l.Pos, Token{EOF, tokens[EOF], ""}
			}

			panic(err)
		}
		l.Pos.Column++

		switch r {
		case '\n':
			l.Pos.Line++
			l.Pos.Column = 0
		case '+':
			return l.Pos, Token{ADD, tokens[ADD], string(r)}
		case '-':
			return l.Pos, Token{SUB, tokens[SUB], string(r)}
		case '>':
			return l.Pos, Token{INCP, tokens[INCP], string(r)}
		case '<':
			return l.Pos, Token{DECP, tokens[DECP], string(r)}
		case '.':
			return l.Pos, Token{OUT, tokens[OUT], string(r)}
		case ',':
			return l.Pos, Token{IN, tokens[IN], string(r)}
		case '[':
			return l.Pos, Token{JMPF, tokens[JMPF], string(r)}
		case ']':
			return l.Pos, Token{JMPB, tokens[JMPB], string(r)}
		default:
			continue
		}
	}
}
