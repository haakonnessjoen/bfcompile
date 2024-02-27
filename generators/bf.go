package generators

import (
	l "bcomp/lexer"
	"fmt"
)

// PrintBF prints the tokens as Brainfuck code
func PrintBF(tokens []ParseToken, includeComments bool) {
	for _, t := range tokens {
		if includeComments {
			fmt.Printf("\n%d:%d: %v ", t.Pos.Line, t.Pos.Column, t.Tok.TokenName)
		}

		switch t.Tok.Tok {
		case l.ADD:
			for i := 0; i < t.Extra; i++ {
				fmt.Print("+")
			}
		case l.SUB:
			for i := 0; i < t.Extra; i++ {
				fmt.Print("-")
			}
		case l.INCP:
			for i := 0; i < t.Extra; i++ {
				fmt.Print(">")
			}
		case l.DECP:
			for i := 0; i < t.Extra; i++ {
				fmt.Print("<")
			}
		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				fmt.Print(".")
			}
		case l.IN:
			for i := 0; i < t.Extra; i++ {
				fmt.Print(",")
			}
		case l.JMPF:
			fmt.Print("[")
		case l.JMPB:
			fmt.Print("]")
		}
	}
}
