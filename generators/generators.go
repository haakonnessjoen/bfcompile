package generators

import (
	l "bcomp/lexer"
	"fmt"
	"os"
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

type GeneratorOutput struct {
	file *os.File
}

func NewGeneratorOutput(filename string) *GeneratorOutput {
	return &GeneratorOutput{openFile(filename)}
}

func (g *GeneratorOutput) Printf(format string, a ...interface{}) {
	fmt.Fprintf(g.file, format, a...)
}

func (g *GeneratorOutput) Println(text string) {
	fmt.Fprintln(g.file, text)
}

func (g *GeneratorOutput) Print(text string) {
	fmt.Fprint(g.file, text)
}

func openFile(filename string) *os.File {
	if filename == "" || filename == "-" {
		return os.Stdout
	}

	// Open file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		os.Exit(1)
	}

	return file
}
