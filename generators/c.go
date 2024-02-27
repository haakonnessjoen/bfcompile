package generators

import (
	l "bcomp/lexer"
	"fmt"
)

// PrintC prints the tokens as C code
func PrintC(tokens []ParseToken, includeComments bool) {
	fmt.Println("#include <stdio.h>")
	fmt.Println("#include <stdint.h>")

	fmt.Printf("uint8_t mem[30000];\n")
	fmt.Println("int main() {")
	fmt.Println("	uint8_t *p = mem;")

	indentLevel := 1
	for _, t := range tokens {
		if includeComments {
			fmt.Printf("%s// Line %d, Pos %d: %v\n", indent(indentLevel), t.Pos.Line, t.Pos.Column, t.Tok)
		}

		switch t.Tok.Tok {
		case l.ADD:
			if t.Extra == 1 {
				fmt.Printf("%s(*p)++;\n", indent(indentLevel))
			} else {
				fmt.Printf("%s*p += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.SUB:
			if t.Extra == 1 {
				fmt.Printf("%s(*p)--;\n", indent(indentLevel))
			} else {
				fmt.Printf("%s*p -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.INCP:
			if t.Extra == 1 {
				fmt.Printf("%sp++;\n", indent(indentLevel))
			} else {
				fmt.Printf("%sp += %d;\n", indent(indentLevel), t.Extra)
			}
		case l.DECP:
			if t.Extra == 1 {
				fmt.Printf("%sp--;\n", indent(indentLevel))
			} else {
				fmt.Printf("%sp -= %d;\n", indent(indentLevel), t.Extra)
			}
		case l.OUT:
			if t.Extra == 1 {
				fmt.Printf("%sputchar(*p);\n", indent(indentLevel))
			} else {
				fmt.Printf("%sfor (int i = 0; i < %d; i++) {\n%s	putchar(*p);\n%s}\n", indent(indentLevel), t.Extra, indent(indentLevel), indent(indentLevel))
			}
			fmt.Printf("%sputchar(*p);\n", indent(indentLevel))
		case l.IN:
			if t.Extra == 1 {
				fmt.Printf("%s*p = getchar();\n", indent(indentLevel))
			} else {
				fmt.Printf("%sfor (int i = 0; i < %d; i++) {\n%s	*p = getchar();\n%s}\n", indent(indentLevel), t.Extra, indent(indentLevel), indent(indentLevel))
			}
		case l.JMPF:
			fmt.Printf("%swhile (*p) {\n", indent(indentLevel))
			indentLevel++
		case l.JMPB:
			indentLevel--
			fmt.Printf("%s}\n", indent(indentLevel))
		}
	}
	fmt.Printf("	return 0;\n")
	fmt.Println("}")
}
