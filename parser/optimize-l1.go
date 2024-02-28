package parser

import (
	g "bcomp/generators"
	l "bcomp/lexer"
)

func Optimize(tokens []g.ParseToken) []g.ParseToken {
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
