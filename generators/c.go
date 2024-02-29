package generators

import (
	l "bcomp/lexer"
	"fmt"
	"strings"
)

// PrintC prints the tokens as C code
func PrintC(outputFile string, tokens []ParseToken, includeComments bool, memorySize int) {
	f := NewGeneratorOutput(outputFile)
	f.Println("#include <stdio.h>")
	f.Println("#include <stdint.h>")

	f.Printf("uint8_t mem[%d];\n", memorySize)
	f.Println("int main() {")
	f.Println("	uint8_t *p = mem;")

	indentLevel := 1
	for _, t := range tokens {
		if includeComments {
			f.Printf("%s// Line %d, Pos %d: %v\n", indent(indentLevel), t.Pos.Line, t.Pos.Column, t.Tok)
		}

		switch t.Tok.Tok {
		case l.ADD:
			if t.Extra == 1 {
				f.Printf("%s(*p)++;\n", indent(indentLevel))
			} else {
				f.Printf("%s*p += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.SUB:
			if t.Extra == 1 {
				f.Printf("%s(*p)--;\n", indent(indentLevel))
			} else {
				f.Printf("%s*p -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.INCP:
			if t.Extra == 1 {
				f.Printf("%sp++;\n", indent(indentLevel))
			} else {
				f.Printf("%sp += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.DECP:
			if t.Extra == 1 {
				f.Printf("%sp--;\n", indent(indentLevel))
			} else {
				f.Printf("%sp -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.OUT:
			if t.Extra == 1 {
				f.Printf("%sputchar(*p);\n", indent(indentLevel))
			} else {
				f.Printf("%sfor (int i = 0; i < %d; i++) {\n%s	putchar(*p);\n%s}\n", indent(indentLevel), t.Extra, indent(indentLevel), indent(indentLevel))
			}
		case l.IN:
			if t.Extra == 1 {
				f.Printf("%s*p = getchar();\n", indent(indentLevel))
			} else {
				f.Printf("%sfor (int i = 0; i < %d; i++) {\n%s	*p = getchar();\n%s}\n", indent(indentLevel), t.Extra, indent(indentLevel), indent(indentLevel))
			}
		case l.JMPF:
			f.Printf("%swhile (*p) {\n", indent(indentLevel))
			indentLevel++
		case l.JMPB:
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		case l.MUL:
			if t.Extra == -1 && t.Extra2 == 0 {
				f.Printf("%s*p = 0;\n", indent(indentLevel))
				continue
			}

			output := ""
			if t.Extra == 1 {
				output = fmt.Sprintf("%sp[%d] += *p;\n", indent(indentLevel), t.Extra2)
			} else if t.Extra == -1 {
				output = fmt.Sprintf("%sp[%d] -= *p;\n", indent(indentLevel), t.Extra2)
			} else {
				output = fmt.Sprintf("%sp[%d] += *p * %d;\n", indent(indentLevel), t.Extra2, t.Extra)
			}
			f.Print(strings.ReplaceAll(output, "p[0]", "*p"))
		case l.DIV:
			output := ""
			output = fmt.Sprintf("%sp[%d] /= %d;\n", indent(indentLevel), t.Extra2, t.Extra)
			f.Print(strings.ReplaceAll(output, "p[0]", "*p"))
		case l.BZ: // NB: We are ignoring labels right now, if we are going to need them for something else we need to switch this out with a goto
			f.Printf("%sif (*p) {\n", indent(indentLevel))
			indentLevel++
		case l.LBL:
			indentLevel--
			f.Printf("%s}\n", indent(indentLevel))
		}
	}
	f.Println("	return 0;")
	f.Println("}")
}
