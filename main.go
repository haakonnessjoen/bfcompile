package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// parse command line arguments
	args := os.Args
	filename := ""

	// Get first argument
	if len(args) > 1 {
		if !strings.HasSuffix(args[1], ".bf") {
			fmt.Fprintln(os.Stderr, "First argument must be a .bf file")
			os.Exit(1)
		}
		filename = args[1]
	} else {
		fmt.Fprintf(os.Stderr, "Usage: %s <arg1.bf>\n", args[0])
		os.Exit(1)
	}

	parseFile(filename)
}

type ParseToken struct {
	pos     Position
	tok     Token
	address int
	extra   int
}

type JumpStack struct {
	elements []int
}

func (s *JumpStack) Push(v int) {
	s.elements = append(s.elements, v)
}

func (s *JumpStack) Pop() int {
	v := s.elements[len(s.elements)-1]
	s.elements = s.elements[:len(s.elements)-1]
	return v
}

func (s *JumpStack) Len() int {
	return len(s.elements)
}

func parseFile(filename string) {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting file info:", err)
		os.Exit(1)
	}
	fileSize := fileInfo.Size()

	var tokens []ParseToken = make([]ParseToken, 0, fileSize)

	jumpstack := &JumpStack{elements: make([]int, 0)}
	jumps := 0

	lexer := NewLexer(file)
	for {
		pos, tok, _ := lexer.Lex()
		if tok == EOF {
			break
		}

		//fmt.Printf("%d:%d\t%s\t%s\n", pos.line, pos.column, tok.String(), lit)
		address := len(tokens)

		switch tok {
		case ADD, SUB, INCP, DECP, OUT, IN:
			tokens = append(tokens, ParseToken{pos, tok, address, 0})
		case JMPF:
			jumps++
			jumpstack.Push(jumps)
			tokens = append(tokens, ParseToken{pos, tok, address, jumps})
		case JMPB:
			if jumpstack.Len() == 0 {
				fmt.Fprintf(os.Stderr, "Line %d, Pos %d: Unmatched ']'\n", pos.line, pos.column)
				os.Exit(1)
			}
			jumpto := jumpstack.Pop()
			tokens = append(tokens, ParseToken{pos, tok, address, jumpto})
		}
	}

	PrintIL(tokens)
}

func PrintIL(tokens []ParseToken) {
	fmt.Println("data $ch = { w 0 }")
	fmt.Println("data $MEM = { z 30000 }")
	fmt.Println("export function w $main() {")
	fmt.Println("@start")
	fmt.Println("	%P =l copy $MEM")
	for _, t := range tokens {
		switch t.tok {
		case ADD:
			fmt.Printf("	%%v =w loadub %%P\n")
			fmt.Printf("	%%v =w add %%v, 1\n")
			fmt.Printf("	storew %%v, %%P\n")
		case SUB:
			fmt.Printf("	%%v =w loadub %%P\n")
			fmt.Printf("	%%v =w sub %%v, 1\n")
			fmt.Printf("	storew %%v, %%P\n")
		case INCP:
			fmt.Printf("	%%P =l add %%P, 1\n")
		case DECP:
			fmt.Printf("	%%P =l sub %%P, 1\n")
		case OUT:
			fmt.Printf("	%%fd =w copy 1\n")
			fmt.Printf("	%%ch =w loadub %%P\n")
			fmt.Printf("	storew %%ch, $ch\n")
			fmt.Printf("	%%r =w call $write(w %%fd, l $ch, w 1)\n")
		case IN:
			// TODO
		case JMPF:
			fmt.Printf("@JMP%df\n", t.extra)
			fmt.Printf("	%%v =w loadub %%P\n")
			fmt.Printf("	jnz %%v, @JMP%dfd, @JMP%db\n", t.extra, t.extra)
			fmt.Printf("@JMP%dfd\n", t.extra)
		case JMPB:
			fmt.Printf("@JMP%db\n", t.extra)
			fmt.Printf("	%%v =w loadub %%P\n")
			fmt.Printf("	jnz %%v, @JMP%df, @JMP%dbd\n", t.extra, t.extra)
			fmt.Printf("@JMP%dbd\n", t.extra)
		}
	}
	fmt.Println("	ret 0")
	fmt.Println("}")
}
