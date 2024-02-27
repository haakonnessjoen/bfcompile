package generators

import (
	l "bcomp/lexer"
	"fmt"
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
			fmt.Printf("# Line %d, Pos %d: %v\n", t.Pos.Line, t.Pos.Column, t.Tok)
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
			// EXTRA = multiplier
			// EXTRA2 = pointer

			// p[%d] += *p * %d;
			if t.Extra > 1 || t.Extra < -1 {
				fmt.Printf("	%%v3 =w mul %%v, %d\n", t.Extra)

				if t.Extra2 == 0 {
					fmt.Printf("	%%v =w add %%v, %%v3\n")
					fmt.Printf("	storeb %%v, %%p\n")
					fmt.Printf("    %%v =w extub %%v\n")
				} else {
					if t.Extra2 > 0 {
						fmt.Printf("	%%p2 =l add %%p, %d\n", t.Extra2)
					} else if t.Extra2 < 0 {
						fmt.Printf("	%%p2 =l sub %%p, %d\n", -t.Extra2)
					}

					fmt.Printf("	%%v2 =w loadub %%p2\n")
					fmt.Printf("	%%v2 =w add %%v3, %%v2\n")
					fmt.Printf("	storeb %%v2, %%p2\n")
				}
			} else if t.Extra == 1 || t.Extra == -1 {
				// p[%d] += *p;
				if t.Extra2 == 0 {
					if t.Extra > 0 {
						fmt.Printf("	%%v =w add %%v, %%v\n")
					} else if t.Extra < 0 {
						fmt.Printf("	%%v =w sub %%v, %%v\n")
					}
					fmt.Printf("	storeb %%v, %%p\n")
					fmt.Printf("    %%v =w extub %%v\n")
				} else {
					if t.Extra2 > 0 {
						fmt.Printf("	%%p2 =l add %%p, %d\n", t.Extra2)
					} else if t.Extra2 < 0 {
						fmt.Printf("	%%p2 =l sub %%p, %d\n", -t.Extra2)
					}
					fmt.Printf("	%%v2 =w loadub %%p2\n")
					if t.Extra == 1 {
						fmt.Printf("	%%v2 =w add %%v, %%v2\n")
					} else if t.Extra == -1 {
						fmt.Printf("	%%v2 =w sub %%v, %%v2\n")
					}
					fmt.Printf("	storeb %%v2, %%p2\n")
				}
			} else {
				fmt.Fprintf(os.Stderr, "Internal error: MUL operation with wrong multiplier: %d\n", t.Extra)
			}
		case l.DIV:
			// p[%d] /= %d;
			if t.Extra2 == 0 {
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
