package generators

import (
	l "bcomp/lexer"
	"fmt"
	"math"
	"os"
)

// PrintIL prints the tokens as IL code
func PrintIL(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int) {
	f.Printf("data $MEM = { z %d }\n", memorySize)

	f.Println("export function w $main() {")
	f.Println("@start")
	f.Printf("	%%p =l copy $MEM\n")
	f.Printf("	%%v =l copy 0\n")
	for _, t := range tokens {
		if includeComments {
			f.Printf("# Pos %d:%d %s (%s, %d, %d)\n", t.Pos.Line, t.Pos.Column, t.Tok.Character, t.Tok.TokenName, t.Extra, t.Extra2)
		}

		switch t.Tok.Tok {
		case l.ADD:
			f.Printf("	%%v =w add %%v, %d\n", t.Extra)
			f.Printf("	storeb %%v, %%p\n")
			f.Printf("	%%v =w extub %%v\n")
		case l.SUB:
			f.Printf("	%%v =w sub %%v, %d\n", t.Extra)
			f.Printf("	storeb %%v, %%p\n")
			f.Printf("	%%v =w extub %%v\n")
		case l.INCP:
			f.Printf("	%%p =l add %%p, %d\n", t.Extra)
			f.Printf("	%%v =w loadub %%p\n")
		case l.DECP:
			f.Printf("	%%p =l sub %%p, %d\n", t.Extra)
			f.Printf("	%%v =w loadub %%p\n")
		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				f.Printf("	call $write(w 1, l %%p, w 1)\n")
			}
		case l.IN:
			for i := 0; i < t.Extra; i++ {
				f.Printf("    call $read(w 0, l %%p, w 1)\n")
			}
			// Since they will all be overwritten, we only push the last value back to the memory
			f.Printf("	%%v =w loadub %%p\n")
		case l.JMPF:
			f.Printf("@JMP%df\n", t.Extra)
			f.Printf("	jnz %%v, @JMP%dfd, @JMP%dbd\n", t.Extra, t.Extra)
			f.Printf("@JMP%dfd\n", t.Extra)
		case l.JMPB:
			f.Printf("	jmp @JMP%df\n", t.Extra)
			f.Printf("@JMP%dbd\n", t.Extra)
		case l.MUL:
			// p[%d] += *p * %d;
			multiplier := t.Extra
			ptr := t.Extra2

			if ptr == 0 && multiplier == -1 {
				f.Printf("	%%v =w copy 0\n")
				f.Printf("	storeb %%v, %%p\n")
				continue
			}

			sourcevar := "%v2"
			destvar := "%p2"

			if multiplier == 1 || multiplier == -1 {
				sourcevar = "%v"
			} else {
				f.Printf("	%%v2 =w mul %%v, %d\n", int(math.Abs(float64(multiplier))))
				f.Printf("	%%v2 =w extub %%v2\n")
			}

			if ptr == 0 {
				destvar = "%p"
			} else {
				if ptr > 0 {
					f.Printf("	%%p2 =l add %%p, %d\n", ptr)
				} else {
					f.Printf("	%%p2 =l sub %%p, %d\n", -ptr)
				}
			}
			f.Printf("	%%v3 =w loadub %s\n", destvar)

			if multiplier > 0 {
				f.Printf("	%%v3 =w add %%v3, %s\n", sourcevar)
			} else {
				f.Printf("	%%v3 =w sub %%v3, %s\n", sourcevar)
			}

			f.Printf("	storeb %%v3, %s\n", destvar)

			if ptr == 0 {
				f.Printf("	%%v =w extub %%v3\n")
			}
		case l.DIV:
			// p[%d] /= %d;
			ptr := t.Extra2
			if ptr == 0 {
				f.Printf("	%%v =w div %%v, %d\n", t.Extra)
				f.Printf("	storeb %%v, %%p\n")
				f.Printf("	%%v =w extub %%v\n")
			} else {
				fmt.Fprint(os.Stderr, "Internal error: DIV operation with pointer other than 0 is not implemented\n")
			}
		case l.BZ:
			f.Printf("	jnz %%v, @JMP%df, @JMP%d\n", t.Extra, t.Extra)
			f.Printf("@JMP%df\n", t.Extra)
		case l.LBL:
			f.Printf("@JMP%d\n", t.Extra)
		}
	}
	f.Println("	ret 0")
	f.Println("}")
}
