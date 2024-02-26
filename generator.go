package main

import "fmt"

// PrintIL prints the tokens as IL code
func PrintIL(tokens []ParseToken) {
	fmt.Println("data $MEM = { z 30000 }")

	fmt.Println("export function w $main() {")
	fmt.Println("@start")
	fmt.Printf("	%%p =l copy $MEM\n")
	for _, t := range tokens {
		// Uncomment to get inline comments showing the current operation
		fmt.Printf("# Line %d, Pos %d: %v\n", t.pos.line, t.pos.column, t.tok)
		switch t.tok.Tok {
		case ADD:
			fmt.Printf("	%%v =w loadub %%p\n")
			fmt.Printf("	%%v =w add %%v, %d\n", t.extra)
			fmt.Printf("	storeb %%v, %%p\n")
		case SUB:
			fmt.Printf("	%%v =w loadub %%p\n")
			fmt.Printf("	%%v =w sub %%v, %d\n", t.extra)
			fmt.Printf("	storeb %%v, %%p\n")
		case INCP:
			fmt.Printf("	%%p =l add %%p, %d\n", t.extra)
		case DECP:
			fmt.Printf("	%%p =l sub %%p, %d\n", t.extra)
		case OUT:
			fmt.Printf("	%%r =w call $write(w 1, l %%p, w 1)\n")
		case IN:
			fmt.Printf("    %%r =w call $read(w 0, l %%p, w 1)\n")
		case JMPF:
			fmt.Printf("@JMP%df\n", t.extra)
			fmt.Printf("	%%v =w loadub %%p\n")
			fmt.Printf("	jnz %%v, @JMP%dfd, @JMP%dbd\n", t.extra, t.extra)
			fmt.Printf("@JMP%dfd\n", t.extra)
		case JMPB:
			fmt.Printf("	jmp @JMP%df\n", t.extra)
			fmt.Printf("@JMP%dbd\n", t.extra)
		}
	}
	fmt.Println("	ret 0")
	fmt.Println("}")
}

// PrintC prints the tokens as C code
func PrintC(tokens []ParseToken) {
	fmt.Println("#include <stdio.h>")
	fmt.Println("#include <stdint.h>")

	fmt.Printf("uint8_t mem[30000];\n")
	fmt.Println("int main() {")
	fmt.Println("	uint8_t *p = mem;")

	indentLevel := 0
	for _, t := range tokens {
		switch t.tok.Tok {
		case ADD:
			if t.extra == 1 {
				fmt.Printf("%s	(*p)++;\n", indent(indentLevel))
			} else {
				fmt.Printf("%s	*p += %d;\n", indent(indentLevel), t.extra)
			}
		case SUB:
			if t.extra == 1 {
				fmt.Printf("%s	(*p)--;\n", indent(indentLevel))
			} else {
				fmt.Printf("%s	*p -= %d;\n", indent(indentLevel), t.extra)
			}
		case INCP:
			if t.extra == 1 {
				fmt.Printf("%s	p++;\n", indent(indentLevel))
			} else {
				fmt.Printf("%s	p += %d;\n", indent(indentLevel), t.extra)
			}
		case DECP:
			if t.extra == 1 {
				fmt.Printf("%s	p--;\n", indent(indentLevel))
			} else {
				fmt.Printf("%s	p -= %d;\n", indent(indentLevel), t.extra)
			}
		case OUT:
			fmt.Printf("%s	putchar(*p);\n", indent(indentLevel))
		case IN:
			fmt.Printf("%s	*p = getchar();\n", indent(indentLevel))
		case JMPF:
			fmt.Printf("%s	while (*p) {\n", indent(indentLevel))
			indentLevel++
		case JMPB:
			indentLevel--
			fmt.Printf("%s	}\n", indent(indentLevel))
		}
	}
	fmt.Printf("	return 0;\n")
	fmt.Println("}")
}
