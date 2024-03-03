package parser

import (
	g "bcomp/generators"
	l "bcomp/lexer"

	"fmt"
	"os"
)

var Debug = false

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

func D(token g.ParseToken, format string, a ...interface{}) {
	if Debug {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%d:%d %s (%d, %d) => %s\n", token.Pos.Line, token.Pos.Column, token.Tok.TokenName, token.Extra, token.Extra2, format), a...)
	}
}

func Peek(tokens *[]g.ParseToken, idx int) *g.ParseToken {
	if idx < 0 || idx > len(*tokens)-1 {
		return &g.ParseToken{Tok: l.Token{Tok: l.NOP, TokenName: "", Character: ""}, Pos: l.Position{}}
	}
	return &(*tokens)[idx]
}

func findLoopEnd(tokens []g.ParseToken) int {
	depth := 1
	for i, t := range tokens {
		if t.Tok.Tok == l.JMPF {
			depth++
		} else if t.Tok.Tok == l.JMPB {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

func setValueAfterLoop(t g.ParseToken, tokens []g.ParseToken) (newToken *g.ParseToken, lastOpWasLoop bool, insts int) {
	value := 0
	insts = 0
	lastOpWasLoop = true

	if Peek(&tokens, 1).Tok.Tok == l.ADD {
		value = Peek(&tokens, 1).Extra
		D(t, "SLO: Pushing a MOV with %d, 0 to set final value of mem[p]", value)
		insts++
	} else if Peek(&tokens, 1).Tok.Tok == l.SUB {
		value = (0 - Peek(&tokens, 1).Extra) % 256
		D(t, "SLO: Pushing a MOV with %d, 0 to set final value of mem[p]", value)
		insts++
	} else {
		D(t, "SLO: No ADD/SUB found, pushing a MUL with -1, 0 to just empty the current mem[p]")
	}

	if value > 0 {
		newToken = &g.ParseToken{
			Pos:    tokens[1].Pos,
			Tok:    l.Token{Tok: l.MOV, TokenName: "MOV", Character: ""},
			Extra:  value,
			Extra2: 0,
		}
		lastOpWasLoop = false
	} else {
		newToken = &g.ParseToken{
			Pos:    tokens[0].Pos,
			Tok:    l.Token{Tok: l.MUL, TokenName: "MUL", Character: ""},
			Extra:  -1,
			Extra2: 0,
		}
		lastOpWasLoop = false
	}

	return
}

// This will optimize the code and add multiplication
// so this optimizer will generate new tokens not supported by the Brainfuck generator
func Optimize2(tokens []g.ParseToken, generator string) []g.ParseToken {
	newTokens := make([]g.ParseToken, 0, len(tokens))
	lastOpWasLoop := false
mainloop:
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		token := t.Tok.Tok

		if token == l.JMPF {
			// If a loop start immediately after another one, it will never be entered.
			// So we can remove it and everything it contains.
			if lastOpWasLoop {
				// Find next JMPB on the same scope
				insts := findLoopEnd(tokens[i+1:])
				if insts == -1 {
					fmt.Fprintf(os.Stderr, "Parse error: Unterminated loop at %d:%d", t.Pos.Line, t.Pos.Column)
				}
				// Skip over loop, and let "lastOpWasLoop" be true
				// as this also was a loop.
				D(t, "Skipping entire loop, since we know mem[p] is 0")
				i += insts + 1
				continue
			}

			if Peek(&tokens, i+1).Tok.Tok == l.JMPB {
				D(t, "Aborting, putting JMPF back")
				// Special case, a loop with no operation, cannot optimize as we cannot divide by 0.
				newTokens = append(newTokens, t)
				lastOpWasLoop = false
				continue
			}

			// Found this idea here: http://calmerthanyouare.org/2015/01/07/optimizing-brainfuck.html
			// C stdlib has memchr() to go through data fast, (but seems like memrchr() is only in gnu stdlib)
			if generator == "c" {
				if Peek(&tokens, i+2).Tok.Tok == l.JMPB && (Peek(&tokens, i+1).Tok.Tok == l.INCP || Peek(&tokens, i+1).Tok.Tok == l.DECP) && Peek(&tokens, i+1).Extra == 1 {
					D(t, "C optimization, found a simple scanloop")
					if Peek(&tokens, i+1).Tok.Tok == l.INCP {
						newTokens = append(newTokens, g.ParseToken{
							Pos: t.Pos,
							Tok: l.Token{Tok: l.SCANR, TokenName: "SCANR", Character: ""},
						})
						/*} else {
							newTokens = append(newTokens, g.ParseToken{
								Pos: t.Pos,
								Tok: l.Token{Tok: l.SCANL, TokenName: "SCANL", Character: ""},
							})
						}*/
						i += 2
						lastOpWasLoop = true
						continue
					}
				}

				// Find [.>], it's a simple puts
				if Peek(&tokens, i+1).Tok.Tok == l.OUT && Peek(&tokens, i+2).Tok.Tok == l.INCP && Peek(&tokens, i+3).Tok.Tok == l.JMPB {
					newTokens = append(newTokens, g.ParseToken{
						Pos: t.Pos,
						Tok: l.Token{Tok: l.PRNT, TokenName: "PRNT", Character: ""},
					})
					i += 3
					lastOpWasLoop = true
					continue
				}
			}

			isSimple, insts := isSimpleLoop(tokens[i+1:])
			if isSimple {
				// We found a simple loop that we can optimize away by using multiplication instead

				// If the operation is [-], just optimize it to *p = 0
				// or if the next operation is an ADD or SUB we can just set the value directly
				if insts == 1 && (Peek(&tokens, i+1).Tok.Tok == l.SUB || Peek(&tokens, i+1).Tok.Tok == l.ADD) {
					D(t, "SLO: Optimizing away zero-loop, setting resetting mem[p] directly")

					prevToken := Peek(&newTokens, len(newTokens)-1)
					if prevToken.Tok.Tok == l.ADD || prevToken.Tok.Tok == l.SUB || (prevToken.Tok.Tok == l.MOV && prevToken.Extra2 == 0) {
						// Setting p* right before this loop is not needed, as this loop just resets the value anyways
						newTokens = newTokens[:len(newTokens)-1]
					}

					newTok, isStilLoop, instsadd := setValueAfterLoop(t, tokens[i+2:])
					newTokens = append(newTokens, *newTok)
					insts += instsadd

					if !isStilLoop {
						lastOpWasLoop = false
					}

					i += insts + 1
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
						lastOpWasLoop = false
						continue mainloop
					} else if pointer == 0 && ttoken == l.SUB {
						// Count the number of decrements of p[0]
						decrementer += tt.Extra
					}
				}

				pointer = 0
				// If the decrementer is not 1, we need to divide p[0] with the decrementer to get the correct multiplier
				if decrementer != 1 && decrementer != 0 {
					D(t, "SLO: Loop with decrementer %d, adding DIV with (%d, 0)", decrementer, decrementer)
					newTokens = append(newTokens, g.ParseToken{
						Pos:    t.Pos,
						Tok:    l.Token{Tok: l.DIV, TokenName: "DIV", Character: ""},
						Extra:  decrementer,
						Extra2: 0,
					})
				}
				D(t, "SLO: Loop had %d instructions, with %d decrements of p[0] per round", insts, decrementer)

				D(t, "SLO: Add a BZ to check if the pointer is != 0, or else wi might try to assign 0 to invalid memory locations")
				newTokens = append(newTokens, g.ParseToken{
					Pos:   t.Pos,
					Tok:   l.Token{Tok: l.BZ, TokenName: "BZ", Character: ""},
					Extra: t.Extra, // We re-use the jump label
				})

				if i+1 >= len(tokens) {
					fmt.Fprintf(os.Stderr, "Parse error: Unterminated loop at %d:%d", t.Pos.Line, t.Pos.Column)
					os.Exit(1)
				}

				if i+1+insts >= len(tokens) {
					fmt.Fprintf(os.Stderr, "Internal error: Unexpected unterminated loop at %d:%d", t.Pos.Line, t.Pos.Column)
					os.Exit(1)
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
						D(tt, "SLO: INCP with %d, New P is %d", tt.Extra, pointer+tt.Extra)
						// Keep track of the pointer, this is previously optimized so .Extra holds the number of increments
						pointer += tt.Extra
					} else if ttoken == l.DECP {
						D(tt, "SLO: DECP with %d, New P is %d", tt.Extra, pointer-tt.Extra)
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

						D(tt, "SLO: Adding MUL with %d, %d", count, pointer)
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

				newTok, isStilLoop, instsadd := setValueAfterLoop(t, tokens[i+insts+1:])

				if !isStilLoop {
					lastOpWasLoop = false
				}

				// Add back a label to handle the case where the value was zero before the loop
				newTokens = append(newTokens, g.ParseToken{
					Pos:    t.Pos,
					Tok:    l.Token{Tok: l.LBL, TokenName: "LBL", Character: ""},
					Extra:  t.Extra,
					Extra2: 0,
				})

				newTokens = append(newTokens, *newTok)
				insts += instsadd

				D(t, "SLO: Loop optimized, skipping %d instructions", insts)

				// Skip over the loop tokens we just processed, and continue in the outer loop
				i += insts + 1
				continue
			}
			lastOpWasLoop = false
		} else if token == l.JMPB {
			lastOpWasLoop = true
		} else {
			lastOpWasLoop = false
		}

		newTokens = append(newTokens, t)
	}

	return newTokens
}
