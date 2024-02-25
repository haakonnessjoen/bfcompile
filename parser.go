package main

import (
	"fmt"
	"os"
	"strings"
)

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

func parseFile(filename string) []ParseToken {
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
			tokens = append(tokens, ParseToken{pos, tok, address, 1})
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

	return tokens
}

func optimize(tokens []ParseToken) []ParseToken {
	newTokens := make([]ParseToken, 0, len(tokens))

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		if t.tok == ADD || t.tok == SUB || t.tok == INCP || t.tok == DECP {
			count := 1
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].tok != t.tok {
					break
				}
				count++
			}
			newTokens = append(newTokens, ParseToken{t.pos, t.tok, t.address, count})
			i += count - 1
		} else {
			newTokens = append(newTokens, t)
		}
	}

	return newTokens
}

func indent(n int) string {
	return strings.Repeat("\t", n)
}
