package generators

import (
	l "bcomp/lexer"
	"log"
)

// PrintBF prints the tokens as Brainfuck code
func PrintBF(outputFile string, tokens []ParseToken, includeComments bool) {
	f := NewGeneratorOutput(outputFile)
	for _, t := range tokens {
		if includeComments {
			f.Printf("\n%d:%d: %v ", t.Pos.Line, t.Pos.Column, t.Tok.TokenName)
		}

		switch t.Tok.Tok {
		case l.ADD:
			for i := 0; i < t.Extra; i++ {
				f.Print("+")
			}
		case l.SUB:
			for i := 0; i < t.Extra; i++ {
				f.Print("-")
			}
		case l.INCP:
			for i := 0; i < t.Extra; i++ {
				f.Print(">")
			}
		case l.DECP:
			for i := 0; i < t.Extra; i++ {
				f.Print("<")
			}
		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				f.Print(".")
			}
		case l.IN:
			for i := 0; i < t.Extra; i++ {
				f.Print(",")
			}
		case l.JMPF:
			f.Print("[")
		case l.JMPB:
			f.Print("]")
		default:
			log.Fatalf("Unknown token at %d:%d\n", t.Pos.Line, t.Pos.Column)
		}
	}
}
