package main

import (
	"bufio"
	"fmt"
	"io"
)

type Token int

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

var humantokens = []string{
	EOF: "",

	ADD:  "+",
	SUB:  "-",
	INCP: ">",
	DECP: "<",
	OUT:  ".",
	IN:   ",",
	JMPF: "[",
	JMPB: "]",
}

func (t Token) String() string {
	return fmt.Sprintf("%s (%s)", humantokens[t], tokens[t])
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

// Lex scans the input for the next token. It returns the position of the token,
// the token's type, and the literal value.
func (l *Lexer) Lex() (Position, Token, string) {
	// keep looping until we return a token
	for {
		r, _, err := l.reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return l.pos, EOF, ""
			}

			// at this point there isn't much we can do, and the compiler
			// should just return the raw error to the user
			panic(err)
		}
		// update the column to the position of the newly read in rune
		l.pos.column++

		switch r {
		case '\n':
			l.pos.line++
			l.pos.column = 0
		case '+':
			return l.pos, ADD, string(r)
		case '-':
			return l.pos, SUB, string(r)
		case '>':
			return l.pos, INCP, string(r)
		case '<':
			return l.pos, DECP, string(r)
		case '.':
			return l.pos, OUT, string(r)
		case ',':
			return l.pos, IN, string(r)
		case '[':
			return l.pos, JMPF, string(r)
		case ']':
			return l.pos, JMPB, string(r)
		default:
			continue
		}
	}
}
