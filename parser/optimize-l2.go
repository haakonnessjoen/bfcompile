package parser

import (
	g "bcomp/generators"
	l "bcomp/lexer"

	"fmt"
	"os"
)

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
func Optimize2(tokens []g.ParseToken) []g.ParseToken {
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
