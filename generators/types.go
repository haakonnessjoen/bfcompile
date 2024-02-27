package generators

import (
	l "bcomp/lexer"
	"strings"
)

type ParseToken struct {
	Pos     l.Position
	Tok     l.Token
	Address int
	Extra   int
}

func indent(n int) string {
	return strings.Repeat("\t", n)
}
