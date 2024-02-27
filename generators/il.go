package generators

import (
	l "bcomp/lexer"
	"fmt"
	"math"
	"os"
)

// PrintIL prints the tokens as IL code
func PrintIL(tokens []ParseToken, includeComments bool) {
	fmt.Println("data $MEM = { z 30000 }")

	fmt.Println("export function w $main() {")
	fmt.Println("@start")
	fmt.Printf("	%%p =l copy $MEM\n")
	fmt.Printf("	%%v =l copy 0\n")
	for _, t := range tokens {
		if includeComments {
			fmt.Printf("# Line %d, Pos %d: %s (%s, %d, %d)\n", t.Pos.Line, t.Pos.Column, t.Tok.Character, t.Tok.TokenName, t.Extra, t.Extra2)
		}

		switch t.Tok.Tok {
		case l.ADD:
			fmt.Printf("	%%v =w add %%v, %d\n", t.Extra)
			fmt.Printf("	storeb %%v, %%p\n")
			fmt.Printf("    %%v =w extub %%v\n")
		case l.SUB:
			fmt.Printf("	%%v =w sub %%v, %d\n", t.Extra)
			fmt.Printf("	storeb %%v, %%p\n")
			fmt.Printf("    %%v =w extub %%v\n")
		case l.INCP:
			fmt.Printf("	%%p =l add %%p, %d\n", t.Extra)
			fmt.Printf("	%%v =w loadub %%p\n")
		case l.DECP:
			fmt.Printf("	%%p =l sub %%p, %d\n", t.Extra)
			fmt.Printf("	%%v =w loadub %%p\n")
		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				fmt.Printf("	call $write(w 1, l %%p, w 1)\n")
			}
		case l.IN:
			for i := 0; i < t.Extra; i++ {
				fmt.Printf("    call $read(w 0, l %%p, w 1)\n")
			}
			// Since they will all be overwritten, we only push the last value back to the memory
			fmt.Printf("	%%v =w loadub %%p\n")
		case l.JMPF:
			fmt.Printf("@JMP%df\n", t.Extra)
			fmt.Printf("	jnz %%v, @JMP%dfd, @JMP%dbd\n", t.Extra, t.Extra)
			fmt.Printf("@JMP%dfd\n", t.Extra)
		case l.JMPB:
			fmt.Printf("	jmp @JMP%df\n", t.Extra)
			fmt.Printf("@JMP%dbd\n", t.Extra)
		case l.MUL:
			// p[%d] += *p * %d;
			multiplier := t.Extra
			ptr := t.Extra2

			fmt.Printf("	%%v2 =w mul %%v, %d\n", int(math.Abs(float64(multiplier))))
			fmt.Printf("    %%v2 =w extub %%v2\n")

			fmt.Printf("	%%p2 =l add %%p, %d\n", ptr)

			fmt.Printf("	%%v3 =w loadub %%p2\n")

			if multiplier > 0 {
				fmt.Printf("	%%v3 =w add %%v3, %%v2\n")
			} else {
				fmt.Printf("	%%v3 =w sub %%v3, %%v2\n")
			}
			fmt.Printf("	storeb %%v3, %%p2\n")
			fmt.Printf("    %%v3 =w extub %%v3\n")
		case l.DIV:
			// p[%d] /= %d;
			ptr := t.Extra2
			if ptr == 0 {
				fmt.Printf("	%%v =w div %%v, %d\n", t.Extra)
				fmt.Printf("	storeb %%v, %%p\n")
				fmt.Printf("    %%v =w extub %%v\n")
			} else {
				fmt.Fprint(os.Stderr, "Internal error: DIV operation with pointer other than 0 is not implemented\n")
			}

		}
	}
	fmt.Println("	ret 0")
	fmt.Println("}")
}
