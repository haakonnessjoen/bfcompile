package main

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

func parseFile(filename string) []g.ParseToken {
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

		address := len(tokens)

		switch tok.Tok {
		case l.ADD, l.SUB, l.INCP, l.DECP, l.OUT, l.IN:
			tokens = append(tokens, g.ParseToken{Pos: pos, Tok: tok, Address: address, Extra: 1})
		case l.JMPF:
			jumps++
			jumpstack.Push(jumps)
			tokens = append(tokens, g.ParseToken{Pos: pos, Tok: tok, Address: address, Extra: jumps})
		case l.JMPB:
			if jumpstack.Len() == 0 {
				fmt.Fprintf(os.Stderr, "Line %d, Pos %d: Unmatched ']'\n", pos.Line, pos.Column)
				os.Exit(1)
			}
			jumpto := jumpstack.Pop()
			tokens = append(tokens, g.ParseToken{Pos: pos, Tok: tok, Address: address, Extra: jumpto})
		}
	}

	return tokens
}

func optimize(tokens []g.ParseToken) []g.ParseToken {
	newTokens := make([]g.ParseToken, 0, len(tokens))

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		token := t.Tok.Tok
		count := t.Extra
		instructions := 0

		// Optimize IN/OUT
		if token == l.IN || token == l.OUT {
			// Aggregate equal instructions
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Tok.Tok != token {
					break
				}
				count += tokens[j].Extra
				instructions++
			}

			newTokens = append(newTokens, g.ParseToken{
				Pos:     t.Pos,
				Tok:     t.Tok,
				Address: t.Address,
				Extra:   count,
			})

		} else

		// Optimize ADD, SUB, INCP, DECP
		if token == l.ADD || token == l.SUB || token == l.INCP || token == l.DECP {
			var j int

			// Aggregate equal instructions
			for j = i + 1; j < len(tokens); j++ {
				if tokens[j].Tok.Tok != token {
					break
				}
				count += tokens[j].Extra
				instructions++
			}

			// Remove redundant instructions, for example: +- or >< would cancel each other out
			if j < len(tokens) && (token == l.INCP || token == l.ADD) {
				if tokens[j].Tok.Tok == token+1 { // DECP or SUB
					count -= tokens[j].Extra
					instructions++
				}
			} else
			// Remove opposite redundant instructions, for example: -+ or <> would cancel each other out
			if j < len(tokens) && (token == l.DECP || token == l.SUB) {
				if tokens[j].Tok.Tok == token-1 { // INCP or ADD
					count -= tokens[j].Extra
					instructions++
				}
			}

			// Operation has reversed itself, swap operation
			if count < 0 {
				switch token {
				case l.ADD:
					t.Tok = l.Token{Tok: l.SUB, TokenName: "SUB", Character: "-"}
				case l.SUB:
					t.Tok = l.Token{Tok: l.ADD, TokenName: "ADD", Character: "+"}
				case l.INCP:
					t.Tok = l.Token{Tok: l.DECP, TokenName: "DECP", Character: "<"}
				case l.DECP:
					t.Tok = l.Token{Tok: l.INCP, TokenName: "INCP", Character: ">"}
				}
				count = -count
			}

			if count > 0 {
				newTokens = append(newTokens, g.ParseToken{
					Pos:     t.Pos,
					Tok:     t.Tok,
					Address: t.Address,
					Extra:   count,
				})
			}

		} else {
			newTokens = append(newTokens, t)
		}

		// Skip already processed tokens
		i += instructions
	}

	return newTokens
}
