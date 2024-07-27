package generators

import (
	l "bcomp/lexer"
	"fmt"
	"log"
	"os"
	"strings"
)

// PrintC prints the tokens as C code
func PrintC(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int, wordSize int) {
	wordType := ""
	if wordSize == 8 {
		wordType = "uint8_t"
	} else if wordSize == 16 {
		wordType = "uint16_t"
	} else if wordSize == 32 {
		wordType = "uint32_t"
	} else if wordSize == 64 {
		wordType = "uint64_t"
	} else {
		log.Fatalf("Error: Unknown word size %d\n", wordSize)
	}

	f.Println("#include <stdio.h>")
	f.Println("#include <stdint.h>")
	f.Println("#include <string.h>")

	f.Printf("%s mem[%d];\n", wordType, memorySize)
	f.Println("int main() {")
	f.Printf("	%s *p = mem;\n", wordType)

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
		case l.SCANL:
			// Not implemented, because memrchr only seems to be included in GNU standard library
		case l.SCANR:
			f.Printf("%sp = (%s *)(memchr(p, 0, sizeof(mem) - (p-mem)));\n", indent(indentLevel), wordType)
		case l.MOV:
			if t.Extra2 == 0 {
				f.Printf("%s*p = %d;\n", indent(indentLevel), t.Extra)
			} else {
				f.Printf("%sp[%d] = %d;\n", indent(indentLevel), t.Extra2, t.Extra)
			}
		case l.PRNT:
			if wordSize == 8 {
				f.Printf("%sp += fputs((char *)p, stdout);\n", indent(indentLevel))
			} else {
				f.Printf("%swhile (p* != 0) { putchar(*p); p++; }\n", indent(indentLevel))
			}
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
			f.Printf("%s}\n", indent(indentLevel))
		}
	}
	f.Println("	return 0;")
	f.Println("}")
}
