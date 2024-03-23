package generators

import (
	l "bcomp/lexer"
	"log"
	"math"
)

func printILStore(f *GeneratorOutput, wordSize int, to, from string) {
	if wordSize == 8 {
		f.Printf("	storeb %s, %s\n", to, from)
	} else if wordSize == 16 {
		f.Printf("	storeh %s, %s\n", to, from)
	} else if wordSize == 32 {
		f.Printf("	storew %s, %s\n", to, from)
	}
}

func printILExt(f *GeneratorOutput, wordSize int, to, from string) {
	if wordSize == 8 {
		f.Printf("	%s =w extub %s\n", to, from)
	} else if wordSize == 16 {
		f.Printf("	%s =w extuh %s\n", to, from)
	}
}

func printILLoad(f *GeneratorOutput, wordSize int, to, from string) {
	if wordSize == 8 {
		f.Printf("	%s =w loadub %s\n", to, from)
	} else if wordSize == 16 {
		f.Printf("	%s =w loaduh %s\n", to, from)
	} else if wordSize == 32 {
		f.Printf("	%s =w loadw %s\n", to, from)
	}
}

// PrintIL prints the tokens as IL code
func PrintIL(f *GeneratorOutput, tokens []ParseToken, includeComments bool, memorySize int, wordSize int) {
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

			printILStore(f, wordSize, "%v", "%p")
			printILExt(f, wordSize, "%v", "%v")
		case l.SUB:
			f.Printf("	%%v =w sub %%v, %d\n", t.Extra)

			printILStore(f, wordSize, "%v", "%p")
			printILExt(f, wordSize, "%v", "%v")
		case l.INCP:
			f.Printf("	%%p =l add %%p, %d\n", t.Extra*wordSize/8)

			printILLoad(f, wordSize, "%v", "%p")
		case l.DECP:
			f.Printf("	%%p =l sub %%p, %d\n", t.Extra*wordSize/8)

			printILLoad(f, wordSize, "%v", "%p")
		case l.OUT:
			for i := 0; i < t.Extra; i++ {
				f.Printf("	call $write(w 1, l %%p, w 1)\n")
			}
		case l.IN:
			for i := 0; i < t.Extra; i++ {
				f.Printf("    call $read(w 0, l %%p, w 1)\n")
			}
			// Since they will all be overwritten, we only push the last value back to the memory
			printILLoad(f, wordSize, "%v", "%p")
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

			sourcevar := "%v2"
			destvar := "%p2"

			if multiplier == 1 || multiplier == -1 {
				sourcevar = "%v"
			} else {
				f.Printf("	%%v2 =w mul %%v, %d\n", int(math.Abs(float64(multiplier))))

				printILExt(f, wordSize, "%v2", "%v2")
			}

			if ptr == 0 {
				destvar = "%p"
			} else {
				if ptr > 0 {
					f.Printf("	%%p2 =l add %%p, %d\n", ptr*wordSize/8)
				} else {
					f.Printf("	%%p2 =l sub %%p, %d\n", -ptr*wordSize/8)
				}
			}

			printILLoad(f, wordSize, "%v3", destvar)

			if multiplier > 0 {
				f.Printf("	%%v3 =w add %%v3, %s\n", sourcevar)
			} else {
				f.Printf("	%%v3 =w sub %%v3, %s\n", sourcevar)
			}

			printILStore(f, wordSize, "%v3", destvar)

			if ptr == 0 {
				printILExt(f, wordSize, "%v", "%v3")
			}
		case l.DIV:
			// p[%d] /= %d;
			ptr := t.Extra2
			if ptr == 0 {
				f.Printf("	%%v =w div %%v, %d\n", t.Extra)

				printILStore(f, wordSize, "%v", "%p")
				printILExt(f, wordSize, "%v", "%v")
			} else {
				log.Fatalf("Internal error: DIV operation with pointer other than 0 is not implemented\n")
			}
		case l.BZ:
			f.Printf("	jnz %%v, @JMP%df, @JMP%d\n", t.Extra, t.Extra)
			f.Printf("@JMP%df\n", t.Extra)
		case l.LBL:
			f.Printf("@JMP%d\n", t.Extra)
		case l.MOV:
			ptr := t.Extra2
			value := t.Extra
			if ptr == 0 {
				f.Printf("	%%v =w copy %d\n", value)

				printILStore(f, wordSize, "%v", "%p")
			} else {
				if ptr > 0 {
					f.Printf("	%%p2 =l add %%p, %d\n", ptr*wordSize/8)
				} else {
					f.Printf("	%%p2 =l sub %%p, %d\n", -ptr*wordSize/8)
				}
				f.Printf("	%%v2 =w copy %d\n", value)

				printILStore(f, wordSize, "%v", "%p2")
			}
		default:
			log.Fatalf("Error: Unknown token %v\n", t.Tok)
		}
	}
	f.Println("	ret 0")
	f.Println("}")
}
