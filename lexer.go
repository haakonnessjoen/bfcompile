package main

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
}

func (t Token) String() string {
	return fmt.Sprintf("%s (%s)", t.Character, t.TokenName)
}

type Position struct {
	line   int
	column int
}

type Lexer struct {
	pos    Position
	reader *bufio.Reader
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{
		pos:    Position{line: 1, column: 0},
		reader: bufio.NewReader(reader),
	}
}

func (l *Lexer) Lex() (Position, Token) {
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return l.pos, Token{EOF, tokens[EOF], ""}
			}

			panic(err)
		}
		l.pos.column++

		switch r {
		case '\n':
			l.pos.line++
			l.pos.column = 0
		case '+':
			return l.pos, Token{ADD, tokens[ADD], string(r)}
		case '-':
			return l.pos, Token{SUB, tokens[SUB], string(r)}
		case '>':
			return l.pos, Token{INCP, tokens[INCP], string(r)}
		case '<':
			return l.pos, Token{DECP, tokens[DECP], string(r)}
		case '.':
			return l.pos, Token{OUT, tokens[OUT], string(r)}
		case ',':
			return l.pos, Token{IN, tokens[IN], string(r)}
		case '[':
			return l.pos, Token{JMPF, tokens[JMPF], string(r)}
		case ']':
			return l.pos, Token{JMPB, tokens[JMPB], string(r)}
		default:
			continue
		}
	}
}
