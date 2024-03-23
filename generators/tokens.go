package generators

import (
	l "bcomp/lexer"
	"fmt"
	"log"
	"os"
)

// PrintTokens prints the tokens as human readable instructions
func PrintTokens(f *GeneratorOutput, tokens []ParseToken, includeComments bool) {
	indentLevel := 1
	for _, t := range tokens {
		if includeComments {
			f.Printf("%s# Line %d, Pos %d: %v\n", indent(indentLevel), t.Pos.Line, t.Pos.Column, t.Tok)
		}

		switch t.Tok.Tok {
		case l.ADD:
			f.Printf("%sADD %d\n", indent(indentLevel), t.Extra)
		case l.SUB:
			f.Printf("%sSUB %d\n", indent(indentLevel), t.Extra)
		case l.INCP:
			f.Printf("%sINCP %d\n", indent(indentLevel), t.Extra)
		case l.DECP:
			f.Printf("%sDECP %d\n", indent(indentLevel), t.Extra)
		case l.OUT:
			f.Printf("%sOUT %d\n", indent(indentLevel), t.Extra)
		case l.IN:
			f.Printf("%sIN %d\n", indent(indentLevel), t.Extra)
		case l.JMPF:
			f.Printf("%sJMPF @%d\n", indent(indentLevel), t.Extra)
			indentLevel++
		case l.JMPB:
			indentLevel--
			f.Printf("%sJMPB @%d\n", indent(indentLevel), t.Extra)
		case l.MUL:
			f.Printf("%sMUL %d, %d\n", indent(indentLevel), t.Extra, t.Extra2)
		case l.DIV:
			f.Printf("%sDIV %d, %d\n", indent(indentLevel), t.Extra, t.Extra2)
		case l.BZ:
			f.Printf("%sBZ @%d\n", indent(indentLevel), t.Extra)
			indentLevel++
		case l.LBL:
			indentLevel--
			f.Printf("%sLBL @%d\n", indent(indentLevel), t.Extra)
		case l.MOV:
			f.Printf("%sMOV %d, %d\n", indent(indentLevel), t.Extra, t.Extra2)
		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}
	if indentLevel > 1 {
		if PrintWarnings {
			fmt.Fprintf(os.Stderr, "Warning: Unbalanced brackets in code\n")
		}
		for indentLevel > 1 {
			indentLevel--
			f.Printf("%sJMPB ??\n", indent(indentLevel))
		}
	}
}
