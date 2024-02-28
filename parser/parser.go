package parser

import (
	g "bcomp/generators"
	l "bcomp/lexer"
	"fmt"
	"os"
)

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

func ParseFile(filename string) []g.ParseToken {
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

	var tokens []g.ParseToken = make([]g.ParseToken, 0, fileSize)

	jumpstack := &JumpStack{elements: make([]int, 0)}
	jumps := 0

	lexer := l.NewLexer(file)
	for {
		pos, tok := lexer.Lex()
		if tok.Tok == l.EOF {
			break
		}

		switch tok.Tok {
		case l.ADD, l.SUB, l.INCP, l.DECP, l.OUT, l.IN:
			tokens = append(tokens, g.ParseToken{Pos: pos, Tok: tok, Extra: 1})
		case l.JMPF:
			jumps++
			jumpstack.Push(jumps)
			tokens = append(tokens, g.ParseToken{Pos: pos, Tok: tok, Extra: jumps})
		case l.JMPB:
			if jumpstack.Len() == 0 {
				fmt.Fprintf(os.Stderr, "Line %d, Pos %d: Unmatched ']'\n", pos.Line, pos.Column)
				os.Exit(1)
			}
			jumpto := jumpstack.Pop()
			tokens = append(tokens, g.ParseToken{Pos: pos, Tok: tok, Extra: jumpto})
		}
	}

	return tokens
}
