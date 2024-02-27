package generators

import (
	l "bcomp/lexer"
	"strings"
)

type ParseToken struct {
	Pos    l.Position
	Tok    l.Token
	Extra  int
	Extra2 int
}

func indent(n int) string {
	return strings.Repeat("\t", n)
}
