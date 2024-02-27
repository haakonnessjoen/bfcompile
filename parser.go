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

// Check if all code inside a loop is inc/dec/incp/decp
func isSimpleLoop(tokens []g.ParseToken) (bool, int) {
	pointer := 0
	// We will iterate the tokens starting from the first token of our given slice
	for i, t := range tokens {
		// Keep track of the pointer
		if t.Tok.Tok == l.INCP {
			pointer += t.Extra
		} else if t.Tok.Tok == l.DECP {
			pointer -= t.Extra
		}

		// We found the end of this loop
		if t.Tok.Tok == l.JMPB {
			// If the pointer is back to 0 at this stage, we have a simple loop!
			if pointer == 0 {
				return true, i
			}

			// The pointer changes during the loop, we cannot optimize this
			return false, 0
		}

		// We found a jump, input or output, we cannot optimize this loop
		if t.Tok.Tok == l.JMPF || t.Tok.Tok == l.IN || t.Tok.Tok == l.OUT {
			return false, 0
		}
	}
	// This should not be reached
	fmt.Fprintln(os.Stderr, "Internal error: isSimpleLoop() reached end of code without finding the end of the loop.")
	os.Exit(1)
	return false, 0
}

// This will optimize the code and add multiplication
// so this optimizer will generate new tokens not supported by the Brainfuck generator
func optimize2(tokens []g.ParseToken) []g.ParseToken {
	newTokens := make([]g.ParseToken, 0, len(tokens))

mainloop:
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		token := t.Tok.Tok

		if token == l.JMPF {
			if tokens[i+1].Tok.Tok == l.JMPB {
				// Special case, a loop with no operation, cannot optimize as we cannot divide by 0.
				newTokens = append(newTokens, t)
				continue
			}

			isSimple, insts := isSimpleLoop(tokens[i+1:])
			if isSimple {
				// We found a simple loop that we can optimize away

				// If the operation is [-], just optimize it to *p -= *p
				if insts == 1 && tokens[i+1].Tok.Tok == l.SUB {
					newTokens = append(newTokens, g.ParseToken{
						Pos:    t.Pos,
						Tok:    l.Token{Tok: l.MUL, TokenName: "MUL", Character: ""},
						Extra:  -tokens[i+1].Extra,
						Extra2: 0,
					})
					i += 2
					continue
				}

				pointer := 0
				decrementer := 0
				// First find the number of decrements per loop, ex: ++++[>+<--] would only increase p[1] with 2
				for j := i + 1; j < i+1+insts; j++ {
					tt := tokens[j]
					ttoken := tt.Tok.Tok

					// Count the current pointer
					if ttoken == l.INCP {
						pointer += tt.Extra
					} else if ttoken == l.DECP {
						pointer -= tt.Extra
					} else if pointer == 0 && ttoken == l.ADD {
						// At the moment, we don't handle loops that counts upwards
						// We abort this optimization and just add the tokens as they are
						newTokens = append(newTokens, t)
						continue mainloop
					} else if pointer == 0 && ttoken == l.SUB {
						// Count the number of decrements of p[0]
						decrementer += tt.Extra
					}
				}

				pointer = 0
				// If the decrementer is not 1, we need to divide p[0] with the decrementer to get the correct multiplier
				if decrementer != 1 && decrementer != 0 {
					newTokens = append(newTokens, g.ParseToken{
						Pos:    t.Pos,
						Tok:    l.Token{Tok: l.DIV, TokenName: "DIV", Character: ""},
						Extra:  decrementer,
						Extra2: 0,
					})
				}

				// Start the actual optimization, lets add the multiplication operations
				for j := i + 1; j < i+1+insts; j++ {
					tt := tokens[j]
					ttoken := tt.Tok.Tok
					if pointer == 0 && ttoken == l.ADD {
						fmt.Fprintf(os.Stderr, "Internal error: Should not reach ADD at position %d:%d\n", tt.Pos.Line, tt.Pos.Column)
						os.Exit(1)
						continue mainloop
					} else if pointer == 0 && ttoken == l.SUB {
						// Ignore, we already handled this
					} else if ttoken == l.INCP {
						// Keep track of the pointer, this is previously optimized so .Extra holds the number of increments
						pointer += tt.Extra
					} else if ttoken == l.DECP {
						// Keep track of the pointer
						pointer -= tt.Extra
					} else {
						if ttoken != l.ADD && ttoken != l.SUB {
							fmt.Fprintf(os.Stderr, "Internal error: Unexpected token %v at position %d:%d\n", tt.Tok.TokenName, tt.Pos.Line, tt.Pos.Column)
						}
						// We should now either be at a ADD or SUB in a pointer other than 0
						var count = tt.Extra

						// If this is a SUB, we need to negate the count
						if ttoken == l.SUB {
							count = -count
						}

						// Add the multiplication operation
						// This would be translated to for example: p[pointer] += p[0] * count
						newTokens = append(newTokens, g.ParseToken{
							Pos:    tt.Pos,
							Tok:    l.Token{Tok: l.MUL, TokenName: "MUL", Character: ""},
							Extra:  count,
							Extra2: pointer,
						})
					}
				}

				// Set p[0] to 0, as it has just exited the "loop" and would be zero
				newTokens = append(newTokens, g.ParseToken{
					Pos:    t.Pos,
					Tok:    l.Token{Tok: l.MUL, TokenName: "MUL", Character: ""},
					Extra:  -1,
					Extra2: 0,
				})

				// Skip over the loop tokens we just processed, and continue in the outer loop
				i += insts + 1
				continue
			}
		}

		newTokens = append(newTokens, t)
	}

	return newTokens
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
				Pos:   t.Pos,
				Tok:   t.Tok,
				Extra: count,
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
					Pos:   t.Pos,
					Tok:   t.Tok,
					Extra: count,
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
